package dynaml

import (
	"fmt"
	"math"
	"sort"
	"strconv"
)

const endSymbol rune = 1114112

/* The rule types inferred from the grammar are below. */
type pegRule uint8

const (
	ruleUnknown pegRule = iota
	ruleDynaml
	rulePrefer
	ruleMarkedExpression
	ruleSubsequentMarker
	ruleMarker
	ruleMarkerExpression
	ruleExpression
	ruleScoped
	ruleScope
	ruleCreateScope
	ruleLevel7
	ruleOr
	ruleOrOp
	ruleLevel6
	ruleConditional
	ruleLevel5
	ruleConcatenation
	ruleLevel4
	ruleLogOr
	ruleLogAnd
	ruleLevel3
	ruleComparison
	ruleCompareOp
	ruleLevel2
	ruleAddition
	ruleSubtraction
	ruleLevel1
	ruleMultiplication
	ruleDivision
	ruleModulo
	ruleLevel0
	ruleChained
	ruleChainedQualifiedExpression
	ruleChainedRef
	ruleChainedDynRef
	ruleSlice
	ruleCurrying
	ruleChainedCall
	ruleStartArguments
	ruleNameArgumentList
	ruleNextNameArgument
	ruleExpressionList
	ruleNextExpression
	ruleListExpansion
	ruleProjection
	ruleProjectionValue
	ruleSubstitution
	ruleNot
	ruleGrouped
	ruleRange
	ruleStartRange
	ruleRangeOp
	ruleNumber
	ruleString
	ruleBoolean
	ruleNil
	ruleUndefined
	ruleSymbol
	ruleList
	ruleStartList
	ruleMap
	ruleCreateMap
	ruleAssignments
	ruleAssignment
	ruleMerge
	ruleRefMerge
	ruleSimpleMerge
	ruleReplace
	ruleRequired
	ruleOn
	ruleAuto
	ruleDefault
	ruleSync
	ruleLambdaExt
	ruleLambdaOrExpr
	ruleCatch
	ruleMapMapping
	ruleMapping
	ruleMapSelection
	ruleSelection
	ruleSum
	ruleLambda
	ruleLambdaRef
	ruleLambdaExpr
	ruleParams
	ruleStartParams
	ruleNames
	ruleNextName
	ruleName
	ruleDefaultValue
	ruleVarParams
	ruleReference
	ruleFollowUpRef
	rulePathComponent
	ruleKey
	ruleIndex
	ruleIP
	rulews
	rulereq_ws
	ruleAction0
	ruleAction1
	ruleAction2

	rulePre
	ruleIn
	ruleSuf
)

var rul3s = [...]string{
	"Unknown",
	"Dynaml",
	"Prefer",
	"MarkedExpression",
	"SubsequentMarker",
	"Marker",
	"MarkerExpression",
	"Expression",
	"Scoped",
	"Scope",
	"CreateScope",
	"Level7",
	"Or",
	"OrOp",
	"Level6",
	"Conditional",
	"Level5",
	"Concatenation",
	"Level4",
	"LogOr",
	"LogAnd",
	"Level3",
	"Comparison",
	"CompareOp",
	"Level2",
	"Addition",
	"Subtraction",
	"Level1",
	"Multiplication",
	"Division",
	"Modulo",
	"Level0",
	"Chained",
	"ChainedQualifiedExpression",
	"ChainedRef",
	"ChainedDynRef",
	"Slice",
	"Currying",
	"ChainedCall",
	"StartArguments",
	"NameArgumentList",
	"NextNameArgument",
	"ExpressionList",
	"NextExpression",
	"ListExpansion",
	"Projection",
	"ProjectionValue",
	"Substitution",
	"Not",
	"Grouped",
	"Range",
	"StartRange",
	"RangeOp",
	"Number",
	"String",
	"Boolean",
	"Nil",
	"Undefined",
	"Symbol",
	"List",
	"StartList",
	"Map",
	"CreateMap",
	"Assignments",
	"Assignment",
	"Merge",
	"RefMerge",
	"SimpleMerge",
	"Replace",
	"Required",
	"On",
	"Auto",
	"Default",
	"Sync",
	"LambdaExt",
	"LambdaOrExpr",
	"Catch",
	"MapMapping",
	"Mapping",
	"MapSelection",
	"Selection",
	"Sum",
	"Lambda",
	"LambdaRef",
	"LambdaExpr",
	"Params",
	"StartParams",
	"Names",
	"NextName",
	"Name",
	"DefaultValue",
	"VarParams",
	"Reference",
	"FollowUpRef",
	"PathComponent",
	"Key",
	"Index",
	"IP",
	"ws",
	"req_ws",
	"Action0",
	"Action1",
	"Action2",

	"Pre_",
	"_In_",
	"_Suf",
}

type tokenTree interface {
	Print()
	PrintSyntax()
	PrintSyntaxTree(buffer string)
	Add(rule pegRule, begin, end, next uint32, depth int)
	Expand(index int) tokenTree
	Tokens() <-chan token32
	AST() *node32
	Error() []token32
	trim(length int)
}

type node32 struct {
	token32
	up, next *node32
}

func (node *node32) print(depth int, buffer string) {
	for node != nil {
		for c := 0; c < depth; c++ {
			fmt.Printf(" ")
		}
		fmt.Printf("\x1B[34m%v\x1B[m %v\n", rul3s[node.pegRule], strconv.Quote(string(([]rune(buffer)[node.begin:node.end]))))
		if node.up != nil {
			node.up.print(depth+1, buffer)
		}
		node = node.next
	}
}

func (node *node32) Print(buffer string) {
	node.print(0, buffer)
}

type element struct {
	node *node32
	down *element
}

/* ${@} bit structure for abstract syntax tree */
type token32 struct {
	pegRule
	begin, end, next uint32
}

func (t *token32) isZero() bool {
	return t.pegRule == ruleUnknown && t.begin == 0 && t.end == 0 && t.next == 0
}

func (t *token32) isParentOf(u token32) bool {
	return t.begin <= u.begin && t.end >= u.end && t.next > u.next
}

func (t *token32) getToken32() token32 {
	return token32{pegRule: t.pegRule, begin: uint32(t.begin), end: uint32(t.end), next: uint32(t.next)}
}

func (t *token32) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v %v", rul3s[t.pegRule], t.begin, t.end, t.next)
}

type tokens32 struct {
	tree    []token32
	ordered [][]token32
}

func (t *tokens32) trim(length int) {
	t.tree = t.tree[0:length]
}

func (t *tokens32) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens32) Order() [][]token32 {
	if t.ordered != nil {
		return t.ordered
	}

	depths := make([]int32, 1, math.MaxInt16)
	for i, token := range t.tree {
		if token.pegRule == ruleUnknown {
			t.tree = t.tree[:i]
			break
		}
		depth := int(token.next)
		if length := len(depths); depth >= length {
			depths = depths[:depth+1]
		}
		depths[depth]++
	}
	depths = append(depths, 0)

	ordered, pool := make([][]token32, len(depths)), make([]token32, len(t.tree)+len(depths))
	for i, depth := range depths {
		depth++
		ordered[i], pool, depths[i] = pool[:depth], pool[depth:], 0
	}

	for i, token := range t.tree {
		depth := token.next
		token.next = uint32(i)
		ordered[depth][depths[depth]] = token
		depths[depth]++
	}
	t.ordered = ordered
	return ordered
}

type state32 struct {
	token32
	depths []int32
	leaf   bool
}

func (t *tokens32) AST() *node32 {
	tokens := t.Tokens()
	stack := &element{node: &node32{token32: <-tokens}}
	for token := range tokens {
		if token.begin == token.end {
			continue
		}
		node := &node32{token32: token}
		for stack != nil && stack.node.begin >= token.begin && stack.node.end <= token.end {
			stack.node.next = node.up
			node.up = stack.node
			stack = stack.down
		}
		stack = &element{node: node, down: stack}
	}
	return stack.node
}

func (t *tokens32) PreOrder() (<-chan state32, [][]token32) {
	s, ordered := make(chan state32, 6), t.Order()
	go func() {
		var states [8]state32
		for i := range states {
			states[i].depths = make([]int32, len(ordered))
		}
		depths, state, depth := make([]int32, len(ordered)), 0, 1
		write := func(t token32, leaf bool) {
			S := states[state]
			state, S.pegRule, S.begin, S.end, S.next, S.leaf = (state+1)%8, t.pegRule, t.begin, t.end, uint32(depth), leaf
			copy(S.depths, depths)
			s <- S
		}

		states[state].token32 = ordered[0][0]
		depths[0]++
		state++
		a, b := ordered[depth-1][depths[depth-1]-1], ordered[depth][depths[depth]]
	depthFirstSearch:
		for {
			for {
				if i := depths[depth]; i > 0 {
					if c, j := ordered[depth][i-1], depths[depth-1]; a.isParentOf(c) &&
						(j < 2 || !ordered[depth-1][j-2].isParentOf(c)) {
						if c.end != b.begin {
							write(token32{pegRule: ruleIn, begin: c.end, end: b.begin}, true)
						}
						break
					}
				}

				if a.begin < b.begin {
					write(token32{pegRule: rulePre, begin: a.begin, end: b.begin}, true)
				}
				break
			}

			next := depth + 1
			if c := ordered[next][depths[next]]; c.pegRule != ruleUnknown && b.isParentOf(c) {
				write(b, false)
				depths[depth]++
				depth, a, b = next, b, c
				continue
			}

			write(b, true)
			depths[depth]++
			c, parent := ordered[depth][depths[depth]], true
			for {
				if c.pegRule != ruleUnknown && a.isParentOf(c) {
					b = c
					continue depthFirstSearch
				} else if parent && b.end != a.end {
					write(token32{pegRule: ruleSuf, begin: b.end, end: a.end}, true)
				}

				depth--
				if depth > 0 {
					a, b, c = ordered[depth-1][depths[depth-1]-1], a, ordered[depth][depths[depth]]
					parent = a.isParentOf(b)
					continue
				}

				break depthFirstSearch
			}
		}

		close(s)
	}()
	return s, ordered
}

func (t *tokens32) PrintSyntax() {
	tokens, ordered := t.PreOrder()
	max := -1
	for token := range tokens {
		if !token.leaf {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[36m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
			}
			fmt.Printf(" \x1B[36m%v\x1B[m\n", rul3s[token.pegRule])
		} else if token.begin == token.end {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[31m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
			}
			fmt.Printf(" \x1B[31m%v\x1B[m\n", rul3s[token.pegRule])
		} else {
			for c, end := token.begin, token.end; c < end; c++ {
				if i := int(c); max+1 < i {
					for j := max; j < i; j++ {
						fmt.Printf("skip %v %v\n", j, token.String())
					}
					max = i
				} else if i := int(c); i <= max {
					for j := i; j <= max; j++ {
						fmt.Printf("dupe %v %v\n", j, token.String())
					}
				} else {
					max = int(c)
				}
				fmt.Printf("%v", c)
				for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
					fmt.Printf(" \x1B[34m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
				}
				fmt.Printf(" \x1B[34m%v\x1B[m\n", rul3s[token.pegRule])
			}
			fmt.Printf("\n")
		}
	}
}

func (t *tokens32) PrintSyntaxTree(buffer string) {
	tokens, _ := t.PreOrder()
	for token := range tokens {
		for c := 0; c < int(token.next); c++ {
			fmt.Printf(" ")
		}
		fmt.Printf("\x1B[34m%v\x1B[m %v\n", rul3s[token.pegRule], strconv.Quote(string(([]rune(buffer)[token.begin:token.end]))))
	}
}

func (t *tokens32) Add(rule pegRule, begin, end, depth uint32, index int) {
	t.tree[index] = token32{pegRule: rule, begin: uint32(begin), end: uint32(end), next: uint32(depth)}
}

func (t *tokens32) Tokens() <-chan token32 {
	s := make(chan token32, 16)
	go func() {
		for _, v := range t.tree {
			s <- v.getToken32()
		}
		close(s)
	}()
	return s
}

func (t *tokens32) Error() []token32 {
	ordered := t.Order()
	length := len(ordered)
	tokens, length := make([]token32, length), length-1
	for i := range tokens {
		o := ordered[length-i]
		if len(o) > 1 {
			tokens[i] = o[len(o)-2].getToken32()
		}
	}
	return tokens
}

/*func (t *tokens16) Expand(index int) tokenTree {
	tree := t.tree
	if index >= len(tree) {
		expanded := make([]token32, 2 * len(tree))
		for i, v := range tree {
			expanded[i] = v.getToken32()
		}
		return &tokens32{tree: expanded}
	}
	return nil
}*/

func (t *tokens32) Expand(index int) tokenTree {
	tree := t.tree
	if index >= len(tree) {
		expanded := make([]token32, 2*len(tree))
		copy(expanded, tree)
		t.tree = expanded
	}
	return nil
}

type DynamlGrammar struct {
	Buffer string
	buffer []rune
	rules  [103]func() bool
	Parse  func(rule ...int) error
	Reset  func()
	Pretty bool
	tokenTree
}

type textPosition struct {
	line, symbol int
}

type textPositionMap map[int]textPosition

func translatePositions(buffer []rune, positions []int) textPositionMap {
	length, translations, j, line, symbol := len(positions), make(textPositionMap, len(positions)), 0, 1, 0
	sort.Ints(positions)

search:
	for i, c := range buffer {
		if c == '\n' {
			line, symbol = line+1, 0
		} else {
			symbol++
		}
		if i == positions[j] {
			translations[positions[j]] = textPosition{line, symbol}
			for j++; j < length; j++ {
				if i != positions[j] {
					continue search
				}
			}
			break search
		}
	}

	return translations
}

type parseError struct {
	p   *DynamlGrammar
	max token32
}

