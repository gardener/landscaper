package wireguard

import (
	. "github.com/mandelsoft/spiff/dynaml"
)

const F_PubKey = "wgpublickey"

func init() {
	RegisterFunction(F_PubKey, func_pubkey)
}

func func_pubkey(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()

	if len(arguments) != 1 {
		return info.Error("one argument required for %q", F_PubKey)
	}

	str, ok := arguments[0].(string)
	if !ok {
		return info.Error("argument for %s must be a provate wireguard key (string)", F_PubKey)
	}
	key, err := ParseKey(str)
	if err != nil {
		return info.Error("error parsing key %q: %s", str, err)
	}
	return key.PublicKey().String(), info, true
}
