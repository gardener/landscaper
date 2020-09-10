package dynaml

import (
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/mandelsoft/spiff/yaml"
)

type Status interface {
	error
	Issue(fmt string, args ...interface{}) (issue yaml.Issue, localError bool, failed bool)
	HasError() bool
}

type SourceProvider interface {
	SourceName() string
}

type State interface {
	GetTempName(data []byte) (string, error)
	GetFileContent(file string, cached bool) ([]byte, error)
	GetEncryptionKey() string
	OSAccessAllowed() bool
	FileAccessAllowed() bool
	FileSystem() vfs.VFS
	GetFunctions() Registry
}

type Binding interface {
	SourceProvider
	GetStaticBinding() map[string]yaml.Node
	GetRootBinding() map[string]yaml.Node

	FindFromRoot([]string) (yaml.Node, bool)
	FindReference([]string) (yaml.Node, bool)
	FindInStubs([]string) (yaml.Node, bool)

	WithScope(step map[string]yaml.Node) Binding
	WithLocalScope(step map[string]yaml.Node) Binding
	WithPath(step string) Binding
	WithSource(source string) Binding
	WithNewRoot() Binding
	RedirectOverwrite(path []string) Binding

	Outer() Binding
	Path() []string
	StubPath() []string
	NoMerge() bool

	GetState() State
	GetTempName(data []byte) (string, error)
	GetFileContent(file string, cached bool) ([]byte, error)

	Flow(source yaml.Node, shouldOverride bool) (yaml.Node, Status)
	Cascade(outer Binding, template yaml.Node, partial bool, templates ...yaml.Node) (yaml.Node, error)
}

type Cleanup interface {
	Cleanup() error
}

type EvaluationInfo struct {
	RedirectPath []string
	Replace      bool
	Merged       bool
	Preferred    bool
	KeyName      string
	Source       string
	LocalError   bool
	Failed       bool
	Undefined    bool
	Raw          bool
	Issue        yaml.Issue
	Cleanups     []Cleanup
	yaml.NodeFlags
}

type EvaluationError struct {
	resolved bool
	EvaluationInfo
	ok bool
}

func (e EvaluationError) Error() string {
	return e.Issue.Issue
}

func RaiseEvaluationError(resolved bool, info EvaluationInfo, ok bool) {
	panic(EvaluationError{resolved, info, ok})
}

func RaiseEvaluationErrorf(format string, args ...interface{}) {
	info := DefaultInfo()
	info.SetError(format, args...)
	panic(EvaluationError{true, info, false})
}

func CatchEvaluationError(result *interface{}, info *EvaluationInfo, ok *bool, msgfmt string, args ...interface{}) {
	err := recover()
	if err != nil {
		if eerr, my := err.(EvaluationError); my {
			*result = nil
			*info = eerr.EvaluationInfo
			if msgfmt != "" {
				(*info).SetError(msgfmt, args...)
				(*info).Issue.Sequence = true
				if eerr.Issue.Issue != "" {
					(*info).Issue.Nested = []yaml.Issue{eerr.Issue}
				}
			}
			*ok = eerr.ok
		} else {
			panic(err)
		}
	}
}

func (e EvaluationInfo) SourceName() string {
	return e.Source
}

func DefaultInfo() EvaluationInfo {
	return EvaluationInfo{nil, false, false,
		false, "", "",
		false, false, false, false,
		yaml.Issue{}, nil, 0}
}

type Expression interface {
	Evaluate(Binding, bool) (interface{}, EvaluationInfo, bool)
}

type StaticallyScopedValue interface {
	StaticResolver() Binding
	SetStaticResolver(binding Binding) StaticallyScopedValue
}

func (i *EvaluationInfo) Cleanup() error {
	var err error
	for _, c := range i.Cleanups {
		e := c.Cleanup()
		if e != nil {
			err = e
		}
	}
	i.Cleanups = nil
	return err
}

func (i *EvaluationInfo) DenyOSOperation(name string) (interface{}, EvaluationInfo, bool) {
	return i.Error("%s: no OS operations supported in this execution environment", name)
}

func (i *EvaluationInfo) Error(msgfmt interface{}, args ...interface{}) (interface{}, EvaluationInfo, bool) {
	i.SetError(msgfmt, args...)
	return nil, *i, false
}

