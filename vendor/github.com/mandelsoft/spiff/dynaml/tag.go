package dynaml

import (
	"fmt"

	"github.com/mandelsoft/spiff/yaml"
)

const TAG_LOCAL = TagScope(0x01)
const TAG_SCOPE = TagScope(0x06)
const TAG_SCOPE_GLOBAL = TagScope(0x00)
const TAG_SCOPE_STREAM = TagScope(0x02)

type TagScope int

type Tag struct {
	name  string
	node  yaml.Node
	path  []string
	scope TagScope
}

func NewTag(name string, node yaml.Node, path []string, scope TagScope) *Tag {
	return &Tag{name, node, path, scope}
}

func (t *Tag) Name() string {
	return t.name
}

func (t *Tag) Node() yaml.Node {
	return t.node
}

func (t *Tag) Path() []string {
	return t.path
}

func (t *Tag) Scope() TagScope {
	return t.scope
}

func (t *Tag) IsLocal() bool {
	return t.scope&TAG_LOCAL != 0
}

func (t *Tag) IsStream() bool {
	return t.scope&TAG_SCOPE == TAG_SCOPE_STREAM
}

func (t *Tag) IsGlobal() bool {
	return t.scope&TAG_SCOPE == TAG_SCOPE_GLOBAL
}

func (t *Tag) ResetLocal() {
	if t.IsLocal() {
		t.scope &= ^TAG_LOCAL
	}
}

type TagInfo struct {
	tag   *Tag
	level int
	comps []string
}

func (t *TagInfo) Tag() *Tag {
	return t.tag
}

func (t *TagInfo) Level() int {
	return t.level
}

func (t *TagInfo) Comps() []string {
	return t.comps
}

func (t *TagInfo) Name() string {
	return t.tag.name
}

func (t *TagInfo) Node() yaml.Node {
	return t.tag.node
}

func (t *TagInfo) Path() []string {
	return t.tag.path
}

func (t *TagInfo) Scope() TagScope {
	return t.tag.scope
}

func (t *TagInfo) IsLocal() bool {
	return t.tag.IsLocal()
}

func (t *TagInfo) IsStream() bool {
	return t.tag.IsStream()
}

func (t *TagInfo) IsGlobal() bool {
	return t.tag.IsGlobal()
}

func (t *TagInfo) ResetLocal() {
	t.tag.ResetLocal()
}

func NewTagInfo(tag *Tag) *TagInfo {
	l := 0
	comp := ""
	comps := []string{}
	for _, c := range tag.name {
		if c == ':' || c == '.' {
			comps = append(comps, comp)
			comp = ""
			l++
		} else {
			comp += string(c)
		}
	}
	comps = append(comps, comp)
	return &TagInfo{
		tag:   tag,
		level: l,
		comps: comps,
	}
}

func CheckTagName(name string) error {
	l := 0
	for _, c := range name {
		switch c {
		case ':', '.':
			if l == 0 {
				return fmt.Errorf("empty tag component not allowed")
			}
			l = 0
		default:
			l++
			if c >= '0' && c <= '9' {
				if l == 1 {
					return fmt.Errorf("tag component must start with alnum rune")
				}
				continue
			}
			if c >= 'a' && c <= 'z' {
				continue
			}
			if c >= 'A' && c <= 'Z' {
				continue
			}
			return fmt.Errorf("invalid character %q in tag component", string(c))
		}
	}
	return nil
}