func (e *parseError) Error() string {
	tokens, error := []token32{e.max}, "\n"
	positions, p := make([]int, 2*len(tokens)), 0
	for _, token := range tokens {
		positions[p], p = int(token.begin), p+1
		positions[p], p = int(token.end), p+1
	}
	translations := translatePositions(e.p.buffer, positions)
	format := "parse error near %v (line %v symbol %v - line %v symbol %v):\n%v\n"
	if e.p.Pretty {
		format = "parse error near \x1B[34m%v\x1B[m (line %v symbol %v - line %v symbol %v):\n%v\n"
	}
	for _, token := range tokens {
		begin, end := int(token.begin), int(token.end)
		error += fmt.Sprintf(format,
			rul3s[token.pegRule],
			translations[begin].line, translations[begin].symbol,
			translations[end].line, translations[end].symbol,
			strconv.Quote(string(e.p.buffer[begin:end])))
	}

	return error
}

func (p *DynamlGrammar) PrintSyntaxTree() {
	p.tokenTree.PrintSyntaxTree(p.Buffer)
}

func (p *DynamlGrammar) Highlighter() {
	p.tokenTree.PrintSyntax()
}

func (p *DynamlGrammar) Execute() {
	buffer, _buffer, text, begin, end := p.Buffer, p.buffer, "", 0, 0
	for token := range p.tokenTree.Tokens() {
		switch token.pegRule {

		case ruleAction0:

		case ruleAction1:

		case ruleAction2:

		}
	}
	_, _, _, _, _ = buffer, _buffer, text, begin, end
}

