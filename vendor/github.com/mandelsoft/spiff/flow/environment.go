package flow

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/mandelsoft/spiff/debug"
	"github.com/mandelsoft/spiff/dynaml"
	"github.com/mandelsoft/spiff/features"
	"github.com/mandelsoft/spiff/yaml"
)

type Scope struct {
	list   []yaml.Node
	local  map[string]yaml.Node
	static map[string]yaml.Node
	path   []string
	next   *Scope
	root   *Scope
}

func newFakeScope(outer *Scope, path []string, local map[string]yaml.Node) *Scope {
	return newScope(outer, path, local, nil)
}

func newScope(outer *Scope, path []string, local, static map[string]yaml.Node) *Scope {
	scope := &Scope{nil, local, static, path, outer, nil}
	if outer == nil || outer.root == nil {
		scope.root = scope
	} else {
		scope.root = outer.root
	}
	return scope
}

func newListScope(outer *Scope, path []string, local []yaml.Node, static map[string]yaml.Node) *Scope {
	scope := &Scope{local, nil, static, path, outer, nil}
	if outer == nil || outer.root == nil {
		scope.root = scope
	} else {
		scope.root = outer.root
		scope.local = outer.local
	}
	return scope
}

type DefaultEnvironment struct {
	state *State
	scope *Scope
	path  []string

	stubs      []yaml.Node
	stubPath   []string
	nomerge    bool
	sourceName string

	currentSourceName string

	static map[string]yaml.Node
	outer  dynaml.Binding

	active  bool
	binding bool
}

func keys(s map[string]yaml.Node) string {
	keys := "[ "
	sep := ""
	for k := range s {
		keys = keys + sep + k
		sep = ", "
	}
	return keys + "]"
}

func (e *DefaultEnvironment) String() string {
	result := fmt.Sprintf("SCOPES: <%s> static: %s", strings.Join(e.path, "."), keys(e.static))
	s := e.scope
	for s != nil {
		result = result + "\n  <" + strings.Join(s.path, ".") + ">" + keys(s.local) + keys(s.static)
		s = s.next
	}
	return result
}

func (e *DefaultEnvironment) GetState() dynaml.State {
	if e.state == nil {
		if e.outer != nil {
			return e.outer.GetState()
		}
	}
	return e.state
}

func (e *DefaultEnvironment) GetFeatures() features.FeatureFlags {
	return e.GetState().GetFeatures()
}

func (e *DefaultEnvironment) GetTempName(data []byte) (string, error) {
	if e.outer != nil {
		return e.outer.GetTempName(data)
	}
	return e.state.GetTempName(data)
}

func (e *DefaultEnvironment) GetFileContent(file string, cached bool) ([]byte, error) {
	if e.outer != nil {
		return e.outer.GetFileContent(file, cached)
	}
	return e.state.GetFileContent(file, cached)
}

func (e *DefaultEnvironment) Outer() dynaml.Binding {
	return e.outer
}

func (e *DefaultEnvironment) Active() bool {
	return e.active
}

func (e *DefaultEnvironment) Deactivate() dynaml.Binding {
	e.active = false
	return e
}

func (e *DefaultEnvironment) Path() []string {
	return e.path
}

func (e *DefaultEnvironment) StubPath() []string {
	return e.stubPath
}

func (e *DefaultEnvironment) NoMerge() bool {
	return e.nomerge
}

func (e *DefaultEnvironment) SourceName() string {
	return e.sourceName
}

func (e *DefaultEnvironment) CurrentSourceName() string {
	return e.currentSourceName
}

func (e *DefaultEnvironment) GetRootBinding() map[string]yaml.Node {
	if e.scope == nil {
		return nil
	}
	return e.scope.root.local
}

func (e *DefaultEnvironment) GetScope() *Scope {
	return e.scope
}

func (e *DefaultEnvironment) GetStaticBinding() map[string]yaml.Node {
	return e.static
}

