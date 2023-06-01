package dynaml

import (
	"math"
	"reflect"
	"runtime"
	"strings"
)

var float_functions = []func(float64) float64{
	math.Floor, math.Ceil, math.Round, math.RoundToEven,
	math.Abs,
	math.Sin, math.Cos, math.Sinh, math.Cosh,
	math.Asin, math.Acos, math.Asinh, math.Acosh,
	math.Sqrt, math.Exp, math.Log, math.Log10,
}

func init() {
	for _, f := range float_functions {
		s := strings.Split(runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name(), ".")
		n := strings.ToLower(s[len(s)-1])
		f := f
		RegisterFunction(n, func(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
			return _float(n, f, arguments)
		})
	}
}

func _float(name string, f func(float64) float64, arguments []interface{}) (val interface{}, info EvaluationInfo, ok bool) {
	info = DefaultInfo()
	defer func() {
		if r := recover(); r != nil {
			val, info, ok = info.Error("%s", r)
		}
	}()
	if len(arguments) != 1 {
		return info.Error("%s requires one argument", name)
	}
	if name == "ceil" || name == "floor" || name == "round" || name == "roundtoeven" {
		switch v := arguments[0].(type) {
		case int64:
			return v, info, true
		case float64:
			return int64(f(v)), info, true
		case bool:
			if v {
				return 1, info, true
			}
			return 0, info, true
		default:
			return info.Error("invalid argument type for %s: %T", name, v)
		}
	} else {
		if name == "abs" {
			switch v := arguments[0].(type) {
			case int64:
				if v < 0 {
					return -v, info, true
				}
				return v, info, true
			}
		}
		var r float64
		switch v := arguments[0].(type) {
		case int64:
			r = f(float64(v))
		case float64:
			r = f(v)
		default:
			return info.Error("invalid argument type for %s: %T", name, v)
		}
		if math.IsNaN(r) {
			return info.Error("%s: NaN", name)
		}
		return r, info, true
	}
}
