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
	ruleTagMarker
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
	ruleTopIndex
	ruleIndices
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
	ruleTagPrefix
	ruleTag
	ruleTagComponent
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
	"TagMarker",
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
	"TopIndex",
	"Indices",
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
	"TagPrefix",
	"Tag",
	"TagComponent",
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
	rules  [109]func() bool
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
		/* 4 Marker <- <('&' (('t' 'e' 'm' 'p' 'l' 'a' 't' 'e') / ('t' 'e' 'm' 'p' 'o' 'r' 'a' 'r' 'y') / ('l' 'o' 'c' 'a' 'l') / ('i' 'n' 'j' 'e' 'c' 't') / ('s' 't' 'a' 't' 'e') / ('d' 'e' 'f' 'a' 'u' 'l' 't') / ('d' 'y' 'n' 'a' 'm' 'i' 'c') / TagMarker))> */
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
						goto l24
					}
					position++
					if buffer[position] != rune('e') {
						goto l24
					}
					position++
					if buffer[position] != rune('f') {
						goto l24
					}
					position++
					if buffer[position] != rune('a') {
						goto l24
					}
					position++
					if buffer[position] != rune('u') {
						goto l24
					}
					position++
					if buffer[position] != rune('l') {
						goto l24
					}
					position++
					if buffer[position] != rune('t') {
						goto l24
					}
					position++
					goto l18
				l24:
					position, tokenIndex, depth = position18, tokenIndex18, depth18
					if buffer[position] != rune('d') {
						goto l25
					}
					position++
					if buffer[position] != rune('y') {
						goto l25
					}
					position++
					if buffer[position] != rune('n') {
						goto l25
					}
					position++
					if buffer[position] != rune('a') {
						goto l25
					}
					position++
					if buffer[position] != rune('m') {
						goto l25
					}
					position++
					if buffer[position] != rune('i') {
						goto l25
					}
					position++
					if buffer[position] != rune('c') {
						goto l25
					}
					position++
					goto l18
				l25:
					position, tokenIndex, depth = position18, tokenIndex18, depth18
					if !_rules[ruleTagMarker]() {
						goto l16
					}
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
		/* 5 TagMarker <- <('t' 'a' 'g' ':' '*'? Tag)> */
		func() bool {
			position26, tokenIndex26, depth26 := position, tokenIndex, depth
			{
				position27 := position
				depth++
				if buffer[position] != rune('t') {
					goto l26
				}
				position++
				if buffer[position] != rune('a') {
					goto l26
				}
				position++
				if buffer[position] != rune('g') {
					goto l26
				}
				position++
				if buffer[position] != rune(':') {
					goto l26
				}
				position++
				{
					position28, tokenIndex28, depth28 := position, tokenIndex, depth
					if buffer[position] != rune('*') {
						goto l28
					}
					position++
					goto l29
				l28:
					position, tokenIndex, depth = position28, tokenIndex28, depth28
				}
			l29:
				if !_rules[ruleTag]() {
					goto l26
				}
				depth--
				add(ruleTagMarker, position27)
			}
			return true
		l26:
			position, tokenIndex, depth = position26, tokenIndex26, depth26
			return false
		},
		/* 6 MarkerExpression <- <Grouped> */
		func() bool {
			position30, tokenIndex30, depth30 := position, tokenIndex, depth
			{
				position31 := position
				depth++
				if !_rules[ruleGrouped]() {
					goto l30
				}
				depth--
				add(ruleMarkerExpression, position31)
			}
			return true
		l30:
			position, tokenIndex, depth = position30, tokenIndex30, depth30
			return false
		},
		/* 7 Expression <- <((Scoped / LambdaExpr / Level7) ws)> */
		func() bool {
			position32, tokenIndex32, depth32 := position, tokenIndex, depth
			{
				position33 := position
				depth++
				{
					position34, tokenIndex34, depth34 := position, tokenIndex, depth
					if !_rules[ruleScoped]() {
						goto l35
					}
					goto l34
				l35:
					position, tokenIndex, depth = position34, tokenIndex34, depth34
					if !_rules[ruleLambdaExpr]() {
						goto l36
					}
					goto l34
				l36:
					position, tokenIndex, depth = position34, tokenIndex34, depth34
					if !_rules[ruleLevel7]() {
						goto l32
					}
				}
			l34:
				if !_rules[rulews]() {
					goto l32
				}
				depth--
				add(ruleExpression, position33)
			}
			return true
		l32:
			position, tokenIndex, depth = position32, tokenIndex32, depth32
			return false
		},
		/* 8 Scoped <- <(ws Scope ws Expression)> */
		func() bool {
			position37, tokenIndex37, depth37 := position, tokenIndex, depth
			{
				position38 := position
				depth++
				if !_rules[rulews]() {
					goto l37
				}
				if !_rules[ruleScope]() {
					goto l37
				}
				if !_rules[rulews]() {
					goto l37
				}
				if !_rules[ruleExpression]() {
					goto l37
				}
				depth--
				add(ruleScoped, position38)
			}
			return true
		l37:
			position, tokenIndex, depth = position37, tokenIndex37, depth37
			return false
		},
		/* 9 Scope <- <(CreateScope ws Assignments? ')')> */
		func() bool {
			position39, tokenIndex39, depth39 := position, tokenIndex, depth
			{
				position40 := position
				depth++
				if !_rules[ruleCreateScope]() {
					goto l39
				}
				if !_rules[rulews]() {
					goto l39
				}
				{
					position41, tokenIndex41, depth41 := position, tokenIndex, depth
					if !_rules[ruleAssignments]() {
						goto l41
					}
					goto l42
				l41:
					position, tokenIndex, depth = position41, tokenIndex41, depth41
				}
			l42:
				if buffer[position] != rune(')') {
					goto l39
				}
				position++
				depth--
				add(ruleScope, position40)
			}
			return true
		l39:
			position, tokenIndex, depth = position39, tokenIndex39, depth39
			return false
		},
		/* 10 CreateScope <- <'('> */
		func() bool {
			position43, tokenIndex43, depth43 := position, tokenIndex, depth
			{
				position44 := position
				depth++
				if buffer[position] != rune('(') {
					goto l43
				}
				position++
				depth--
				add(ruleCreateScope, position44)
			}
			return true
		l43:
			position, tokenIndex, depth = position43, tokenIndex43, depth43
			return false
		},
		/* 11 Level7 <- <(ws Level6 (req_ws Or)*)> */
		func() bool {
			position45, tokenIndex45, depth45 := position, tokenIndex, depth
			{
				position46 := position
				depth++
				if !_rules[rulews]() {
					goto l45
				}
				if !_rules[ruleLevel6]() {
					goto l45
				}
			l47:
				{
					position48, tokenIndex48, depth48 := position, tokenIndex, depth
					if !_rules[rulereq_ws]() {
						goto l48
					}
					if !_rules[ruleOr]() {
						goto l48
					}
					goto l47
				l48:
					position, tokenIndex, depth = position48, tokenIndex48, depth48
				}
				depth--
				add(ruleLevel7, position46)
			}
			return true
		l45:
			position, tokenIndex, depth = position45, tokenIndex45, depth45
			return false
		},
		/* 12 Or <- <(OrOp req_ws Level6)> */
		func() bool {
			position49, tokenIndex49, depth49 := position, tokenIndex, depth
			{
				position50 := position
				depth++
				if !_rules[ruleOrOp]() {
					goto l49
				}
				if !_rules[rulereq_ws]() {
					goto l49
				}
				if !_rules[ruleLevel6]() {
					goto l49
				}
				depth--
				add(ruleOr, position50)
			}
			return true
		l49:
			position, tokenIndex, depth = position49, tokenIndex49, depth49
			return false
		},
		/* 13 OrOp <- <(('|' '|') / ('/' '/'))> */
		func() bool {
			position51, tokenIndex51, depth51 := position, tokenIndex, depth
			{
				position52 := position
				depth++
				{
					position53, tokenIndex53, depth53 := position, tokenIndex, depth
					if buffer[position] != rune('|') {
						goto l54
					}
					position++
					if buffer[position] != rune('|') {
						goto l54
					}
					position++
					goto l53
				l54:
					position, tokenIndex, depth = position53, tokenIndex53, depth53
					if buffer[position] != rune('/') {
						goto l51
					}
					position++
					if buffer[position] != rune('/') {
						goto l51
					}
					position++
				}
			l53:
				depth--
				add(ruleOrOp, position52)
			}
			return true
		l51:
			position, tokenIndex, depth = position51, tokenIndex51, depth51
			return false
		},
		/* 14 Level6 <- <(Conditional / Level5)> */
		func() bool {
			position55, tokenIndex55, depth55 := position, tokenIndex, depth
			{
				position56 := position
				depth++
				{
					position57, tokenIndex57, depth57 := position, tokenIndex, depth
					if !_rules[ruleConditional]() {
						goto l58
					}
					goto l57
				l58:
					position, tokenIndex, depth = position57, tokenIndex57, depth57
					if !_rules[ruleLevel5]() {
						goto l55
					}
				}
			l57:
				depth--
				add(ruleLevel6, position56)
			}
			return true
		l55:
			position, tokenIndex, depth = position55, tokenIndex55, depth55
			return false
		},
		/* 15 Conditional <- <(Level5 ws '?' Expression ':' Expression)> */
		func() bool {
			position59, tokenIndex59, depth59 := position, tokenIndex, depth
			{
				position60 := position
				depth++
				if !_rules[ruleLevel5]() {
					goto l59
				}
				if !_rules[rulews]() {
					goto l59
				}
				if buffer[position] != rune('?') {
					goto l59
				}
				position++
				if !_rules[ruleExpression]() {
					goto l59
				}
				if buffer[position] != rune(':') {
					goto l59
				}
				position++
				if !_rules[ruleExpression]() {
					goto l59
				}
				depth--
				add(ruleConditional, position60)
			}
			return true
		l59:
			position, tokenIndex, depth = position59, tokenIndex59, depth59
			return false
		},
		/* 16 Level5 <- <(Level4 Concatenation*)> */
		func() bool {
			position61, tokenIndex61, depth61 := position, tokenIndex, depth
			{
				position62 := position
				depth++
				if !_rules[ruleLevel4]() {
					goto l61
				}
			l63:
				{
					position64, tokenIndex64, depth64 := position, tokenIndex, depth
					if !_rules[ruleConcatenation]() {
						goto l64
					}
					goto l63
				l64:
					position, tokenIndex, depth = position64, tokenIndex64, depth64
				}
				depth--
				add(ruleLevel5, position62)
			}
			return true
		l61:
			position, tokenIndex, depth = position61, tokenIndex61, depth61
			return false
		},
		/* 17 Concatenation <- <(req_ws Level4)> */
		func() bool {
			position65, tokenIndex65, depth65 := position, tokenIndex, depth
			{
				position66 := position
				depth++
				if !_rules[rulereq_ws]() {
					goto l65
				}
				if !_rules[ruleLevel4]() {
					goto l65
				}
				depth--
				add(ruleConcatenation, position66)
			}
			return true
		l65:
			position, tokenIndex, depth = position65, tokenIndex65, depth65
			return false
		},
		/* 18 Level4 <- <(Level3 (req_ws (LogOr / LogAnd))*)> */
		func() bool {
			position67, tokenIndex67, depth67 := position, tokenIndex, depth
			{
				position68 := position
				depth++
				if !_rules[ruleLevel3]() {
					goto l67
				}
			l69:
				{
					position70, tokenIndex70, depth70 := position, tokenIndex, depth
					if !_rules[rulereq_ws]() {
						goto l70
					}
					{
						position71, tokenIndex71, depth71 := position, tokenIndex, depth
						if !_rules[ruleLogOr]() {
							goto l72
						}
						goto l71
					l72:
						position, tokenIndex, depth = position71, tokenIndex71, depth71
						if !_rules[ruleLogAnd]() {
							goto l70
						}
					}
				l71:
					goto l69
				l70:
					position, tokenIndex, depth = position70, tokenIndex70, depth70
				}
				depth--
				add(ruleLevel4, position68)
			}
			return true
		l67:
			position, tokenIndex, depth = position67, tokenIndex67, depth67
			return false
		},
		/* 19 LogOr <- <('-' 'o' 'r' req_ws Level3)> */
		func() bool {
			position73, tokenIndex73, depth73 := position, tokenIndex, depth
			{
				position74 := position
				depth++
				if buffer[position] != rune('-') {
					goto l73
				}
				position++
				if buffer[position] != rune('o') {
					goto l73
				}
				position++
				if buffer[position] != rune('r') {
					goto l73
				}
				position++
				if !_rules[rulereq_ws]() {
					goto l73
				}
				if !_rules[ruleLevel3]() {
					goto l73
				}
				depth--
				add(ruleLogOr, position74)
			}
			return true
		l73:
			position, tokenIndex, depth = position73, tokenIndex73, depth73
			return false
		},
		/* 20 LogAnd <- <('-' 'a' 'n' 'd' req_ws Level3)> */
		func() bool {
			position75, tokenIndex75, depth75 := position, tokenIndex, depth
			{
				position76 := position
				depth++
				if buffer[position] != rune('-') {
					goto l75
				}
				position++
				if buffer[position] != rune('a') {
					goto l75
				}
				position++
				if buffer[position] != rune('n') {
					goto l75
				}
				position++
				if buffer[position] != rune('d') {
					goto l75
				}
				position++
				if !_rules[rulereq_ws]() {
					goto l75
				}
				if !_rules[ruleLevel3]() {
					goto l75
				}
				depth--
				add(ruleLogAnd, position76)
			}
			return true
		l75:
			position, tokenIndex, depth = position75, tokenIndex75, depth75
			return false
		},
		/* 21 Level3 <- <(Level2 (req_ws Comparison)*)> */
		func() bool {
			position77, tokenIndex77, depth77 := position, tokenIndex, depth
			{
				position78 := position
				depth++
				if !_rules[ruleLevel2]() {
					goto l77
				}
			l79:
				{
					position80, tokenIndex80, depth80 := position, tokenIndex, depth
					if !_rules[rulereq_ws]() {
						goto l80
					}
					if !_rules[ruleComparison]() {
						goto l80
					}
					goto l79
				l80:
					position, tokenIndex, depth = position80, tokenIndex80, depth80
				}
				depth--
				add(ruleLevel3, position78)
			}
			return true
		l77:
			position, tokenIndex, depth = position77, tokenIndex77, depth77
			return false
		},
		/* 22 Comparison <- <(CompareOp req_ws Level2)> */
		func() bool {
			position81, tokenIndex81, depth81 := position, tokenIndex, depth
			{
				position82 := position
				depth++
				if !_rules[ruleCompareOp]() {
					goto l81
				}
				if !_rules[rulereq_ws]() {
					goto l81
				}
				if !_rules[ruleLevel2]() {
					goto l81
				}
				depth--
				add(ruleComparison, position82)
			}
			return true
		l81:
			position, tokenIndex, depth = position81, tokenIndex81, depth81
			return false
		},
		/* 23 CompareOp <- <(('=' '=') / ('!' '=') / ('<' '=') / ('>' '=') / '>' / '<' / '>')> */
		func() bool {
			position83, tokenIndex83, depth83 := position, tokenIndex, depth
			{
				position84 := position
				depth++
				{
					position85, tokenIndex85, depth85 := position, tokenIndex, depth
					if buffer[position] != rune('=') {
						goto l86
					}
					position++
					if buffer[position] != rune('=') {
						goto l86
					}
					position++
					goto l85
				l86:
					position, tokenIndex, depth = position85, tokenIndex85, depth85
					if buffer[position] != rune('!') {
						goto l87
					}
					position++
					if buffer[position] != rune('=') {
						goto l87
					}
					position++
					goto l85
				l87:
					position, tokenIndex, depth = position85, tokenIndex85, depth85
					if buffer[position] != rune('<') {
						goto l88
					}
					position++
					if buffer[position] != rune('=') {
						goto l88
					}
					position++
					goto l85
				l88:
					position, tokenIndex, depth = position85, tokenIndex85, depth85
					if buffer[position] != rune('>') {
						goto l89
					}
					position++
					if buffer[position] != rune('=') {
						goto l89
					}
					position++
					goto l85
				l89:
					position, tokenIndex, depth = position85, tokenIndex85, depth85
					if buffer[position] != rune('>') {
						goto l90
					}
					position++
					goto l85
				l90:
					position, tokenIndex, depth = position85, tokenIndex85, depth85
					if buffer[position] != rune('<') {
						goto l91
					}
					position++
					goto l85
				l91:
					position, tokenIndex, depth = position85, tokenIndex85, depth85
					if buffer[position] != rune('>') {
						goto l83
					}
					position++
				}
			l85:
				depth--
				add(ruleCompareOp, position84)
			}
			return true
		l83:
			position, tokenIndex, depth = position83, tokenIndex83, depth83
			return false
		},
		/* 24 Level2 <- <(Level1 (req_ws (Addition / Subtraction))*)> */
		func() bool {
			position92, tokenIndex92, depth92 := position, tokenIndex, depth
			{
				position93 := position
				depth++
				if !_rules[ruleLevel1]() {
					goto l92
				}
			l94:
				{
					position95, tokenIndex95, depth95 := position, tokenIndex, depth
					if !_rules[rulereq_ws]() {
						goto l95
					}
					{
						position96, tokenIndex96, depth96 := position, tokenIndex, depth
						if !_rules[ruleAddition]() {
							goto l97
						}
						goto l96
					l97:
						position, tokenIndex, depth = position96, tokenIndex96, depth96
						if !_rules[ruleSubtraction]() {
							goto l95
						}
					}
				l96:
					goto l94
				l95:
					position, tokenIndex, depth = position95, tokenIndex95, depth95
				}
				depth--
				add(ruleLevel2, position93)
			}
			return true
		l92:
			position, tokenIndex, depth = position92, tokenIndex92, depth92
			return false
		},
		/* 25 Addition <- <('+' req_ws Level1)> */
		func() bool {
			position98, tokenIndex98, depth98 := position, tokenIndex, depth
			{
				position99 := position
				depth++
				if buffer[position] != rune('+') {
					goto l98
				}
				position++
				if !_rules[rulereq_ws]() {
					goto l98
				}
				if !_rules[ruleLevel1]() {
					goto l98
				}
				depth--
				add(ruleAddition, position99)
			}
			return true
		l98:
			position, tokenIndex, depth = position98, tokenIndex98, depth98
			return false
		},
		/* 26 Subtraction <- <('-' req_ws Level1)> */
		func() bool {
			position100, tokenIndex100, depth100 := position, tokenIndex, depth
			{
				position101 := position
				depth++
				if buffer[position] != rune('-') {
					goto l100
				}
				position++
				if !_rules[rulereq_ws]() {
					goto l100
				}
				if !_rules[ruleLevel1]() {
					goto l100
				}
				depth--
				add(ruleSubtraction, position101)
			}
			return true
		l100:
			position, tokenIndex, depth = position100, tokenIndex100, depth100
			return false
		},
		/* 27 Level1 <- <(Level0 (req_ws (Multiplication / Division / Modulo))*)> */
		func() bool {
			position102, tokenIndex102, depth102 := position, tokenIndex, depth
			{
				position103 := position
				depth++
				if !_rules[ruleLevel0]() {
					goto l102
				}
			l104:
				{
					position105, tokenIndex105, depth105 := position, tokenIndex, depth
					if !_rules[rulereq_ws]() {
						goto l105
					}
					{
						position106, tokenIndex106, depth106 := position, tokenIndex, depth
						if !_rules[ruleMultiplication]() {
							goto l107
						}
						goto l106
					l107:
						position, tokenIndex, depth = position106, tokenIndex106, depth106
						if !_rules[ruleDivision]() {
							goto l108
						}
						goto l106
					l108:
						position, tokenIndex, depth = position106, tokenIndex106, depth106
						if !_rules[ruleModulo]() {
							goto l105
						}
					}
				l106:
					goto l104
				l105:
					position, tokenIndex, depth = position105, tokenIndex105, depth105
				}
				depth--
				add(ruleLevel1, position103)
			}
			return true
		l102:
			position, tokenIndex, depth = position102, tokenIndex102, depth102
			return false
		},
		/* 28 Multiplication <- <('*' req_ws Level0)> */
		func() bool {
			position109, tokenIndex109, depth109 := position, tokenIndex, depth
			{
				position110 := position
				depth++
				if buffer[position] != rune('*') {
					goto l109
				}
				position++
				if !_rules[rulereq_ws]() {
					goto l109
				}
				if !_rules[ruleLevel0]() {
					goto l109
				}
				depth--
				add(ruleMultiplication, position110)
			}
			return true
		l109:
			position, tokenIndex, depth = position109, tokenIndex109, depth109
			return false
		},
		/* 29 Division <- <('/' req_ws Level0)> */
		func() bool {
			position111, tokenIndex111, depth111 := position, tokenIndex, depth
			{
				position112 := position
				depth++
				if buffer[position] != rune('/') {
					goto l111
				}
				position++
				if !_rules[rulereq_ws]() {
					goto l111
				}
				if !_rules[ruleLevel0]() {
					goto l111
				}
				depth--
				add(ruleDivision, position112)
			}
			return true
		l111:
			position, tokenIndex, depth = position111, tokenIndex111, depth111
			return false
		},
		/* 30 Modulo <- <('%' req_ws Level0)> */
		func() bool {
			position113, tokenIndex113, depth113 := position, tokenIndex, depth
			{
				position114 := position
				depth++
				if buffer[position] != rune('%') {
					goto l113
				}
				position++
				if !_rules[rulereq_ws]() {
					goto l113
				}
				if !_rules[ruleLevel0]() {
					goto l113
				}
				depth--
				add(ruleModulo, position114)
			}
			return true
		l113:
			position, tokenIndex, depth = position113, tokenIndex113, depth113
			return false
		},
		/* 31 Level0 <- <(IP / String / Number / Boolean / Undefined / Nil / Symbol / Not / Substitution / Merge / Auto / Lambda / Chained)> */
		func() bool {
			position115, tokenIndex115, depth115 := position, tokenIndex, depth
			{
				position116 := position
				depth++
				{
					position117, tokenIndex117, depth117 := position, tokenIndex, depth
					if !_rules[ruleIP]() {
						goto l118
					}
					goto l117
				l118:
					position, tokenIndex, depth = position117, tokenIndex117, depth117
					if !_rules[ruleString]() {
						goto l119
					}
					goto l117
				l119:
					position, tokenIndex, depth = position117, tokenIndex117, depth117
					if !_rules[ruleNumber]() {
						goto l120
					}
					goto l117
				l120:
					position, tokenIndex, depth = position117, tokenIndex117, depth117
					if !_rules[ruleBoolean]() {
						goto l121
					}
					goto l117
				l121:
					position, tokenIndex, depth = position117, tokenIndex117, depth117
					if !_rules[ruleUndefined]() {
						goto l122
					}
					goto l117
				l122:
					position, tokenIndex, depth = position117, tokenIndex117, depth117
					if !_rules[ruleNil]() {
						goto l123
					}
					goto l117
				l123:
					position, tokenIndex, depth = position117, tokenIndex117, depth117
					if !_rules[ruleSymbol]() {
						goto l124
					}
					goto l117
				l124:
					position, tokenIndex, depth = position117, tokenIndex117, depth117
					if !_rules[ruleNot]() {
						goto l125
					}
					goto l117
				l125:
					position, tokenIndex, depth = position117, tokenIndex117, depth117
					if !_rules[ruleSubstitution]() {
						goto l126
					}
					goto l117
				l126:
					position, tokenIndex, depth = position117, tokenIndex117, depth117
					if !_rules[ruleMerge]() {
						goto l127
					}
					goto l117
				l127:
					position, tokenIndex, depth = position117, tokenIndex117, depth117
					if !_rules[ruleAuto]() {
						goto l128
					}
					goto l117
				l128:
					position, tokenIndex, depth = position117, tokenIndex117, depth117
					if !_rules[ruleLambda]() {
						goto l129
					}
					goto l117
				l129:
					position, tokenIndex, depth = position117, tokenIndex117, depth117
					if !_rules[ruleChained]() {
						goto l115
					}
				}
			l117:
				depth--
				add(ruleLevel0, position116)
			}
			return true
		l115:
			position, tokenIndex, depth = position115, tokenIndex115, depth115
			return false
		},
		/* 32 Chained <- <((MapMapping / Sync / Catch / Mapping / MapSelection / Selection / Sum / List / Map / Range / Grouped / Reference / TopIndex) ChainedQualifiedExpression*)> */
		func() bool {
			position130, tokenIndex130, depth130 := position, tokenIndex, depth
			{
				position131 := position
				depth++
				{
					position132, tokenIndex132, depth132 := position, tokenIndex, depth
					if !_rules[ruleMapMapping]() {
						goto l133
					}
					goto l132
				l133:
					position, tokenIndex, depth = position132, tokenIndex132, depth132
					if !_rules[ruleSync]() {
						goto l134
					}
					goto l132
				l134:
					position, tokenIndex, depth = position132, tokenIndex132, depth132
					if !_rules[ruleCatch]() {
						goto l135
					}
					goto l132
				l135:
					position, tokenIndex, depth = position132, tokenIndex132, depth132
					if !_rules[ruleMapping]() {
						goto l136
					}
					goto l132
				l136:
					position, tokenIndex, depth = position132, tokenIndex132, depth132
					if !_rules[ruleMapSelection]() {
						goto l137
					}
					goto l132
				l137:
					position, tokenIndex, depth = position132, tokenIndex132, depth132
					if !_rules[ruleSelection]() {
						goto l138
					}
					goto l132
				l138:
					position, tokenIndex, depth = position132, tokenIndex132, depth132
					if !_rules[ruleSum]() {
						goto l139
					}
					goto l132
				l139:
					position, tokenIndex, depth = position132, tokenIndex132, depth132
					if !_rules[ruleList]() {
						goto l140
					}
					goto l132
				l140:
					position, tokenIndex, depth = position132, tokenIndex132, depth132
					if !_rules[ruleMap]() {
						goto l141
					}
					goto l132
				l141:
					position, tokenIndex, depth = position132, tokenIndex132, depth132
					if !_rules[ruleRange]() {
						goto l142
					}
					goto l132
				l142:
					position, tokenIndex, depth = position132, tokenIndex132, depth132
					if !_rules[ruleGrouped]() {
						goto l143
					}
					goto l132
				l143:
					position, tokenIndex, depth = position132, tokenIndex132, depth132
					if !_rules[ruleReference]() {
						goto l144
					}
					goto l132
				l144:
					position, tokenIndex, depth = position132, tokenIndex132, depth132
					if !_rules[ruleTopIndex]() {
						goto l130
					}
				}
			l132:
			l145:
				{
					position146, tokenIndex146, depth146 := position, tokenIndex, depth
					if !_rules[ruleChainedQualifiedExpression]() {
						goto l146
					}
					goto l145
				l146:
					position, tokenIndex, depth = position146, tokenIndex146, depth146
				}
				depth--
				add(ruleChained, position131)
			}
			return true
		l130:
			position, tokenIndex, depth = position130, tokenIndex130, depth130
			return false
		},
		/* 33 ChainedQualifiedExpression <- <(ChainedCall / Currying / ChainedRef / ChainedDynRef / Projection)> */
		func() bool {
			position147, tokenIndex147, depth147 := position, tokenIndex, depth
			{
				position148 := position
				depth++
				{
					position149, tokenIndex149, depth149 := position, tokenIndex, depth
					if !_rules[ruleChainedCall]() {
						goto l150
					}
					goto l149
				l150:
					position, tokenIndex, depth = position149, tokenIndex149, depth149
					if !_rules[ruleCurrying]() {
						goto l151
					}
					goto l149
				l151:
					position, tokenIndex, depth = position149, tokenIndex149, depth149
					if !_rules[ruleChainedRef]() {
						goto l152
					}
					goto l149
				l152:
					position, tokenIndex, depth = position149, tokenIndex149, depth149
					if !_rules[ruleChainedDynRef]() {
						goto l153
					}
					goto l149
				l153:
					position, tokenIndex, depth = position149, tokenIndex149, depth149
					if !_rules[ruleProjection]() {
						goto l147
					}
				}
			l149:
				depth--
				add(ruleChainedQualifiedExpression, position148)
			}
			return true
		l147:
			position, tokenIndex, depth = position147, tokenIndex147, depth147
			return false
		},
		/* 34 ChainedRef <- <(PathComponent FollowUpRef)> */
		func() bool {
			position154, tokenIndex154, depth154 := position, tokenIndex, depth
			{
				position155 := position
				depth++
				if !_rules[rulePathComponent]() {
					goto l154
				}
				if !_rules[ruleFollowUpRef]() {
					goto l154
				}
				depth--
				add(ruleChainedRef, position155)
			}
			return true
		l154:
			position, tokenIndex, depth = position154, tokenIndex154, depth154
			return false
		},
		/* 35 ChainedDynRef <- <('.'? Indices)> */
		func() bool {
			position156, tokenIndex156, depth156 := position, tokenIndex, depth
			{
				position157 := position
				depth++
				{
					position158, tokenIndex158, depth158 := position, tokenIndex, depth
					if buffer[position] != rune('.') {
						goto l158
					}
					position++
					goto l159
				l158:
					position, tokenIndex, depth = position158, tokenIndex158, depth158
				}
			l159:
				if !_rules[ruleIndices]() {
					goto l156
				}
				depth--
				add(ruleChainedDynRef, position157)
			}
			return true
		l156:
			position, tokenIndex, depth = position156, tokenIndex156, depth156
			return false
		},
		/* 36 TopIndex <- <('.' Indices)> */
		func() bool {
			position160, tokenIndex160, depth160 := position, tokenIndex, depth
			{
				position161 := position
				depth++
				if buffer[position] != rune('.') {
					goto l160
				}
				position++
				if !_rules[ruleIndices]() {
					goto l160
				}
				depth--
				add(ruleTopIndex, position161)
			}
			return true
		l160:
			position, tokenIndex, depth = position160, tokenIndex160, depth160
			return false
		},
		/* 37 Indices <- <(StartList ExpressionList ']')> */
		func() bool {
			position162, tokenIndex162, depth162 := position, tokenIndex, depth
			{
				position163 := position
				depth++
				if !_rules[ruleStartList]() {
					goto l162
				}
				if !_rules[ruleExpressionList]() {
					goto l162
				}
				if buffer[position] != rune(']') {
					goto l162
				}
				position++
				depth--
				add(ruleIndices, position163)
			}
			return true
		l162:
			position, tokenIndex, depth = position162, tokenIndex162, depth162
			return false
		},
		/* 38 Slice <- <Range> */
		func() bool {
			position164, tokenIndex164, depth164 := position, tokenIndex, depth
			{
				position165 := position
				depth++
				if !_rules[ruleRange]() {
					goto l164
				}
				depth--
				add(ruleSlice, position165)
			}
			return true
		l164:
			position, tokenIndex, depth = position164, tokenIndex164, depth164
			return false
		},
		/* 39 Currying <- <('*' ChainedCall)> */
		func() bool {
			position166, tokenIndex166, depth166 := position, tokenIndex, depth
			{
				position167 := position
				depth++
				if buffer[position] != rune('*') {
					goto l166
				}
				position++
				if !_rules[ruleChainedCall]() {
					goto l166
				}
				depth--
				add(ruleCurrying, position167)
			}
			return true
		l166:
			position, tokenIndex, depth = position166, tokenIndex166, depth166
			return false
		},
		/* 40 ChainedCall <- <(StartArguments NameArgumentList? ')')> */
		func() bool {
			position168, tokenIndex168, depth168 := position, tokenIndex, depth
			{
				position169 := position
				depth++
				if !_rules[ruleStartArguments]() {
					goto l168
				}
				{
					position170, tokenIndex170, depth170 := position, tokenIndex, depth
					if !_rules[ruleNameArgumentList]() {
						goto l170
					}
					goto l171
				l170:
					position, tokenIndex, depth = position170, tokenIndex170, depth170
				}
			l171:
				if buffer[position] != rune(')') {
					goto l168
				}
				position++
				depth--
				add(ruleChainedCall, position169)
			}
			return true
		l168:
			position, tokenIndex, depth = position168, tokenIndex168, depth168
			return false
		},
		/* 41 StartArguments <- <('(' ws)> */
		func() bool {
			position172, tokenIndex172, depth172 := position, tokenIndex, depth
			{
				position173 := position
				depth++
				if buffer[position] != rune('(') {
					goto l172
				}
				position++
				if !_rules[rulews]() {
					goto l172
				}
				depth--
				add(ruleStartArguments, position173)
			}
			return true
		l172:
			position, tokenIndex, depth = position172, tokenIndex172, depth172
			return false
		},
		/* 42 NameArgumentList <- <(((NextNameArgument (',' NextNameArgument)*) / NextExpression) (',' NextExpression)*)> */
		func() bool {
			position174, tokenIndex174, depth174 := position, tokenIndex, depth
			{
				position175 := position
				depth++
				{
					position176, tokenIndex176, depth176 := position, tokenIndex, depth
					if !_rules[ruleNextNameArgument]() {
						goto l177
					}
				l178:
					{
						position179, tokenIndex179, depth179 := position, tokenIndex, depth
						if buffer[position] != rune(',') {
							goto l179
						}
						position++
						if !_rules[ruleNextNameArgument]() {
							goto l179
						}
						goto l178
					l179:
						position, tokenIndex, depth = position179, tokenIndex179, depth179
					}
					goto l176
				l177:
					position, tokenIndex, depth = position176, tokenIndex176, depth176
					if !_rules[ruleNextExpression]() {
						goto l174
					}
				}
			l176:
			l180:
				{
					position181, tokenIndex181, depth181 := position, tokenIndex, depth
					if buffer[position] != rune(',') {
						goto l181
					}
					position++
					if !_rules[ruleNextExpression]() {
						goto l181
					}
					goto l180
				l181:
					position, tokenIndex, depth = position181, tokenIndex181, depth181
				}
				depth--
				add(ruleNameArgumentList, position175)
			}
			return true
		l174:
			position, tokenIndex, depth = position174, tokenIndex174, depth174
			return false
		},
		/* 43 NextNameArgument <- <(ws Name ws '=' ws Expression ws)> */
		func() bool {
			position182, tokenIndex182, depth182 := position, tokenIndex, depth
			{
				position183 := position
				depth++
				if !_rules[rulews]() {
					goto l182
				}
				if !_rules[ruleName]() {
					goto l182
				}
				if !_rules[rulews]() {
					goto l182
				}
				if buffer[position] != rune('=') {
					goto l182
				}
				position++
				if !_rules[rulews]() {
					goto l182
				}
				if !_rules[ruleExpression]() {
					goto l182
				}
				if !_rules[rulews]() {
					goto l182
				}
				depth--
				add(ruleNextNameArgument, position183)
			}
			return true
		l182:
			position, tokenIndex, depth = position182, tokenIndex182, depth182
			return false
		},
		/* 44 ExpressionList <- <(NextExpression (',' NextExpression)*)> */
		func() bool {
			position184, tokenIndex184, depth184 := position, tokenIndex, depth
			{
				position185 := position
				depth++
				if !_rules[ruleNextExpression]() {
					goto l184
				}
			l186:
				{
					position187, tokenIndex187, depth187 := position, tokenIndex, depth
					if buffer[position] != rune(',') {
						goto l187
					}
					position++
					if !_rules[ruleNextExpression]() {
						goto l187
					}
					goto l186
				l187:
					position, tokenIndex, depth = position187, tokenIndex187, depth187
				}
				depth--
				add(ruleExpressionList, position185)
			}
			return true
		l184:
			position, tokenIndex, depth = position184, tokenIndex184, depth184
			return false
		},
		/* 45 NextExpression <- <(Expression ListExpansion?)> */
		func() bool {
			position188, tokenIndex188, depth188 := position, tokenIndex, depth
			{
				position189 := position
				depth++
				if !_rules[ruleExpression]() {
					goto l188
				}
				{
					position190, tokenIndex190, depth190 := position, tokenIndex, depth
					if !_rules[ruleListExpansion]() {
						goto l190
					}
					goto l191
				l190:
					position, tokenIndex, depth = position190, tokenIndex190, depth190
				}
			l191:
				depth--
				add(ruleNextExpression, position189)
			}
			return true
		l188:
			position, tokenIndex, depth = position188, tokenIndex188, depth188
			return false
		},
		/* 46 ListExpansion <- <('.' '.' '.' ws)> */
		func() bool {
			position192, tokenIndex192, depth192 := position, tokenIndex, depth
			{
				position193 := position
				depth++
				if buffer[position] != rune('.') {
					goto l192
				}
				position++
				if buffer[position] != rune('.') {
					goto l192
				}
				position++
				if buffer[position] != rune('.') {
					goto l192
				}
				position++
				if !_rules[rulews]() {
					goto l192
				}
				depth--
				add(ruleListExpansion, position193)
			}
			return true
		l192:
			position, tokenIndex, depth = position192, tokenIndex192, depth192
			return false
		},
		/* 47 Projection <- <('.'? (('[' '*' ']') / Slice) ProjectionValue ChainedQualifiedExpression*)> */
		func() bool {
			position194, tokenIndex194, depth194 := position, tokenIndex, depth
			{
				position195 := position
				depth++
				{
					position196, tokenIndex196, depth196 := position, tokenIndex, depth
					if buffer[position] != rune('.') {
						goto l196
					}
					position++
					goto l197
				l196:
					position, tokenIndex, depth = position196, tokenIndex196, depth196
				}
			l197:
				{
					position198, tokenIndex198, depth198 := position, tokenIndex, depth
					if buffer[position] != rune('[') {
						goto l199
					}
					position++
					if buffer[position] != rune('*') {
						goto l199
					}
					position++
					if buffer[position] != rune(']') {
						goto l199
					}
					position++
					goto l198
				l199:
					position, tokenIndex, depth = position198, tokenIndex198, depth198
					if !_rules[ruleSlice]() {
						goto l194
					}
				}
			l198:
				if !_rules[ruleProjectionValue]() {
					goto l194
				}
			l200:
				{
					position201, tokenIndex201, depth201 := position, tokenIndex, depth
					if !_rules[ruleChainedQualifiedExpression]() {
						goto l201
					}
					goto l200
				l201:
					position, tokenIndex, depth = position201, tokenIndex201, depth201
				}
				depth--
				add(ruleProjection, position195)
			}
			return true
		l194:
			position, tokenIndex, depth = position194, tokenIndex194, depth194
			return false
		},
		/* 48 ProjectionValue <- <Action0> */
		func() bool {
			position202, tokenIndex202, depth202 := position, tokenIndex, depth
			{
				position203 := position
				depth++
				if !_rules[ruleAction0]() {
					goto l202
				}
				depth--
				add(ruleProjectionValue, position203)
			}
			return true
		l202:
			position, tokenIndex, depth = position202, tokenIndex202, depth202
			return false
		},
		/* 49 Substitution <- <('*' Level0)> */
		func() bool {
			position204, tokenIndex204, depth204 := position, tokenIndex, depth
			{
				position205 := position
				depth++
				if buffer[position] != rune('*') {
					goto l204
				}
				position++
				if !_rules[ruleLevel0]() {
					goto l204
				}
				depth--
				add(ruleSubstitution, position205)
			}
			return true
		l204:
			position, tokenIndex, depth = position204, tokenIndex204, depth204
			return false
		},
		/* 50 Not <- <('!' ws Level0)> */
		func() bool {
			position206, tokenIndex206, depth206 := position, tokenIndex, depth
			{
				position207 := position
				depth++
				if buffer[position] != rune('!') {
					goto l206
				}
				position++
				if !_rules[rulews]() {
					goto l206
				}
				if !_rules[ruleLevel0]() {
					goto l206
				}
				depth--
				add(ruleNot, position207)
			}
			return true
		l206:
			position, tokenIndex, depth = position206, tokenIndex206, depth206
			return false
		},
		/* 51 Grouped <- <('(' Expression ')')> */
		func() bool {
			position208, tokenIndex208, depth208 := position, tokenIndex, depth
			{
				position209 := position
				depth++
				if buffer[position] != rune('(') {
					goto l208
				}
				position++
				if !_rules[ruleExpression]() {
					goto l208
				}
				if buffer[position] != rune(')') {
					goto l208
				}
				position++
				depth--
				add(ruleGrouped, position209)
			}
			return true
		l208:
			position, tokenIndex, depth = position208, tokenIndex208, depth208
			return false
		},
		/* 52 Range <- <(StartRange Expression? RangeOp Expression? ']')> */
		func() bool {
			position210, tokenIndex210, depth210 := position, tokenIndex, depth
			{
				position211 := position
				depth++
				if !_rules[ruleStartRange]() {
					goto l210
				}
				{
					position212, tokenIndex212, depth212 := position, tokenIndex, depth
					if !_rules[ruleExpression]() {
						goto l212
					}
					goto l213
				l212:
					position, tokenIndex, depth = position212, tokenIndex212, depth212
				}
			l213:
				if !_rules[ruleRangeOp]() {
					goto l210
				}
				{
					position214, tokenIndex214, depth214 := position, tokenIndex, depth
					if !_rules[ruleExpression]() {
						goto l214
					}
					goto l215
				l214:
					position, tokenIndex, depth = position214, tokenIndex214, depth214
				}
			l215:
				if buffer[position] != rune(']') {
					goto l210
				}
				position++
				depth--
				add(ruleRange, position211)
			}
			return true
		l210:
			position, tokenIndex, depth = position210, tokenIndex210, depth210
			return false
		},
		/* 53 StartRange <- <'['> */
		func() bool {
			position216, tokenIndex216, depth216 := position, tokenIndex, depth
			{
				position217 := position
				depth++
				if buffer[position] != rune('[') {
					goto l216
				}
				position++
				depth--
				add(ruleStartRange, position217)
			}
			return true
		l216:
			position, tokenIndex, depth = position216, tokenIndex216, depth216
			return false
		},
		/* 54 RangeOp <- <('.' '.')> */
		func() bool {
			position218, tokenIndex218, depth218 := position, tokenIndex, depth
			{
				position219 := position
				depth++
				if buffer[position] != rune('.') {
					goto l218
				}
				position++
				if buffer[position] != rune('.') {
					goto l218
				}
				position++
				depth--
				add(ruleRangeOp, position219)
			}
			return true
		l218:
			position, tokenIndex, depth = position218, tokenIndex218, depth218
			return false
		},
		/* 55 Number <- <('-'? [0-9] ([0-9] / '_')* ('.' [0-9] [0-9]*)? (('e' / 'E') '-'? [0-9] [0-9]*)? !(':' ':'))> */
		func() bool {
			position220, tokenIndex220, depth220 := position, tokenIndex, depth
			{
				position221 := position
				depth++
				{
					position222, tokenIndex222, depth222 := position, tokenIndex, depth
					if buffer[position] != rune('-') {
						goto l222
					}
					position++
					goto l223
				l222:
					position, tokenIndex, depth = position222, tokenIndex222, depth222
				}
			l223:
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l220
				}
				position++
			l224:
				{
					position225, tokenIndex225, depth225 := position, tokenIndex, depth
					{
						position226, tokenIndex226, depth226 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l227
						}
						position++
						goto l226
					l227:
						position, tokenIndex, depth = position226, tokenIndex226, depth226
						if buffer[position] != rune('_') {
							goto l225
						}
						position++
					}
				l226:
					goto l224
				l225:
					position, tokenIndex, depth = position225, tokenIndex225, depth225
				}
				{
					position228, tokenIndex228, depth228 := position, tokenIndex, depth
					if buffer[position] != rune('.') {
						goto l228
					}
					position++
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l228
					}
					position++
				l230:
					{
						position231, tokenIndex231, depth231 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l231
						}
						position++
						goto l230
					l231:
						position, tokenIndex, depth = position231, tokenIndex231, depth231
					}
					goto l229
				l228:
					position, tokenIndex, depth = position228, tokenIndex228, depth228
				}
			l229:
				{
					position232, tokenIndex232, depth232 := position, tokenIndex, depth
					{
						position234, tokenIndex234, depth234 := position, tokenIndex, depth
						if buffer[position] != rune('e') {
							goto l235
						}
						position++
						goto l234
					l235:
						position, tokenIndex, depth = position234, tokenIndex234, depth234
						if buffer[position] != rune('E') {
							goto l232
						}
						position++
					}
				l234:
					{
						position236, tokenIndex236, depth236 := position, tokenIndex, depth
						if buffer[position] != rune('-') {
							goto l236
						}
						position++
						goto l237
					l236:
						position, tokenIndex, depth = position236, tokenIndex236, depth236
					}
				l237:
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l232
					}
					position++
				l238:
					{
						position239, tokenIndex239, depth239 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l239
						}
						position++
						goto l238
					l239:
						position, tokenIndex, depth = position239, tokenIndex239, depth239
					}
					goto l233
				l232:
					position, tokenIndex, depth = position232, tokenIndex232, depth232
				}
			l233:
				{
					position240, tokenIndex240, depth240 := position, tokenIndex, depth
					if buffer[position] != rune(':') {
						goto l240
					}
					position++
					if buffer[position] != rune(':') {
						goto l240
					}
					position++
					goto l220
				l240:
					position, tokenIndex, depth = position240, tokenIndex240, depth240
				}
				depth--
				add(ruleNumber, position221)
			}
			return true
		l220:
			position, tokenIndex, depth = position220, tokenIndex220, depth220
			return false
		},
		/* 56 String <- <('"' (('\\' '"') / (!'"' .))* '"')> */
		func() bool {
			position241, tokenIndex241, depth241 := position, tokenIndex, depth
			{
				position242 := position
				depth++
				if buffer[position] != rune('"') {
					goto l241
				}
				position++
			l243:
				{
					position244, tokenIndex244, depth244 := position, tokenIndex, depth
					{
						position245, tokenIndex245, depth245 := position, tokenIndex, depth
						if buffer[position] != rune('\\') {
							goto l246
						}
						position++
						if buffer[position] != rune('"') {
							goto l246
						}
						position++
						goto l245
					l246:
						position, tokenIndex, depth = position245, tokenIndex245, depth245
						{
							position247, tokenIndex247, depth247 := position, tokenIndex, depth
							if buffer[position] != rune('"') {
								goto l247
							}
							position++
							goto l244
						l247:
							position, tokenIndex, depth = position247, tokenIndex247, depth247
						}
						if !matchDot() {
							goto l244
						}
					}
				l245:
					goto l243
				l244:
					position, tokenIndex, depth = position244, tokenIndex244, depth244
				}
				if buffer[position] != rune('"') {
					goto l241
				}
				position++
				depth--
				add(ruleString, position242)
			}
			return true
		l241:
			position, tokenIndex, depth = position241, tokenIndex241, depth241
			return false
		},
		/* 57 Boolean <- <(('t' 'r' 'u' 'e') / ('f' 'a' 'l' 's' 'e'))> */
		func() bool {
			position248, tokenIndex248, depth248 := position, tokenIndex, depth
			{
				position249 := position
				depth++
				{
					position250, tokenIndex250, depth250 := position, tokenIndex, depth
					if buffer[position] != rune('t') {
						goto l251
					}
					position++
					if buffer[position] != rune('r') {
						goto l251
					}
					position++
					if buffer[position] != rune('u') {
						goto l251
					}
					position++
					if buffer[position] != rune('e') {
						goto l251
					}
					position++
					goto l250
				l251:
					position, tokenIndex, depth = position250, tokenIndex250, depth250
					if buffer[position] != rune('f') {
						goto l248
					}
					position++
					if buffer[position] != rune('a') {
						goto l248
					}
					position++
					if buffer[position] != rune('l') {
						goto l248
					}
					position++
					if buffer[position] != rune('s') {
						goto l248
					}
					position++
					if buffer[position] != rune('e') {
						goto l248
					}
					position++
				}
			l250:
				depth--
				add(ruleBoolean, position249)
			}
			return true
		l248:
			position, tokenIndex, depth = position248, tokenIndex248, depth248
			return false
		},
		/* 58 Nil <- <(('n' 'i' 'l') / '~')> */
		func() bool {
			position252, tokenIndex252, depth252 := position, tokenIndex, depth
			{
				position253 := position
				depth++
				{
					position254, tokenIndex254, depth254 := position, tokenIndex, depth
					if buffer[position] != rune('n') {
						goto l255
					}
					position++
					if buffer[position] != rune('i') {
						goto l255
					}
					position++
					if buffer[position] != rune('l') {
						goto l255
					}
					position++
					goto l254
				l255:
					position, tokenIndex, depth = position254, tokenIndex254, depth254
					if buffer[position] != rune('~') {
						goto l252
					}
					position++
				}
			l254:
				depth--
				add(ruleNil, position253)
			}
			return true
		l252:
			position, tokenIndex, depth = position252, tokenIndex252, depth252
			return false
		},
		/* 59 Undefined <- <('~' '~')> */
		func() bool {
			position256, tokenIndex256, depth256 := position, tokenIndex, depth
			{
				position257 := position
				depth++
				if buffer[position] != rune('~') {
					goto l256
				}
				position++
				if buffer[position] != rune('~') {
					goto l256
				}
				position++
				depth--
				add(ruleUndefined, position257)
			}
			return true
		l256:
			position, tokenIndex, depth = position256, tokenIndex256, depth256
			return false
		},
		/* 60 Symbol <- <('$' Name)> */
		func() bool {
			position258, tokenIndex258, depth258 := position, tokenIndex, depth
			{
				position259 := position
				depth++
				if buffer[position] != rune('$') {
					goto l258
				}
				position++
				if !_rules[ruleName]() {
					goto l258
				}
				depth--
				add(ruleSymbol, position259)
			}
			return true
		l258:
			position, tokenIndex, depth = position258, tokenIndex258, depth258
			return false
		},
		/* 61 List <- <(StartList ExpressionList? ']')> */
		func() bool {
			position260, tokenIndex260, depth260 := position, tokenIndex, depth
			{
				position261 := position
				depth++
				if !_rules[ruleStartList]() {
					goto l260
				}
				{
					position262, tokenIndex262, depth262 := position, tokenIndex, depth
					if !_rules[ruleExpressionList]() {
						goto l262
					}
					goto l263
				l262:
					position, tokenIndex, depth = position262, tokenIndex262, depth262
				}
			l263:
				if buffer[position] != rune(']') {
					goto l260
				}
				position++
				depth--
				add(ruleList, position261)
			}
			return true
		l260:
			position, tokenIndex, depth = position260, tokenIndex260, depth260
			return false
		},
		/* 62 StartList <- <('[' ws)> */
		func() bool {
			position264, tokenIndex264, depth264 := position, tokenIndex, depth
			{
				position265 := position
				depth++
				if buffer[position] != rune('[') {
					goto l264
				}
				position++
				if !_rules[rulews]() {
					goto l264
				}
				depth--
				add(ruleStartList, position265)
			}
			return true
		l264:
			position, tokenIndex, depth = position264, tokenIndex264, depth264
			return false
		},
		/* 63 Map <- <(CreateMap ws Assignments? '}')> */
		func() bool {
			position266, tokenIndex266, depth266 := position, tokenIndex, depth
			{
				position267 := position
				depth++
				if !_rules[ruleCreateMap]() {
					goto l266
				}
				if !_rules[rulews]() {
					goto l266
				}
				{
					position268, tokenIndex268, depth268 := position, tokenIndex, depth
					if !_rules[ruleAssignments]() {
						goto l268
					}
					goto l269
				l268:
					position, tokenIndex, depth = position268, tokenIndex268, depth268
				}
			l269:
				if buffer[position] != rune('}') {
					goto l266
				}
				position++
				depth--
				add(ruleMap, position267)
			}
			return true
		l266:
			position, tokenIndex, depth = position266, tokenIndex266, depth266
			return false
		},
		/* 64 CreateMap <- <'{'> */
		func() bool {
			position270, tokenIndex270, depth270 := position, tokenIndex, depth
			{
				position271 := position
				depth++
				if buffer[position] != rune('{') {
					goto l270
				}
				position++
				depth--
				add(ruleCreateMap, position271)
			}
			return true
		l270:
			position, tokenIndex, depth = position270, tokenIndex270, depth270
			return false
		},
		/* 65 Assignments <- <(Assignment (',' Assignment)*)> */
		func() bool {
			position272, tokenIndex272, depth272 := position, tokenIndex, depth
			{
				position273 := position
				depth++
				if !_rules[ruleAssignment]() {
					goto l272
				}
			l274:
				{
					position275, tokenIndex275, depth275 := position, tokenIndex, depth
					if buffer[position] != rune(',') {
						goto l275
					}
					position++
					if !_rules[ruleAssignment]() {
						goto l275
					}
					goto l274
				l275:
					position, tokenIndex, depth = position275, tokenIndex275, depth275
				}
				depth--
				add(ruleAssignments, position273)
			}
			return true
		l272:
			position, tokenIndex, depth = position272, tokenIndex272, depth272
			return false
		},
		/* 66 Assignment <- <(Expression '=' Expression)> */
		func() bool {
			position276, tokenIndex276, depth276 := position, tokenIndex, depth
			{
				position277 := position
				depth++
				if !_rules[ruleExpression]() {
					goto l276
				}
				if buffer[position] != rune('=') {
					goto l276
				}
				position++
				if !_rules[ruleExpression]() {
					goto l276
				}
				depth--
				add(ruleAssignment, position277)
			}
			return true
		l276:
			position, tokenIndex, depth = position276, tokenIndex276, depth276
			return false
		},
		/* 67 Merge <- <(RefMerge / SimpleMerge)> */
		func() bool {
			position278, tokenIndex278, depth278 := position, tokenIndex, depth
			{
				position279 := position
				depth++
				{
					position280, tokenIndex280, depth280 := position, tokenIndex, depth
					if !_rules[ruleRefMerge]() {
						goto l281
					}
					goto l280
				l281:
					position, tokenIndex, depth = position280, tokenIndex280, depth280
					if !_rules[ruleSimpleMerge]() {
						goto l278
					}
				}
			l280:
				depth--
				add(ruleMerge, position279)
			}
			return true
		l278:
			position, tokenIndex, depth = position278, tokenIndex278, depth278
			return false
		},
		/* 68 RefMerge <- <('m' 'e' 'r' 'g' 'e' !(req_ws Required) (req_ws (Replace / On))? req_ws Reference)> */
		func() bool {
			position282, tokenIndex282, depth282 := position, tokenIndex, depth
			{
				position283 := position
				depth++
				if buffer[position] != rune('m') {
					goto l282
				}
				position++
				if buffer[position] != rune('e') {
					goto l282
				}
				position++
				if buffer[position] != rune('r') {
					goto l282
				}
				position++
				if buffer[position] != rune('g') {
					goto l282
				}
				position++
				if buffer[position] != rune('e') {
					goto l282
				}
				position++
				{
					position284, tokenIndex284, depth284 := position, tokenIndex, depth
					if !_rules[rulereq_ws]() {
						goto l284
					}
					if !_rules[ruleRequired]() {
						goto l284
					}
					goto l282
				l284:
					position, tokenIndex, depth = position284, tokenIndex284, depth284
				}
				{
					position285, tokenIndex285, depth285 := position, tokenIndex, depth
					if !_rules[rulereq_ws]() {
						goto l285
					}
					{
						position287, tokenIndex287, depth287 := position, tokenIndex, depth
						if !_rules[ruleReplace]() {
							goto l288
						}
						goto l287
					l288:
						position, tokenIndex, depth = position287, tokenIndex287, depth287
						if !_rules[ruleOn]() {
							goto l285
						}
					}
				l287:
					goto l286
				l285:
					position, tokenIndex, depth = position285, tokenIndex285, depth285
				}
			l286:
				if !_rules[rulereq_ws]() {
					goto l282
				}
				if !_rules[ruleReference]() {
					goto l282
				}
				depth--
				add(ruleRefMerge, position283)
			}
			return true
		l282:
			position, tokenIndex, depth = position282, tokenIndex282, depth282
			return false
		},
		/* 69 SimpleMerge <- <('m' 'e' 'r' 'g' 'e' !'(' (req_ws (Replace / Required / On))?)> */
		func() bool {
			position289, tokenIndex289, depth289 := position, tokenIndex, depth
			{
				position290 := position
				depth++
				if buffer[position] != rune('m') {
					goto l289
				}
				position++
				if buffer[position] != rune('e') {
					goto l289
				}
				position++
				if buffer[position] != rune('r') {
					goto l289
				}
				position++
				if buffer[position] != rune('g') {
					goto l289
				}
				position++
				if buffer[position] != rune('e') {
					goto l289
				}
				position++
				{
					position291, tokenIndex291, depth291 := position, tokenIndex, depth
					if buffer[position] != rune('(') {
						goto l291
					}
					position++
					goto l289
				l291:
					position, tokenIndex, depth = position291, tokenIndex291, depth291
				}
				{
					position292, tokenIndex292, depth292 := position, tokenIndex, depth
					if !_rules[rulereq_ws]() {
						goto l292
					}
					{
						position294, tokenIndex294, depth294 := position, tokenIndex, depth
						if !_rules[ruleReplace]() {
							goto l295
						}
						goto l294
					l295:
						position, tokenIndex, depth = position294, tokenIndex294, depth294
						if !_rules[ruleRequired]() {
							goto l296
						}
						goto l294
					l296:
						position, tokenIndex, depth = position294, tokenIndex294, depth294
						if !_rules[ruleOn]() {
							goto l292
						}
					}
				l294:
					goto l293
				l292:
					position, tokenIndex, depth = position292, tokenIndex292, depth292
				}
			l293:
				depth--
				add(ruleSimpleMerge, position290)
			}
			return true
		l289:
			position, tokenIndex, depth = position289, tokenIndex289, depth289
			return false
		},
		/* 70 Replace <- <('r' 'e' 'p' 'l' 'a' 'c' 'e')> */
		func() bool {
			position297, tokenIndex297, depth297 := position, tokenIndex, depth
			{
				position298 := position
				depth++
				if buffer[position] != rune('r') {
					goto l297
				}
				position++
				if buffer[position] != rune('e') {
					goto l297
				}
				position++
				if buffer[position] != rune('p') {
					goto l297
				}
				position++
				if buffer[position] != rune('l') {
					goto l297
				}
				position++
				if buffer[position] != rune('a') {
					goto l297
				}
				position++
				if buffer[position] != rune('c') {
					goto l297
				}
				position++
				if buffer[position] != rune('e') {
					goto l297
				}
				position++
				depth--
				add(ruleReplace, position298)
			}
			return true
		l297:
			position, tokenIndex, depth = position297, tokenIndex297, depth297
			return false
		},
		/* 71 Required <- <('r' 'e' 'q' 'u' 'i' 'r' 'e' 'd')> */
		func() bool {
			position299, tokenIndex299, depth299 := position, tokenIndex, depth
			{
				position300 := position
				depth++
				if buffer[position] != rune('r') {
					goto l299
				}
				position++
				if buffer[position] != rune('e') {
					goto l299
				}
				position++
				if buffer[position] != rune('q') {
					goto l299
				}
				position++
				if buffer[position] != rune('u') {
					goto l299
				}
				position++
				if buffer[position] != rune('i') {
					goto l299
				}
				position++
				if buffer[position] != rune('r') {
					goto l299
				}
				position++
				if buffer[position] != rune('e') {
					goto l299
				}
				position++
				if buffer[position] != rune('d') {
					goto l299
				}
				position++
				depth--
				add(ruleRequired, position300)
			}
			return true
		l299:
			position, tokenIndex, depth = position299, tokenIndex299, depth299
			return false
		},
		/* 72 On <- <('o' 'n' req_ws Name)> */
		func() bool {
			position301, tokenIndex301, depth301 := position, tokenIndex, depth
			{
				position302 := position
				depth++
				if buffer[position] != rune('o') {
					goto l301
				}
				position++
				if buffer[position] != rune('n') {
					goto l301
				}
				position++
				if !_rules[rulereq_ws]() {
					goto l301
				}
				if !_rules[ruleName]() {
					goto l301
				}
				depth--
				add(ruleOn, position302)
			}
			return true
		l301:
			position, tokenIndex, depth = position301, tokenIndex301, depth301
			return false
		},
		/* 73 Auto <- <('a' 'u' 't' 'o')> */
		func() bool {
			position303, tokenIndex303, depth303 := position, tokenIndex, depth
			{
				position304 := position
				depth++
				if buffer[position] != rune('a') {
					goto l303
				}
				position++
				if buffer[position] != rune('u') {
					goto l303
				}
				position++
				if buffer[position] != rune('t') {
					goto l303
				}
				position++
				if buffer[position] != rune('o') {
					goto l303
				}
				position++
				depth--
				add(ruleAuto, position304)
			}
			return true
		l303:
			position, tokenIndex, depth = position303, tokenIndex303, depth303
			return false
		},
		/* 74 Default <- <Action1> */
		func() bool {
			position305, tokenIndex305, depth305 := position, tokenIndex, depth
			{
				position306 := position
				depth++
				if !_rules[ruleAction1]() {
					goto l305
				}
				depth--
				add(ruleDefault, position306)
			}
			return true
		l305:
			position, tokenIndex, depth = position305, tokenIndex305, depth305
			return false
		},
		/* 75 Sync <- <('s' 'y' 'n' 'c' '[' Level7 ((((LambdaExpr LambdaExt) / (LambdaOrExpr LambdaOrExpr)) (('|' Expression) / Default)) / (LambdaOrExpr Default Default)) ']')> */
		func() bool {
			position307, tokenIndex307, depth307 := position, tokenIndex, depth
			{
				position308 := position
				depth++
				if buffer[position] != rune('s') {
					goto l307
				}
				position++
				if buffer[position] != rune('y') {
					goto l307
				}
				position++
				if buffer[position] != rune('n') {
					goto l307
				}
				position++
				if buffer[position] != rune('c') {
					goto l307
				}
				position++
				if buffer[position] != rune('[') {
					goto l307
				}
				position++
				if !_rules[ruleLevel7]() {
					goto l307
				}
				{
					position309, tokenIndex309, depth309 := position, tokenIndex, depth
					{
						position311, tokenIndex311, depth311 := position, tokenIndex, depth
						if !_rules[ruleLambdaExpr]() {
							goto l312
						}
						if !_rules[ruleLambdaExt]() {
							goto l312
						}
						goto l311
					l312:
						position, tokenIndex, depth = position311, tokenIndex311, depth311
						if !_rules[ruleLambdaOrExpr]() {
							goto l310
						}
						if !_rules[ruleLambdaOrExpr]() {
							goto l310
						}
					}
				l311:
					{
						position313, tokenIndex313, depth313 := position, tokenIndex, depth
						if buffer[position] != rune('|') {
							goto l314
						}
						position++
						if !_rules[ruleExpression]() {
							goto l314
						}
						goto l313
					l314:
						position, tokenIndex, depth = position313, tokenIndex313, depth313
						if !_rules[ruleDefault]() {
							goto l310
						}
					}
				l313:
					goto l309
				l310:
					position, tokenIndex, depth = position309, tokenIndex309, depth309
					if !_rules[ruleLambdaOrExpr]() {
						goto l307
					}
					if !_rules[ruleDefault]() {
						goto l307
					}
					if !_rules[ruleDefault]() {
						goto l307
					}
				}
			l309:
				if buffer[position] != rune(']') {
					goto l307
				}
				position++
				depth--
				add(ruleSync, position308)
			}
			return true
		l307:
			position, tokenIndex, depth = position307, tokenIndex307, depth307
			return false
		},
		/* 76 LambdaExt <- <(',' Expression)> */
		func() bool {
			position315, tokenIndex315, depth315 := position, tokenIndex, depth
			{
				position316 := position
				depth++
				if buffer[position] != rune(',') {
					goto l315
				}
				position++
				if !_rules[ruleExpression]() {
					goto l315
				}
				depth--
				add(ruleLambdaExt, position316)
			}
			return true
		l315:
			position, tokenIndex, depth = position315, tokenIndex315, depth315
			return false
		},
		/* 77 LambdaOrExpr <- <(LambdaExpr / ('|' Expression))> */
		func() bool {
			position317, tokenIndex317, depth317 := position, tokenIndex, depth
			{
				position318 := position
				depth++
				{
					position319, tokenIndex319, depth319 := position, tokenIndex, depth
					if !_rules[ruleLambdaExpr]() {
						goto l320
					}
					goto l319
				l320:
					position, tokenIndex, depth = position319, tokenIndex319, depth319
					if buffer[position] != rune('|') {
						goto l317
					}
					position++
					if !_rules[ruleExpression]() {
						goto l317
					}
				}
			l319:
				depth--
				add(ruleLambdaOrExpr, position318)
			}
			return true
		l317:
			position, tokenIndex, depth = position317, tokenIndex317, depth317
			return false
		},
		/* 78 Catch <- <('c' 'a' 't' 'c' 'h' '[' Level7 LambdaOrExpr ']')> */
		func() bool {
			position321, tokenIndex321, depth321 := position, tokenIndex, depth
			{
				position322 := position
				depth++
				if buffer[position] != rune('c') {
					goto l321
				}
				position++
				if buffer[position] != rune('a') {
					goto l321
				}
				position++
				if buffer[position] != rune('t') {
					goto l321
				}
				position++
				if buffer[position] != rune('c') {
					goto l321
				}
				position++
				if buffer[position] != rune('h') {
					goto l321
				}
				position++
				if buffer[position] != rune('[') {
					goto l321
				}
				position++
				if !_rules[ruleLevel7]() {
					goto l321
				}
				if !_rules[ruleLambdaOrExpr]() {
					goto l321
				}
				if buffer[position] != rune(']') {
					goto l321
				}
				position++
				depth--
				add(ruleCatch, position322)
			}
			return true
		l321:
			position, tokenIndex, depth = position321, tokenIndex321, depth321
			return false
		},
		/* 79 MapMapping <- <('m' 'a' 'p' '{' Level7 LambdaOrExpr '}')> */
		func() bool {
			position323, tokenIndex323, depth323 := position, tokenIndex, depth
			{
				position324 := position
				depth++
				if buffer[position] != rune('m') {
					goto l323
				}
				position++
				if buffer[position] != rune('a') {
					goto l323
				}
				position++
				if buffer[position] != rune('p') {
					goto l323
				}
				position++
				if buffer[position] != rune('{') {
					goto l323
				}
				position++
				if !_rules[ruleLevel7]() {
					goto l323
				}
				if !_rules[ruleLambdaOrExpr]() {
					goto l323
				}
				if buffer[position] != rune('}') {
					goto l323
				}
				position++
				depth--
				add(ruleMapMapping, position324)
			}
			return true
		l323:
			position, tokenIndex, depth = position323, tokenIndex323, depth323
			return false
		},
		/* 80 Mapping <- <('m' 'a' 'p' '[' Level7 LambdaOrExpr ']')> */
		func() bool {
			position325, tokenIndex325, depth325 := position, tokenIndex, depth
			{
				position326 := position
				depth++
				if buffer[position] != rune('m') {
					goto l325
				}
				position++
				if buffer[position] != rune('a') {
					goto l325
				}
				position++
				if buffer[position] != rune('p') {
					goto l325
				}
				position++
				if buffer[position] != rune('[') {
					goto l325
				}
				position++
				if !_rules[ruleLevel7]() {
					goto l325
				}
				if !_rules[ruleLambdaOrExpr]() {
					goto l325
				}
				if buffer[position] != rune(']') {
					goto l325
				}
				position++
				depth--
				add(ruleMapping, position326)
			}
			return true
		l325:
			position, tokenIndex, depth = position325, tokenIndex325, depth325
			return false
		},
		/* 81 MapSelection <- <('s' 'e' 'l' 'e' 'c' 't' '{' Level7 LambdaOrExpr '}')> */
		func() bool {
			position327, tokenIndex327, depth327 := position, tokenIndex, depth
			{
				position328 := position
				depth++
				if buffer[position] != rune('s') {
					goto l327
				}
				position++
				if buffer[position] != rune('e') {
					goto l327
				}
				position++
				if buffer[position] != rune('l') {
					goto l327
				}
				position++
				if buffer[position] != rune('e') {
					goto l327
				}
				position++
				if buffer[position] != rune('c') {
					goto l327
				}
				position++
				if buffer[position] != rune('t') {
					goto l327
				}
				position++
				if buffer[position] != rune('{') {
					goto l327
				}
				position++
				if !_rules[ruleLevel7]() {
					goto l327
				}
				if !_rules[ruleLambdaOrExpr]() {
					goto l327
				}
				if buffer[position] != rune('}') {
					goto l327
				}
				position++
				depth--
				add(ruleMapSelection, position328)
			}
			return true
		l327:
			position, tokenIndex, depth = position327, tokenIndex327, depth327
			return false
		},
		/* 82 Selection <- <('s' 'e' 'l' 'e' 'c' 't' '[' Level7 LambdaOrExpr ']')> */
		func() bool {
			position329, tokenIndex329, depth329 := position, tokenIndex, depth
			{
				position330 := position
				depth++
				if buffer[position] != rune('s') {
					goto l329
				}
				position++
				if buffer[position] != rune('e') {
					goto l329
				}
				position++
				if buffer[position] != rune('l') {
					goto l329
				}
				position++
				if buffer[position] != rune('e') {
					goto l329
				}
				position++
				if buffer[position] != rune('c') {
					goto l329
				}
				position++
				if buffer[position] != rune('t') {
					goto l329
				}
				position++
				if buffer[position] != rune('[') {
					goto l329
				}
				position++
				if !_rules[ruleLevel7]() {
					goto l329
				}
				if !_rules[ruleLambdaOrExpr]() {
					goto l329
				}
				if buffer[position] != rune(']') {
					goto l329
				}
				position++
				depth--
				add(ruleSelection, position330)
			}
			return true
		l329:
			position, tokenIndex, depth = position329, tokenIndex329, depth329
			return false
		},
		/* 83 Sum <- <('s' 'u' 'm' '[' Level7 '|' Level7 LambdaOrExpr ']')> */
		func() bool {
			position331, tokenIndex331, depth331 := position, tokenIndex, depth
			{
				position332 := position
				depth++
				if buffer[position] != rune('s') {
					goto l331
				}
				position++
				if buffer[position] != rune('u') {
					goto l331
				}
				position++
				if buffer[position] != rune('m') {
					goto l331
				}
				position++
				if buffer[position] != rune('[') {
					goto l331
				}
				position++
				if !_rules[ruleLevel7]() {
					goto l331
				}
				if buffer[position] != rune('|') {
					goto l331
				}
				position++
				if !_rules[ruleLevel7]() {
					goto l331
				}
				if !_rules[ruleLambdaOrExpr]() {
					goto l331
				}
				if buffer[position] != rune(']') {
					goto l331
				}
				position++
				depth--
				add(ruleSum, position332)
			}
			return true
		l331:
			position, tokenIndex, depth = position331, tokenIndex331, depth331
			return false
		},
		/* 84 Lambda <- <('l' 'a' 'm' 'b' 'd' 'a' (LambdaRef / LambdaExpr))> */
		func() bool {
			position333, tokenIndex333, depth333 := position, tokenIndex, depth
			{
				position334 := position
				depth++
				if buffer[position] != rune('l') {
					goto l333
				}
				position++
				if buffer[position] != rune('a') {
					goto l333
				}
				position++
				if buffer[position] != rune('m') {
					goto l333
				}
				position++
				if buffer[position] != rune('b') {
					goto l333
				}
				position++
				if buffer[position] != rune('d') {
					goto l333
				}
				position++
				if buffer[position] != rune('a') {
					goto l333
				}
				position++
				{
					position335, tokenIndex335, depth335 := position, tokenIndex, depth
					if !_rules[ruleLambdaRef]() {
						goto l336
					}
					goto l335
				l336:
					position, tokenIndex, depth = position335, tokenIndex335, depth335
					if !_rules[ruleLambdaExpr]() {
						goto l333
					}
				}
			l335:
				depth--
				add(ruleLambda, position334)
			}
			return true
		l333:
			position, tokenIndex, depth = position333, tokenIndex333, depth333
			return false
		},
		/* 85 LambdaRef <- <(req_ws Expression)> */
		func() bool {
			position337, tokenIndex337, depth337 := position, tokenIndex, depth
			{
				position338 := position
				depth++
				if !_rules[rulereq_ws]() {
					goto l337
				}
				if !_rules[ruleExpression]() {
					goto l337
				}
				depth--
				add(ruleLambdaRef, position338)
			}
			return true
		l337:
			position, tokenIndex, depth = position337, tokenIndex337, depth337
			return false
		},
		/* 86 LambdaExpr <- <(ws Params ws ('-' '>') Expression)> */
		func() bool {
			position339, tokenIndex339, depth339 := position, tokenIndex, depth
			{
				position340 := position
				depth++
				if !_rules[rulews]() {
					goto l339
				}
				if !_rules[ruleParams]() {
					goto l339
				}
				if !_rules[rulews]() {
					goto l339
				}
				if buffer[position] != rune('-') {
					goto l339
				}
				position++
				if buffer[position] != rune('>') {
					goto l339
				}
				position++
				if !_rules[ruleExpression]() {
					goto l339
				}
				depth--
				add(ruleLambdaExpr, position340)
			}
			return true
		l339:
			position, tokenIndex, depth = position339, tokenIndex339, depth339
			return false
		},
		/* 87 Params <- <('|' StartParams ws Names? '|')> */
		func() bool {
			position341, tokenIndex341, depth341 := position, tokenIndex, depth
			{
				position342 := position
				depth++
				if buffer[position] != rune('|') {
					goto l341
				}
				position++
				if !_rules[ruleStartParams]() {
					goto l341
				}
				if !_rules[rulews]() {
					goto l341
				}
				{
					position343, tokenIndex343, depth343 := position, tokenIndex, depth
					if !_rules[ruleNames]() {
						goto l343
					}
					goto l344
				l343:
					position, tokenIndex, depth = position343, tokenIndex343, depth343
				}
			l344:
				if buffer[position] != rune('|') {
					goto l341
				}
				position++
				depth--
				add(ruleParams, position342)
			}
			return true
		l341:
			position, tokenIndex, depth = position341, tokenIndex341, depth341
			return false
		},
		/* 88 StartParams <- <Action2> */
		func() bool {
			position345, tokenIndex345, depth345 := position, tokenIndex, depth
			{
				position346 := position
				depth++
				if !_rules[ruleAction2]() {
					goto l345
				}
				depth--
				add(ruleStartParams, position346)
			}
			return true
		l345:
			position, tokenIndex, depth = position345, tokenIndex345, depth345
			return false
		},
		/* 89 Names <- <(NextName (',' NextName)* DefaultValue? (',' NextName DefaultValue)* VarParams?)> */
		func() bool {
			position347, tokenIndex347, depth347 := position, tokenIndex, depth
			{
				position348 := position
				depth++
				if !_rules[ruleNextName]() {
					goto l347
				}
			l349:
				{
					position350, tokenIndex350, depth350 := position, tokenIndex, depth
					if buffer[position] != rune(',') {
						goto l350
					}
					position++
					if !_rules[ruleNextName]() {
						goto l350
					}
					goto l349
				l350:
					position, tokenIndex, depth = position350, tokenIndex350, depth350
				}
				{
					position351, tokenIndex351, depth351 := position, tokenIndex, depth
					if !_rules[ruleDefaultValue]() {
						goto l351
					}
					goto l352
				l351:
					position, tokenIndex, depth = position351, tokenIndex351, depth351
				}
			l352:
			l353:
				{
					position354, tokenIndex354, depth354 := position, tokenIndex, depth
					if buffer[position] != rune(',') {
						goto l354
					}
					position++
					if !_rules[ruleNextName]() {
						goto l354
					}
					if !_rules[ruleDefaultValue]() {
						goto l354
					}
					goto l353
				l354:
					position, tokenIndex, depth = position354, tokenIndex354, depth354
				}
				{
					position355, tokenIndex355, depth355 := position, tokenIndex, depth
					if !_rules[ruleVarParams]() {
						goto l355
					}
					goto l356
				l355:
					position, tokenIndex, depth = position355, tokenIndex355, depth355
				}
			l356:
				depth--
				add(ruleNames, position348)
			}
			return true
		l347:
			position, tokenIndex, depth = position347, tokenIndex347, depth347
			return false
		},
		/* 90 NextName <- <(ws Name ws)> */
		func() bool {
			position357, tokenIndex357, depth357 := position, tokenIndex, depth
			{
				position358 := position
				depth++
				if !_rules[rulews]() {
					goto l357
				}
				if !_rules[ruleName]() {
					goto l357
				}
				if !_rules[rulews]() {
					goto l357
				}
				depth--
				add(ruleNextName, position358)
			}
			return true
		l357:
			position, tokenIndex, depth = position357, tokenIndex357, depth357
			return false
		},
		/* 91 Name <- <([a-z] / [A-Z] / [0-9] / '_')+> */
		func() bool {
			position359, tokenIndex359, depth359 := position, tokenIndex, depth
			{
				position360 := position
				depth++
				{
					position363, tokenIndex363, depth363 := position, tokenIndex, depth
					if c := buffer[position]; c < rune('a') || c > rune('z') {
						goto l364
					}
					position++
					goto l363
				l364:
					position, tokenIndex, depth = position363, tokenIndex363, depth363
					if c := buffer[position]; c < rune('A') || c > rune('Z') {
						goto l365
					}
					position++
					goto l363
				l365:
					position, tokenIndex, depth = position363, tokenIndex363, depth363
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l366
					}
					position++
					goto l363
				l366:
					position, tokenIndex, depth = position363, tokenIndex363, depth363
					if buffer[position] != rune('_') {
						goto l359
					}
					position++
				}
			l363:
			l361:
				{
					position362, tokenIndex362, depth362 := position, tokenIndex, depth
					{
						position367, tokenIndex367, depth367 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l368
						}
						position++
						goto l367
					l368:
						position, tokenIndex, depth = position367, tokenIndex367, depth367
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l369
						}
						position++
						goto l367
					l369:
						position, tokenIndex, depth = position367, tokenIndex367, depth367
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l370
						}
						position++
						goto l367
					l370:
						position, tokenIndex, depth = position367, tokenIndex367, depth367
						if buffer[position] != rune('_') {
							goto l362
						}
						position++
					}
				l367:
					goto l361
				l362:
					position, tokenIndex, depth = position362, tokenIndex362, depth362
				}
				depth--
				add(ruleName, position360)
			}
			return true
		l359:
			position, tokenIndex, depth = position359, tokenIndex359, depth359
			return false
		},
		/* 92 DefaultValue <- <('=' Expression)> */
		func() bool {
			position371, tokenIndex371, depth371 := position, tokenIndex, depth
			{
				position372 := position
				depth++
				if buffer[position] != rune('=') {
					goto l371
				}
				position++
				if !_rules[ruleExpression]() {
					goto l371
				}
				depth--
				add(ruleDefaultValue, position372)
			}
			return true
		l371:
			position, tokenIndex, depth = position371, tokenIndex371, depth371
			return false
		},
		/* 93 VarParams <- <('.' '.' '.' ws)> */
		func() bool {
			position373, tokenIndex373, depth373 := position, tokenIndex, depth
			{
				position374 := position
				depth++
				if buffer[position] != rune('.') {
					goto l373
				}
				position++
				if buffer[position] != rune('.') {
					goto l373
				}
				position++
				if buffer[position] != rune('.') {
					goto l373
				}
				position++
				if !_rules[rulews]() {
					goto l373
				}
				depth--
				add(ruleVarParams, position374)
			}
			return true
		l373:
			position, tokenIndex, depth = position373, tokenIndex373, depth373
			return false
		},
		/* 94 Reference <- <(((TagPrefix ('.' / Key)) / ('.'? Key)) FollowUpRef)> */
		func() bool {
			position375, tokenIndex375, depth375 := position, tokenIndex, depth
			{
				position376 := position
				depth++
				{
					position377, tokenIndex377, depth377 := position, tokenIndex, depth
					if !_rules[ruleTagPrefix]() {
						goto l378
					}
					{
						position379, tokenIndex379, depth379 := position, tokenIndex, depth
						if buffer[position] != rune('.') {
							goto l380
						}
						position++
						goto l379
					l380:
						position, tokenIndex, depth = position379, tokenIndex379, depth379
						if !_rules[ruleKey]() {
							goto l378
						}
					}
				l379:
					goto l377
				l378:
					position, tokenIndex, depth = position377, tokenIndex377, depth377
					{
						position381, tokenIndex381, depth381 := position, tokenIndex, depth
						if buffer[position] != rune('.') {
							goto l381
						}
						position++
						goto l382
					l381:
						position, tokenIndex, depth = position381, tokenIndex381, depth381
					}
				l382:
					if !_rules[ruleKey]() {
						goto l375
					}
				}
			l377:
				if !_rules[ruleFollowUpRef]() {
					goto l375
				}
				depth--
				add(ruleReference, position376)
			}
			return true
		l375:
			position, tokenIndex, depth = position375, tokenIndex375, depth375
			return false
		},
		/* 95 TagPrefix <- <((('d' 'o' 'c' ('.' / ':') '-'? [0-9]+) / Tag) (':' ':'))> */
		func() bool {
			position383, tokenIndex383, depth383 := position, tokenIndex, depth
			{
				position384 := position
				depth++
				{
					position385, tokenIndex385, depth385 := position, tokenIndex, depth
					if buffer[position] != rune('d') {
						goto l386
					}
					position++
					if buffer[position] != rune('o') {
						goto l386
					}
					position++
					if buffer[position] != rune('c') {
						goto l386
					}
					position++
					{
						position387, tokenIndex387, depth387 := position, tokenIndex, depth
						if buffer[position] != rune('.') {
							goto l388
						}
						position++
						goto l387
					l388:
						position, tokenIndex, depth = position387, tokenIndex387, depth387
						if buffer[position] != rune(':') {
							goto l386
						}
						position++
					}
				l387:
					{
						position389, tokenIndex389, depth389 := position, tokenIndex, depth
						if buffer[position] != rune('-') {
							goto l389
						}
						position++
						goto l390
					l389:
						position, tokenIndex, depth = position389, tokenIndex389, depth389
					}
				l390:
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l386
					}
					position++
				l391:
					{
						position392, tokenIndex392, depth392 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l392
						}
						position++
						goto l391
					l392:
						position, tokenIndex, depth = position392, tokenIndex392, depth392
					}
					goto l385
				l386:
					position, tokenIndex, depth = position385, tokenIndex385, depth385
					if !_rules[ruleTag]() {
						goto l383
					}
				}
			l385:
				if buffer[position] != rune(':') {
					goto l383
				}
				position++
				if buffer[position] != rune(':') {
					goto l383
				}
				position++
				depth--
				add(ruleTagPrefix, position384)
			}
			return true
		l383:
			position, tokenIndex, depth = position383, tokenIndex383, depth383
			return false
		},
		/* 96 Tag <- <(TagComponent (('.' / ':') TagComponent)*)> */
		func() bool {
			position393, tokenIndex393, depth393 := position, tokenIndex, depth
			{
				position394 := position
				depth++
				if !_rules[ruleTagComponent]() {
					goto l393
				}
			l395:
				{
					position396, tokenIndex396, depth396 := position, tokenIndex, depth
					{
						position397, tokenIndex397, depth397 := position, tokenIndex, depth
						if buffer[position] != rune('.') {
							goto l398
						}
						position++
						goto l397
					l398:
						position, tokenIndex, depth = position397, tokenIndex397, depth397
						if buffer[position] != rune(':') {
							goto l396
						}
						position++
					}
				l397:
					if !_rules[ruleTagComponent]() {
						goto l396
					}
					goto l395
				l396:
					position, tokenIndex, depth = position396, tokenIndex396, depth396
				}
				depth--
				add(ruleTag, position394)
			}
			return true
		l393:
			position, tokenIndex, depth = position393, tokenIndex393, depth393
			return false
		},
		/* 97 TagComponent <- <(([a-z] / [A-Z] / '_') ([a-z] / [A-Z] / [0-9] / '_')*)> */
		func() bool {
			position399, tokenIndex399, depth399 := position, tokenIndex, depth
			{
				position400 := position
				depth++
				{
					position401, tokenIndex401, depth401 := position, tokenIndex, depth
					if c := buffer[position]; c < rune('a') || c > rune('z') {
						goto l402
					}
					position++
					goto l401
				l402:
					position, tokenIndex, depth = position401, tokenIndex401, depth401
					if c := buffer[position]; c < rune('A') || c > rune('Z') {
						goto l403
					}
					position++
					goto l401
				l403:
					position, tokenIndex, depth = position401, tokenIndex401, depth401
					if buffer[position] != rune('_') {
						goto l399
					}
					position++
				}
			l401:
			l404:
				{
					position405, tokenIndex405, depth405 := position, tokenIndex, depth
					{
						position406, tokenIndex406, depth406 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l407
						}
						position++
						goto l406
					l407:
						position, tokenIndex, depth = position406, tokenIndex406, depth406
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l408
						}
						position++
						goto l406
					l408:
						position, tokenIndex, depth = position406, tokenIndex406, depth406
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l409
						}
						position++
						goto l406
					l409:
						position, tokenIndex, depth = position406, tokenIndex406, depth406
						if buffer[position] != rune('_') {
							goto l405
						}
						position++
					}
				l406:
					goto l404
				l405:
					position, tokenIndex, depth = position405, tokenIndex405, depth405
				}
				depth--
				add(ruleTagComponent, position400)
			}
			return true
		l399:
			position, tokenIndex, depth = position399, tokenIndex399, depth399
			return false
		},
		/* 98 FollowUpRef <- <PathComponent*> */
		func() bool {
			{
				position411 := position
				depth++
			l412:
				{
					position413, tokenIndex413, depth413 := position, tokenIndex, depth
					if !_rules[rulePathComponent]() {
						goto l413
					}
					goto l412
				l413:
					position, tokenIndex, depth = position413, tokenIndex413, depth413
				}
				depth--
				add(ruleFollowUpRef, position411)
			}
			return true
		},
		/* 99 PathComponent <- <(('.' Key) / ('.'? Index))> */
		func() bool {
			position414, tokenIndex414, depth414 := position, tokenIndex, depth
			{
				position415 := position
				depth++
				{
					position416, tokenIndex416, depth416 := position, tokenIndex, depth
					if buffer[position] != rune('.') {
						goto l417
					}
					position++
					if !_rules[ruleKey]() {
						goto l417
					}
					goto l416
				l417:
					position, tokenIndex, depth = position416, tokenIndex416, depth416
					{
						position418, tokenIndex418, depth418 := position, tokenIndex, depth
						if buffer[position] != rune('.') {
							goto l418
						}
						position++
						goto l419
					l418:
						position, tokenIndex, depth = position418, tokenIndex418, depth418
					}
				l419:
					if !_rules[ruleIndex]() {
						goto l414
					}
				}
			l416:
				depth--
				add(rulePathComponent, position415)
			}
			return true
		l414:
			position, tokenIndex, depth = position414, tokenIndex414, depth414
			return false
		},
		/* 100 Key <- <(([a-z] / [A-Z] / [0-9] / '_') ([a-z] / [A-Z] / [0-9] / '_' / '-')* (':' ([a-z] / [A-Z] / [0-9] / '_') ([a-z] / [A-Z] / [0-9] / '_' / '-')*)?)> */
		func() bool {
			position420, tokenIndex420, depth420 := position, tokenIndex, depth
			{
				position421 := position
				depth++
				{
					position422, tokenIndex422, depth422 := position, tokenIndex, depth
					if c := buffer[position]; c < rune('a') || c > rune('z') {
						goto l423
					}
					position++
					goto l422
				l423:
					position, tokenIndex, depth = position422, tokenIndex422, depth422
					if c := buffer[position]; c < rune('A') || c > rune('Z') {
						goto l424
					}
					position++
					goto l422
				l424:
					position, tokenIndex, depth = position422, tokenIndex422, depth422
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l425
					}
					position++
					goto l422
				l425:
					position, tokenIndex, depth = position422, tokenIndex422, depth422
					if buffer[position] != rune('_') {
						goto l420
					}
					position++
				}
			l422:
			l426:
				{
					position427, tokenIndex427, depth427 := position, tokenIndex, depth
					{
						position428, tokenIndex428, depth428 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l429
						}
						position++
						goto l428
					l429:
						position, tokenIndex, depth = position428, tokenIndex428, depth428
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l430
						}
						position++
						goto l428
					l430:
						position, tokenIndex, depth = position428, tokenIndex428, depth428
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l431
						}
						position++
						goto l428
					l431:
						position, tokenIndex, depth = position428, tokenIndex428, depth428
						if buffer[position] != rune('_') {
							goto l432
						}
						position++
						goto l428
					l432:
						position, tokenIndex, depth = position428, tokenIndex428, depth428
						if buffer[position] != rune('-') {
							goto l427
						}
						position++
					}
				l428:
					goto l426
				l427:
					position, tokenIndex, depth = position427, tokenIndex427, depth427
				}
				{
					position433, tokenIndex433, depth433 := position, tokenIndex, depth
					if buffer[position] != rune(':') {
						goto l433
					}
					position++
					{
						position435, tokenIndex435, depth435 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l436
						}
						position++
						goto l435
					l436:
						position, tokenIndex, depth = position435, tokenIndex435, depth435
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l437
						}
						position++
						goto l435
					l437:
						position, tokenIndex, depth = position435, tokenIndex435, depth435
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l438
						}
						position++
						goto l435
					l438:
						position, tokenIndex, depth = position435, tokenIndex435, depth435
						if buffer[position] != rune('_') {
							goto l433
						}
						position++
					}
				l435:
				l439:
					{
						position440, tokenIndex440, depth440 := position, tokenIndex, depth
						{
							position441, tokenIndex441, depth441 := position, tokenIndex, depth
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l442
							}
							position++
							goto l441
						l442:
							position, tokenIndex, depth = position441, tokenIndex441, depth441
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l443
							}
							position++
							goto l441
						l443:
							position, tokenIndex, depth = position441, tokenIndex441, depth441
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l444
							}
							position++
							goto l441
						l444:
							position, tokenIndex, depth = position441, tokenIndex441, depth441
							if buffer[position] != rune('_') {
								goto l445
							}
							position++
							goto l441
						l445:
							position, tokenIndex, depth = position441, tokenIndex441, depth441
							if buffer[position] != rune('-') {
								goto l440
							}
							position++
						}
					l441:
						goto l439
					l440:
						position, tokenIndex, depth = position440, tokenIndex440, depth440
					}
					goto l434
				l433:
					position, tokenIndex, depth = position433, tokenIndex433, depth433
				}
			l434:
				depth--
				add(ruleKey, position421)
			}
			return true
		l420:
			position, tokenIndex, depth = position420, tokenIndex420, depth420
			return false
		},
		/* 101 Index <- <('[' '-'? [0-9]+ ']')> */
		func() bool {
			position446, tokenIndex446, depth446 := position, tokenIndex, depth
			{
				position447 := position
				depth++
				if buffer[position] != rune('[') {
					goto l446
				}
				position++
				{
					position448, tokenIndex448, depth448 := position, tokenIndex, depth
					if buffer[position] != rune('-') {
						goto l448
					}
					position++
					goto l449
				l448:
					position, tokenIndex, depth = position448, tokenIndex448, depth448
				}
			l449:
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l446
				}
				position++
			l450:
				{
					position451, tokenIndex451, depth451 := position, tokenIndex, depth
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l451
					}
					position++
					goto l450
				l451:
					position, tokenIndex, depth = position451, tokenIndex451, depth451
				}
				if buffer[position] != rune(']') {
					goto l446
				}
				position++
				depth--
				add(ruleIndex, position447)
			}
			return true
		l446:
			position, tokenIndex, depth = position446, tokenIndex446, depth446
			return false
		},
		/* 102 IP <- <([0-9]+ '.' [0-9]+ '.' [0-9]+ '.' [0-9]+)> */
		func() bool {
			position452, tokenIndex452, depth452 := position, tokenIndex, depth
			{
				position453 := position
				depth++
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l452
				}
				position++
			l454:
				{
					position455, tokenIndex455, depth455 := position, tokenIndex, depth
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l455
					}
					position++
					goto l454
				l455:
					position, tokenIndex, depth = position455, tokenIndex455, depth455
				}
				if buffer[position] != rune('.') {
					goto l452
				}
				position++
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l452
				}
				position++
			l456:
				{
					position457, tokenIndex457, depth457 := position, tokenIndex, depth
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l457
					}
					position++
					goto l456
				l457:
					position, tokenIndex, depth = position457, tokenIndex457, depth457
				}
				if buffer[position] != rune('.') {
					goto l452
				}
				position++
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l452
				}
				position++
			l458:
				{
					position459, tokenIndex459, depth459 := position, tokenIndex, depth
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l459
					}
					position++
					goto l458
				l459:
					position, tokenIndex, depth = position459, tokenIndex459, depth459
				}
				if buffer[position] != rune('.') {
					goto l452
				}
				position++
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l452
				}
				position++
			l460:
				{
					position461, tokenIndex461, depth461 := position, tokenIndex, depth
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l461
					}
					position++
					goto l460
				l461:
					position, tokenIndex, depth = position461, tokenIndex461, depth461
				}
				depth--
				add(ruleIP, position453)
			}
			return true
		l452:
			position, tokenIndex, depth = position452, tokenIndex452, depth452
			return false
		},
		/* 103 ws <- <(' ' / '\t' / '\n' / '\r')*> */
		func() bool {
			{
				position463 := position
				depth++
			l464:
				{
					position465, tokenIndex465, depth465 := position, tokenIndex, depth
					{
						position466, tokenIndex466, depth466 := position, tokenIndex, depth
						if buffer[position] != rune(' ') {
							goto l467
						}
						position++
						goto l466
					l467:
						position, tokenIndex, depth = position466, tokenIndex466, depth466
						if buffer[position] != rune('\t') {
							goto l468
						}
						position++
						goto l466
					l468:
						position, tokenIndex, depth = position466, tokenIndex466, depth466
						if buffer[position] != rune('\n') {
							goto l469
						}
						position++
						goto l466
					l469:
						position, tokenIndex, depth = position466, tokenIndex466, depth466
						if buffer[position] != rune('\r') {
							goto l465
						}
						position++
					}
				l466:
					goto l464
				l465:
					position, tokenIndex, depth = position465, tokenIndex465, depth465
				}
				depth--
				add(rulews, position463)
			}
			return true
		},
		/* 104 req_ws <- <(' ' / '\t' / '\n' / '\r')+> */
		func() bool {
			position470, tokenIndex470, depth470 := position, tokenIndex, depth
			{
				position471 := position
				depth++
				{
					position474, tokenIndex474, depth474 := position, tokenIndex, depth
					if buffer[position] != rune(' ') {
						goto l475
					}
					position++
					goto l474
				l475:
					position, tokenIndex, depth = position474, tokenIndex474, depth474
					if buffer[position] != rune('\t') {
						goto l476
					}
					position++
					goto l474
				l476:
					position, tokenIndex, depth = position474, tokenIndex474, depth474
					if buffer[position] != rune('\n') {
						goto l477
					}
					position++
					goto l474
				l477:
					position, tokenIndex, depth = position474, tokenIndex474, depth474
					if buffer[position] != rune('\r') {
						goto l470
					}
					position++
				}
			l474:
			l472:
				{
					position473, tokenIndex473, depth473 := position, tokenIndex, depth
					{
						position478, tokenIndex478, depth478 := position, tokenIndex, depth
						if buffer[position] != rune(' ') {
							goto l479
						}
						position++
						goto l478
					l479:
						position, tokenIndex, depth = position478, tokenIndex478, depth478
						if buffer[position] != rune('\t') {
							goto l480
						}
						position++
						goto l478
					l480:
						position, tokenIndex, depth = position478, tokenIndex478, depth478
						if buffer[position] != rune('\n') {
							goto l481
						}
						position++
						goto l478
					l481:
						position, tokenIndex, depth = position478, tokenIndex478, depth478
						if buffer[position] != rune('\r') {
							goto l473
						}
						position++
					}
				l478:
					goto l472
				l473:
					position, tokenIndex, depth = position473, tokenIndex473, depth473
				}
				depth--
				add(rulereq_ws, position471)
			}
			return true
		l470:
			position, tokenIndex, depth = position470, tokenIndex470, depth470
			return false
		},
		/* 106 Action0 <- <{}> */
		func() bool {
			{
				add(ruleAction0, position)
			}
			return true
		},
		/* 107 Action1 <- <{}> */
		func() bool {
			{
				add(ruleAction1, position)
			}
			return true
		},
		/* 108 Action2 <- <{}> */
		func() bool {
			{
				add(ruleAction2, position)
			}
			return true
		},
	}
	p.rules = _rules
}
