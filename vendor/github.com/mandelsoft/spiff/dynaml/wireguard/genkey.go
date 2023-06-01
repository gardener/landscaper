package wireguard

import (
	. "github.com/mandelsoft/spiff/dynaml"
)

const F_GenKey = "wggenkey"

func init() {
	RegisterFunction(F_GenKey, func_genkey)
}

func func_genkey(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) > 1 {
		return info.Error("a maximum of one argument expected for %s", F_GenKey)
	}
	ktype := "private"
	if len(arguments) == 1 {
		str, ok := arguments[0].(string)
		if !ok {
			return info.Error("argument for %s must be a string (private or preshared)", F_GenKey)
		}
		ktype = str
	}
	var key Key
	var err error
	switch ktype {
	case "private":
		key, err = GeneratePrivateKey()
	case "preshared":
		key, err = GenerateKey()
	default:
		return info.Error("invalid key type %q, use private or preshared", ktype)
	}
	if err != nil {
		return info.Error("error generating key: %s", err)
	}
	return key.String(), info, true
}
