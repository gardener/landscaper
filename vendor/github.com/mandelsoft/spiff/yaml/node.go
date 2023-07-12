package yaml

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/mandelsoft/spiff/legacy/candiedyaml"
)

const SELF = "_"
const DOCNODE = "__"
const ROOT = "___"
const MERGEKEY = "<<<"

type RefResolver interface {
	FindReference([]string) (Node, bool)
}

type Node interface {
	candiedyaml.Marshaler

	Value() interface{}
	Template() interface{}
	SourceName() string
	RedirectPath() []string
	Flags() NodeFlags
	Temporary() bool
	Local() bool
	State() bool
	ReplaceFlag() bool
	Preferred() bool
	Merged() bool
	StandardOverride() bool
	KeyName() string
	HasError() bool
	Failed() bool
	Undefined() bool
	Issue() Issue

	Resolver() RefResolver

	GetAnnotation() Annotation
	EquivalentToNode(Node) bool
}

type AnnotatedNode struct {
	value      interface{}
	template   interface{}
	resolver   RefResolver
	sourceName string
	Annotation
}

type Issue struct {
	Issue    string
	OrigPath []string
	Nested   []Issue
	Sequence bool
}

func NewIssue(msg string, args ...interface{}) Issue {
	return Issue{fmt.Sprintf(msg, args...), nil, []Issue{}, false}
}

func NewPathIssue(path []string, msg string, args ...interface{}) Issue {
	return Issue{fmt.Sprintf(msg, args...), path, []Issue{}, false}
}

const (
	FLAG_TEMPORARY = 0x001
	FLAG_LOCAL     = 0x002
	FLAG_INJECT    = 0x004
	FLAG_STATE     = 0x008
	FLAG_DEFAULT   = 0x010
	FLAG_DYNAMIC   = 0x020

	FLAG_INJECTED = 0x040
	FLAG_IMPLIED  = 0x080
)

type NodeFlags int

func (f *NodeFlags) AddFlags(flags NodeFlags) *NodeFlags {
	*f |= flags
	return f
}

func (f NodeFlags) Temporary() bool {
	return (f & FLAG_TEMPORARY) != 0
}
func (f *NodeFlags) SetTemporary() *NodeFlags {
	*f |= FLAG_TEMPORARY
	return f
}

func (f NodeFlags) Local() bool {
	return (f & FLAG_LOCAL) != 0
}
func (f *NodeFlags) SetLocal() *NodeFlags {
	*f |= FLAG_LOCAL
	return f
}

func (f NodeFlags) Inject() bool {
	return (f & (FLAG_INJECT | FLAG_DEFAULT)) != 0
}
func (f *NodeFlags) SetInject() *NodeFlags {
	*f |= FLAG_INJECT
	return f
}

func (f NodeFlags) Default() bool {
	return (f & (FLAG_DEFAULT | FLAG_INJECT)) == FLAG_DEFAULT
}
func (f *NodeFlags) SetDefault() *NodeFlags {
	*f |= FLAG_DEFAULT
	return f
}

func (f NodeFlags) Implied() bool {
	return (f & (FLAG_IMPLIED | FLAG_DEFAULT)) == FLAG_IMPLIED
}
func (f *NodeFlags) SetImplied() *NodeFlags {
	*f |= FLAG_IMPLIED
	return f
}
func (f NodeFlags) PropagateImplied() bool {
	return (f & (FLAG_IMPLIED | FLAG_DEFAULT)) != 0
}

func (f *NodeFlags) Overridden() NodeFlags {
	if f.Default() {
		return (*f | FLAG_INJECT) &^ FLAG_DEFAULT
	}
	return *f
}

func (f NodeFlags) State() bool {
	return (f & FLAG_STATE) != 0
}
func (f *NodeFlags) SetState() *NodeFlags {
	*f |= FLAG_STATE
	return f
}

func (f NodeFlags) Dynamic() bool {
	return (f & FLAG_DYNAMIC) != 0
}
func (f *NodeFlags) SetDynamic() *NodeFlags {
	*f |= FLAG_DYNAMIC
	return f
}

func (f NodeFlags) Injected() bool {
	return (f & FLAG_INJECTED) != 0
}
func (f *NodeFlags) SetInjected() *NodeFlags {
	*f |= FLAG_INJECTED
	return f
}

type Annotation struct {
	redirectPath []string
	replace      bool
	preferred    bool
	merged       bool
	keyName      string
	error        bool
	failed       bool
	undefined    bool
	issue        Issue
	tag          string
	NodeFlags
}

func copyNode(node Node) AnnotatedNode {
	return AnnotatedNode{node.Value(), node.Template(), node.Resolver(), node.SourceName(), node.GetAnnotation()}
}
func copyNodeAnnotated(node Node, anno Annotation) AnnotatedNode {
	return AnnotatedNode{node.Value(), node.Template(), node.Resolver(), node.SourceName(), anno}
}

