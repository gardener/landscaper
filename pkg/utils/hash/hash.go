package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"reflect"
	"sort"
)

// EXPERIMENTAL !
func ComputeHash(obj interface{}) (string, error) {
	hasher := newHasher()

	if err := hasher.addInterface(obj); err != nil {
		return "", err
	}

	resultBytes := hasher.hash.Sum(nil)
	return hex.EncodeToString(resultBytes), nil
}

func newHasher() *hasher {
	return &hasher{
		hash: sha256.New(),
	}
}

type hasher struct {
	hash hash.Hash
}

func (h *hasher) addInterface(obj interface{}) error {
	if obj == nil {
		_, err := fmt.Fprint(h.hash, reflect.ValueOf(nil))
		if err != nil {
			return err
		}

		return nil
	}

	value := reflect.Indirect(reflect.ValueOf(obj))
	return h.addAnyValue(value)
}

func (h *hasher) addAnyValue(val reflect.Value) error {
	//val := reflect.Indirect(reflect.ValueOf(value))
	if !val.IsValid() || val.IsZero() {
		return nil
	}

	kind := val.Type().Kind()
	switch kind {
	case reflect.Struct, reflect.Ptr:
		if err := h.addStructure(val); err != nil {
			return err
		}

	case reflect.Slice, reflect.Array:
		if err := h.addArray(val); err != nil {
			return err
		}

	case reflect.Map:
		if err := h.addMap(val); err != nil {
			return err
		}

	default:
		if err := h.addElementaryValue(val); err != nil {
			return err
		}

	}

	return nil
}

func (h *hasher) addElementaryValue(value reflect.Value) error {
	_, err := fmt.Fprint(h.hash, reflect.ValueOf(value).Interface())
	return err
}

func (h *hasher) addStructure(value reflect.Value) error {
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)

		ignore := value.Type().Field(i).Tag.Get("hash") == "ignore"
		if field.IsZero() || !field.IsValid() || ignore {
			continue
		}

		var valOf interface{}
		// check if field of struct is unexported
		if reflect.Indirect(field).CanInterface() {
			valOf = reflect.Indirect(field).Interface()
		} else {
			return nil
		}

		if err := h.addInterface(valOf); err != nil {
			return err
		}
	}

	return nil
}

func (h *hasher) addArray(value reflect.Value) error {
	for i := 0; i < value.Len(); i++ {
		item := reflect.Indirect(value.Index(i)).Interface()

		itemHash, err := ComputeHash(item)
		if err != nil {
			return err
		}

		if err := h.addInterface(itemHash); err != nil {
			return err
		}
	}

	return nil
}

func (h *hasher) addMap(value reflect.Value) error {
	// sort key-value pairs based on hash string of each key
	keyHash := make([]string, len(value.MapKeys()))
	keyHashValue := make(map[string]reflect.Value)

	for i, key := range value.MapKeys() {
		kh, err := ComputeHash(key.Interface())
		if err != nil {
			return err
		}
		keyHash[i] = kh
		keyHashValue[kh] = value.MapIndex(key)
	}
	sort.Strings(keyHash)

	for _, kh := range keyHash {
		_, err := fmt.Fprint(h.hash, kh)
		if err != nil {
			return err
		}
		vh, err := ComputeHash(keyHashValue[kh].Interface())
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(h.hash, vh)
		if err != nil {
			return err
		}
	}

	return nil
}