func (e *DefaultEnvironment) FindFromRoot(path []string) (yaml.Node, bool) {
	if e.scope == nil {
		return nil, false
	}

	var root interface{}

	if e.scope.root.list != nil {
		if len(path) == 0 {
			// for root return root list
			return yaml.NewNode(e.scope.root.list, "root"), true
		}
		// special case: for list documents, we enable root path expressions based on the top level map found.
		if path[0] == "__map" {
			lroot := e.scope
			for lroot != nil {
				if lroot.local != nil {
					root = lroot.local
				}
				lroot = lroot.next
			}
			path = path[1:]
		} else {
			root = e.scope.root.list
		}
	} else {
		root = e.scope.root.local
	}
	return yaml.FindR(true, yaml.NewNode(root, "scope"), e.GetFeatures(), path...)
}

func (e *DefaultEnvironment) FindInScopes(nodescope *Scope, path []string) (yaml.Node, bool) {
	if len(path) > 0 {
		scope := nodescope
		for scope != nil {
			val := scope.local[path[0]]
			if val != nil {
				return yaml.FindR(true, val, e.GetFeatures(), path[1:]...)
			}
			scope = scope.next
		}
		return nil, false
	}
	return yaml.FindR(true, node(nodescope.local), e.GetFeatures(), path...)
}

func (e *DefaultEnvironment) FindReference(path []string) (yaml.Node, bool) {
	root, found, nodescope := resolveSymbol(e, path[0], e.scope)
	if !found {
		if path[0] == yaml.ROOT {
			var outer dynaml.Binding = e
			for outer.Outer() != nil {
				outer = outer.Outer()
			}
			return yaml.FindR(true, node(outer.GetRootBinding()), e.GetFeatures(), path[1:]...)
		}
		//fmt.Printf("FIND %s: %s\n", strings.Join(path,"."), e)
		//fmt.Printf("FOUND %s: %v\n", strings.Join(path,"."),  keys(nodescope))
		if path[0] == yaml.DOCNODE && nodescope != nil {
			return e.FindInScopes(nodescope, path[1:])
		}
		if e.outer != nil {
			return e.outer.FindReference(path)
		}
		return nil, false
	}

	//fmt.Printf("RESOLVE: %s: %s\n",path[0], dynaml.ExpressionType(root.Value()))
	if len(path) > 1 && path[0] == yaml.SELF {
		resolver := root.Resolver()
		return resolver.FindReference(path[1:])
	}
	return yaml.FindR(true, root, e.GetFeatures(), path[1:]...)
}

func (e *DefaultEnvironment) FindInStubs(path []string) (yaml.Node, bool) {
	debug.Debug("lookup %v in stubs\n", path)
	for _, stub := range e.stubs {
		debug.Debug("checking stub %s\n", stub.SourceName())
		val, found := yaml.Find(stub, e.GetFeatures(), path...)
		if found {
			if !val.Flags().Implied() {
				debug.Debug("found %v\n", path)
				return val, true
			}
			debug.Debug("skipping found stub %v\n", path)
		}
	}

	return nil, false
}

func (e *DefaultEnvironment) WithSource(source string) dynaml.Binding {
	n := *e
	n.sourceName = source
	return &n
}

func (e *DefaultEnvironment) WithScope(step map[string]yaml.Node) dynaml.Binding {
	n := *e
	n.scope = newScope(e.scope, e.path, step, e.static)
	return &n
}

func (e *DefaultEnvironment) WithListScope(step []yaml.Node) dynaml.Binding {
	n := *e
	n.scope = newListScope(e.scope, e.path, step, e.static)
	return &n
}

func (e *DefaultEnvironment) WithNewRoot() dynaml.Binding {
	n := *e
	static := map[string]yaml.Node{}
	n.scope = newScope(e.scope, e.path, static, e.static)
	n.scope.root = nil
	return &n
}

func (e *DefaultEnvironment) WithLocalScope(step map[string]yaml.Node) dynaml.Binding {
	n := *e
	static := map[string]yaml.Node{}
	for k, v := range e.static {
		static[k] = v
	}
	for k, v := range step {
		static[k] = v
	}
	n.static = static
	n.scope = newScope(e.scope, nil, step, static)
	return &n
}