func NewNode(value interface{}, sourcePath string) Node {
	return AnnotatedNode{MassageType(value), nil, nil, sourcePath, EmptyAnnotation()}
}

func NewDynamicNode(value, template interface{}, sourcePath string) Node {
	return AnnotatedNode{MassageType(value), template, nil, sourcePath, EmptyAnnotation().SetInjected().SetDynamic()}
}

func ResolverNode(node Node, resolver RefResolver) Node {
	n := copyNode(node)
	n.resolver = resolver
	return n
}

func ReplaceValue(value interface{}, node Node) Node {
	n := copyNode(node)
	n.value = value
	return n
}
func ReferencedNode(node Node) Node {
	return copyNodeAnnotated(node, NewReferencedAnnotation(node))
}

func SubstituteNode(value interface{}, node Node) Node {
	n := copyNode(node)
	n.value = MassageType(value)
	return n
}

func RedirectNode(value interface{}, node Node, redirect []string) Node {
	n := copyNodeAnnotated(node, node.GetAnnotation().SetRedirectPath(redirect))
	n.value = MassageType(value)
	return n
}

func ReplaceNode(value interface{}, node Node, redirect []string) Node {
	n := copyNodeAnnotated(node, node.GetAnnotation().SetReplaceFlag().SetRedirectPath(redirect))
	n.value = MassageType(value)
	return n
}

func PreferredNode(node Node) Node {
	return copyNodeAnnotated(node, node.GetAnnotation().SetPreferred())
}

func MergedNode(node Node) Node {
	return copyNodeAnnotated(node, node.GetAnnotation().SetMerged())
}

func KeyNameNode(node Node, keyName string) Node {
	return copyNodeAnnotated(node, node.GetAnnotation().AddKeyName(keyName))
}

func IssueNode(node Node, error bool, failed bool, issue Issue) Node {
	return copyNodeAnnotated(node, node.GetAnnotation().AddIssue(error, failed, issue))
}

func UndefinedNode(node Node) Node {
	return copyNodeAnnotated(node, node.GetAnnotation().SetUndefined())
}

func AddFlags(node Node, flags NodeFlags) Node {
	return copyNodeAnnotated(node, node.GetAnnotation().AddFlags(flags))
}

func SetTag(node Node, tag string) Node {
	return copyNodeAnnotated(node, node.GetAnnotation().SetTag(tag))
}

func TemporaryNode(node Node) Node {
	return copyNodeAnnotated(node, node.GetAnnotation().SetTemporary())
}

func InjectedNode(node Node) Node {
	return copyNodeAnnotated(node, node.GetAnnotation().SetInjected())
}

func DefaultedNode(node Node) Node {
	return copyNodeAnnotated(node, node.GetAnnotation().SetDefault())
}

func LocalNode(node Node) Node {
	return copyNodeAnnotated(node, node.GetAnnotation().SetLocal())
}

func StateNode(node Node) Node {
	return copyNodeAnnotated(node, node.GetAnnotation().SetState())
}

func MassageType(value interface{}) interface{} {
	switch value.(type) {
	case int, int8, int16, int32:
		value = reflect.ValueOf(value).Int()
	}
	return value
}

func EmptyAnnotation() Annotation {
	return Annotation{nil, false, false, false, "", false, false, false, Issue{}, "", 0}
}

func NewReferencedAnnotation(node Node) Annotation {
	return Annotation{nil, false, false, false, node.KeyName(), node.HasError(), node.Failed(), node.Undefined(), node.Issue(), "", 0}
}

func (n Annotation) Flags() NodeFlags {
	return n.NodeFlags
}

func (n Annotation) RedirectPath() []string {
	return n.redirectPath
}

func (n Annotation) ReplaceFlag() bool {
	return n.replace
}

func (n Annotation) Preferred() bool {
	return n.preferred
}

func (n Annotation) Merged() bool {
	return n.merged //|| n.ReplaceFlag() || len(n.RedirectPath()) > 0
}

func (n Annotation) StandardOverride() bool {
	return !n.merged && !n.ReplaceFlag() && len(n.RedirectPath()) == 0
}

func (n Annotation) KeyName() string {
	return n.keyName
}

func (n Annotation) Tag() string {
	return n.tag
}

func (n Annotation) HasError() bool {
	return n.error
}

func (n Annotation) Failed() bool {
	return n.failed
}

func (n Annotation) Undefined() bool {
	return n.undefined
}

func (n Annotation) Issue() Issue {
	return n.issue
}

func (n Annotation) AddFlags(flags NodeFlags) Annotation {
	n.NodeFlags |= flags
	return n
}

func (n Annotation) SetTemporary() Annotation {
	n.NodeFlags.SetTemporary()
	return n
}

func (n Annotation) SetLocal() Annotation {
	n.NodeFlags.SetLocal()
	return n
}

