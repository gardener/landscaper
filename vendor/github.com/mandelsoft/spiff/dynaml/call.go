package dynaml

import (
	"fmt"
	"strings"

	"github.com/mandelsoft/spiff/yaml"

	"github.com/mandelsoft/spiff/debug"
)

type Function func(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool)

type Functions interface {
	RegisterFunction(name string, f Function)
	LookupFunction(name string) Function
}

type functionRegistry struct {
	functions map[string]Function
}

func NewFunctions() Functions {
	return &functionRegistry{map[string]Function{}}
}

func (r *functionRegistry) RegisterFunction(name string, f Function) {
	r.functions[name] = f
}

func (r *functionRegistry) LookupFunction(name string) Function {
	f := r.functions[name]
	if f != nil || r == function_registry {
		return f
	}
	return function_registry.(*functionRegistry).functions[name]
}

func RegisterFunction(name string, f Function) {
	function_registry.RegisterFunction(name, f)
}

var function_registry = NewFunctions()

type NameArgument struct {
	Name string
	Expression
}

func (a NameArgument) String() string {
	return fmt.Sprintf("%s=%s", a.Name, a.Expression)
}

type CallExpr struct {
	Function  Expression
	Arguments []Expression
	Curry     bool
}

func (e CallExpr) Evaluate(binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	resolved := true
	funcName := ""
	var value interface{}
	var info EvaluationInfo

	ref, okf := e.Function.(ReferenceExpr)
	if okf && ref.Tag == "" && len(ref.Path) == 1 && ref.Path[0] != "" && ref.Path[0] != "_" {
		funcName = ref.Path[0]
	} else {
		value, info, okf = ResolveExpressionOrPushEvaluation(&e.Function, &resolved, &info, binding, false)
		if okf && resolved {
			_, okf = value.(LambdaValue)
			if !okf {
				debug.Debug("function: no string or lambda value: %T\n", value)
				return info.Error("function call '%s' requires function name or lambda value", e.Function)
			}
		}
	}

	info.Cleanup()
	if !okf {
		debug.Debug("failed to resolve function: %s\n", info.Issue.Issue)
		return nil, info, false
	}

	cleaned := false

	var f func(binding Binding) (interface{}, EvaluationInfo, bool)
	switch funcName {
	case "defined":
		f = e.defined
	case "require":
		f = e.require
	case "valid":
		f = e.valid
	case "stub":
		f = e.stub
	case "catch":
		f = e.catch
	case "sync":
		f = e.sync
	}

	if f != nil {
		if e.Curry {
			return info.Error("no currying for intrinsic builtin function (%s)", e.Function)
		}
		return f(binding)
	}

	values, info, ok := ResolveExpressionListOrPushEvaluation(&e.Arguments, &resolved, nil, binding, false)

	if !ok {
		debug.Debug("call args failed\n")
		return nil, info, false
	}

	if !resolved {
		return e, info, true
	}

	named := map[string]yaml.Node{}
	found := -1
	for i := range e.Arguments {
		if n, ok := e.Arguments[i].(NameArgument); ok {
			named[n.Name] = NewNode(values[i], binding)
			found = i
		} else {
			break
		}
	}
	if found >= 0 {
		values = values[found+1:]
	}

	var result interface{}
	var sub EvaluationInfo

	if funcName != "" && len(named) > 0 {
		return info.Error("no named arguments for builtin function (%s)", e.Function)
	}
	if funcName != "" && e.Curry {
		params := []Parameter{Parameter{Name: "__args"}}
		args := make([]Expression, len(values)+1)
		for i, v := range values {
			args[i] = ValueExpr{v}
		}
		args[len(values)] = ListExpansionExpr{NewReferenceExpr("__args")}
		expr := CallExpr{
			Function:  NewReferenceExpr(funcName),
			Arguments: args,
		}

		return LambdaValue{params, LambdaExpr{params, true, expr}, nil, binding}, DefaultInfo(), true
	}
	switch funcName {
	case "":
		debug.Debug("calling lambda function %#v\n", value)
		resolved, result, sub, ok = value.(LambdaValue).Evaluate(false, e.Curry, true, named, values, binding, false)

	case "static_ips":
		result, sub, ok = func_static_ips(e.Arguments, binding)
		if ok && result == nil {
			resolved = false
		}

	case "join":
		result, sub, ok = func_join(values, binding)

	case "split":
		result, sub, ok = func_split(values, binding)
	case "split_match":
		result, sub, ok = func_splitMatch(values, binding)

	case "trim":
		result, sub, ok = func_trim(values, binding)

	case "length":
		result, sub, ok = func_length(values, binding)

	case "uniq":
		result, sub, ok = func_uniq(values, binding)

	case "element":
		result, sub, ok = func_element(values, binding)

	case "contains":
		result, sub, ok = func_contains(values, binding)

	case "index":
		result, sub, ok = func_index(values, binding)

	case "lastindex":
		result, sub, ok = func_lastindex(values, binding)

	case "replace":
		result, sub, ok = func_replace(values, binding)
	case "replace_match":
		result, sub, ok = func_replaceMatch(values, binding)

	case "match":
		result, sub, ok = func_match(values, binding)
	case "sort":
		result, sub, ok = func_sort(values, binding)

	case "exec":
		result, sub, ok = func_exec(true, values, binding)
		cleaned = true
	case "exec_uncached":
		result, sub, ok = func_exec(false, values, binding)
		cleaned = true
	case "pipe":
		result, sub, ok = func_pipe(true, values, binding)
		cleaned = true
	case "pipe_uncached":
		result, sub, ok = func_pipe(false, values, binding)
		cleaned = true

	case "eval":
		result, sub, ok = func_eval(values, binding, locally)

	case "env":
		result, sub, ok = func_env(values, binding)

	case "rand":
		result, sub, ok = func_rand(values, binding)

	case "read":
		result, sub, ok = func_read(true, values, binding)
		cleaned = true
	case "read_uncached":
		result, sub, ok = func_read(false, values, binding)
		cleaned = true
	case "write":
		result, sub, ok = func_write(values, binding)
		cleaned = true
	case "lookup_file":
		result, sub, ok = func_lookup(false, values, binding)
	case "lookup_dir":
		result, sub, ok = func_lookup(true, values, binding)
	case "list_files":
		result, sub, ok = func_listFiles(false, values, binding)
	case "list_dirs":
		result, sub, ok = func_listFiles(true, values, binding)
	case "tempfile":
		result, sub, ok = func_tempfile(values, binding)

	case "format":
		result, sub, ok = func_format(values, binding)

	case "error":
		result, sub, ok = func_error(values, binding)

	case "min_ip":
		result, sub, ok = func_minIP(values, binding)

	case "max_ip":
		result, sub, ok = func_maxIP(values, binding)

	case "num_ip":
		result, sub, ok = func_numIP(values, binding)

	case "contains_ip":
		result, sub, ok = func_containsIP(values, binding)

	case "makemap":
		result, sub, ok = func_makemap(values, binding)

	case "list_to_map":
		result, sub, ok = func_list_to_map(e.Arguments[0], values, binding)

	case "ipset":
		result, sub, ok = func_ipset(values, binding)

	case "merge":
		result, sub, ok = func_merge(values, binding)

	case "base64":
		result, sub, ok = func_base64(values, binding)
	case "base64_decode":
		result, sub, ok = func_base64_decode(values, binding)

	case "md5":
		result, sub, ok = func_md5(values, binding)
	case "hash":
		result, sub, ok = func_hash(values, binding)

	case "bcrypt":
		result, sub, ok = func_bcrypt(values, binding)
	case "bcrypt_check":
		result, sub, ok = func_bcrypt_check(values, binding)

	case "md5crypt":
		result, sub, ok = func_md5crypt(values, binding)
	case "md5crypt_check":
		result, sub, ok = func_md5crypt_check(values, binding)

	case "asjson":
		result, sub, ok = func_as_json(values, binding)
	case "asyaml":
		result, sub, ok = func_as_yaml(values, binding)
	case "parse":
		result, sub, ok = func_parse_yaml(values, binding)

	case "substr":
		result, sub, ok = func_substr(values, binding)
	case "lower":
		result, sub, ok = func_lower(values, binding)
	case "upper":
		result, sub, ok = func_upper(values, binding)

	case "keys":
		result, sub, ok = func_keys(values, binding)

	case "archive":
		result, sub, ok = func_archive(values, binding)

	case "validate":
		resolved, result, sub, ok = func_validate(values, binding)
	case "check":
		resolved, result, sub, ok = func_check(values, binding)

	case "type":
		if info.Undefined {
			info.Undefined = false
			return "undef", info, ok
		} else {
			result, sub, ok = func_type(values, binding)
		}

	default:
		f := binding.GetState().GetRegistry().LookupFunction(funcName)
		if f == nil {
			return info.Error("unknown function '%s'", funcName)
		}
		result, sub, ok = f(values, binding)
	}

	if cleaned {
		info.Cleanup()
	}
	if ok && (!resolved || IsExpression(result)) {
		return e, sub.Join(info), true
	}
	return result, sub.Join(info), ok
}

func (e CallExpr) String() string {
	args := make([]string, len(e.Arguments))
	for i, a := range e.Arguments {
		args[i] = fmt.Sprintf("%s", a)
	}
	curry := ""
	if e.Curry {
		curry = "*"
	}
	return fmt.Sprintf("%s%s(%s)", e.Function, curry, strings.Join(args, ", "))
}
