package dynaml

import (
	"crypto/rand"
	"math/big"
	"regexp"
)

const MaxUint = ^uint64(0)
const MaxInt = int64(MaxUint >> 1)

func randNumber(max int64) int64 {
	big := big.NewInt(max)
	v, _ := rand.Int(rand.Reader, big)
	return v.Int64()
}

func func_rand(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	var result interface{}
	info := DefaultInfo()

	if len(arguments) > 2 {
		return info.Error("rand takes a maximum of 2 arguments")
	}

	if len(arguments) == 0 {
		result = randNumber(MaxInt)
	} else {
		switch v := arguments[0].(type) {
		case int64:
			if len(arguments) > 1 {
				return info.Error("rand int takes only one argument")
			}
			if v < 0 {
				result = -randNumber(-v)
			} else {
				if v > 0 {
					result = randNumber(v)
				} else {
					return info.Error("zero range not possible for integer random values")
				}
			}
		case bool:
			if len(arguments) > 1 {
				return info.Error("rand bool takes only one argument")
			}
			result = randNumber(2) == 1
		case string:
			exp, err := regexp.Compile("^[" + v + "]")
			if err != nil {
				return info.Error("invalid rand character set specification %q: %s", v, err)
			}
			length := 1
			if len(arguments) == 2 {
				l, ok := arguments[1].(int64)
				if !ok {
					return info.Error("rand length must be an integer, found %s", ExpressionType(arguments[1]))
				}
				if l <= 0 {
					return info.Error("rand length must be positive, found %d", l)
				}
				length = int(l)
			}
			r := []byte{}
			var buf [4]byte
			for i := 0; i < length; {
				rand.Read(buf[:])
				if found := exp.Find(buf[:]); found != nil {
					r = append(r, found...)
					i++
				}
			}
			result = string(r)
		}
	}

	return result, info, true
}
