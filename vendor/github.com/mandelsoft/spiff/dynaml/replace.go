package dynaml

import (
	"bytes"
	"fmt"
	"github.com/mandelsoft/spiff/yaml"
	"regexp"
	"strings"
	"unicode/utf8"
)

func func_replace(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	return _replace("replace", ReplaceString, arguments, binding)
}

func ReplaceString(str string, src string, dst interface{}, cnt int, binding Binding) (bool, string, error) {
	templ, ok := dst.(string)
	if ok {
		return true, strings.Replace(str, src, templ, cnt), nil
	}
	lambda, ok := dst.(LambdaValue)
	if !ok {
		return false, "", fmt.Errorf("replace substitution must be string or lambda")
	}
	expand := LambdaExpander(lambda, binding)

	return processReplace(str, StringFinder(src), expand, cnt)
}

///////////////////////////////////////////////////////////////////////////////

func func_replaceMatch(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	return _replace("replace_match", ReplaceRegExp, arguments, binding)
}

func ReplaceRegExp(str string, src string, dst interface{}, cnt int, binding Binding) (bool, string, error) {
	var expand Expander
	exp, err := regexp.Compile(src)
	if err != nil {
		return false, "", err
	}

	templ, ok := dst.(string)
	if ok {
		// fmt.Printf("%s -> %s\n", str, exp.ReplaceAllString(str, templ))
		expand = RegExpExpander(exp, []byte(templ))
	} else {
		lambda, ok := dst.(LambdaValue)
		if ok {
			expand = LambdaExpander(lambda, binding)
		} else {
			return false, "", fmt.Errorf("replace substitution must be string or lambda")
		}
	}

	return processReplace(str, RegExpFinder(exp), expand, cnt)
}

///////////////////////////////////////////////////////////////////////////////

type Expander func(dst []byte, src []byte, match []int) (bool, []byte, error)
type Finder func(src []byte) []int

func StringFinder(str string) Finder {
	pat := []byte(str)
	return func(src []byte) []int {
		index := bytes.Index(src, pat)
		if index < 0 {
			return nil
		}
		return []int{index, index + len(pat)}
	}
}

func RegExpFinder(exp *regexp.Regexp) Finder {
	return func(src []byte) []int {
		return exp.FindSubmatchIndex(src)
	}
}

func RegExpExpander(exp *regexp.Regexp, templ []byte) Expander {
	return func(dst []byte, src []byte, match []int) (bool, []byte, error) {
		return true, exp.Expand(dst, templ, src, match), nil
	}
}

func LambdaExpander(lambda LambdaValue, binding Binding) Expander {
	return func(dst []byte, src []byte, match []int) (bool, []byte, error) {
		matches := []yaml.Node{}
		for i := 0; i < len(match); i += 2 {
			matches = append(matches, NewNode(string(src[match[i]:match[i+1]]), binding))
		}
		inp := []interface{}{matches}
		resolved, v, info, ok := lambda.Evaluate(false, false, false, nil, inp, binding, false)
		if !ok {
			return resolved, nil, fmt.Errorf("replace: %s", info.Issue.Issue)
		}
		if !resolved {
			return false, nil, nil
		}
		str, ok := v.(string)
		if !ok {
			return false, nil, fmt.Errorf("replace: lambda must return a string")
		}
		return resolved, append(dst, []byte(str)...), nil
	}
}

func processReplace(str string, find Finder, expand Expander, cnt int) (bool, string, error) {
	b := []byte(str)
	n := []byte{}
	emptyMatch := true

	for cnt < 0 || cnt > 0 {
		loc := find(b)
		if len(loc) == 0 {
			break
		}
		if cnt > 0 {
			cnt--
		}
		n = append(n, b[0:loc[0]]...)

		if emptyMatch || loc[1] > 0 {
			resolved, m, err := expand(n, b, loc)
			if !resolved {
				return false, "", err
			}
			n = m
		}
		b = b[loc[1]:]

		// Advance past this match; always advance at least one character.
		emptyMatch = loc[1] == loc[0]
		if emptyMatch {
			_, width := utf8.DecodeRune(b)
			if width == 0 {
				break
			}
			n = append(n, b[:width]...)
			b = b[width:]
		}
	}
	r := string(append(n, b...))
	return true, r, nil
}

func _replace(name string, replace func(string, string, interface{}, int, Binding) (bool, string, error), arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) < 3 {
		return info.Error("%s requires at least 3 arguments", name)
	}
	if len(arguments) > 4 {
		return info.Error("%s does not take more than 4 arguments", name)
	}

	str, ok := arguments[0].(string)
	if !ok {
		return info.Error("first argument for %s must be a string", name)
	}
	src, ok := arguments[1].(string)
	if !ok {
		return info.Error("second argument for %s must be a string", name)
	}

	n := int64(-1)
	if len(arguments) > 3 {
		n, ok = arguments[3].(int64)
		if !ok {
			return info.Error("fourth argument for %s must be an integer", name)
		}
	}

	resolved, e, err := replace(str, src, arguments[2], int(n), binding)
	if err != nil {
		return info.Error("%s: %s", name, err)
	}
	if !resolved {
		return nil, info, false
	}
	return e, info, true
}