func (p *DynamlGrammar) Init() {
	p.buffer = []rune(p.Buffer)
	if len(p.buffer) == 0 || p.buffer[len(p.buffer)-1] != endSymbol {
		p.buffer = append(p.buffer, endSymbol)
	}

	var tree tokenTree = &tokens32{tree: make([]token32, math.MaxInt16)}
	var max token32
	position, depth, tokenIndex, buffer, _rules := uint32(0), uint32(0), 0, p.buffer, p.rules

	p.Parse = func(rule ...int) error {
		r := 1
		if len(rule) > 0 {
			r = rule[0]
		}
		matches := p.rules[r]()
		p.tokenTree = tree
		if matches {
			p.tokenTree.trim(tokenIndex)
			return nil
		}
		return &parseError{p, max}
	}

	p.Reset = func() {
		position, tokenIndex, depth = 0, 0, 0
	}

	add := func(rule pegRule, begin uint32) {
		if t := tree.Expand(tokenIndex); t != nil {
			tree = t
		}
		tree.Add(rule, begin, position, depth, tokenIndex)
		tokenIndex++
		if begin != position && position > max.end {
			max = token32{rule, begin, position, depth}
		}
	}

	matchDot := func() bool {
		if buffer[position] != endSymbol {
			position++
			return true
		}
		return false
	}

	/*matchChar := func(c byte) bool {
		if buffer[position] == c {
			position++
			return true
		}
		return false
	}*/

	/*matchRange := func(lower byte, upper byte) bool {
		if c := buffer[position]; c >= lower && c <= upper {
			position++
			return true
		}
		return false
	}*/

	_rules = [...]func() bool{
		nil,
		/* 0 Dynaml <- <((Prefer / MarkedExpression / Expression) !.)> */
		func() bool {
			position0, tokenIndex0, depth0 := position, tokenIndex, depth
			{
				position1 := position
				depth++
				{
					position2, tokenIndex2, depth2 := position, tokenIndex, depth
					if !_rules[rulePrefer]() {
						goto l3
					}
					goto l2
				l3:
					position, tokenIndex, depth = position2, tokenIndex2, depth2
					if !_rules[ruleMarkedExpression]() {
						goto l4
					}
					goto l2
				l4:
					position, tokenIndex, depth = position2, tokenIndex2, depth2
					if !_rules[ruleExpression]() {
						goto l0
					}
				}
			l2:
				{
					position5, tokenIndex5, depth5 := position, tokenIndex, depth
					if !matchDot() {
						goto l5
					}
					goto l0
				l5:
					position, tokenIndex, depth = position5, tokenIndex5, depth5
				}
				depth--
				add(ruleDynaml, position1)
			}
			return true
		l0:
			position, tokenIndex, depth = position0, tokenIndex0, depth0
			return false
		},
		/* 1 Prefer <- <(ws ('p' 'r' 'e' 'f' 'e' 'r') req_ws Expression)> */
		func() bool {
			position6, tokenIndex6, depth6 := position, tokenIndex, depth
			{
				position7 := position
				depth++
				if !_rules[rulews]() {
					goto l6
				}
				if buffer[position] != rune('p') {
					goto l6
				}
				position++
				if buffer[position] != rune('r') {
					goto l6
				}
				position++
				if buffer[position] != rune('e') {
					goto l6
				}
				position++
				if buffer[position] != rune('f') {
					goto l6
				}
				position++
				if buffer[position] != rune('e') {
					goto l6
				}
				position++
				if buffer[position] != rune('r') {
					goto l6
				}
				position++
				if !_rules[rulereq_ws]() {
					goto l6
				}
				if !_rules[ruleExpression]() {
					goto l6
				}
				depth--
				add(rulePrefer, position7)
			}
			return true
		l6:
			position, tokenIndex, depth = position6, tokenIndex6, depth6
			return false
		},
		/* 2 MarkedExpression <- <(ws Marker (req_ws SubsequentMarker)* ws MarkerExpression? ws)> */
		func() bool {
			position8, tokenIndex8, depth8 := position, tokenIndex, depth
			{
				position9 := position
				depth++
				if !_rules[rulews]() {
					goto l8
				}
				if !_rules[ruleMarker]() {
					goto l8
				}
			l10:
				{
					position11, tokenIndex11, depth11 := position, tokenIndex, depth
					if !_rules[rulereq_ws]() {
						goto l11
					}
					if !_rules[ruleSubsequentMarker]() {
						goto l11
					}
					goto l10
				l11:
					position, tokenIndex, depth = position11, tokenIndex11, depth11
				}
				if !_rules[rulews]() {
					goto l8
				}
				{
					position12, tokenIndex12, depth12 := position, tokenIndex, depth
					if !_rules[ruleMarkerExpression]() {
						goto l12
					}
					goto l13
				l12:
					position, tokenIndex, depth = position12, tokenIndex12, depth12
				}
			l13:
				if !_rules[rulews]() {
					goto l8
				}
				depth--
				add(ruleMarkedExpression, position9)
			}
			return true
		l8:
			position, tokenIndex, depth = position8, tokenIndex8, depth8
			return false
		},
		/* 3 SubsequentMarker <- <Marker> */
		func() bool {
			position14, tokenIndex14, depth14 := position, tokenIndex, depth
			{
				position15 := position
				depth++
				if !_rules[ruleMarker]() {
					goto l14
				}
				depth--
				add(ruleSubsequentMarker, position15)
			}
			return true
		l14:
			position, tokenIndex, depth = position14, tokenIndex14, depth14
			return false
		},
		/* 4 Marker <- <('&' (('t' 'e' 'm' 'p' 'l' 'a' 't' 'e') / ('t' 'e' 'm' 'p' 'o' 'r' 'a' 'r' 'y') / ('l' 'o' 'c' 'a' 'l') / ('i' 'n' 'j' 'e' 'c' 't') / ('s' 't' 'a' 't' 'e') / ('d' 'e' 'f' 'a' 'u' 'l' 't')))> */
		func() bool {
			position16, tokenIndex16, depth16 := position, tokenIndex, depth
			{
				position17 := position
				depth++
				if buffer[position] != rune('&') {
					goto l16
				}
				position++
				{
					position18, tokenIndex18, depth18 := position, tokenIndex, depth
					if buffer[position] != rune('t') {
						goto l19
					}
					position++
					if buffer[position] != rune('e') {
						goto l19
					}
					position++
					if buffer[position] != rune('m') {
						goto l19
					}
					position++
					if buffer[position] != rune('p') {
						goto l19
					}
					position++
					if buffer[position] != rune('l') {
						goto l19
					}
					position++
					if buffer[position] != rune('a') {
						goto l19
					}
					position++
					if buffer[position] != rune('t') {
						goto l19
					}
					position++
					if buffer[position] != rune('e') {
						goto l19
					}
					position++
					goto l18
				l19:
					position, tokenIndex, depth = position18, tokenIndex18, depth18
					if buffer[position] != rune('t') {
						goto l20
					}
					position++
					if buffer[position] != rune('e') {
						goto l20
					}
					position++
					if buffer[position] != rune('m') {
						goto l20
					}
					position++
					if buffer[position] != rune('p') {
						goto l20
					}
					position++
					if buffer[position] != rune('o') {
						goto l20
					}
					position++
					if buffer[position] != rune('r') {
						goto l20
					}
					position++
					if buffer[position] != rune('a') {
						goto l20
					}
					position++
					if buffer[position] != rune('r') {
						goto l20
					}
					position++
					if buffer[position] != rune('y') {
						goto l20
					}
					position++
					goto l18
				l20:
					position, tokenIndex, depth = position18, tokenIndex18, depth18
					if buffer[position] != rune('l') {
						goto l21
					}
					position++
					if buffer[position] != rune('o') {
						goto l21
					}
					position++
					if buffer[position] != rune('c') {
						goto l21
					}
					position++
					if buffer[position] != rune('a') {
						goto l21
					}
					position++
					if buffer[position] != rune('l') {
						goto l21
					}
					position++
					goto l18
				l21:
					position, tokenIndex, depth = position18, tokenIndex18, depth18
					if buffer[position] != rune('i') {
						goto l22
					}
					position++
					if buffer[position] != rune('n') {
						goto l22
					}
					position++
					if buffer[position] != rune('j') {
						goto l22
					}
					position++
					if buffer[position] != rune('e') {
						goto l22
					}
					position++
					if buffer[position] != rune('c') {
						goto l22
					}
					position++
					if buffer[position] != rune('t') {
						goto l22
					}
					position++
					goto l18
				l22:
					position, tokenIndex, depth = position18, tokenIndex18, depth18
					if buffer[position] != rune('s') {
						goto l23
					}
					position++
					if buffer[position] != rune('t') {
						goto l23
					}
					position++
					if buffer[position] != rune('a') {
						goto l23
					}
					position++
					if buffer[position] != rune('t') {
						goto l23
					}
					position++
					if buffer[position] != rune('e') {
						goto l23
					}
					position++
					goto l18
				l23:
					position, tokenIndex, depth = position18, tokenIndex18, depth18
					if buffer[position] != rune('d') {
						goto l16
					}
					position++
					if buffer[position] != rune('e') {
						goto l16
					}
					position++
					if buffer[position] != rune('f') {
						goto l16
					}
					position++
					if buffer[position] != rune('a') {
						goto l16
					}
					position++
					if buffer[position] != rune('u') {
						goto l16
					}
					position++
					if buffer[position] != rune('l') {
						goto l16
					}
					position++
					if buffer[position] != rune('t') {
						goto l16
					}
					position++
				}
			l18:
				depth--
				add(ruleMarker, position17)
			}
			return true
		l16:
			position, tokenIndex, depth = position16, tokenIndex16, depth16
			return false
		},
		/* 5 MarkerExpression <- <Grouped> */
		func() bool {
			position24, tokenIndex24, depth24 := position, tokenIndex, depth
			{
				position25 := position
				depth++
				if !_rules[ruleGrouped]() {
					goto l24
				}
				depth--
				add(ruleMarkerExpression, position25)
			}
			return true
		l24:
			position, tokenIndex, depth = position24, tokenIndex24, depth24
			return false
		},
		/* 6 Expression <- <((Scoped / LambdaExpr / Level7) ws)> */
		func() bool {
			position26, tokenIndex26, depth26 := position, tokenIndex, depth
			{
				position27 := position
				depth++
				{
					position28, tokenIndex28, depth28 := position, tokenIndex, depth
					if !_rules[ruleScoped]() {
						goto l29
					}
					goto l28
				l29:
					position, tokenIndex, depth = position28, tokenIndex28, depth28
					if !_rules[ruleLambdaExpr]() {
						goto l30
					}
					goto l28
				l30:
					position, tokenIndex, depth = position28, tokenIndex28, depth28
					if !_rules[ruleLevel7]() {
						goto l26
					}
				}
			l28:
				if !_rules[rulews]() {
					goto l26
				}
				depth--
				add(ruleExpression, position27)
			}
			return true
		l26:
			position, tokenIndex, depth = position26, tokenIndex26, depth26
			return false
		},
		/* 7 Scoped <- <(ws Scope ws Expression)> */
		func() bool {
			position31, tokenIndex31, depth31 := position, tokenIndex, depth
			{
				position32 := position
				depth++
				if !_rules[rulews]() {
					goto l31
				}
				if !_rules[ruleScope]() {
					goto l31
				}
				if !_rules[rulews]() {
					goto l31
				}
				if !_rules[ruleExpression]() {
					goto l31
				}
				depth--
				add(ruleScoped, position32)
			}
			return true
		l31:
			position, tokenIndex, depth = position31, tokenIndex31, depth31
			return false
		},
		/* 8 Scope <- <(CreateScope ws Assignments? ')')> */
		func() bool {
			position33, tokenIndex33, depth33 := position, tokenIndex, depth
			{
				position34 := position
				depth++
				if !_rules[ruleCreateScope]() {
					goto l33
				}
				if !_rules[rulews]() {
					goto l33
				}
				{
					position35, tokenIndex35, depth35 := position, tokenIndex, depth
					if !_rules[ruleAssignments]() {
						goto l35
					}
					goto l36
				l35:
					position, tokenIndex, depth = position35, tokenIndex35, depth35
				}
			l36:
				if buffer[position] != rune(')') {
					goto l33
				}
				position++
				depth--
				add(ruleScope, position34)
			}
			return true
		l33:
			position, tokenIndex, depth = position33, tokenIndex33, depth33
			return false
		},
		/* 9 CreateScope <- <'('> */
		func() bool {
			position37, tokenIndex37, depth37 := position, tokenIndex, depth
			{
				position38 := position
				depth++
				if buffer[position] != rune('(') {
					goto l37
				}
				position++
				depth--
				add(ruleCreateScope, position38)
			}
			return true
		l37:
			position, tokenIndex, depth = position37, tokenIndex37, depth37
			return false
		},
		/* 10 Level7 <- <(ws Level6 (req_ws Or)*)> */
		func() bool {
			position39, tokenIndex39, depth39 := position, tokenIndex, depth
			{
				position40 := position
				depth++
				if !_rules[rulews]() {
					goto l39
				}
				if !_rules[ruleLevel6]() {
					goto l39
				}
			l41:
				{
					position42, tokenIndex42, depth42 := position, tokenIndex, depth
					if !_rules[rulereq_ws]() {
						goto l42
					}
					if !_rules[ruleOr]() {
						goto l42
					}
					goto l41
				l42:
					position, tokenIndex, depth = position42, tokenIndex42, depth42
				}
				depth--
				add(ruleLevel7, position40)
			}
			return true
		l39:
			position, tokenIndex, depth = position39, tokenIndex39, depth39
			return false
		},
		/* 11 Or <- <(OrOp req_ws Level6)> */
		func() bool {
			position43, tokenIndex43, depth43 := position, tokenIndex, depth
			{
				position44 := position
				depth++
				if !_rules[ruleOrOp]() {
					goto l43
				}
				if !_rules[rulereq_ws]() {
					goto l43
				}
				if !_rules[ruleLevel6]() {
					goto l43
				}
				depth--
				add(ruleOr, position44)
			}
			return true
		l43:
			position, tokenIndex, depth = position43, tokenIndex43, depth43
			return false
		},
		/* 12 OrOp <- <(('|' '|') / ('/' '/'))> */
		func() bool {
			position45, tokenIndex45, depth45 := position, tokenIndex, depth
			{
				position46 := position
				depth++
				{
					position47, tokenIndex47, depth47 := position, tokenIndex, depth
					if buffer[position] != rune('|') {
						goto l48
					}
					position++
					if buffer[position] != rune('|') {
						goto l48
					}
					position++
					goto l47
				l48:
					position, tokenIndex, depth = position47, tokenIndex47, depth47
					if buffer[position] != rune('/') {
						goto l45
					}
					position++
					if buffer[position] != rune('/') {
						goto l45
					}
					position++
				}
			l47:
				depth--
				add(ruleOrOp, position46)
			}
			return true
		l45:
			position, tokenIndex, depth = position45, tokenIndex45, depth45
			return false
		},
		/* 13 Level6 <- <(Conditional / Level5)> */
		func() bool {
			position49, tokenIndex49, depth49 := position, tokenIndex, depth
			{
				position50 := position
				depth++
				{
					position51, tokenIndex51, depth51 := position, tokenIndex, depth
					if !_rules[ruleConditional]() {
						goto l52
					}
					goto l51
				l52:
					position, tokenIndex, depth = position51, tokenIndex51, depth51
					if !_rules[ruleLevel5]() {
						goto l49
					}
				}
			l51:
				depth--
				add(ruleLevel6, position50)
			}
			return true
		l49:
			position, tokenIndex, depth = position49, tokenIndex49, depth49
			return false
		},
		/* 14 Conditional <- <(Level5 ws '?' Expression ':' Expression)> */
		func() bool {
			position53, tokenIndex53, depth53 := position, tokenIndex, depth
			{
				position54 := position
				depth++
				if !_rules[ruleLevel5]() {
					goto l53
				}
				if !_rules[rulews]() {
					goto l53
				}
				if buffer[position] != rune('?') {
					goto l53
				}
				position++
				if !_rules[ruleExpression]() {
					goto l53
				}
				if buffer[position] != rune(':') {
					goto l53
				}
				position++
				if !_rules[ruleExpression]() {
					goto l53
				}
				depth--
				add(ruleConditional, position54)
			}
			return true
		l53:
			position, tokenIndex, depth = position53, tokenIndex53, depth53
			return false
		},
		/* 15 Level5 <- <(Level4 Concatenation*)> */
		func() bool {
			position55, tokenIndex55, depth55 := position, tokenIndex, depth
			{
				position56 := position
				depth++
				if !_rules[ruleLevel4]() {
					goto l55
				}
			l57:
				{
					position58, tokenIndex58, depth58 := position, tokenIndex, depth
					if !_rules[ruleConcatenation]() {
						goto l58
					}
					goto l57
				l58:
					position, tokenIndex, depth = position58, tokenIndex58, depth58
				}
				depth--
				add(ruleLevel5, position56)
			}
			return true
		l55:
			position, tokenIndex, depth = position55, tokenIndex55, depth55
			return false
		},
		/* 16 Concatenation <- <(req_ws Level4)> */
		func() bool {
			position59, tokenIndex59, depth59 := position, tokenIndex, depth
			{
				position60 := position
				depth++
				if !_rules[rulereq_ws]() {
					goto l59
				}
				if !_rules[ruleLevel4]() {
					goto l59
				}
				depth--
				add(ruleConcatenation, position60)
			}
			return true
		l59:
			position, tokenIndex, depth = position59, tokenIndex59, depth59
			return false
		},
		/* 17 Level4 <- <(Level3 (req_ws (LogOr / LogAnd))*)> */
		func() bool {
			position61, tokenIndex61, depth61 := position, tokenIndex, depth
			{
				position62 := position
				depth++
				if !_rules[ruleLevel3]() {
					goto l61
				}
			l63:
				{
					position64, tokenIndex64, depth64 := position, tokenIndex, depth
					if !_rules[rulereq_ws]() {
						goto l64
					}
					{
						position65, tokenIndex65, depth65 := position, tokenIndex, depth
						if !_rules[ruleLogOr]() {
							goto l66
						}
						goto l65
					l66:
						position, tokenIndex, depth = position65, tokenIndex65, depth65
						if !_rules[ruleLogAnd]() {
							goto l64
						}
					}
				l65:
					goto l63
				l64:
					position, tokenIndex, depth = position64, tokenIndex64, depth64
				}
				depth--
				add(ruleLevel4, position62)
			}
			return true
		l61:
			position, tokenIndex, depth = position61, tokenIndex61, depth61
			return false
		},
		/* 18 LogOr <- <('-' 'o' 'r' req_ws Level3)> */
		func() bool {
			position67, tokenIndex67, depth67 := position, tokenIndex, depth
			{
				position68 := position
				depth++
				if buffer[position] != rune('-') {
					goto l67
				}
				position++
				if buffer[position] != rune('o') {
					goto l67
				}
				position++
				if buffer[position] != rune('r') {
					goto l67
				}
				position++
				if !_rules[rulereq_ws]() {
					goto l67
				}
				if !_rules[ruleLevel3]() {
					goto l67
				}
				depth--
				add(ruleLogOr, position68)
			}
			return true
		l67:
			position, tokenIndex, depth = position67, tokenIndex67, depth67
			return false
		},
		/* 19 LogAnd <- <('-' 'a' 'n' 'd' req_ws Level3)> */
		func() bool {
			position69, tokenIndex69, depth69 := position, tokenIndex, depth
			{
				position70 := position
				depth++
				if buffer[position] != rune('-') {
					goto l69
				}
				position++
				if buffer[position] != rune('a') {
					goto l69
				}
				position++
				if buffer[position] != rune('n') {
					goto l69
				}
				position++
				if buffer[position] != rune('d') {
					goto l69
				}
				position++
				if !_rules[rulereq_ws]() {
					goto l69
				}
				if !_rules[ruleLevel3]() {
					goto l69
				}
				depth--
				add(ruleLogAnd, position70)
			}
			return true
		l69:
			position, tokenIndex, depth = position69, tokenIndex69, depth69
			return false
		},
		/* 20 Level3 <- <(Level2 (req_ws Comparison)*)> */
		func() bool {
			position71, tokenIndex71, depth71 := position, tokenIndex, depth
			{
				position72 := position
				depth++
				if !_rules[ruleLevel2]() {
					goto l71
				}
			l73:
				{
					position74, tokenIndex74, depth74 := position, tokenIndex, depth
					if !_rules[rulereq_ws]() {
						goto l74
					}
					if !_rules[ruleComparison]() {
						goto l74
					}
					goto l73
				l74:
					position, tokenIndex, depth = position74, tokenIndex74, depth74
				}
				depth--
				add(ruleLevel3, position72)
			}
			return true
		l71:
			position, tokenIndex, depth = position71, tokenIndex71, depth71
			return false
		},
		/* 21 Comparison <- <(CompareOp req_ws Level2)> */
		func() bool {
			position75, tokenIndex75, depth75 := position, tokenIndex, depth
			{
				position76 := position
				depth++
				if !_rules[ruleCompareOp]() {
					goto l75
				}
				if !_rules[rulereq_ws]() {
					goto l75
				}
				if !_rules[ruleLevel2]() {
					goto l75
				}
				depth--
				add(ruleComparison, position76)
			}
			return true
		l75:
			position, tokenIndex, depth = position75, tokenIndex75, depth75
			return false
		},
		/* 22 CompareOp <- <(('=' '=') / ('!' '=') / ('<' '=') / ('>' '=') / '>' / '<' / '>')> */
		func() bool {
			position77, tokenIndex77, depth77 := position, tokenIndex, depth
			{
				position78 := position
				depth++
				{
					position79, tokenIndex79, depth79 := position, tokenIndex, depth
					if buffer[position] != rune('=') {
						goto l80
					}
					position++
					if buffer[position] != rune('=') {
						goto l80
					}
					position++
					goto l79
				l80:
					position, tokenIndex, depth = position79, tokenIndex79, depth79
					if buffer[position] != rune('!') {
						goto l81
					}
					position++
					if buffer[position] != rune('=') {
						goto l81
					}
					position++
					goto l79
				l81:
					position, tokenIndex, depth = position79, tokenIndex79, depth79
					if buffer[position] != rune('<') {
						goto l82
					}
					position++
					if buffer[position] != rune('=') {
						goto l82
					}
					position++
					goto l79
				l82:
					position, tokenIndex, depth = position79, tokenIndex79, depth79
					if buffer[position] != rune('>') {
						goto l83
					}
					position++
					if buffer[position] != rune('=') {
						goto l83
					}
					position++
					goto l79
				l83:
					position, tokenIndex, depth = position79, tokenIndex79, depth79
					if buffer[position] != rune('>') {
						goto l84
					}
					position++
					goto l79
				l84:
					position, tokenIndex, depth = position79, tokenIndex79, depth79
					if buffer[position] != rune('<') {
						goto l85
					}
					position++
					goto l79
				l85:
					position, tokenIndex, depth = position79, tokenIndex79, depth79
					if buffer[position] != rune('>') {
						goto l77
					}
					position++
				}
			l79:
				depth--
				add(ruleCompareOp, position78)
			}
			return true
		l77:
			position, tokenIndex, depth = position77, tokenIndex77, depth77
			return false
		},
		/* 23 Level2 <- <(Level1 (req_ws (Addition / Subtraction))*)> */
		func() bool {
			position86, tokenIndex86, depth86 := position, tokenIndex, depth
			{
				position87 := position
				depth++
				if !_rules[ruleLevel1]() {
					goto l86
				}
			l88:
				{
					position89, tokenIndex89, depth89 := position, tokenIndex, depth
					if !_rules[rulereq_ws]() {
						goto l89
					}
					{
						position90, tokenIndex90, depth90 := position, tokenIndex, depth
						if !_rules[ruleAddition]() {
							goto l91
						}
						goto l90
					l91:
						position, tokenIndex, depth = position90, tokenIndex90, depth90
						if !_rules[ruleSubtraction]() {
							goto l89
						}
					}
				l90:
					goto l88
				l89:
					position, tokenIndex, depth = position89, tokenIndex89, depth89
				}
				depth--
				add(ruleLevel2, position87)
			}
			return true
		l86:
			position, tokenIndex, depth = position86, tokenIndex86, depth86
			return false
		},
		/* 24 Addition <- <('+' req_ws Level1)> */
		func() bool {
			position92, tokenIndex92, depth92 := position, tokenIndex, depth
			{
				position93 := position
				depth++
				if buffer[position] != rune('+') {
					goto l92
				}
				position++
				if !_rules[rulereq_ws]() {
					goto l92
				}
				if !_rules[ruleLevel1]() {
					goto l92
				}
				depth--
				add(ruleAddition, position93)
			}
			return true
		l92:
			position, tokenIndex, depth = position92, tokenIndex92, depth92
			return false
		},
		/* 25 Subtraction <- <('-' req_ws Level1)> */
		func() bool {
			position94, tokenIndex94, depth94 := position, tokenIndex, depth
			{
				position95 := position
				depth++
				if buffer[position] != rune('-') {
					goto l94
				}
				position++
				if !_rules[rulereq_ws]() {
					goto l94
				}
				if !_rules[ruleLevel1]() {
					goto l94
				}
				depth--
				add(ruleSubtraction, position95)
			}
			return true
		l94:
			position, tokenIndex, depth = position94, tokenIndex94, depth94
			return false
		},
		/* 26 Level1 <- <(Level0 (req_ws (Multiplication / Division / Modulo))*)> */
		func() bool {
			position96, tokenIndex96, depth96 := position, tokenIndex, depth
			{
				position97 := position
				depth++
				if !_rules[ruleLevel0]() {
					goto l96
				}
			l98:
				{
					position99, tokenIndex99, depth99 := position, tokenIndex, depth
					if !_rules[rulereq_ws]() {
						goto l99
					}
					{
						position100, tokenIndex100, depth100 := position, tokenIndex, depth
						if !_rules[ruleMultiplication]() {
							goto l101
						}
						goto l100
					l101:
						position, tokenIndex, depth = position100, tokenIndex100, depth100
						if !_rules[ruleDivision]() {
							goto l102
						}
						goto l100
					l102:
						position, tokenIndex, depth = position100, tokenIndex100, depth100
						if !_rules[ruleModulo]() {
							goto l99
						}
					}
				l100:
					goto l98
				l99:
					position, tokenIndex, depth = position99, tokenIndex99, depth99
				}
				depth--
				add(ruleLevel1, position97)
			}
			return true
		l96:
			position, tokenIndex, depth = position96, tokenIndex96, depth96
			return false
		},
		/* 27 Multiplication <- <('*' req_ws Level0)> */
		func() bool {
			position103, tokenIndex103, depth103 := position, tokenIndex, depth
			{
				position104 := position
				depth++
				if buffer[position] != rune('*') {
					goto l103
				}
				position++
				if !_rules[rulereq_ws]() {
					goto l103
				}
				if !_rules[ruleLevel0]() {
					goto l103
				}
				depth--
				add(ruleMultiplication, position104)
			}
			return true
		l103:
			position, tokenIndex, depth = position103, tokenIndex103, depth103
			return false
		},
		/* 28 Division <- <('/' req_ws Level0)> */
		func() bool {
			position105, tokenIndex105, depth105 := position, tokenIndex, depth
			{
				position106 := position
				depth++
				if buffer[position] != rune('/') {
					goto l105
				}
				position++
				if !_rules[rulereq_ws]() {
					goto l105
				}
				if !_rules[ruleLevel0]() {
					goto l105
				}
				depth--
				add(ruleDivision, position106)
			}
			return true
		l105:
			position, tokenIndex, depth = position105, tokenIndex105, depth105
			return false
		},
		/* 29 Modulo <- <('%' req_ws Level0)> */
		func() bool {
			position107, tokenIndex107, depth107 := position, tokenIndex, depth
			{
				position108 := position
				depth++
				if buffer[position] != rune('%') {
					goto l107
				}
				position++
				if !_rules[rulereq_ws]() {
					goto l107
				}
				if !_rules[ruleLevel0]() {
					goto l107
				}
				depth--
				add(ruleModulo, position108)
			}
			return true
		l107:
			position, tokenIndex, depth = position107, tokenIndex107, depth107
			return false
		},
		/* 30 Level0 <- <(IP / String / Number / Boolean / Undefined / Nil / Symbol / Not / Substitution / Merge / Auto / Lambda / Chained)> */
		func() bool {
			position109, tokenIndex109, depth109 := position, tokenIndex, depth
			{
				position110 := position
				depth++
				{
					position111, tokenIndex111, depth111 := position, tokenIndex, depth
					if !_rules[ruleIP]() {
						goto l112
					}
					goto l111
				l112:
					position, tokenIndex, depth = position111, tokenIndex111, depth111
					if !_rules[ruleString]() {
						goto l113
					}
					goto l111
				l113:
					position, tokenIndex, depth = position111, tokenIndex111, depth111
					if !_rules[ruleNumber]() {
						goto l114
					}
					goto l111
				l114:
					position, tokenIndex, depth = position111, tokenIndex111, depth111
					if !_rules[ruleBoolean]() {
						goto l115
					}
					goto l111
				l115:
					position, tokenIndex, depth = position111, tokenIndex111, depth111
					if !_rules[ruleUndefined]() {
						goto l116
					}
					goto l111
				l116:
					position, tokenIndex, depth = position111, tokenIndex111, depth111
					if !_rules[ruleNil]() {
						goto l117
					}
					goto l111
				l117:
					position, tokenIndex, depth = position111, tokenIndex111, depth111
					if !_rules[ruleSymbol]() {
						goto l118
					}
					goto l111
				l118:
					position, tokenIndex, depth = position111, tokenIndex111, depth111
					if !_rules[ruleNot]() {
						goto l119
					}
					goto l111
				l119:
					position, tokenIndex, depth = position111, tokenIndex111, depth111
					if !_rules[ruleSubstitution]() {
						goto l120
					}
					goto l111
				l120:
					position, tokenIndex, depth = position111, tokenIndex111, depth111
					if !_rules[ruleMerge]() {
						goto l121
					}
					goto l111
				l121:
					position, tokenIndex, depth = position111, tokenIndex111, depth111
					if !_rules[ruleAuto]() {
						goto l122
					}
					goto l111
				l122:
					position, tokenIndex, depth = position111, tokenIndex111, depth111
					if !_rules[ruleLambda]() {
						goto l123
					}
					goto l111
				l123:
					position, tokenIndex, depth = position111, tokenIndex111, depth111
					if !_rules[ruleChained]() {
						goto l109
					}
				}
			l111:
				depth--
				add(ruleLevel0, position110)
			}
			return true
		l109:
			position, tokenIndex, depth = position109, tokenIndex109, depth109
			return false
		},
		/* 31 Chained <- <((MapMapping / Sync / Catch / Mapping / MapSelection / Selection / Sum / List / Map / Range / Grouped / Reference) ChainedQualifiedExpression*)> */
		func() bool {
			position124, tokenIndex124, depth124 := position, tokenIndex, depth
			{
				position125 := position
				depth++
				{
					position126, tokenIndex126, depth126 := position, tokenIndex, depth
					if !_rules[ruleMapMapping]() {
						goto l127
					}
					goto l126
				l127:
					position, tokenIndex, depth = position126, tokenIndex126, depth126
					if !_rules[ruleSync]() {
						goto l128
					}
					goto l126
				l128:
					position, tokenIndex, depth = position126, tokenIndex126, depth126
					if !_rules[ruleCatch]() {
						goto l129
					}
					goto l126
				l129:
					position, tokenIndex, depth = position126, tokenIndex126, depth126
					if !_rules[ruleMapping]() {
						goto l130
					}
					goto l126
				l130:
					position, tokenIndex, depth = position126, tokenIndex126, depth126
					if !_rules[ruleMapSelection]() {
						goto l131
					}
					goto l126
				l131:
					position, tokenIndex, depth = position126, tokenIndex126, depth126
					if !_rules[ruleSelection]() {
						goto l132
					}
					goto l126
				l132:
					position, tokenIndex, depth = position126, tokenIndex126, depth126
					if !_rules[ruleSum]() {
						goto l133
					}
					goto l126
				l133:
					position, tokenIndex, depth = position126, tokenIndex126, depth126
					if !_rules[ruleList]() {
						goto l134
					}
					goto l126
				l134:
					position, tokenIndex, depth = position126, tokenIndex126, depth126
					if !_rules[ruleMap]() {
						goto l135
					}
					goto l126
				l135:
					position, tokenIndex, depth = position126, tokenIndex126, depth126
					if !_rules[ruleRange]() {
						goto l136
					}
					goto l126
				l136:
					position, tokenIndex, depth = position126, tokenIndex126, depth126
					if !_rules[ruleGrouped]() {
						goto l137
					}
					goto l126
				l137:
					position, tokenIndex, depth = position126, tokenIndex126, depth126
					if !_rules[ruleReference]() {
						goto l124
					}
				}
			l126:
			l138:
				{
					position139, tokenIndex139, depth139 := position, tokenIndex, depth
					if !_rules[ruleChainedQualifiedExpression]() {
						goto l139
					}
					goto l138
				l139:
					position, tokenIndex, depth = position139, tokenIndex139, depth139
				}
				depth--
				add(ruleChained, position125)
			}
			return true
		l124:
			position, tokenIndex, depth = position124, tokenIndex124, depth124
			return false
		},
		/* 32 ChainedQualifiedExpression <- <(ChainedCall / Currying / ChainedRef / ChainedDynRef / Projection)> */
		func() bool {
			position140, tokenIndex140, depth140 := position, tokenIndex, depth
			{
				position141 := position
				depth++
				{
					position142, tokenIndex142, depth142 := position, tokenIndex, depth
					if !_rules[ruleChainedCall]() {
						goto l143
					}
					goto l142
				l143:
					position, tokenIndex, depth = position142, tokenIndex142, depth142
					if !_rules[ruleCurrying]() {
						goto l144
					}
					goto l142
				l144:
					position, tokenIndex, depth = position142, tokenIndex142, depth142
					if !_rules[ruleChainedRef]() {
						goto l145
					}
					goto l142
				l145:
					position, tokenIndex, depth = position142, tokenIndex142, depth142
					if !_rules[ruleChainedDynRef]() {
						goto l146
					}
					goto l142
				l146:
					position, tokenIndex, depth = position142, tokenIndex142, depth142
					if !_rules[ruleProjection]() {
						goto l140
					}
				}
			l142:
				depth--
				add(ruleChainedQualifiedExpression, position141)
			}
			return true
		l140:
			position, tokenIndex, depth = position140, tokenIndex140, depth140
			return false
		},
		/* 33 ChainedRef <- <(PathComponent FollowUpRef)> */
		func() bool {
			position147, tokenIndex147, depth147 := position, tokenIndex, depth
			{
				position148 := position
				depth++
				if !_rules[rulePathComponent]() {
					goto l147
				}
				if !_rules[ruleFollowUpRef]() {
					goto l147
				}
				depth--
				add(ruleChainedRef, position148)
			}
			return true
		l147:
			position, tokenIndex, depth = position147, tokenIndex147, depth147
			return false
		},
		/* 34 ChainedDynRef <- <('.'? '[' Expression ']')> */
		func() bool {
			position149, tokenIndex149, depth149 := position, tokenIndex, depth
			{
				position150 := position
				depth++
				{
					position151, tokenIndex151, depth151 := position, tokenIndex, depth
					if buffer[position] != rune('.') {
						goto l151
					}
					position++
					goto l152
				l151:
					position, tokenIndex, depth = position151, tokenIndex151, depth151
				}
			l152:
				if buffer[position] != rune('[') {
					goto l149
				}
				position++
				if !_rules[ruleExpression]() {
					goto l149
				}
				if buffer[position] != rune(']') {
					goto l149
				}
				position++
				depth--
				add(ruleChainedDynRef, position150)
			}
			return true
		l149:
			position, tokenIndex, depth = position149, tokenIndex149, depth149
			return false
		},
		/* 35 Slice <- <Range> */
		func() bool {
			position153, tokenIndex153, depth153 := position, tokenIndex, depth
			{
				position154 := position
				depth++
				if !_rules[ruleRange]() {
					goto l153
				}
				depth--
				add(ruleSlice, position154)
			}
			return true
		l153:
			position, tokenIndex, depth = position153, tokenIndex153, depth153
			return false
		},
		/* 36 Currying <- <('*' ChainedCall)> */
		func() bool {
			position155, tokenIndex155, depth155 := position, tokenIndex, depth
			{
				position156 := position
				depth++
				if buffer[position] != rune('*') {
					goto l155
				}
				position++
				if !_rules[ruleChainedCall]() {
					goto l155
				}
				depth--
				add(ruleCurrying, position156)
			}
			return true
		l155:
			position, tokenIndex, depth = position155, tokenIndex155, depth155
			return false
		},
		/* 37 ChainedCall <- <(StartArguments NameArgumentList? ')')> */
		func() bool {
			position157, tokenIndex157, depth157 := position, tokenIndex, depth
			{
				position158 := position
				depth++
				if !_rules[ruleStartArguments]() {
					goto l157
				}
				{
					position159, tokenIndex159, depth159 := position, tokenIndex, depth
					if !_rules[ruleNameArgumentList]() {
						goto l159
					}
					goto l160
				l159:
					position, tokenIndex, depth = position159, tokenIndex159, depth159
				}
			l160:
				if buffer[position] != rune(')') {
					goto l157
				}
				position++
				depth--
				add(ruleChainedCall, position158)
			}
			return true
		l157:
			position, tokenIndex, depth = position157, tokenIndex157, depth157
			return false
		},
		/* 38 StartArguments <- <('(' ws)> */
		func() bool {
			position161, tokenIndex161, depth161 := position, tokenIndex, depth
			{
				position162 := position
				depth++
				if buffer[position] != rune('(') {
					goto l161
				}
				position++
				if !_rules[rulews]() {
					goto l161
				}
				depth--
				add(ruleStartArguments, position162)
			}
			return true
		l161:
			position, tokenIndex, depth = position161, tokenIndex161, depth161
			return false
		},
		/* 39 NameArgumentList <- <(((NextNameArgument (',' NextNameArgument)*) / NextExpression) (',' NextExpression)*)> */
		func() bool {
			position163, tokenIndex163, depth163 := position, tokenIndex, depth
			{
				position164 := position
				depth++
				{
					position165, tokenIndex165, depth165 := position, tokenIndex, depth
					if !_rules[ruleNextNameArgument]() {
						goto l166
					}
				l167:
					{
						position168, tokenIndex168, depth168 := position, tokenIndex, depth
						if buffer[position] != rune(',') {
							goto l168
						}
						position++
						if !_rules[ruleNextNameArgument]() {
							goto l168
						}
						goto l167
					l168:
						position, tokenIndex, depth = position168, tokenIndex168, depth168
					}
					goto l165
				l166:
					position, tokenIndex, depth = position165, tokenIndex165, depth165
					if !_rules[ruleNextExpression]() {
						goto l163
					}
				}
			l165:
			l169:
				{
					position170, tokenIndex170, depth170 := position, tokenIndex, depth
					if buffer[position] != rune(',') {
						goto l170
					}
					position++
					if !_rules[ruleNextExpression]() {
						goto l170
					}
					goto l169
				l170:
					position, tokenIndex, depth = position170, tokenIndex170, depth170
				}
				depth--
				add(ruleNameArgumentList, position164)
			}
			return true
		l163:
			position, tokenIndex, depth = position163, tokenIndex163, depth163
			return false
		},
		/* 40 NextNameArgument <- <(ws Name ws '=' ws Expression ws)> */
		func() bool {
			position171, tokenIndex171, depth171 := position, tokenIndex, depth
			{
				position172 := position
				depth++
				if !_rules[rulews]() {
					goto l171
				}
				if !_rules[ruleName]() {
					goto l171
				}
				if !_rules[rulews]() {
					goto l171
				}
				if buffer[position] != rune('=') {
					goto l171
				}
				position++
				if !_rules[rulews]() {
					goto l171
				}
				if !_rules[ruleExpression]() {
					goto l171
				}
				if !_rules[rulews]() {
					goto l171
				}
				depth--
				add(ruleNextNameArgument, position172)
			}
			return true
		l171:
			position, tokenIndex, depth = position171, tokenIndex171, depth171
			return false
		},
		/* 41 ExpressionList <- <(NextExpression (',' NextExpression)*)> */
		func() bool {
			position173, tokenIndex173, depth173 := position, tokenIndex, depth
			{
				position174 := position
				depth++
				if !_rules[ruleNextExpression]() {
					goto l173
				}
			l175:
				{
					position176, tokenIndex176, depth176 := position, tokenIndex, depth
					if buffer[position] != rune(',') {
						goto l176
					}
					position++
					if !_rules[ruleNextExpression]() {
						goto l176
					}
					goto l175
				l176:
					position, tokenIndex, depth = position176, tokenIndex176, depth176
				}
				depth--
				add(ruleExpressionList, position174)
			}
			return true
		l173:
			position, tokenIndex, depth = position173, tokenIndex173, depth173
			return false
		},
		/* 42 NextExpression <- <(Expression ListExpansion?)> */
		func() bool {
			position177, tokenIndex177, depth177 := position, tokenIndex, depth
			{
				position178 := position
				depth++
				if !_rules[ruleExpression]() {
					goto l177
				}
				{
					position179, tokenIndex179, depth179 := position, tokenIndex, depth
					if !_rules[ruleListExpansion]() {
						goto l179
					}
					goto l180
				l179:
					position, tokenIndex, depth = position179, tokenIndex179, depth179
				}
			l180:
				depth--
				add(ruleNextExpression, position178)
			}
			return true
		l177:
			position, tokenIndex, depth = position177, tokenIndex177, depth177
			return false
		},
		/* 43 ListExpansion <- <('.' '.' '.' ws)> */
		func() bool {
			position181, tokenIndex181, depth181 := position, tokenIndex, depth
			{
				position182 := position
				depth++
				if buffer[position] != rune('.') {
					goto l181
				}
				position++
				if buffer[position] != rune('.') {
					goto l181
				}
				position++
				if buffer[position] != rune('.') {
					goto l181
				}
				position++
				if !_rules[rulews]() {
					goto l181
				}
				depth--
				add(ruleListExpansion, position182)
			}
			return true
		l181:
			position, tokenIndex, depth = position181, tokenIndex181, depth181
			return false
		},
		/* 44 Projection <- <('.'? (('[' '*' ']') / Slice) ProjectionValue ChainedQualifiedExpression*)> */
		func() bool {
			position183, tokenIndex183, depth183 := position, tokenIndex, depth
			{
				position184 := position
				depth++
				{
					position185, tokenIndex185, depth185 := position, tokenIndex, depth
					if buffer[position] != rune('.') {
						goto l185
					}
					position++
					goto l186
				l185:
					position, tokenIndex, depth = position185, tokenIndex185, depth185
				}
			l186:
				{
					position187, tokenIndex187, depth187 := position, tokenIndex, depth
					if buffer[position] != rune('[') {
						goto l188
					}
					position++
					if buffer[position] != rune('*') {
						goto l188
					}
					position++
					if buffer[position] != rune(']') {
						goto l188
					}
					position++
					goto l187
				l188:
					position, tokenIndex, depth = position187, tokenIndex187, depth187
					if !_rules[ruleSlice]() {
						goto l183
					}
				}
			l187:
				if !_rules[ruleProjectionValue]() {
					goto l183
				}
			l189:
				{
					position190, tokenIndex190, depth190 := position, tokenIndex, depth
					if !_rules[ruleChainedQualifiedExpression]() {
						goto l190
					}
					goto l189
				l190:
					position, tokenIndex, depth = position190, tokenIndex190, depth190
				}
				depth--
				add(ruleProjection, position184)
			}
			return true
		l183:
			position, tokenIndex, depth = position183, tokenIndex183, depth183
			return false
		},
		/* 45 ProjectionValue <- <Action0> */
		func() bool {
			position191, tokenIndex191, depth191 := position, tokenIndex, depth
			{
				position192 := position
				depth++
				if !_rules[ruleAction0]() {
					goto l191
				}
				depth--
				add(ruleProjectionValue, position192)
			}
			return true
		l191:
			position, tokenIndex, depth = position191, tokenIndex191, depth191
			return false
		},
		/* 46 Substitution <- <('*' Level0)> */
		func() bool {
			position193, tokenIndex193, depth193 := position, tokenIndex, depth
			{
				position194 := position
				depth++
				if buffer[position] != rune('*') {
					goto l193
				}
				position++
				if !_rules[ruleLevel0]() {
					goto l193
				}
				depth--
				add(ruleSubstitution, position194)
			}
			return true
		l193:
			position, tokenIndex, depth = position193, tokenIndex193, depth193
			return false
		},
		/* 47 Not <- <('!' ws Level0)> */
		func() bool {
			position195, tokenIndex195, depth195 := position, tokenIndex, depth
			{
				position196 := position
				depth++
				if buffer[position] != rune('!') {
					goto l195
				}
				position++
				if !_rules[rulews]() {
					goto l195
				}
				if !_rules[ruleLevel0]() {
					goto l195
				}
				depth--
				add(ruleNot, position196)
			}
			return true
		l195:
			position, tokenIndex, depth = position195, tokenIndex195, depth195
			return false
		},
		/* 48 Grouped <- <('(' Expression ')')> */
		func() bool {
			position197, tokenIndex197, depth197 := position, tokenIndex, depth
			{
				position198 := position
				depth++
				if buffer[position] != rune('(') {
					goto l197
				}
				position++
				if !_rules[ruleExpression]() {
					goto l197
				}
				if buffer[position] != rune(')') {
					goto l197
				}
				position++
				depth--
				add(ruleGrouped, position198)
			}
			return true
		l197:
			position, tokenIndex, depth = position197, tokenIndex197, depth197
			return false
		},
		/* 49 Range <- <(StartRange Expression? RangeOp Expression? ']')> */
		func() bool {
			position199, tokenIndex199, depth199 := position, tokenIndex, depth
			{
				position200 := position
				depth++
				if !_rules[ruleStartRange]() {
					goto l199
				}
				{
					position201, tokenIndex201, depth201 := position, tokenIndex, depth
					if !_rules[ruleExpression]() {
						goto l201
					}
					goto l202
				l201:
					position, tokenIndex, depth = position201, tokenIndex201, depth201
				}
			l202:
				if !_rules[ruleRangeOp]() {
					goto l199
				}
				{
					position203, tokenIndex203, depth203 := position, tokenIndex, depth
					if !_rules[ruleExpression]() {
						goto l203
					}
					goto l204
				l203:
					position, tokenIndex, depth = position203, tokenIndex203, depth203
				}
			l204:
				if buffer[position] != rune(']') {
					goto l199
				}
				position++
				depth--
				add(ruleRange, position200)
			}
			return true
		l199:
			position, tokenIndex, depth = position199, tokenIndex199, depth199
			return false
		},
		/* 50 StartRange <- <'['> */
		func() bool {
			position205, tokenIndex205, depth205 := position, tokenIndex, depth
			{
				position206 := position
				depth++
				if buffer[position] != rune('[') {
					goto l205
				}
				position++
				depth--
				add(ruleStartRange, position206)
			}
			return true
		l205:
			position, tokenIndex, depth = position205, tokenIndex205, depth205
			return false
		},
		/* 51 RangeOp <- <('.' '.')> */
		func() bool {
			position207, tokenIndex207, depth207 := position, tokenIndex, depth
			{
				position208 := position
				depth++
				if buffer[position] != rune('.') {
					goto l207
				}
				position++
				if buffer[position] != rune('.') {
					goto l207
				}
				position++
				depth--
				add(ruleRangeOp, position208)
			}
			return true
		l207:
			position, tokenIndex, depth = position207, tokenIndex207, depth207
			return false
		},
		/* 52 Number <- <('-'? [0-9] ([0-9] / '_')* ('.' [0-9] [0-9]*)? (('e' / 'E') '-'? [0-9] [0-9]*)?)> */
		func() bool {
			position209, tokenIndex209, depth209 := position, tokenIndex, depth
			{
				position210 := position
				depth++
				{
					position211, tokenIndex211, depth211 := position, tokenIndex, depth
					if buffer[position] != rune('-') {
						goto l211
					}
					position++
					goto l212
				l211:
					position, tokenIndex, depth = position211, tokenIndex211, depth211
				}
			l212:
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l209
				}
				position++
			l213:
				{
					position214, tokenIndex214, depth214 := position, tokenIndex, depth
					{
						position215, tokenIndex215, depth215 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l216
						}
						position++
						goto l215
					l216:
						position, tokenIndex, depth = position215, tokenIndex215, depth215
						if buffer[position] != rune('_') {
							goto l214
						}
						position++
					}
				l215:
					goto l213
				l214:
					position, tokenIndex, depth = position214, tokenIndex214, depth214
				}
				{
					position217, tokenIndex217, depth217 := position, tokenIndex, depth
					if buffer[position] != rune('.') {
						goto l217
					}
					position++
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l217
					}
					position++
				l219:
					{
						position220, tokenIndex220, depth220 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l220
						}
						position++
						goto l219
					l220:
						position, tokenIndex, depth = position220, tokenIndex220, depth220
					}
					goto l218
				l217:
					position, tokenIndex, depth = position217, tokenIndex217, depth217
				}
			l218:
				{
					position221, tokenIndex221, depth221 := position, tokenIndex, depth
					{
						position223, tokenIndex223, depth223 := position, tokenIndex, depth
						if buffer[position] != rune('e') {
							goto l224
						}
						position++
						goto l223
					l224:
						position, tokenIndex, depth = position223, tokenIndex223, depth223
						if buffer[position] != rune('E') {
							goto l221
						}
						position++
					}
				l223:
					{
						position225, tokenIndex225, depth225 := position, tokenIndex, depth
						if buffer[position] != rune('-') {
							goto l225
						}
						position++
						goto l226
					l225:
						position, tokenIndex, depth = position225, tokenIndex225, depth225
					}
				l226:
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l221
					}
					position++
				l227:
					{
						position228, tokenIndex228, depth228 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l228
						}
						position++
						goto l227
					l228:
						position, tokenIndex, depth = position228, tokenIndex228, depth228
					}
					goto l222
				l221:
					position, tokenIndex, depth = position221, tokenIndex221, depth221
				}
			l222:
				depth--
				add(ruleNumber, position210)
			}
			return true
		l209:
			position, tokenIndex, depth = position209, tokenIndex209, depth209
			return false
		},
		/* 53 String <- <('"' (('\\' '"') / (!'"' .))* '"')> */
		func() bool {
			position229, tokenIndex229, depth229 := position, tokenIndex, depth
			{
				position230 := position
				depth++
				if buffer[position] != rune('"') {
					goto l229
				}
				position++
			l231:
				{
					position232, tokenIndex232, depth232 := position, tokenIndex, depth
					{
						position233, tokenIndex233, depth233 := position, tokenIndex, depth
						if buffer[position] != rune('\\') {
							goto l234
						}
						position++
						if buffer[position] != rune('"') {
							goto l234
						}
						position++
						goto l233
					l234:
						position, tokenIndex, depth = position233, tokenIndex233, depth233
						{
							position235, tokenIndex235, depth235 := position, tokenIndex, depth
							if buffer[position] != rune('"') {
								goto l235
							}
							position++
							goto l232
						l235:
							position, tokenIndex, depth = position235, tokenIndex235, depth235
						}
						if !matchDot() {
							goto l232
						}
					}
				l233:
					goto l231
				l232:
					position, tokenIndex, depth = position232, tokenIndex232, depth232
				}
				if buffer[position] != rune('"') {
					goto l229
				}
				position++
				depth--
				add(ruleString, position230)
			}
			return true
		l229:
			position, tokenIndex, depth = position229, tokenIndex229, depth229
			return false
		},
		/* 54 Boolean <- <(('t' 'r' 'u' 'e') / ('f' 'a' 'l' 's' 'e'))> */
		func() bool {
			position236, tokenIndex236, depth236 := position, tokenIndex, depth
			{
				position237 := position
				depth++
				{
					position238, tokenIndex238, depth238 := position, tokenIndex, depth
					if buffer[position] != rune('t') {
						goto l239
					}
					position++
					if buffer[position] != rune('r') {
						goto l239
					}
					position++
					if buffer[position] != rune('u') {
						goto l239
					}
					position++
					if buffer[position] != rune('e') {
						goto l239
					}
					position++
					goto l238
				l239:
					position, tokenIndex, depth = position238, tokenIndex238, depth238
					if buffer[position] != rune('f') {
						goto l236
					}
					position++
					if buffer[position] != rune('a') {
						goto l236
					}
					position++
					if buffer[position] != rune('l') {
						goto l236
					}
					position++
					if buffer[position] != rune('s') {
						goto l236
					}
					position++
					if buffer[position] != rune('e') {
						goto l236
					}
					position++
				}
			l238:
				depth--
				add(ruleBoolean, position237)
			}
			return true
		l236:
			position, tokenIndex, depth = position236, tokenIndex236, depth236
			return false
		},
		/* 55 Nil <- <(('n' 'i' 'l') / '~')> */
		func() bool {
			position240, tokenIndex240, depth240 := position, tokenIndex, depth
			{
				position241 := position
				depth++
				{
					position242, tokenIndex242, depth242 := position, tokenIndex, depth
					if buffer[position] != rune('n') {
						goto l243
					}
					position++
					if buffer[position] != rune('i') {
						goto l243
					}
					position++
					if buffer[position] != rune('l') {
						goto l243
					}
					position++
					goto l242
				l243:
					position, tokenIndex, depth = position242, tokenIndex242, depth242
					if buffer[position] != rune('~') {
						goto l240
					}
					position++
				}
			l242:
				depth--
				add(ruleNil, position241)
			}
			return true
		l240:
			position, tokenIndex, depth = position240, tokenIndex240, depth240
			return false
		},
		/* 56 Undefined <- <('~' '~')> */
		func() bool {
			position244, tokenIndex244, depth244 := position, tokenIndex, depth
			{
				position245 := position
				depth++
				if buffer[position] != rune('~') {
					goto l244
				}
				position++
				if buffer[position] != rune('~') {
					goto l244
				}
				position++
				depth--
				add(ruleUndefined, position245)
			}
			return true
		l244:
			position, tokenIndex, depth = position244, tokenIndex244, depth244
			return false
		},
		/* 57 Symbol <- <('$' Name)> */
		func() bool {
			position246, tokenIndex246, depth246 := position, tokenIndex, depth
			{
				position247 := position
				depth++
				if buffer[position] != rune('$') {
					goto l246
				}
				position++
				if !_rules[ruleName]() {
					goto l246
				}
				depth--
				add(ruleSymbol, position247)
			}
			return true
		l246:
			position, tokenIndex, depth = position246, tokenIndex246, depth246
			return false
		},
		/* 58 List <- <(StartList ExpressionList? ']')> */
		func() bool {
			position248, tokenIndex248, depth248 := position, tokenIndex, depth
			{
				position249 := position
				depth++
				if !_rules[ruleStartList]() {
					goto l248
				}
				{
					position250, tokenIndex250, depth250 := position, tokenIndex, depth
					if !_rules[ruleExpressionList]() {
						goto l250
					}
					goto l251
				l250:
					position, tokenIndex, depth = position250, tokenIndex250, depth250
				}
			l251:
				if buffer[position] != rune(']') {
					goto l248
				}
				position++
				depth--
				add(ruleList, position249)
			}
			return true
		l248:
			position, tokenIndex, depth = position248, tokenIndex248, depth248
			return false
		},
		/* 59 StartList <- <('[' ws)> */
		func() bool {
			position252, tokenIndex252, depth252 := position, tokenIndex, depth
			{
				position253 := position
				depth++
				if buffer[position] != rune('[') {
					goto l252
				}
				position++
				if !_rules[rulews]() {
					goto l252
				}
				depth--
				add(ruleStartList, position253)
			}
			return true
		l252:
			position, tokenIndex, depth = position252, tokenIndex252, depth252
			return false
		},
		/* 60 Map <- <(CreateMap ws Assignments? '}')> */
		func() bool {
			position254, tokenIndex254, depth254 := position, tokenIndex, depth
			{
				position255 := position
				depth++
				if !_rules[ruleCreateMap]() {
					goto l254
				}
				if !_rules[rulews]() {
					goto l254
				}
				{
					position256, tokenIndex256, depth256 := position, tokenIndex, depth
					if !_rules[ruleAssignments]() {
						goto l256
					}
					goto l257
				l256:
					position, tokenIndex, depth = position256, tokenIndex256, depth256
				}
			l257:
				if buffer[position] != rune('}') {
					goto l254
				}
				position++
				depth--
				add(ruleMap, position255)
			}
			return true
		l254:
			position, tokenIndex, depth = position254, tokenIndex254, depth254
			return false
		},
		/* 61 CreateMap <- <'{'> */
		func() bool {
			position258, tokenIndex258, depth258 := position, tokenIndex, depth
			{
				position259 := position
				depth++
				if buffer[position] != rune('{') {
					goto l258
				}
				position++
				depth--
				add(ruleCreateMap, position259)
			}
			return true
		l258:
			position, tokenIndex, depth = position258, tokenIndex258, depth258
			return false
		},
		/* 62 Assignments <- <(Assignment (',' Assignment)*)> */
		func() bool {
			position260, tokenIndex260, depth260 := position, tokenIndex, depth
			{
				position261 := position
				depth++
				if !_rules[ruleAssignment]() {
					goto l260
				}
			l262:
				{
					position263, tokenIndex263, depth263 := position, tokenIndex, depth
					if buffer[position] != rune(',') {
						goto l263
					}
					position++
					if !_rules[ruleAssignment]() {
						goto l263
					}
					goto l262
				l263:
					position, tokenIndex, depth = position263, tokenIndex263, depth263
				}
				depth--
				add(ruleAssignments, position261)
			}
			return true
		l260:
			position, tokenIndex, depth = position260, tokenIndex260, depth260
			return false
		},
		/* 63 Assignment <- <(Expression '=' Expression)> */
		func() bool {
			position264, tokenIndex264, depth264 := position, tokenIndex, depth
			{
				position265 := position
				depth++
				if !_rules[ruleExpression]() {
					goto l264
				}
				if buffer[position] != rune('=') {
					goto l264
				}
				position++
				if !_rules[ruleExpression]() {
					goto l264
				}
				depth--
				add(ruleAssignment, position265)
			}
			return true
		l264:
			position, tokenIndex, depth = position264, tokenIndex264, depth264
			return false
		},
		/* 64 Merge <- <(RefMerge / SimpleMerge)> */
		func() bool {
			position266, tokenIndex266, depth266 := position, tokenIndex, depth
			{
				position267 := position
				depth++
				{
					position268, tokenIndex268, depth268 := position, tokenIndex, depth
					if !_rules[ruleRefMerge]() {
						goto l269
					}
					goto l268
				l269:
					position, tokenIndex, depth = position268, tokenIndex268, depth268
					if !_rules[ruleSimpleMerge]() {
						goto l266
					}
				}
			l268:
				depth--
				add(ruleMerge, position267)
			}
			return true
		l266:
			position, tokenIndex, depth = position266, tokenIndex266, depth266
			return false
		},
		/* 65 RefMerge <- <('m' 'e' 'r' 'g' 'e' !(req_ws Required) (req_ws (Replace / On))? req_ws Reference)> */
		func() bool {
			position270, tokenIndex270, depth270 := position, tokenIndex, depth
			{
				position271 := position
				depth++
				if buffer[position] != rune('m') {
					goto l270
				}
				position++
				if buffer[position] != rune('e') {
					goto l270
				}
				position++
				if buffer[position] != rune('r') {
					goto l270
				}
				position++
				if buffer[position] != rune('g') {
					goto l270
				}
				position++
				if buffer[position] != rune('e') {
					goto l270
				}
				position++
				{
					position272, tokenIndex272, depth272 := position, tokenIndex, depth
					if !_rules[rulereq_ws]() {
						goto l272
					}
					if !_rules[ruleRequired]() {
						goto l272
					}
					goto l270
				l272:
					position, tokenIndex, depth = position272, tokenIndex272, depth272
				}
				{
					position273, tokenIndex273, depth273 := position, tokenIndex, depth
					if !_rules[rulereq_ws]() {
						goto l273
					}
					{
						position275, tokenIndex275, depth275 := position, tokenIndex, depth
						if !_rules[ruleReplace]() {
							goto l276
						}
						goto l275
					l276:
						position, tokenIndex, depth = position275, tokenIndex275, depth275
						if !_rules[ruleOn]() {
							goto l273
						}
					}
				l275:
					goto l274
				l273:
					position, tokenIndex, depth = position273, tokenIndex273, depth273
				}
			l274:
				if !_rules[rulereq_ws]() {
					goto l270
				}
				if !_rules[ruleReference]() {
					goto l270
				}
				depth--
				add(ruleRefMerge, position271)
			}
			return true
		l270:
			position, tokenIndex, depth = position270, tokenIndex270, depth270
			return false
		},
		/* 66 SimpleMerge <- <('m' 'e' 'r' 'g' 'e' !'(' (req_ws (Replace / Required / On))?)> */
		func() bool {
			position277, tokenIndex277, depth277 := position, tokenIndex, depth
			{
				position278 := position
				depth++
				if buffer[position] != rune('m') {
					goto l277
				}
				position++
				if buffer[position] != rune('e') {
					goto l277
				}
				position++
				if buffer[position] != rune('r') {
					goto l277
				}
				position++
				if buffer[position] != rune('g') {
					goto l277
				}
				position++
				if buffer[position] != rune('e') {
					goto l277
				}
				position++
				{
					position279, tokenIndex279, depth279 := position, tokenIndex, depth
					if buffer[position] != rune('(') {
						goto l279
					}
					position++
					goto l277
				l279:
					position, tokenIndex, depth = position279, tokenIndex279, depth279
				}
				{
					position280, tokenIndex280, depth280 := position, tokenIndex, depth
					if !_rules[rulereq_ws]() {
						goto l280
					}
					{
						position282, tokenIndex282, depth282 := position, tokenIndex, depth
						if !_rules[ruleReplace]() {
							goto l283
						}
						goto l282
					l283:
						position, tokenIndex, depth = position282, tokenIndex282, depth282
						if !_rules[ruleRequired]() {
							goto l284
						}
						goto l282
					l284:
						position, tokenIndex, depth = position282, tokenIndex282, depth282
						if !_rules[ruleOn]() {
							goto l280
						}
					}
				l282:
					goto l281
				l280:
					position, tokenIndex, depth = position280, tokenIndex280, depth280
				}
			l281:
				depth--
				add(ruleSimpleMerge, position278)
			}
			return true
		l277:
			position, tokenIndex, depth = position277, tokenIndex277, depth277
			return false
		},
		/* 67 Replace <- <('r' 'e' 'p' 'l' 'a' 'c' 'e')> */
		func() bool {
			position285, tokenIndex285, depth285 := position, tokenIndex, depth
			{
				position286 := position
				depth++
				if buffer[position] != rune('r') {
					goto l285
				}
				position++
				if buffer[position] != rune('e') {
					goto l285
				}
				position++
				if buffer[position] != rune('p') {
					goto l285
				}
				position++
				if buffer[position] != rune('l') {
					goto l285
				}
				position++
				if buffer[position] != rune('a') {
					goto l285
				}
				position++
				if buffer[position] != rune('c') {
					goto l285
				}
				position++
				if buffer[position] != rune('e') {
					goto l285
				}
				position++
				depth--
				add(ruleReplace, position286)
			}
			return true
		l285:
			position, tokenIndex, depth = position285, tokenIndex285, depth285
			return false
		},
		/* 68 Required <- <('r' 'e' 'q' 'u' 'i' 'r' 'e' 'd')> */
		func() bool {
			position287, tokenIndex287, depth287 := position, tokenIndex, depth
			{
				position288 := position
				depth++
				if buffer[position] != rune('r') {
					goto l287
				}
				position++
				if buffer[position] != rune('e') {
					goto l287
				}
				position++
				if buffer[position] != rune('q') {
					goto l287
				}
				position++
				if buffer[position] != rune('u') {
					goto l287
				}
				position++
				if buffer[position] != rune('i') {
					goto l287
				}
				position++
				if buffer[position] != rune('r') {
					goto l287
				}
				position++
				if buffer[position] != rune('e') {
					goto l287
				}
				position++
				if buffer[position] != rune('d') {
					goto l287
				}
				position++
				depth--
				add(ruleRequired, position288)
			}
			return true
		l287:
			position, tokenIndex, depth = position287, tokenIndex287, depth287
			return false
		},
		/* 69 On <- <('o' 'n' req_ws Name)> */
		func() bool {
			position289, tokenIndex289, depth289 := position, tokenIndex, depth
			{
				position290 := position
				depth++
				if buffer[position] != rune('o') {
					goto l289
				}
				position++
				if buffer[position] != rune('n') {
					goto l289
				}
				position++
				if !_rules[rulereq_ws]() {
					goto l289
				}
				if !_rules[ruleName]() {
					goto l289
				}
				depth--
				add(ruleOn, position290)
			}
			return true
		l289:
			position, tokenIndex, depth = position289, tokenIndex289, depth289
			return false
		},
		/* 70 Auto <- <('a' 'u' 't' 'o')> */
		func() bool {
			position291, tokenIndex291, depth291 := position, tokenIndex, depth
			{
				position292 := position
				depth++
				if buffer[position] != rune('a') {
					goto l291
				}
				position++
				if buffer[position] != rune('u') {
					goto l291
				}
				position++
				if buffer[position] != rune('t') {
					goto l291
				}
				position++
				if buffer[position] != rune('o') {
					goto l291
				}
				position++
				depth--
				add(ruleAuto, position292)
			}
			return true
		l291:
			position, tokenIndex, depth = position291, tokenIndex291, depth291
			return false
		},
		/* 71 Default <- <Action1> */
		func() bool {
			position293, tokenIndex293, depth293 := position, tokenIndex, depth
			{
				position294 := position
				depth++
				if !_rules[ruleAction1]() {
					goto l293
				}
				depth--
				add(ruleDefault, position294)
			}
			return true
		l293:
			position, tokenIndex, depth = position293, tokenIndex293, depth293
			return false
		},
		/* 72 Sync <- <('s' 'y' 'n' 'c' '[' Level7 ((((LambdaExpr LambdaExt) / (LambdaOrExpr LambdaOrExpr)) (('|' Expression) / Default)) / (LambdaOrExpr Default Default)) ']')> */
		func() bool {
			position295, tokenIndex295, depth295 := position, tokenIndex, depth
			{
				position296 := position
				depth++
				if buffer[position] != rune('s') {
					goto l295
				}
				position++
				if buffer[position] != rune('y') {
					goto l295
				}
				position++
				if buffer[position] != rune('n') {
					goto l295
				}
				position++
				if buffer[position] != rune('c') {
					goto l295
				}
				position++
				if buffer[position] != rune('[') {
					goto l295
				}
				position++
				if !_rules[ruleLevel7]() {
					goto l295
				}
				{
					position297, tokenIndex297, depth297 := position, tokenIndex, depth
					{
						position299, tokenIndex299, depth299 := position, tokenIndex, depth
						if !_rules[ruleLambdaExpr]() {
							goto l300
						}
						if !_rules[ruleLambdaExt]() {
							goto l300
						}
						goto l299
					l300:
						position, tokenIndex, depth = position299, tokenIndex299, depth299
						if !_rules[ruleLambdaOrExpr]() {
							goto l298
						}
						if !_rules[ruleLambdaOrExpr]() {
							goto l298
						}
					}
				l299:
					{
						position301, tokenIndex301, depth301 := position, tokenIndex, depth
						if buffer[position] != rune('|') {
							goto l302
						}
						position++
						if !_rules[ruleExpression]() {
							goto l302
						}
						goto l301
					l302:
						position, tokenIndex, depth = position301, tokenIndex301, depth301
						if !_rules[ruleDefault]() {
							goto l298
						}
					}
				l301:
					goto l297
				l298:
					position, tokenIndex, depth = position297, tokenIndex297, depth297
					if !_rules[ruleLambdaOrExpr]() {
						goto l295
					}
					if !_rules[ruleDefault]() {
						goto l295
					}
					if !_rules[ruleDefault]() {
						goto l295
					}
				}
			l297:
				if buffer[position] != rune(']') {
					goto l295
				}
				position++
				depth--
				add(ruleSync, position296)
			}
			return true
		l295:
			position, tokenIndex, depth = position295, tokenIndex295, depth295
			return false
		},
		/* 73 LambdaExt <- <(',' Expression)> */
		func() bool {
			position303, tokenIndex303, depth303 := position, tokenIndex, depth
			{
				position304 := position
				depth++
				if buffer[position] != rune(',') {
					goto l303
				}
				position++
				if !_rules[ruleExpression]() {
					goto l303
				}
				depth--
				add(ruleLambdaExt, position304)
			}
			return true
		l303:
			position, tokenIndex, depth = position303, tokenIndex303, depth303
			return false
		},
		/* 74 LambdaOrExpr <- <(LambdaExpr / ('|' Expression))> */
		func() bool {
			position305, tokenIndex305, depth305 := position, tokenIndex, depth
			{
				position306 := position
				depth++
				{
					position307, tokenIndex307, depth307 := position, tokenIndex, depth
					if !_rules[ruleLambdaExpr]() {
						goto l308
					}
					goto l307
				l308:
					position, tokenIndex, depth = position307, tokenIndex307, depth307
					if buffer[position] != rune('|') {
						goto l305
					}
					position++
					if !_rules[ruleExpression]() {
						goto l305
					}
				}
			l307:
				depth--
				add(ruleLambdaOrExpr, position306)
			}
			return true
		l305:
			position, tokenIndex, depth = position305, tokenIndex305, depth305
			return false
		},
		/* 75 Catch <- <('c' 'a' 't' 'c' 'h' '[' Level7 LambdaOrExpr ']')> */
		func() bool {
			position309, tokenIndex309, depth309 := position, tokenIndex, depth
			{
				position310 := position
				depth++
				if buffer[position] != rune('c') {
					goto l309
				}
				position++
				if buffer[position] != rune('a') {
					goto l309
				}
				position++
				if buffer[position] != rune('t') {
					goto l309
				}
				position++
				if buffer[position] != rune('c') {
					goto l309
				}
				position++
				if buffer[position] != rune('h') {
					goto l309
				}
				position++
				if buffer[position] != rune('[') {
					goto l309
				}
				position++
				if !_rules[ruleLevel7]() {
					goto l309
				}
				if !_rules[ruleLambdaOrExpr]() {
					goto l309
				}
				if buffer[position] != rune(']') {
					goto l309
				}
				position++
				depth--
				add(ruleCatch, position310)
			}
			return true
		l309:
			position, tokenIndex, depth = position309, tokenIndex309, depth309
			return false
		},
		/* 76 MapMapping <- <('m' 'a' 'p' '{' Level7 LambdaOrExpr '}')> */
		func() bool {
			position311, tokenIndex311, depth311 := position, tokenIndex, depth
			{
				position312 := position
				depth++
				if buffer[position] != rune('m') {
					goto l311
				}
				position++
				if buffer[position] != rune('a') {
					goto l311
				}
				position++
				if buffer[position] != rune('p') {
					goto l311
				}
				position++
				if buffer[position] != rune('{') {
					goto l311
				}
				position++
				if !_rules[ruleLevel7]() {
					goto l311
				}
				if !_rules[ruleLambdaOrExpr]() {
					goto l311
				}
				if buffer[position] != rune('}') {
					goto l311
				}
				position++
				depth--
				add(ruleMapMapping, position312)
			}
			return true
		l311:
			position, tokenIndex, depth = position311, tokenIndex311, depth311
			return false
		},
		/* 77 Mapping <- <('m' 'a' 'p' '[' Level7 LambdaOrExpr ']')> */
		func() bool {
			position313, tokenIndex313, depth313 := position, tokenIndex, depth
			{
				position314 := position
				depth++
				if buffer[position] != rune('m') {
					goto l313
				}
				position++
				if buffer[position] != rune('a') {
					goto l313
				}
				position++
				if buffer[position] != rune('p') {
					goto l313
				}
				position++
				if buffer[position] != rune('[') {
					goto l313
				}
				position++
				if !_rules[ruleLevel7]() {
					goto l313
				}
				if !_rules[ruleLambdaOrExpr]() {
					goto l313
				}
				if buffer[position] != rune(']') {
					goto l313
				}
				position++
				depth--
				add(ruleMapping, position314)
			}
			return true
		l313:
			position, tokenIndex, depth = position313, tokenIndex313, depth313
			return false
		},
		/* 78 MapSelection <- <('s' 'e' 'l' 'e' 'c' 't' '{' Level7 LambdaOrExpr '}')> */
		func() bool {
			position315, tokenIndex315, depth315 := position, tokenIndex, depth
			{
				position316 := position
				depth++
				if buffer[position] != rune('s') {
					goto l315
				}
				position++
				if buffer[position] != rune('e') {
					goto l315
				}
				position++
				if buffer[position] != rune('l') {
					goto l315
				}
				position++
				if buffer[position] != rune('e') {
					goto l315
				}
				position++
				if buffer[position] != rune('c') {
					goto l315
				}
				position++
				if buffer[position] != rune('t') {
					goto l315
				}
				position++
				if buffer[position] != rune('{') {
					goto l315
				}
				position++
				if !_rules[ruleLevel7]() {
					goto l315
				}
				if !_rules[ruleLambdaOrExpr]() {
					goto l315
				}
				if buffer[position] != rune('}') {
					goto l315
				}
				position++
				depth--
				add(ruleMapSelection, position316)
			}
			return true
		l315:
			position, tokenIndex, depth = position315, tokenIndex315, depth315
			return false
		},
		/* 79 Selection <- <('s' 'e' 'l' 'e' 'c' 't' '[' Level7 LambdaOrExpr ']')> */
		func() bool {
			position317, tokenIndex317, depth317 := position, tokenIndex, depth
			{
				position318 := position
				depth++
				if buffer[position] != rune('s') {
					goto l317
				}
				position++
				if buffer[position] != rune('e') {
					goto l317
				}
				position++
				if buffer[position] != rune('l') {
					goto l317
				}
				position++
				if buffer[position] != rune('e') {
					goto l317
				}
				position++
				if buffer[position] != rune('c') {
					goto l317
				}
				position++
				if buffer[position] != rune('t') {
					goto l317
				}
				position++
				if buffer[position] != rune('[') {
					goto l317
				}
				position++
				if !_rules[ruleLevel7]() {
					goto l317
				}
				if !_rules[ruleLambdaOrExpr]() {
					goto l317
				}
				if buffer[position] != rune(']') {
					goto l317
				}
				position++
				depth--
				add(ruleSelection, position318)
			}
			return true
		l317:
			position, tokenIndex, depth = position317, tokenIndex317, depth317
			return false
		},
		/* 80 Sum <- <('s' 'u' 'm' '[' Level7 '|' Level7 LambdaOrExpr ']')> */
		func() bool {
			position319, tokenIndex319, depth319 := position, tokenIndex, depth
			{
				position320 := position
				depth++
				if buffer[position] != rune('s') {
					goto l319
				}
				position++
				if buffer[position] != rune('u') {
					goto l319
				}
				position++
				if buffer[position] != rune('m') {
					goto l319
				}
				position++
				if buffer[position] != rune('[') {
					goto l319
				}
				position++
				if !_rules[ruleLevel7]() {
					goto l319
				}
				if buffer[position] != rune('|') {
					goto l319
				}
				position++
				if !_rules[ruleLevel7]() {
					goto l319
				}
				if !_rules[ruleLambdaOrExpr]() {
					goto l319
				}
				if buffer[position] != rune(']') {
					goto l319
				}
				position++
				depth--
				add(ruleSum, position320)
			}
			return true
		l319:
			position, tokenIndex, depth = position319, tokenIndex319, depth319
			return false
		},
		/* 81 Lambda <- <('l' 'a' 'm' 'b' 'd' 'a' (LambdaRef / LambdaExpr))> */
		func() bool {
			position321, tokenIndex321, depth321 := position, tokenIndex, depth
			{
				position322 := position
				depth++
				if buffer[position] != rune('l') {
					goto l321
				}
				position++
				if buffer[position] != rune('a') {
					goto l321
				}
				position++
				if buffer[position] != rune('m') {
					goto l321
				}
				position++
				if buffer[position] != rune('b') {
					goto l321
				}
				position++
				if buffer[position] != rune('d') {
					goto l321
				}
				position++
				if buffer[position] != rune('a') {
					goto l321
				}
				position++
				{
					position323, tokenIndex323, depth323 := position, tokenIndex, depth
					if !_rules[ruleLambdaRef]() {
						goto l324
					}
					goto l323
				l324:
					position, tokenIndex, depth = position323, tokenIndex323, depth323
					if !_rules[ruleLambdaExpr]() {
						goto l321
					}
				}
			l323:
				depth--
				add(ruleLambda, position322)
			}
			return true
		l321:
			position, tokenIndex, depth = position321, tokenIndex321, depth321
			return false
		},
		/* 82 LambdaRef <- <(req_ws Expression)> */
		func() bool {
			position325, tokenIndex325, depth325 := position, tokenIndex, depth
			{
				position326 := position
				depth++
				if !_rules[rulereq_ws]() {
					goto l325
				}
				if !_rules[ruleExpression]() {
					goto l325
				}
				depth--
				add(ruleLambdaRef, position326)
			}
			return true
		l325:
			position, tokenIndex, depth = position325, tokenIndex325, depth325
			return false
		},
		/* 83 LambdaExpr <- <(ws Params ws ('-' '>') Expression)> */
		func() bool {
			position327, tokenIndex327, depth327 := position, tokenIndex, depth
			{
				position328 := position
				depth++
				if !_rules[rulews]() {
					goto l327
				}
				if !_rules[ruleParams]() {
					goto l327
				}
				if !_rules[rulews]() {
					goto l327
				}
				if buffer[position] != rune('-') {
					goto l327
				}
				position++
				if buffer[position] != rune('>') {
					goto l327
				}
				position++
				if !_rules[ruleExpression]() {
					goto l327
				}
				depth--
				add(ruleLambdaExpr, position328)
			}
			return true
		l327:
			position, tokenIndex, depth = position327, tokenIndex327, depth327
			return false
		},
		/* 84 Params <- <('|' StartParams ws Names? '|')> */
		func() bool {
			position329, tokenIndex329, depth329 := position, tokenIndex, depth
			{
				position330 := position
				depth++
				if buffer[position] != rune('|') {
					goto l329
				}
				position++
				if !_rules[ruleStartParams]() {
					goto l329
				}
				if !_rules[rulews]() {
					goto l329
				}
				{
					position331, tokenIndex331, depth331 := position, tokenIndex, depth
					if !_rules[ruleNames]() {
						goto l331
					}
					goto l332
				l331:
					position, tokenIndex, depth = position331, tokenIndex331, depth331
				}
			l332:
				if buffer[position] != rune('|') {
					goto l329
				}
				position++
				depth--
				add(ruleParams, position330)
			}
			return true
		l329:
			position, tokenIndex, depth = position329, tokenIndex329, depth329
			return false
		},
		/* 85 StartParams <- <Action2> */
		func() bool {
			position333, tokenIndex333, depth333 := position, tokenIndex, depth
			{
				position334 := position
				depth++
				if !_rules[ruleAction2]() {
					goto l333
				}
				depth--
				add(ruleStartParams, position334)
			}
			return true
		l333:
			position, tokenIndex, depth = position333, tokenIndex333, depth333
			return false
		},
		/* 86 Names <- <(NextName (',' NextName)* DefaultValue? (',' NextName DefaultValue)* VarParams?)> */
		func() bool {
			position335, tokenIndex335, depth335 := position, tokenIndex, depth
			{
				position336 := position
				depth++
				if !_rules[ruleNextName]() {
					goto l335
				}
			l337:
				{
					position338, tokenIndex338, depth338 := position, tokenIndex, depth
					if buffer[position] != rune(',') {
						goto l338
					}
					position++
					if !_rules[ruleNextName]() {
						goto l338
					}
					goto l337
				l338:
					position, tokenIndex, depth = position338, tokenIndex338, depth338
				}
				{
					position339, tokenIndex339, depth339 := position, tokenIndex, depth
					if !_rules[ruleDefaultValue]() {
						goto l339
					}
					goto l340
				l339:
					position, tokenIndex, depth = position339, tokenIndex339, depth339
				}
			l340:
			l341:
				{
					position342, tokenIndex342, depth342 := position, tokenIndex, depth
					if buffer[position] != rune(',') {
						goto l342
					}
					position++
					if !_rules[ruleNextName]() {
						goto l342
					}
					if !_rules[ruleDefaultValue]() {
						goto l342
					}
					goto l341
				l342:
					position, tokenIndex, depth = position342, tokenIndex342, depth342
				}
				{
					position343, tokenIndex343, depth343 := position, tokenIndex, depth
					if !_rules[ruleVarParams]() {
						goto l343
					}
					goto l344
				l343:
					position, tokenIndex, depth = position343, tokenIndex343, depth343
				}
			l344:
				depth--
				add(ruleNames, position336)
			}
			return true
		l335:
			position, tokenIndex, depth = position335, tokenIndex335, depth335
			return false
		},
		/* 87 NextName <- <(ws Name ws)> */
		func() bool {
			position345, tokenIndex345, depth345 := position, tokenIndex, depth
			{
				position346 := position
				depth++
				if !_rules[rulews]() {
					goto l345
				}
				if !_rules[ruleName]() {
					goto l345
				}
				if !_rules[rulews]() {
					goto l345
				}
				depth--
				add(ruleNextName, position346)
			}
			return true
		l345:
			position, tokenIndex, depth = position345, tokenIndex345, depth345
			return false
		},
		/* 88 Name <- <([a-z] / [A-Z] / [0-9] / '_')+> */
		func() bool {
			position347, tokenIndex347, depth347 := position, tokenIndex, depth
			{
				position348 := position
				depth++
				{
					position351, tokenIndex351, depth351 := position, tokenIndex, depth
					if c := buffer[position]; c < rune('a') || c > rune('z') {
						goto l352
					}
					position++
					goto l351
				l352:
					position, tokenIndex, depth = position351, tokenIndex351, depth351
					if c := buffer[position]; c < rune('A') || c > rune('Z') {
						goto l353
					}
					position++
					goto l351
				l353:
					position, tokenIndex, depth = position351, tokenIndex351, depth351
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l354
					}
					position++
					goto l351
				l354:
					position, tokenIndex, depth = position351, tokenIndex351, depth351
					if buffer[position] != rune('_') {
						goto l347
					}
					position++
				}
			l351:
			l349:
				{
					position350, tokenIndex350, depth350 := position, tokenIndex, depth
					{
						position355, tokenIndex355, depth355 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l356
						}
						position++
						goto l355
					l356:
						position, tokenIndex, depth = position355, tokenIndex355, depth355
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l357
						}
						position++
						goto l355
					l357:
						position, tokenIndex, depth = position355, tokenIndex355, depth355
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l358
						}
						position++
						goto l355
					l358:
						position, tokenIndex, depth = position355, tokenIndex355, depth355
						if buffer[position] != rune('_') {
							goto l350
						}
						position++
					}
				l355:
					goto l349
				l350:
					position, tokenIndex, depth = position350, tokenIndex350, depth350
				}
				depth--
				add(ruleName, position348)
			}
			return true
		l347:
			position, tokenIndex, depth = position347, tokenIndex347, depth347
			return false
		},
		/* 89 DefaultValue <- <('=' Expression)> */
		func() bool {
			position359, tokenIndex359, depth359 := position, tokenIndex, depth
			{
				position360 := position
				depth++
				if buffer[position] != rune('=') {
					goto l359
				}
				position++
				if !_rules[ruleExpression]() {
					goto l359
				}
				depth--
				add(ruleDefaultValue, position360)
			}
			return true
		l359:
			position, tokenIndex, depth = position359, tokenIndex359, depth359
			return false
		},
		/* 90 VarParams <- <('.' '.' '.' ws)> */
		func() bool {
			position361, tokenIndex361, depth361 := position, tokenIndex, depth
			{
				position362 := position
				depth++
				if buffer[position] != rune('.') {
					goto l361
				}
				position++
				if buffer[position] != rune('.') {
					goto l361
				}
				position++
				if buffer[position] != rune('.') {
					goto l361
				}
				position++
				if !_rules[rulews]() {
					goto l361
				}
				depth--
				add(ruleVarParams, position362)
			}
			return true
		l361:
			position, tokenIndex, depth = position361, tokenIndex361, depth361
			return false
		},
		/* 91 Reference <- <('.'? Key FollowUpRef)> */
		func() bool {
			position363, tokenIndex363, depth363 := position, tokenIndex, depth
			{
				position364 := position
				depth++
				{
					position365, tokenIndex365, depth365 := position, tokenIndex, depth
					if buffer[position] != rune('.') {
						goto l365
					}
					position++
					goto l366
				l365:
					position, tokenIndex, depth = position365, tokenIndex365, depth365
				}
			l366:
				if !_rules[ruleKey]() {
					goto l363
				}
				if !_rules[ruleFollowUpRef]() {
					goto l363
				}
				depth--
				add(ruleReference, position364)
			}
			return true
		l363:
			position, tokenIndex, depth = position363, tokenIndex363, depth363
			return false
		},
		/* 92 FollowUpRef <- <PathComponent*> */
		func() bool {
			{
				position368 := position
				depth++
			l369:
				{
					position370, tokenIndex370, depth370 := position, tokenIndex, depth
					if !_rules[rulePathComponent]() {
						goto l370
					}
					goto l369
				l370:
					position, tokenIndex, depth = position370, tokenIndex370, depth370
				}
				depth--
				add(ruleFollowUpRef, position368)
			}
			return true
		},
		/* 93 PathComponent <- <(('.' Key) / ('.'? Index))> */
		func() bool {
			position371, tokenIndex371, depth371 := position, tokenIndex, depth
			{
				position372 := position
				depth++
				{
					position373, tokenIndex373, depth373 := position, tokenIndex, depth
					if buffer[position] != rune('.') {
						goto l374
					}
					position++
					if !_rules[ruleKey]() {
						goto l374
					}
					goto l373
				l374:
					position, tokenIndex, depth = position373, tokenIndex373, depth373
					{
						position375, tokenIndex375, depth375 := position, tokenIndex, depth
						if buffer[position] != rune('.') {
							goto l375
						}
						position++
						goto l376
					l375:
						position, tokenIndex, depth = position375, tokenIndex375, depth375
					}
				l376:
					if !_rules[ruleIndex]() {
						goto l371
					}
				}
			l373:
				depth--
				add(rulePathComponent, position372)
			}
			return true
		l371:
			position, tokenIndex, depth = position371, tokenIndex371, depth371
			return false
		},
		/* 94 Key <- <(([a-z] / [A-Z] / [0-9] / '_') ([a-z] / [A-Z] / [0-9] / '_' / '-')* (':' ([a-z] / [A-Z] / [0-9] / '_') ([a-z] / [A-Z] / [0-9] / '_' / '-')*)?)> */
		func() bool {
			position377, tokenIndex377, depth377 := position, tokenIndex, depth
			{
				position378 := position
				depth++
				{
					position379, tokenIndex379, depth379 := position, tokenIndex, depth
					if c := buffer[position]; c < rune('a') || c > rune('z') {
						goto l380
					}
					position++
					goto l379
				l380:
					position, tokenIndex, depth = position379, tokenIndex379, depth379
					if c := buffer[position]; c < rune('A') || c > rune('Z') {
						goto l381
					}
					position++
					goto l379
				l381:
					position, tokenIndex, depth = position379, tokenIndex379, depth379
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l382
					}
					position++
					goto l379
				l382:
					position, tokenIndex, depth = position379, tokenIndex379, depth379
					if buffer[position] != rune('_') {
						goto l377
					}
					position++
				}
			l379:
			l383:
				{
					position384, tokenIndex384, depth384 := position, tokenIndex, depth
					{
						position385, tokenIndex385, depth385 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l386
						}
						position++
						goto l385
					l386:
						position, tokenIndex, depth = position385, tokenIndex385, depth385
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l387
						}
						position++
						goto l385
					l387:
						position, tokenIndex, depth = position385, tokenIndex385, depth385
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l388
						}
						position++
						goto l385
					l388:
						position, tokenIndex, depth = position385, tokenIndex385, depth385
						if buffer[position] != rune('_') {
							goto l389
						}
						position++
						goto l385
					l389:
						position, tokenIndex, depth = position385, tokenIndex385, depth385
						if buffer[position] != rune('-') {
							goto l384
						}
						position++
					}
				l385:
					goto l383
				l384:
					position, tokenIndex, depth = position384, tokenIndex384, depth384
				}
				{
					position390, tokenIndex390, depth390 := position, tokenIndex, depth
					if buffer[position] != rune(':') {
						goto l390
					}
					position++
					{
						position392, tokenIndex392, depth392 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l393
						}
						position++
						goto l392
					l393:
						position, tokenIndex, depth = position392, tokenIndex392, depth392
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l394
						}
						position++
						goto l392
					l394:
						position, tokenIndex, depth = position392, tokenIndex392, depth392
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l395
						}
						position++
						goto l392
					l395:
						position, tokenIndex, depth = position392, tokenIndex392, depth392
						if buffer[position] != rune('_') {
							goto l390
						}
						position++
					}
				l392:
				l396:
					{
						position397, tokenIndex397, depth397 := position, tokenIndex, depth
						{
							position398, tokenIndex398, depth398 := position, tokenIndex, depth
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l399
							}
							position++
							goto l398
						l399:
							position, tokenIndex, depth = position398, tokenIndex398, depth398
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l400
							}
							position++
							goto l398
						l400:
							position, tokenIndex, depth = position398, tokenIndex398, depth398
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l401
							}
							position++
							goto l398
						l401:
							position, tokenIndex, depth = position398, tokenIndex398, depth398
							if buffer[position] != rune('_') {
								goto l402
							}
							position++
							goto l398
						l402:
							position, tokenIndex, depth = position398, tokenIndex398, depth398
							if buffer[position] != rune('-') {
								goto l397
							}
							position++
						}
					l398:
						goto l396
					l397:
						position, tokenIndex, depth = position397, tokenIndex397, depth397
					}
					goto l391
				l390:
					position, tokenIndex, depth = position390, tokenIndex390, depth390
				}
			l391:
				depth--
				add(ruleKey, position378)
			}
			return true
		l377:
			position, tokenIndex, depth = position377, tokenIndex377, depth377
			return false
		},
		/* 95 Index <- <('[' '-'? [0-9]+ ']')> */
		func() bool {
			position403, tokenIndex403, depth403 := position, tokenIndex, depth
			{
				position404 := position
				depth++
				if buffer[position] != rune('[') {
					goto l403
				}
				position++
				{
					position405, tokenIndex405, depth405 := position, tokenIndex, depth
					if buffer[position] != rune('-') {
						goto l405
					}
					position++
					goto l406
				l405:
					position, tokenIndex, depth = position405, tokenIndex405, depth405
				}
			l406:
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l403
				}
				position++
			l407:
				{
					position408, tokenIndex408, depth408 := position, tokenIndex, depth
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l408
					}
					position++
					goto l407
				l408:
					position, tokenIndex, depth = position408, tokenIndex408, depth408
				}
				if buffer[position] != rune(']') {
					goto l403
				}
				position++
				depth--
				add(ruleIndex, position404)
			}
			return true
		l403:
			position, tokenIndex, depth = position403, tokenIndex403, depth403
			return false
		},
		/* 96 IP <- <([0-9]+ '.' [0-9]+ '.' [0-9]+ '.' [0-9]+)> */
		func() bool {
			position409, tokenIndex409, depth409 := position, tokenIndex, depth
			{
				position410 := position
				depth++
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l409
				}
				position++
			l411:
				{
					position412, tokenIndex412, depth412 := position, tokenIndex, depth
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l412
					}
					position++
					goto l411
				l412:
					position, tokenIndex, depth = position412, tokenIndex412, depth412
				}
				if buffer[position] != rune('.') {
					goto l409
				}
				position++
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l409
				}
				position++
			l413:
				{
					position414, tokenIndex414, depth414 := position, tokenIndex, depth
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l414
					}
					position++
					goto l413
				l414:
					position, tokenIndex, depth = position414, tokenIndex414, depth414
				}
				if buffer[position] != rune('.') {
					goto l409
				}
				position++
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l409
				}
				position++
			l415:
				{
					position416, tokenIndex416, depth416 := position, tokenIndex, depth
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l416
					}
					position++
					goto l415
				l416:
					position, tokenIndex, depth = position416, tokenIndex416, depth416
				}
				if buffer[position] != rune('.') {
					goto l409
				}
				position++
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l409
				}
				position++
			l417:
				{
					position418, tokenIndex418, depth418 := position, tokenIndex, depth
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l418
					}
					position++
					goto l417
				l418:
					position, tokenIndex, depth = position418, tokenIndex418, depth418
				}
				depth--
				add(ruleIP, position410)
			}
			return true
		l409:
			position, tokenIndex, depth = position409, tokenIndex409, depth409
			return false
		},
		/* 97 ws <- <(' ' / '\t' / '\n' / '\r')*> */
		func() bool {
			{
				position420 := position
				depth++
			l421:
				{
					position422, tokenIndex422, depth422 := position, tokenIndex, depth
					{
						position423, tokenIndex423, depth423 := position, tokenIndex, depth
						if buffer[position] != rune(' ') {
							goto l424
						}
						position++
						goto l423
					l424:
						position, tokenIndex, depth = position423, tokenIndex423, depth423
						if buffer[position] != rune('\t') {
							goto l425
						}
						position++
						goto l423
					l425:
						position, tokenIndex, depth = position423, tokenIndex423, depth423
						if buffer[position] != rune('\n') {
							goto l426
						}
						position++
						goto l423
					l426:
						position, tokenIndex, depth = position423, tokenIndex423, depth423
						if buffer[position] != rune('\r') {
							goto l422
						}
						position++
					}
				l423:
					goto l421
				l422:
					position, tokenIndex, depth = position422, tokenIndex422, depth422
				}
				depth--
				add(rulews, position420)
			}
			return true
		},
		/* 98 req_ws <- <(' ' / '\t' / '\n' / '\r')+> */
		func() bool {
			position427, tokenIndex427, depth427 := position, tokenIndex, depth
			{
				position428 := position
				depth++
				{
					position431, tokenIndex431, depth431 := position, tokenIndex, depth
					if buffer[position] != rune(' ') {
						goto l432
					}
					position++
					goto l431
				l432:
					position, tokenIndex, depth = position431, tokenIndex431, depth431
					if buffer[position] != rune('\t') {
						goto l433
					}
					position++
					goto l431
				l433:
					position, tokenIndex, depth = position431, tokenIndex431, depth431
					if buffer[position] != rune('\n') {
						goto l434
					}
					position++
					goto l431
				l434:
					position, tokenIndex, depth = position431, tokenIndex431, depth431
					if buffer[position] != rune('\r') {
						goto l427
					}
					position++
				}
			l431:
			l429:
				{
					position430, tokenIndex430, depth430 := position, tokenIndex, depth
					{
						position435, tokenIndex435, depth435 := position, tokenIndex, depth
						if buffer[position] != rune(' ') {
							goto l436
						}
						position++
						goto l435
					l436:
						position, tokenIndex, depth = position435, tokenIndex435, depth435
						if buffer[position] != rune('\t') {
							goto l437
						}
						position++
						goto l435
					l437:
						position, tokenIndex, depth = position435, tokenIndex435, depth435
						if buffer[position] != rune('\n') {
							goto l438
						}
						position++
						goto l435
					l438:
						position, tokenIndex, depth = position435, tokenIndex435, depth435
						if buffer[position] != rune('\r') {
							goto l430
						}
						position++
					}
				l435:
					goto l429
				l430:
					position, tokenIndex, depth = position430, tokenIndex430, depth430
				}
				depth--
				add(rulereq_ws, position428)
			}
			return true
		l427:
			position, tokenIndex, depth = position427, tokenIndex427, depth427
			return false
		},
		/* 100 Action0 <- <{}> */
		func() bool {
			{
				add(ruleAction0, position)
			}
			return true
		},
		/* 101 Action1 <- <{}> */
		func() bool {
			{
				add(ruleAction1, position)
			}
			return true
		},
		/* 102 Action2 <- <{}> */
		func() bool {
			{
				add(ruleAction2, position)
			}
			return true
		},
	}
	p.rules = _rules
}