func (e *DefaultEnvironment) WithPath(step string) dynaml.Binding {
	n := *e
	newPath := make([]string, len(e.path))
	copy(newPath, e.path)
	n.path = append(newPath, step)

	newPath = make([]string, len(e.stubPath))
	copy(newPath, e.stubPath)
	n.stubPath = append(newPath, step)
	return &n
}

func (e *DefaultEnvironment) RedirectOverwrite(path []string) dynaml.Binding {
	n := *e
	if len(path) > 0 {
		n.stubPath = path
		n.nomerge = false
	} else {
		n.nomerge = true
	}
	return &n
}

func (e *DefaultEnvironment) Flow(source yaml.Node, shouldOverride bool) (yaml.Node, dynaml.Status) {
	result := source

	for {
		debug.Debug("@@{ loop:  %+v\n", result)
		var env dynaml.Binding = e
		if list, ok := source.Value().([]yaml.Node); ok {
			env = e.WithListScope(list)
		}
		next := flow(result, env, shouldOverride, false)
		if next.Undefined() {
			result = yaml.UndefinedNode(node(nil))
			break
		}
		debug.Debug("@@} --->   %+v\n", next)

		next = Cleanup(next, updateBinding(next, env))
		b := reflect.DeepEqual(result, next)
		//b,r:=yaml.Equals(result, next,[]string{})
		if b {
			break
		}
		//fmt.Printf("****** found diff: %s\n", r)
		result = next
	}
	debug.Debug("@@@ Done\n")
	result = Cleanup(result, deactivateScopes)
	unresolved := dynaml.FindUnresolvedNodes(result)
	if len(unresolved) > 0 {
		return result, dynaml.UnresolvedNodes{unresolved}
	}

	return result, nil
}

func (e *DefaultEnvironment) Cascade(outer dynaml.Binding, template yaml.Node, partial bool, templates ...yaml.Node) (yaml.Node, error) {
	return Cascade(outer, template, Options{Partial: partial}, templates...)
}

func NewEnvironment(stubs []yaml.Node, source string, optstate ...*State) dynaml.Binding {
	var state *State
	if len(optstate) > 0 {
		state = optstate[0]
	}
	if state == nil {
		state = NewState(features.EncryptionKey(), MODE_OS_ACCESS|MODE_FILE_ACCESS)
	}
	return &DefaultEnvironment{state: state, stubs: stubs, sourceName: source, currentSourceName: source, outer: nil, active: true, binding: true}
}

func NewProcessLocalEnvironment(stubs []yaml.Node, source string) dynaml.Binding {
	state := NewState(features.EncryptionKey(), 0)
	return &DefaultEnvironment{state: state, stubs: stubs, sourceName: source, currentSourceName: source, outer: nil, active: true}
}

func CleanupEnvironment(binding dynaml.Binding) {
	env, ok := binding.(*DefaultEnvironment)
	if ok && env.state != nil {
		env.state.Cleanup()
	}
}

func ResetTags(binding dynaml.Binding) {
	env, ok := binding.(*DefaultEnvironment)
	if ok && env.state != nil {
		env.state.ResetTags()
	}
}

func ResetStream(binding dynaml.Binding) {
	env, ok := binding.(*DefaultEnvironment)
	if ok && env.state != nil {
		env.state.ResetStream()
	}
}

func PushDocument(binding dynaml.Binding, doc yaml.Node) {
	env, ok := binding.(*DefaultEnvironment)
	if ok && env.state != nil {
		env.state.PushDocument(doc)
	}
}

func NewNestedEnvironment(stubs []yaml.Node, source string, outer dynaml.Binding) dynaml.Binding {
	var state *State
	if outer == nil {
		state = NewDefaultState()
	}
	return &DefaultEnvironment{state: state, stubs: stubs, sourceName: source, currentSourceName: source, outer: outer, active: true}
}

type Updateable interface {
	Active() bool
	GetScope() *Scope
	Deactivate() dynaml.Binding
}

