package dynaml

import (
	"fmt"
	"github.com/mandelsoft/spiff/yaml"
	"net"
	"regexp"
	"strings"
)

type Validator func(value interface{}, binding Binding, args ...interface{}) (bool, string, error, bool)

var validators = map[string]Validator{}

func RegisterValidator(name string, f Validator) {
	validators[name] = f
}

func func_validate(arguments []interface{}, binding Binding) (bool, interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()
	if len(arguments) < 2 {
		info.Error("at least two arguments required for validate")
		return true, nil, info, false
	}

	value := arguments[0]

	for i, c := range arguments[1:] {
		r, m, err, ok := EvalValidationExpression(value, NewNode(c, binding), binding)
		if err != nil {
			info.SetError("condition %d has problem: %s", i+1, err)
			return true, nil, info, false
		}
		if !ok {
			return false, nil, info, true
		}
		if !r {
			info.SetError("condition %d failed: %s", i+1, m)
			return true, nil, info, false
		}
	}
	return true, value, info, true
}

func func_check(arguments []interface{}, binding Binding) (bool, interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()
	if len(arguments) < 2 {
		info.Error("at least two arguments required for check")
		return true, nil, info, false
	}

	value := arguments[0]

	for i, c := range arguments[1:] {
		r, _, err, ok := EvalValidationExpression(value, NewNode(c, binding), binding)
		if err != nil {
			info.SetError("condition %d has problem: %s", i+1, err)
			return true, nil, info, false
		}
		if !ok {
			return false, nil, info, true
		}
		if !r {
			return true, false, info, true
		}
	}
	return true, true, info, true
}

// first result: validation successful
// second:       message
// third:        condition error
// fourth:       expression already resolved

func EvalValidationExpression(value interface{}, cond yaml.Node, binding Binding) (bool, string, error, bool) {
	if cond == nil || cond.Value() == nil {
		return ValidatorResult(true, "no condition")
	}
	switch v := cond.Value().(type) {
	case string:
		return _validate(value, v, binding)
	case LambdaValue:
		return _validate(value, v, binding)
	case []yaml.Node:
		if len(v) == 0 {
			return ValidatorErrorf("validation type missing")
		}
		return _validate(value, v[0].Value(), binding, v[1:]...)
	default:
		return ValidatorErrorf("invalid validation check type: %s", ExpressionType(v))
	}
}

func _validate(value interface{}, cond interface{}, binding Binding, args ...yaml.Node) (bool, string, error, bool) {
	var err error
	switch v := cond.(type) {
	case LambdaValue:
		if len(v.lambda.Parameters) != len(args)+1 {
			return ValidatorErrorf("argument count mismatch for lambda %s: expected %d, found %d", v, len(v.lambda.Parameters), len(args)+1)
		}
		vargs := []interface{}{value}
		for _, a := range args {
			vargs = append(vargs, a.Value())
		}
		valid, r, info, ok := v.Evaluate(false, false, false, nil, vargs, binding, false)

		if !valid {
			if !ok {
				err = fmt.Errorf("%s", info.Issue.Issue)
			}
			return false, "", err, false
		}
		l, ok := r.([]yaml.Node)
		if ok {
			switch len(l) {
			case 1:
				r = l[0].Value()
				break
			case 2:
				t, err := StringValue("lambda validator", l[1].Value())
				if err != nil {
					return ValidatorErrorf("lambda validator result index %d: %s", 2, err)
				}
				return toBool(l[0].Value()), t, nil, true
			case 3:
				t, err := StringValue("lambda validator", l[1].Value())
				if err != nil {
					return ValidatorErrorf("lambda validator result index %d: %s", 2, err)
				}
				f, err := StringValue("lambda validator", l[2].Value())
				if err != nil {
					return ValidatorErrorf("lambda validator result index %d: %s", 3, err)
				}
				return SimpleValidatorResult(toBool(l[0].Value()), t, f)
			default:
				return ValidatorErrorf("invalid result length of validator %s, got %d", v, len(l))
			}
		}
		return SimpleValidatorResult(toBool(r), fmt.Sprintf("%s succeeded", v), "%s failed", v)
	case string:
		not := strings.HasPrefix(v, "!")
		if not {
			v = v[1:]
		} else {
			if v == "" {
				return ValidatorErrorf("empty validator type")
			}
		}
		if v == "not" {
			not = !not
		}
		r, m, err, resolved := handleStringType(value, v, binding, args...)
		if !resolved || err != nil {
			return false, "", err, resolved
		}
		if not {
			return !r, m, err, resolved
		} else {
			return r, m, err, resolved
		}
	default:
		return ValidatorErrorf("unexpected validation type %q", ExpressionType(v))
	}
}