func (i *EvaluationInfo) AnnotateError(err EvaluationInfo, msgfmt interface{}, args ...interface{}) (interface{}, EvaluationInfo, bool) {
	i.SetError(msgfmt, args...)
	if err.Issue.Issue != "" {
		i.Issue.Nested = append(i.Issue.Nested, err.Issue)
	}
	return nil, *i, false
}

func (i *EvaluationInfo) SetError(msgfmt interface{}, args ...interface{}) {
	i.LocalError = true
	switch f := msgfmt.(type) {
	case string:
		i.Issue = yaml.NewIssue(f, args...)
	default:
		i.Issue = yaml.NewIssue("%s", msgfmt)
	}
}

func (i *EvaluationInfo) PropagateError(value interface{}, state Status, msgfmt string, args ...interface{}) (interface{}, EvaluationInfo, bool) {
	i.Issue, i.LocalError, i.Failed = state.Issue(msgfmt, args...)
	if i.LocalError {
		value = nil
	}
	return value, *i, false //!i.LocalError
}

func (i EvaluationInfo) Join(o EvaluationInfo) EvaluationInfo {
	if o.RedirectPath != nil {
		i.RedirectPath = o.RedirectPath
	}
	i.Replace = o.Replace // replace only by directly using the merge node
	i.Preferred = i.Preferred || o.Preferred
	i.Merged = i.Merged || o.Merged
	if o.KeyName != "" {
		i.KeyName = o.KeyName
	}
	if o.Issue.Issue != "" {
		i.Issue = o.Issue
	}
	if o.LocalError {
		i.LocalError = true
	}
	if o.Failed {
		i.Failed = true
	}
	if o.Undefined {
		i.Undefined = true
	}
	i.NodeFlags |= o.NodeFlags

	i.Cleanups = append(i.Cleanups, o.Cleanups...)
	return i
}

func ResolveExpressionOrPushEvaluation(e *Expression, resolved *bool, info *EvaluationInfo, binding Binding, locally bool) (interface{}, EvaluationInfo, bool) {
	val, infoe, ok := (*e).Evaluate(binding, locally)
	if info != nil {
		infoe = (*info).Join(infoe)
	}
	if !ok {
		return nil, infoe, false
	}

	if v, ok := val.(Expression); ok {
		*e = KeepArgWrapper(v, *e)
		*resolved = false
		return nil, infoe, true
	} else {
		return val, infoe, true
	}
}

func ResolveIntegerExpressionOrPushEvaluation(e *Expression, resolved *bool, info *EvaluationInfo, binding Binding, locally bool) (int64, EvaluationInfo, bool) {
	value, infoe, ok := ResolveExpressionOrPushEvaluation(e, resolved, info, binding, locally)

	if value == nil {
		return 0, infoe, ok
	}

	i, ok := value.(int64)
	if ok {
		return i, infoe, true
	} else {
		infoe.Issue = yaml.NewIssue("integer operand required")
		return 0, infoe, false
	}
}

func ResolveExpressionListOrPushEvaluation(list *[]Expression, resolved *bool, info *EvaluationInfo, binding Binding, locally bool) ([]interface{}, EvaluationInfo, bool) {
	values := make([]interface{}, len(*list))
	pushed := make([]Expression, len(*list))
	infoe := EvaluationInfo{}
	expand := false
	ok := true

	copy(pushed, *list)

	for i, _ := range pushed {
		values[i], infoe, ok = ResolveExpressionOrPushEvaluation(&pushed[i], resolved, info, binding, locally)
		info = &infoe
		expand = expand || IsListExpansion(pushed[i])
		if !ok {
			return nil, infoe, false
		}
	}

	if expand {
		vlist := []interface{}{}
		for i, v := range values {
			if IsListExpansion(pushed[i]) {
				list, ok := v.([]yaml.Node)
				if !ok {
					_, infoe, ok := infoe.Error("argument expansion required list argument")
					return nil, infoe, ok
				}
				for _, e := range list {
					vlist = append(vlist, e.Value())
				}
			} else {
				vlist = append(vlist, v)
			}
		}
		values = vlist
	}
	*list = pushed
	return values, infoe, true

}
