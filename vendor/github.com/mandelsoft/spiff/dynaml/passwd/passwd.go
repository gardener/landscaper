package passwd

import (
	"fmt"

	. "github.com/mandelsoft/spiff/dynaml"
	"github.com/mandelsoft/spiff/legacy/candiedyaml"
)

type Encoding interface {
	Encode(text string, key string) (string, error)
	Decode(text string, key string) (string, error)
	Name() string
}

var encodings = map[string]Encoding{
	TRIPPLEDES: des1{},
}

const F_Decrypt = "decrypt"
const F_Encrypt = "encrypt"

func init() {
	RegisterFunction(F_Decrypt, func_decrypt)
	RegisterFunction(F_Encrypt, func_encrypt)
}

func RegisterEncryption(name string, e Encoding) {
	encodings[name] = e
}

func GetEncoding(name string) Encoding {
	return encodings[name]
}

func func_decrypt(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()
	if len(arguments) < 1 || len(arguments) > 3 {
		return info.Error("%s expects one, two or three arguments", F_Decrypt)
	}

	value, err := StringValue(F_Decrypt, arguments[0])
	if err != nil {
		return info.Error(err)
	}

	key := binding.GetState().GetEncryptionKey()
	method := TRIPPLEDES
	v := ""
	if len(arguments) > 1 {
		v, err = StringValue(fmt.Sprintf("%s: 2nd argument", F_Decrypt), arguments[1])
		if err != nil {
			return info.Error(err)
		}
	}

	switch len(arguments) {
	case 2:
		if encodings[v] != nil {
			method = v
		} else {
			key = v
		}
	case 3:
		m, err := StringValue(fmt.Sprintf("%s: method", F_Decrypt), arguments[2])
		if err != nil {
			return info.Error(err)
		}
		method = m
	}

	e := encodings[method]
	if e == nil {
		return info.Error("invalid encyption method %q", method)
	}

	if key == "" {
		return info.Error("invalid empty encyption key")
	}
	result, err := e.Decode(value, key)
	if err != nil {
		return info.Error(err)
	}
	return ParseData("<decrypt>", []byte(result), "import", binding)

}

func func_encrypt(arguments []interface{}, binding Binding) (interface{}, EvaluationInfo, bool) {
	info := DefaultInfo()
	if len(arguments) < 1 || len(arguments) > 3 {
		return info.Error("%s expects one, two or three arguments", F_Encrypt)
	}

	value, err := candiedyaml.Marshal(arguments[0])
	if err != nil {
		return info.Error(err)
	}

	key := binding.GetState().GetEncryptionKey()
	method := TRIPPLEDES
	v := ""
	if len(arguments) > 1 {
		v, err = StringValue(fmt.Sprintf("%s: 2nd argument", F_Encrypt), arguments[1])
		if err != nil {
			return info.Error(err)
		}
	}

	switch len(arguments) {
	case 2:
		if GetEncoding(v) != nil {
			method = v
		} else {
			key = v
		}
	case 3:
		m, err := StringValue(fmt.Sprintf("%s: method", F_Encrypt), arguments[2])
		if err != nil {
			return info.Error(err)
		}
		method = m
	}

	e := GetEncoding(method)
	if e == nil {
		return info.Error("invalid encyption method %q", method)
	}

	if key == "" {
		return info.Error("invalid empty encyption key")
	}
	result, err := e.Encode(string(value), key)
	if err != nil {
		return info.Error(err)
	}
	return result, info, true
}