func handleStringType(value interface{}, op string, binding Binding, args ...yaml.Node) (bool, string, error, bool) {
	reason := "("
	optional := false
	switch op {
	case "list":
		l, ok := value.([]yaml.Node)
		if !ok {
			return ValidatorResult(false, "is no list")
		}
		if len(args) == 0 {
			return ValidatorResult(true, "is a list")
		}
		for i, e := range l {
			for j, c := range args {
				r, m, err, valid := EvalValidationExpression(e.Value(), c, binding)
				if err != nil {
					return ValidatorErrorf("list entry %d condition %d: %s", i, j, err)
				}
				if !valid {
					return false, "", nil, false
				}
				if !r {
					return ValidatorResult(false, "entry %d condition %d %s", i, j, m)
				}
			}
		}
		return ValidatorResult(true, "all entries match all conditions")

	case "map":
		l, ok := value.(map[string]yaml.Node)
		if !ok {
			return ValidatorResult(false, "is no map")
		}
		if len(args) == 0 {
			return ValidatorResult(true, "is a map")
		}
		var ck yaml.Node
		if len(args) > 2 {
			return ValidatorErrorf("map validator takes a maximum of two arguments, got %d", len(args))
		}
		if len(args) == 2 {
			ck = args[0]
		}
		ce := args[len(args)-1]

		for k, e := range l {
			if ck != nil {
				r, m, err, valid := EvalValidationExpression(k, ck, binding)
				if err != nil {
					return ValidatorErrorf("map key %q %s", k, err)
				}
				if !valid {
					return false, "", nil, false
				}
				if !r {
					return ValidatorResult(false, "map key %q %s", k, m)
				}
			}

			r, m, err, valid := EvalValidationExpression(e.Value(), ce, binding)
			if err != nil {
				return ValidatorErrorf("map entry %q: %s", k, err)
			}
			if !valid {
				return false, "", nil, false
			}
			if !r {
				return ValidatorResult(false, "map entry %q %s", k, m)
			}
		}
		return ValidatorResult(true, "all map entries and keys match")

	case "optionalfield":
		optional = true
		fallthrough
	case "mapfield":
		l, ok := value.(map[string]yaml.Node)
		if !ok {
			return ValidatorResult(false, "is no map")
		}
		if len(args) < 1 || len(args) > 2 {
			return ValidatorErrorf("%s reqires one or two arguments", op)
		}
		field, err := StringValue(op, args[0].Value())
		if err != nil {
			return ValidatorErrorf("field name must be string")
		}
		val, ok := l[field]
		if !ok {
			if optional {
				return ValidatorResult(true, "has no optional field %q", field)
			}
			return ValidatorResult(false, "has no field %q", field)
		}
		if len(args) == 2 {
			r, m, err, valid := EvalValidationExpression(val.Value(), args[1], binding)
			if err != nil {
				return ValidatorErrorf("map entry %q %s", field, err)
			}
			if !valid {
				return false, "", nil, false
			}
			return ValidatorResult(r, "map entry %q %s", field, m)
		}
		return ValidatorResult(true, "map entry %q exists", field)

	case "and", "not", "":
		if len(args) == 0 {
			return ValidatorErrorf("validator argument required")
		}
		for _, c := range args {
			r, m, err, resolved := EvalValidationExpression(value, c, binding)
			if err != nil || !resolved {
				return false, "", err, resolved
			}
			if reason != "(" {
				reason += " and "
			}
			reason += m
			if !r {
				return ValidatorResult(false, m)
			}
		}
		if len(args) == 1 {
			return ValidatorResult(true, reason[1:])
		}
		return ValidatorResult(true, reason+")")
	case "or":
		if len(args) == 0 {
			return ValidatorErrorf("validator argument required")
		}
		for _, c := range args {
			r, m, err, resolved := EvalValidationExpression(value, c, binding)
			if err != nil || !resolved {
				return false, "", err, resolved
			}
			if reason != "(" {
				reason += " and "
			}
			reason += m
			if r {
				return ValidatorResult(true, m)
			}
		}
		if len(args) == 1 {
			return ValidatorResult(true, reason[1:])
		}
		return ValidatorResult(false, reason+")")

	case "empty":
		switch v := value.(type) {
		case string:
			return SimpleValidatorResult(v == "", "is empty", "is not empty")
		case []yaml.Node:
			return SimpleValidatorResult(len(v) == 0, "is empty", "is not empty")
		case map[string]yaml.Node:
			return SimpleValidatorResult(len(v) == 0, "is empty", "is not empty")
		default:
			return ValidatorErrorf("invalid type for empty: %s", ExpressionType(v))
		}
	case "valueset":
		if len(args) != 1 {
			return ValidatorErrorf("valueset requires a list argument with possible values")
		}
		l, ok := args[0].Value().([]yaml.Node)
		if !ok {
			return ValidatorErrorf("valueset requires a list argument with possible values")
		}
		for _, v := range l {
			if ok, _, _ := compareEquals(value, v.Value()); ok {
				return ValidatorResult(true, "matches valueset")
			}
		}
		s, ok := value.(string)
		if ok {
			return ValidatorResult(false, "invalid value %q", s)
		}
		i, ok := value.(int64)
		if ok {
			return ValidatorResult(false, "invalid value %d", i)
		}
		return ValidatorResult(false, "invalid value")

	case "value", "=":
		if len(args) != 1 {
			return ValidatorErrorf("value requires a value argument")
		}
		s := args[0].Value()

		sv, isStr := value.(string)
		if ok, _, _ := compareEquals(s, value); !ok {
			if isStr {
				return ValidatorResult(false, "invalid value %q", sv)
			}
			return ValidatorResult(false, "invalid value")
		}
		if isStr {
			return ValidatorResult(true, "valid value %q", sv)
		}
		return ValidatorResult(true, "valid value")

	case "gt", ">":
		if len(args) != 1 {
			return ValidatorErrorf("gt requires a value argument")
		}
		s := args[0].Value()

		r := false
		switch cv := s.(type) {
		case string:
			s, ok := value.(string)
			if !ok {
				return ValidatorErrorf("must be string to compare to string")
			}
			r = strings.Compare(s, cv) > 0
		case int64:
			s, ok := value.(int64)
			if !ok {
				s, ok := value.(float64)
				if !ok {
					return ValidatorErrorf("must be number to compare to number")
				}
				r = s > float64(cv)
			} else {
				r = s > cv
			}
		case float64:
			s, ok := value.(int64)
			if !ok {
				s, ok := value.(float64)
				if !ok {
					return ValidatorErrorf("must be number to compare to number")
				}
				r = s > cv
			} else {
				r = float64(s) > cv
			}
		default:
			return ValidatorErrorf("invalid type %T", s)
		}
		if !r {
			return ValidatorResult(false, "less or equal to %v", s)
		}
		return ValidatorResult(true, "greater than %v", s)
	case "lt", "<":
		if len(args) != 1 {
			return ValidatorErrorf("lt requires a value argument")
		}
		s := args[0].Value()

		r := false
		switch cv := s.(type) {
		case string:
			s, ok := value.(string)
			if !ok {
				return ValidatorErrorf("must be string to compare to string")
			}
			r = strings.Compare(s, cv) < 0
		case int64:
			s, ok := value.(int64)
			if !ok {
				s, ok := value.(float64)
				if !ok {
					return ValidatorErrorf("must be number to compare to number")
				}
				r = s < float64(cv)
			} else {
				r = s < cv
			}
		case float64:
			s, ok := value.(int64)
			if !ok {
				s, ok := value.(float64)
				if !ok {
					return ValidatorErrorf("must be number to compare to number")
				}
				r = s < cv
			} else {
				r = float64(s) < cv
			}
		default:
			return ValidatorErrorf("invalid type %T", s)
		}
		if !r {
			return ValidatorResult(false, "greater than %v", s)
		}
		return ValidatorResult(true, "less or equal to %v", s)
	case "ge", ">=":
		if len(args) != 1 {
			return ValidatorErrorf("ge requires a value argument")
		}
		s := args[0].Value()

		r := false
		switch cv := s.(type) {
		case string:
			s, ok := value.(string)
			if !ok {
				return ValidatorErrorf("must be string to compare to string")
			}
			r = strings.Compare(s, cv) >= 0
		case int64:
			s, ok := value.(int64)
			if !ok {
				s, ok := value.(float64)
				if !ok {
					return ValidatorErrorf("must be number to compare to number")
				}
				r = s >= float64(cv)
			} else {
				r = s >= cv
			}
		case float64:
			s, ok := value.(int64)
			if !ok {
				s, ok := value.(float64)
				if !ok {
					return ValidatorErrorf("must be number to compare to number")
				}
				r = s >= cv
			} else {
				r = float64(s) >= cv
			}
		default:
			return ValidatorErrorf("invalid type %T", s)
		}
		if !r {
			return ValidatorResult(false, "less than %v", s)
		}
		return ValidatorResult(true, "greater or equal to %v", s)
	case "le", "<=":
		if len(args) != 1 {
			return ValidatorErrorf("lt requires a value argument")
		}
		s := args[0].Value()

		r := false
		switch cv := s.(type) {
		case string:
			s, ok := value.(string)
			if !ok {
				return ValidatorErrorf("must be string to compare to string")
			}
			r = strings.Compare(s, cv) <= 0
		case int64:
			s, ok := value.(int64)
			if !ok {
				s, ok := value.(float64)
				if !ok {
					return ValidatorErrorf("must be number to compare to number")
				}
				r = s <= float64(cv)
			} else {
				r = s <= cv
			}
		case float64:
			s, ok := value.(int64)
			if !ok {
				s, ok := value.(float64)
				if !ok {
					return ValidatorErrorf("must be number to compare to number")
				}
				r = s <= cv
			} else {
				r = float64(s) <= cv
			}
		default:
			return ValidatorErrorf("invalid type %T", s)
		}
		if !r {
			return ValidatorResult(false, "greater or equal to %v", s)
		}
		return ValidatorResult(true, "less than %v", s)

	case "match", "~=":
		if len(args) != 1 {
			return ValidatorErrorf("match requires a regexp argument")
		}
		s, ok := args[0].Value().(string)
		if !ok {
			return ValidatorErrorf("match requires a regexp argument")
		}

		re, err := regexp.Compile(s)
		if err != nil {
			return ValidatorErrorf("regexp %s: %s", s, err)
		}
		s, ok = value.(string)
		if !ok {
			return ValidatorErrorf("no string to match regexp")
		}
		if !re.MatchString(s) {
			return ValidatorResult(false, "invalid value %q", s)
		}
		return ValidatorResult(true, "valid value %q", s)

	case "type":
		e := ExpressionType(value)
		for _, t := range args {
			s, err := StringValue("type arg", t.Value())
			if err != nil {
				return ValidatorErrorf("%s: %s", op, err)
			}
			if s == e {
				return ValidatorResult(true, "is of type %s", s)
			}
			if reason != "(" {
				reason += " and "
			}
			reason += fmt.Sprintf("is not of type %s", s)
		}
		if len(args) == 1 {
			return ValidatorResult(false, reason[1:])
		}
		return ValidatorResult(false, reason+")")

	case "dnsname":
		s, err := StringValue(op, value)
		if err != nil {
			return ValidatorErrorf("%s: %s", op, err)
		}
		if r := IsWildcardDNS1123Subdomain(s); r != nil {
			if r := IsDNS1123Subdomain(s); r != nil {
				return ValidatorResult(false, "is no dns name: %s", r)
			}
		}
		return ValidatorResult(true, "is dns name")
	case "dnslabel":
		s, err := StringValue(op, value)
		if err != nil {
			return ValidatorErrorf("%s: %s", op, err)
		}
		r := IsDNS1123Label(s)
		return SimpleValidatorResult(r == nil, "is dns label", "is no dns label: %s", r)
	case "dnsdomain":
		s, err := StringValue(op, value)
		if err != nil {
			return ValidatorErrorf("%s: %s", op, err)
		}
		r := IsDNS1123Subdomain(s)
		return SimpleValidatorResult(r == nil, "is dns domain", "is no dns domain: %s", r)
	case "wildcarddnsdomain":
		s, err := StringValue(op, value)
		if err != nil {
			return ValidatorErrorf("%s: %s", op, err)
		}
		r := IsWildcardDNS1123Subdomain(s)
		return SimpleValidatorResult(r == nil, "is wildcard dns domain", "is no wildcard dns domain: %s", r)
	case "ip":
		s, err := StringValue(op, value)
		if err != nil {
			return ValidatorErrorf("%s: %s", op, err)
		}
		ip := net.ParseIP(s)
		return SimpleValidatorResult(ip != nil, "is ip address", "is no ip address: %s", s)
	case "cidr":
		s, err := StringValue(op, value)
		if err != nil {
			return ValidatorErrorf("%s: %s", op, err)
		}
		_, _, err = net.ParseCIDR(s)
		return SimpleValidatorResult(err == nil, "is CIDR", "is no CIDR: %s", err)
	default:
		v := validators[op]
		if v != nil {
			vargs := []interface{}{}
			for _, a := range args {
				vargs = append(vargs, a.Value())
			}
			return v(value, binding, vargs...)
		}
		return ValidatorErrorf("unknown validation operator %q", op)
	}
}

func StringValue(msg string, v interface{}) (string, error) {
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%s requires string, but got %s", msg, ExpressionType(v))
	}
	return s, nil
}

func ValidatorErrorf(msgfmt string, args ...interface{}) (bool, string, error, bool) {
	return false, "", fmt.Errorf(msgfmt, args...), true
}

func ValidatorResult(r bool, msgfmt string, args ...interface{}) (bool, string, error, bool) {
	return r, fmt.Sprintf(msgfmt, args...), nil, true
}

func SimpleValidatorResult(r bool, t, f string, args ...interface{}) (bool, string, error, bool) {
	if r {
		return ValidatorResult(r, t)
	}
	return ValidatorResult(r, fmt.Sprintf(f, args...))
}
