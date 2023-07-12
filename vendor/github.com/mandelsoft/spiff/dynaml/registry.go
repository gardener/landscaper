package dynaml

type Registry interface {
	LookupFunction(name string) Function

	LookupControl(name string) (*Control, bool)
	IsTemplateControlOption(name string) bool

	WithFunctions(Functions) Registry
	WithControls(Controls) Registry
}

type registry struct {
	functions Functions
	controls  Controls
}

func (r *registry) WithFunctions(f Functions) Registry {
	if r == nil {
		return &registry{functions: f}
	}
	return &registry{
		functions: f,
		controls:  r.controls,
	}
}

func (r *registry) WithControls(c Controls) Registry {
	if r == nil {
		return &registry{controls: c}
	}
	return &registry{
		functions: r.functions,
		controls:  c,
	}
}

func (r *registry) LookupFunction(name string) Function {
	if r == nil || r.functions == nil {
		return function_registry.LookupFunction(name)
	}
	return r.functions.LookupFunction(name)
}

func (r *registry) LookupControl(name string) (*Control, bool) {
	if r == nil || r.controls == nil {
		return control_registry.LookupControl(name)
	}
	return r.controls.LookupControl(name)
}

func (r *registry) IsTemplateControlOption(name string) bool {
	if r == nil || r.controls == nil {
		return control_registry.IsTemplateControlOption(name)
	}
	return r.controls.IsTemplateControlOption(name)
}

func DefaultRegistry() Registry {
	var r *registry
	return r // standard behaviour support on nil pointer inter
}
