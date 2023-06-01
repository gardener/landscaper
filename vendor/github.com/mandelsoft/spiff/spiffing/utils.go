package spiffing

import (
	"fmt"
	"strings"

	"github.com/mandelsoft/spiff/yaml"
)

// Process just processes a template with the values set in the execution
// context. It directly takes and delivers byte array containing yaml data.
func Process(s Spiff, template Source) ([]byte, error) {
	templ, err := s.UnmarshalSource(template)
	if err != nil {
		return nil, err
	}
	result, err := s.Cascade(templ, nil)
	if err != nil {
		return nil, err
	}
	return s.Marshal(result)
}

// ProcessFile just processes a template give by a file with the values set in
// the execution context.
// The path name of the file is interpreted in the context of the filesystem
// found in the execution context, which is defaulted by the OS filesystem.
func ProcessFile(s Spiff, path string) ([]byte, error) {
	return Process(s, s.FileSource(path))
}

// EvaluateDynamlExpression just processes a plain dynaml expression with the values set in
// the execution context.
func EvaluateDynamlExpression(s Spiff, expr string) ([]byte, error) {
	r, err := Process(s, NewSourceData("dynaml", []byte("(( "+expr+" ))")))
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(r), "\n")
	if len(lines) == 2 && lines[1] == "" {
		return []byte(lines[0]), nil
	}
	return r, nil
}

// Cascade processes a template source with a list of stub sources and optional state and
// devivers the cascading results and the new state as yaml data
func Cascade(s Spiff, template Source, stubs []Source, optstate ...Source) ([]byte, []byte, error) {
	var nstubs []Node

	for i, src := range stubs {
		stub, err := s.UnmarshalSource(src)
		if err != nil {
			return nil, nil, fmt.Errorf("stub %d [%s] failed: %s", i+1, src.Name(), err)
		}
		nstubs = append(nstubs, stub)
	}
	for i, src := range optstate {
		stub, err := s.UnmarshalSource(src)
		if err != nil {
			return nil, nil, fmt.Errorf("state %d [%s] failed: %s", i+1, src.Name(), err)
		}
		nstubs = append(nstubs, stub)
	}
	node, err := s.UnmarshalSource(template)
	if err != nil {
		return nil, nil, fmt.Errorf("template [%s] failed: %s", template.Name(), err)
	}
	result, err := s.Cascade(node, nstubs)
	if err != nil {
		return nil, nil, err
	}
	rdata, err := s.Marshal(result)
	if err != nil {
		return nil, nil, fmt.Errorf("error marshalling result: %s", err)
	}
	state := s.DetermineState(result)
	if state != nil {
		sdata, err := s.Marshal(state)
		if err != nil {
			return nil, nil, fmt.Errorf("error marshalling result: %s", err)
		}
		return rdata, sdata, err
	}
	return rdata, nil, err
}

func ToNode(name string, data interface{}) (Node, error) {
	return yaml.Sanitize(name, data)
}

func Normalize(n Node) (interface{}, error) {
	return yaml.Normalize(n)
}
