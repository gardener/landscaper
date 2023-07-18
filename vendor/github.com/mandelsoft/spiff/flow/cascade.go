package flow

import (
	"github.com/mandelsoft/spiff/dynaml"
	"github.com/mandelsoft/spiff/yaml"
)

type Options struct {
	PreserveEscapes    bool
	PreserveTemporaray bool
	Partial            bool
}

func PrepareStubs(outer dynaml.Binding, partial bool, stubs ...yaml.Node) ([]yaml.Node, error) {
	for i := len(stubs) - 1; i >= 0; i-- {
		flowed, err := NestedFlow(outer, stubs[i], stubs[i+1:]...)
		if !partial && err != nil {
			return nil, err
		}

		stubs[i] = Cleanup(flowed, discardLocal)
	}
	return stubs, nil
}

func Apply(outer dynaml.Binding, template yaml.Node, prepared []yaml.Node, opts Options) (yaml.Node, error) {
	result, err := NestedFlow(outer, template, prepared...)
	if err == nil {
		if !opts.PreserveTemporaray {
			result = Cleanup(result, discardTemporary)
		}
		if !opts.PreserveEscapes {
			result = Cleanup(result, unescapeDynaml)
		}
	}
	return result, err
}

func Cascade(outer dynaml.Binding, template yaml.Node, opts Options, stubs ...yaml.Node) (yaml.Node, error) {
	prepared, err := PrepareStubs(outer, opts.Partial, stubs...)
	if err != nil {
		return nil, err
	}

	return Apply(outer, template, prepared, opts)
}

func discardTemporary(node yaml.Node) (yaml.Node, CleanupFunction) {
	if node.Temporary() || node.Local() {
		return nil, discardTemporary
	}
	return node, discardTemporary
}

func unescapeDynaml(node yaml.Node) (yaml.Node, CleanupFunction) {
	return yaml.UnescapeDynaml(node), unescapeDynaml
}

func discardLocal(node yaml.Node) (yaml.Node, CleanupFunction) {
	if node.Local() {
		return nil, discardLocal
	}
	return node, discardLocal
}

func keepAll(node yaml.Node) (yaml.Node, CleanupFunction) {
	return node, keepAll
}

func DiscardNonState(node yaml.Node) (yaml.Node, CleanupFunction) {
	if node.State() {
		return node, keepAll
	}
	return nil, DiscardNonState
}

type CleanupFunction func(yaml.Node) (yaml.Node, CleanupFunction)

func Cleanup(node yaml.Node, test CleanupFunction) yaml.Node {
	if node == nil {
		return nil
	}
	value := node.Value()
	switch v := value.(type) {
	case []yaml.Node:
		r := []yaml.Node{}
		for _, e := range v {
			if n, t := test(e); n != nil {
				r = append(r, Cleanup(n, t))
			}
		}
		value = r

	case map[string]yaml.Node:
		r := map[string]yaml.Node{}
		for k, e := range v {
			if n, t := test(e); n != nil {
				r[k] = Cleanup(n, t)
			}
		}
		value = r
	}
	return yaml.ReplaceValue(value, node)
}

func DetermineState(node yaml.Node) yaml.Node {
	return Cleanup(node, DiscardNonState)
}