func (n Annotation) SetState() Annotation {
	n.NodeFlags.SetState()
	return n
}

func (n Annotation) SetInject() Annotation {
	n.NodeFlags.SetInject()
	return n
}

func (n Annotation) SetInjected() Annotation {
	n.NodeFlags.SetInjected()
	return n
}

func (n Annotation) SetDefault() Annotation {
	n.NodeFlags.SetDefault()
	return n
}

func (n Annotation) SetDynamic() Annotation {
	n.NodeFlags.SetDynamic()
	return n
}

func (n Annotation) SetRedirectPath(redirect []string) Annotation {
	n.redirectPath = redirect
	return n
}

func (n Annotation) SetReplaceFlag() Annotation {
	n.replace = true
	return n
}

func (n Annotation) SetPreferred() Annotation {
	n.preferred = true
	return n
}

func (n Annotation) SetMerged() Annotation {
	n.merged = true
	return n
}

func (n Annotation) SetTag(tag string) Annotation {
	n.tag = tag
	return n
}

func (n Annotation) SetUndefined() Annotation {
	n.undefined = true
	return n
}

func (n Annotation) AddKeyName(keyName string) Annotation {
	if keyName != "" {
		n.keyName = keyName
	}
	return n
}

func (n Annotation) AddIssue(error bool, failed bool, issue Issue) Annotation {
	if issue.Issue != "" {
		n.issue = issue
	}
	n.error = error
	n.failed = failed
	return n
}

func (n AnnotatedNode) Value() interface{} {
	return n.value
}

func (n AnnotatedNode) SourceName() string {
	return n.sourceName
}

func (n AnnotatedNode) Template() interface{} {
	return n.template
}

func (n AnnotatedNode) Resolver() RefResolver {
	return n.resolver
}

func (n AnnotatedNode) GetAnnotation() Annotation {
	return n.Annotation
}

func (n AnnotatedNode) MarshalYAML() (string, interface{}, error) {
	v := n.Value()

	m, ok := v.(candiedyaml.Marshaler)
	for ok {
		_, v, _ = m.MarshalYAML()
		m, ok = v.(candiedyaml.Marshaler)
	}
	return "", v, nil
}

func (n AnnotatedNode) EquivalentToNode(o Node) bool {
	if o == nil {
		return false
	}

	at := reflect.TypeOf(n.Value())
	bt := reflect.TypeOf(o.Value())

	if at != bt {
		return false
	}

	switch nv := n.Value().(type) {
	case map[string]Node:
		ov := o.Value().(map[string]Node)

		if len(nv) != len(ov) {
			return false
		}

		for key, nval := range nv {
			oval, found := ov[key]
			if !found {
				return false
			}

			if !nval.EquivalentToNode(oval) {
				return false
			}
		}

		return true

	case []Node:
		ov := o.Value().([]Node)

		if len(nv) != len(ov) {
			return false
		}

		for i, nval := range nv {
			oval := ov[i]

			if !nval.EquivalentToNode(oval) {
				return false
			}
		}

		return true
	case ComparableValue:
		return nv.EquivalentTo(o.Value())
	}

	b := reflect.DeepEqual(n.Value(), o.Value())

	return b
}

func EmbeddedDynaml(root Node, interpol bool) *string {
	rootString, ok := root.Value().(string)
	if !ok {
		return nil
	}
	if strings.HasPrefix(rootString, "((") &&
		strings.HasSuffix(rootString, "))") {
		sub := rootString[2 : len(rootString)-2]
		if !strings.HasPrefix(sub, "!") {
			return &sub
		}
		return nil
	}
	if !interpol {
		return nil
	}
	_, expr := convertToExpression(rootString, false)
	return expr
}

func UnescapeDynaml(root Node, interpol bool) Node {
	if root.Value() == nil {
		return root
	}
	switch value := root.Value().(type) {
	case string:
		if strings.HasPrefix(value, "((") &&
			strings.HasSuffix(value, "))") {
			sub := value[2 : len(value)-2]
			if strings.HasPrefix(sub, "!") {
				return NewNode("(("+sub[1:]+"))", root.SourceName())
			}
			return root
		}
		if interpol {
			str, _ := convertToExpression(value, true)
			if str != nil && *str != value {
				return NewNode(*str, root.SourceName())
			}
		}
	case map[string]Node:
		new := map[string]Node{}
		found := false
		for k, v := range value {
			switch {
			case strings.HasPrefix(k, "<<!"):
				found = true
				new["<<"+k[3:]] = v
			case strings.HasPrefix(k, MERGEKEY+"!"):
				found = true
				new[MERGEKEY+k[len(MERGEKEY)+1:]] = v
			default:
				new[k] = v
			}
		}
		if found {
			return NewNode(new, root.SourceName())
		}
	}
	return root
}