func updateBinding(root yaml.Node, binding dynaml.Binding) CleanupFunction {
	var me CleanupFunction
	me = func(node yaml.Node) (yaml.Node, CleanupFunction) {
		if v := node.Value(); v != nil {
			if static, ok := v.(dynaml.StaticallyScopedValue); ok {
				debug.Debug("update found static scoped %q\n", static)
				if env := static.StaticResolver().(Updateable); env.Active() {
					for scope := env.GetScope(); scope != nil; scope = scope.next {
						debug.Debug("update scope %v\n", scope.path)
						if scope.path != nil {
							ref, ok := yaml.FindR(true, root, binding.GetFeatures(), scope.path...)
							if ok {
								debug.Debug("found %#v\n", ref.Value())
								m := ref.Value().(map[string]yaml.Node)
								scope.local = m
							}
						} else {
							break
						}
					}
				}
			}
		}
		return node, me
	}
	return me
}

func deactivateScopes(node yaml.Node) (yaml.Node, CleanupFunction) {
	if v := node.Value(); v != nil {
		if lambda, ok := v.(dynaml.StaticallyScopedValue); ok {
			debug.Debug("deactivate statically scoped node %q\n", lambda)
			if env := lambda.StaticResolver().(Updateable); env.Active() {
				return yaml.ReplaceValue(lambda.SetStaticResolver(env.Deactivate()), node), deactivateScopes
			}
		}
	}
	return node, deactivateScopes
}

func resolveSymbol(env *DefaultEnvironment, name string, scope *Scope) (yaml.Node, bool, *Scope) {
	var nodescope *Scope
	if name == "__ctx" {
		return createContext(env), true, nil
	}
	for scope != nil {
		if nodescope == nil && scope.path != nil && scope.local != nil {
			//fmt.Printf("SCOPE NODE: <%s> %v %v\n", strings.Join(scope.path,"."), keys(scope.local), keys(scope.nodescope))
			nodescope = scope
		}
		val := scope.local[name]
		if val != nil {
			return val, true, nil
		}
		scope = scope.next
	}

	return nil, false, nodescope
}

func getOuters(env *DefaultEnvironment) (yaml.Node, yaml.Node) {
	var bindings dynaml.Binding
	var list []yaml.Node

	if outer := env.Outer(); outer != nil {
		for outer != nil {
			if e, ok := outer.(*DefaultEnvironment); ok && e.binding {
				bindings = outer
			}
			if b := outer.GetRootBinding(); b != nil {
				list = append(list, node(b))
			}
			outer = outer.Outer()
		}
	}
	if list == nil {
		if bindings != nil {
			return nil, node(bindings.GetRootBinding())
		} else {
			return nil, yaml.NewNode([]map[string]interface{}{}, "context")
		}
	}
	if bindings != nil {
		return node(list), node(bindings.GetRootBinding())
	} else {
		return node(list), yaml.NewNode([]map[string]interface{}{}, "context")
	}
}

func createContext(env *DefaultEnvironment) yaml.Node {
	ctx := make(map[string]yaml.Node)

	read, err := filepath.EvalSymlinks(env.CurrentSourceName())
	if err != nil {
		read = env.CurrentSourceName()
	}
	ctx["VERSION"] = node(VERSION)
	ctx["FILE"] = node(env.CurrentSourceName())
	ctx["DIR"] = node(filepath.Dir(env.CurrentSourceName()))
	ctx["RESOLVED_FILE"] = node(read)
	ctx["RESOLVED_DIR"] = node(filepath.Dir(read))

	ctx["PATHNAME"] = node(strings.Join(env.Path(), "."))

	path := make([]yaml.Node, len(env.Path()))
	for i, v := range env.Path() {
		path[i] = node(v)
	}
	ctx["PATH"] = node(path)
	path = make([]yaml.Node, len(env.StubPath()))
	for i, v := range env.StubPath() {
		path[i] = node(v)
	}
	ctx["STUBPATH"] = node(path)
	list, bindings := getOuters(env)
	if list != nil {
		ctx["OUTER"] = list
	}
	ctx["BINDINGS"] = bindings
	return node(ctx)
}

func node(val interface{}) yaml.Node {
	return yaml.NewNode(val, "__ctx")
}
