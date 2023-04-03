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
	rules  [107]func() bool
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
		/* 32 Chained <- <((MapMapping / Sync / Catch / Mapping / MapSelection / Selection / Sum / List / Map / Range / Grouped / Reference) ChainedQualifiedExpression*)> */
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
						goto l130
					}
				}
			l132:
			l144:
				{
					position145, tokenIndex145, depth145 := position, tokenIndex, depth
					if !_rules[ruleChainedQualifiedExpression]() {
						goto l145
					}
					goto l144
				l145:
					position, tokenIndex, depth = position145, tokenIndex145, depth145
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
			position146, tokenIndex146, depth146 := position, tokenIndex, depth
			{
				position147 := position
				depth++
				{
					position148, tokenIndex148, depth148 := position, tokenIndex, depth
					if !_rules[ruleChainedCall]() {
						goto l149
					}
					goto l148
				l149:
					position, tokenIndex, depth = position148, tokenIndex148, depth148
					if !_rules[ruleCurrying]() {
						goto l150
					}
					goto l148
				l150:
					position, tokenIndex, depth = position148, tokenIndex148, depth148
					if !_rules[ruleChainedRef]() {
						goto l151
					}
					goto l148
				l151:
					position, tokenIndex, depth = position148, tokenIndex148, depth148
					if !_rules[ruleChainedDynRef]() {
						goto l152
					}
					goto l148
				l152:
					position, tokenIndex, depth = position148, tokenIndex148, depth148
					if !_rules[ruleProjection]() {
						goto l146
					}
				}
			l148:
				depth--
				add(ruleChainedQualifiedExpression, position147)
			}
			return true
		l146:
			position, tokenIndex, depth = position146, tokenIndex146, depth146
			return false
		},
		/* 34 ChainedRef <- <(PathComponent FollowUpRef)> */
		func() bool {
			position153, tokenIndex153, depth153 := position, tokenIndex, depth
			{
				position154 := position
				depth++
				if !_rules[rulePathComponent]() {
					goto l153
				}
				if !_rules[ruleFollowUpRef]() {
					goto l153
				}
				depth--
				add(ruleChainedRef, position154)
			}
			return true
		l153:
			position, tokenIndex, depth = position153, tokenIndex153, depth153
			return false
		},
		/* 35 ChainedDynRef <- <('.'? '[' Expression ']')> */
		func() bool {
			position155, tokenIndex155, depth155 := position, tokenIndex, depth
			{
				position156 := position
				depth++
				{
					position157, tokenIndex157, depth157 := position, tokenIndex, depth
					if buffer[position] != rune('.') {
						goto l157
					}
					position++
					goto l158
				l157:
					position, tokenIndex, depth = position157, tokenIndex157, depth157
				}
			l158:
				if buffer[position] != rune('[') {
					goto l155
				}
				position++
				if !_rules[ruleExpression]() {
					goto l155
				}
				if buffer[position] != rune(']') {
					goto l155
				}
				position++
				depth--
				add(ruleChainedDynRef, position156)
			}
			return true
		l155:
			position, tokenIndex, depth = position155, tokenIndex155, depth155
			return false
		},
		/* 36 Slice <- <Range> */
		func() bool {
			position159, tokenIndex159, depth159 := position, tokenIndex, depth
			{
				position160 := position
				depth++
				if !_rules[ruleRange]() {
					goto l159
				}
				depth--
				add(ruleSlice, position160)
			}
			return true
		l159:
			position, tokenIndex, depth = position159, tokenIndex159, depth159
			return false
		},
		/* 37 Currying <- <('*' ChainedCall)> */
		func() bool {
			position161, tokenIndex161, depth161 := position, tokenIndex, depth
			{
				position162 := position
				depth++
				if buffer[position] != rune('*') {
					goto l161
				}
				position++
				if !_rules[ruleChainedCall]() {
					goto l161
				}
				depth--
				add(ruleCurrying, position162)
			}
			return true
		l161:
			position, tokenIndex, depth = position161, tokenIndex161, depth161
			return false
		},
		/* 38 ChainedCall <- <(StartArguments NameArgumentList? ')')> */
		func() bool {
			position163, tokenIndex163, depth163 := position, tokenIndex, depth
			{
				position164 := position
				depth++
				if !_rules[ruleStartArguments]() {
					goto l163
				}
				{
					position165, tokenIndex165, depth165 := position, tokenIndex, depth
					if !_rules[ruleNameArgumentList]() {
						goto l165
					}
					goto l166
				l165:
					position, tokenIndex, depth = position165, tokenIndex165, depth165
				}
			l166:
				if buffer[position] != rune(')') {
					goto l163
				}
				position++
				depth--
				add(ruleChainedCall, position164)
			}
			return true
		l163:
			position, tokenIndex, depth = position163, tokenIndex163, depth163
			return false
		},
		/* 39 StartArguments <- <('(' ws)> */
		func() bool {
			position167, tokenIndex167, depth167 := position, tokenIndex, depth
			{
				position168 := position
				depth++
				if buffer[position] != rune('(') {
					goto l167
				}
				position++
				if !_rules[rulews]() {
					goto l167
				}
				depth--
				add(ruleStartArguments, position168)
			}
			return true
		l167:
			position, tokenIndex, depth = position167, tokenIndex167, depth167
			return false
		},
		/* 40 NameArgumentList <- <(((NextNameArgument (',' NextNameArgument)*) / NextExpression) (',' NextExpression)*)> */
		func() bool {
			position169, tokenIndex169, depth169 := position, tokenIndex, depth
			{
				position170 := position
				depth++
				{
					position171, tokenIndex171, depth171 := position, tokenIndex, depth
					if !_rules[ruleNextNameArgument]() {
						goto l172
					}
				l173:
					{
						position174, tokenIndex174, depth174 := position, tokenIndex, depth
						if buffer[position] != rune(',') {
							goto l174
						}
						position++
						if !_rules[ruleNextNameArgument]() {
							goto l174
						}
						goto l173
					l174:
						position, tokenIndex, depth = position174, tokenIndex174, depth174
					}
					goto l171
				l172:
					position, tokenIndex, depth = position171, tokenIndex171, depth171
					if !_rules[ruleNextExpression]() {
						goto l169
					}
				}
			l171:
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
				add(ruleNameArgumentList, position170)
			}
			return true
		l169:
			position, tokenIndex, depth = position169, tokenIndex169, depth169
			return false
		},
		/* 41 NextNameArgument <- <(ws Name ws '=' ws Expression ws)> */
		func() bool {
			position177, tokenIndex177, depth177 := position, tokenIndex, depth
			{
				position178 := position
				depth++
				if !_rules[rulews]() {
					goto l177
				}
				if !_rules[ruleName]() {
					goto l177
				}
				if !_rules[rulews]() {
					goto l177
				}
				if buffer[position] != rune('=') {
					goto l177
				}
				position++
				if !_rules[rulews]() {
					goto l177
				}
				if !_rules[ruleExpression]() {
					goto l177
				}
				if !_rules[rulews]() {
					goto l177
				}
				depth--
				add(ruleNextNameArgument, position178)
			}
			return true
		l177:
			position, tokenIndex, depth = position177, tokenIndex177, depth177
			return false
		},
		/* 42 ExpressionList <- <(NextExpression (',' NextExpression)*)> */
		func() bool {
			position179, tokenIndex179, depth179 := position, tokenIndex, depth
			{
				position180 := position
				depth++
				if !_rules[ruleNextExpression]() {
					goto l179
				}
			l181:
				{
					position182, tokenIndex182, depth182 := position, tokenIndex, depth
					if buffer[position] != rune(',') {
						goto l182
					}
					position++
					if !_rules[ruleNextExpression]() {
						goto l182
					}
					goto l181
				l182:
					position, tokenIndex, depth = position182, tokenIndex182, depth182
				}
				depth--
				add(ruleExpressionList, position180)
			}
			return true
		l179:
			position, tokenIndex, depth = position179, tokenIndex179, depth179
			return false
		},
		/* 43 NextExpression <- <(Expression ListExpansion?)> */
		func() bool {
			position183, tokenIndex183, depth183 := position, tokenIndex, depth
			{
				position184 := position
				depth++
				if !_rules[ruleExpression]() {
					goto l183
				}
				{
					position185, tokenIndex185, depth185 := position, tokenIndex, depth
					if !_rules[ruleListExpansion]() {
						goto l185
					}
					goto l186
				l185:
					position, tokenIndex, depth = position185, tokenIndex185, depth185
				}
			l186:
				depth--
				add(ruleNextExpression, position184)
			}
			return true
		l183:
			position, tokenIndex, depth = position183, tokenIndex183, depth183
			return false
		},
		/* 44 ListExpansion <- <('.' '.' '.' ws)> */
		func() bool {
			position187, tokenIndex187, depth187 := position, tokenIndex, depth
			{
				position188 := position
				depth++
				if buffer[position] != rune('.') {
					goto l187
				}
				position++
				if buffer[position] != rune('.') {
					goto l187
				}
				position++
				if buffer[position] != rune('.') {
					goto l187
				}
				position++
				if !_rules[rulews]() {
					goto l187
				}
				depth--
				add(ruleListExpansion, position188)
			}
			return true
		l187:
			position, tokenIndex, depth = position187, tokenIndex187, depth187
			return false
		},
		/* 45 Projection <- <('.'? (('[' '*' ']') / Slice) ProjectionValue ChainedQualifiedExpression*)> */
		func() bool {
			position189, tokenIndex189, depth189 := position, tokenIndex, depth
			{
				position190 := position
				depth++
				{
					position191, tokenIndex191, depth191 := position, tokenIndex, depth
					if buffer[position] != rune('.') {
						goto l191
					}
					position++
					goto l192
				l191:
					position, tokenIndex, depth = position191, tokenIndex191, depth191
				}
			l192:
				{
					position193, tokenIndex193, depth193 := position, tokenIndex, depth
					if buffer[position] != rune('[') {
						goto l194
					}
					position++
					if buffer[position] != rune('*') {
						goto l194
					}
					position++
					if buffer[position] != rune(']') {
						goto l194
					}
					position++
					goto l193
				l194:
					position, tokenIndex, depth = position193, tokenIndex193, depth193
					if !_rules[ruleSlice]() {
						goto l189
					}
				}
			l193:
				if !_rules[ruleProjectionValue]() {
					goto l189
				}
			l195:
				{
					position196, tokenIndex196, depth196 := position, tokenIndex, depth
					if !_rules[ruleChainedQualifiedExpression]() {
						goto l196
					}
					goto l195
				l196:
					position, tokenIndex, depth = position196, tokenIndex196, depth196
				}
				depth--
				add(ruleProjection, position190)
			}
			return true
		l189:
			position, tokenIndex, depth = position189, tokenIndex189, depth189
			return false
		},
		/* 46 ProjectionValue <- <Action0> */
		func() bool {
			position197, tokenIndex197, depth197 := position, tokenIndex, depth
			{
				position198 := position
				depth++
				if !_rules[ruleAction0]() {
					goto l197
				}
				depth--
				add(ruleProjectionValue, position198)
			}
			return true
		l197:
			position, tokenIndex, depth = position197, tokenIndex197, depth197
			return false
		},
		/* 47 Substitution <- <('*' Level0)> */
		func() bool {
			position199, tokenIndex199, depth199 := position, tokenIndex, depth
			{
				position200 := position
				depth++
				if buffer[position] != rune('*') {
					goto l199
				}
				position++
				if !_rules[ruleLevel0]() {
					goto l199
				}
				depth--
				add(ruleSubstitution, position200)
			}
			return true
		l199:
			position, tokenIndex, depth = position199, tokenIndex199, depth199
			return false
		},
		/* 48 Not <- <('!' ws Level0)> */
		func() bool {
			position201, tokenIndex201, depth201 := position, tokenIndex, depth
			{
				position202 := position
				depth++
				if buffer[position] != rune('!') {
					goto l201
				}
				position++
				if !_rules[rulews]() {
					goto l201
				}
				if !_rules[ruleLevel0]() {
					goto l201
				}
				depth--
				add(ruleNot, position202)
			}
			return true
		l201:
			position, tokenIndex, depth = position201, tokenIndex201, depth201
			return false
		},
		/* 49 Grouped <- <('(' Expression ')')> */
		func() bool {
			position203, tokenIndex203, depth203 := position, tokenIndex, depth
			{
				position204 := position
				depth++
				if buffer[position] != rune('(') {
					goto l203
				}
				position++
				if !_rules[ruleExpression]() {
					goto l203
				}
				if buffer[position] != rune(')') {
					goto l203
				}
				position++
				depth--
				add(ruleGrouped, position204)
			}
			return true
		l203:
			position, tokenIndex, depth = position203, tokenIndex203, depth203
			return false
		},
		/* 50 Range <- <(StartRange Expression? RangeOp Expression? ']')> */
		func() bool {
			position205, tokenIndex205, depth205 := position, tokenIndex, depth
			{
				position206 := position
				depth++
				if !_rules[ruleStartRange]() {
					goto l205
				}
				{
					position207, tokenIndex207, depth207 := position, tokenIndex, depth
					if !_rules[ruleExpression]() {
						goto l207
					}
					goto l208
				l207:
					position, tokenIndex, depth = position207, tokenIndex207, depth207
				}
			l208:
				if !_rules[ruleRangeOp]() {
					goto l205
				}
				{
					position209, tokenIndex209, depth209 := position, tokenIndex, depth
					if !_rules[ruleExpression]() {
						goto l209
					}
					goto l210
				l209:
					position, tokenIndex, depth = position209, tokenIndex209, depth209
				}
			l210:
				if buffer[position] != rune(']') {
					goto l205
				}
				position++
				depth--
				add(ruleRange, position206)
			}
			return true
		l205:
			position, tokenIndex, depth = position205, tokenIndex205, depth205
			return false
		},
		/* 51 StartRange <- <'['> */
		func() bool {
			position211, tokenIndex211, depth211 := position, tokenIndex, depth
			{
				position212 := position
				depth++
				if buffer[position] != rune('[') {
					goto l211
				}
				position++
				depth--
				add(ruleStartRange, position212)
			}
			return true
		l211:
			position, tokenIndex, depth = position211, tokenIndex211, depth211
			return false
		},
		/* 52 RangeOp <- <('.' '.')> */
		func() bool {
			position213, tokenIndex213, depth213 := position, tokenIndex, depth
			{
				position214 := position
				depth++
				if buffer[position] != rune('.') {
					goto l213
				}
				position++
				if buffer[position] != rune('.') {
					goto l213
				}
				position++
				depth--
				add(ruleRangeOp, position214)
			}
			return true
		l213:
			position, tokenIndex, depth = position213, tokenIndex213, depth213
			return false
		},
		/* 53 Number <- <('-'? [0-9] ([0-9] / '_')* ('.' [0-9] [0-9]*)? (('e' / 'E') '-'? [0-9] [0-9]*)? !(':' ':'))> */
		func() bool {
			position215, tokenIndex215, depth215 := position, tokenIndex, depth
			{
				position216 := position
				depth++
				{
					position217, tokenIndex217, depth217 := position, tokenIndex, depth
					if buffer[position] != rune('-') {
						goto l217
					}
					position++
					goto l218
				l217:
					position, tokenIndex, depth = position217, tokenIndex217, depth217
				}
			l218:
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l215
				}
				position++
			l219:
				{
					position220, tokenIndex220, depth220 := position, tokenIndex, depth
					{
						position221, tokenIndex221, depth221 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l222
						}
						position++
						goto l221
					l222:
						position, tokenIndex, depth = position221, tokenIndex221, depth221
						if buffer[position] != rune('_') {
							goto l220
						}
						position++
					}
				l221:
					goto l219
				l220:
					position, tokenIndex, depth = position220, tokenIndex220, depth220
				}
				{
					position223, tokenIndex223, depth223 := position, tokenIndex, depth
					if buffer[position] != rune('.') {
						goto l223
					}
					position++
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l223
					}
					position++
				l225:
					{
						position226, tokenIndex226, depth226 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l226
						}
						position++
						goto l225
					l226:
						position, tokenIndex, depth = position226, tokenIndex226, depth226
					}
					goto l224
				l223:
					position, tokenIndex, depth = position223, tokenIndex223, depth223
				}
			l224:
				{
					position227, tokenIndex227, depth227 := position, tokenIndex, depth
					{
						position229, tokenIndex229, depth229 := position, tokenIndex, depth
						if buffer[position] != rune('e') {
							goto l230
						}
						position++
						goto l229
					l230:
						position, tokenIndex, depth = position229, tokenIndex229, depth229
						if buffer[position] != rune('E') {
							goto l227
						}
						position++
					}
				l229:
					{
						position231, tokenIndex231, depth231 := position, tokenIndex, depth
						if buffer[position] != rune('-') {
							goto l231
						}
						position++
						goto l232
					l231:
						position, tokenIndex, depth = position231, tokenIndex231, depth231
					}
				l232:
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l227
					}
					position++
				l233:
					{
						position234, tokenIndex234, depth234 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l234
						}
						position++
						goto l233
					l234:
						position, tokenIndex, depth = position234, tokenIndex234, depth234
					}
					goto l228
				l227:
					position, tokenIndex, depth = position227, tokenIndex227, depth227
				}
			l228:
				{
					position235, tokenIndex235, depth235 := position, tokenIndex, depth
					if buffer[position] != rune(':') {
						goto l235
					}
					position++
					if buffer[position] != rune(':') {
						goto l235
					}
					position++
					goto l215
				l235:
					position, tokenIndex, depth = position235, tokenIndex235, depth235
				}
				depth--
				add(ruleNumber, position216)
			}
			return true
		l215:
			position, tokenIndex, depth = position215, tokenIndex215, depth215
			return false
		},
		/* 54 String <- <('"' (('\\' '"') / (!'"' .))* '"')> */
		func() bool {
			position236, tokenIndex236, depth236 := position, tokenIndex, depth
			{
				position237 := position
				depth++
				if buffer[position] != rune('"') {
					goto l236
				}
				position++
			l238:
				{
					position239, tokenIndex239, depth239 := position, tokenIndex, depth
					{
						position240, tokenIndex240, depth240 := position, tokenIndex, depth
						if buffer[position] != rune('\\') {
							goto l241
						}
						position++
						if buffer[position] != rune('"') {
							goto l241
						}
						position++
						goto l240
					l241:
						position, tokenIndex, depth = position240, tokenIndex240, depth240
						{
							position242, tokenIndex242, depth242 := position, tokenIndex, depth
							if buffer[position] != rune('"') {
								goto l242
							}
							position++
							goto l239
						l242:
							position, tokenIndex, depth = position242, tokenIndex242, depth242
						}
						if !matchDot() {
							goto l239
						}
					}
				l240:
					goto l238
				l239:
					position, tokenIndex, depth = position239, tokenIndex239, depth239
				}
				if buffer[position] != rune('"') {
					goto l236
				}
				position++
				depth--
				add(ruleString, position237)
			}
			return true
		l236:
			position, tokenIndex, depth = position236, tokenIndex236, depth236
			return false
		},
		/* 55 Boolean <- <(('t' 'r' 'u' 'e') / ('f' 'a' 'l' 's' 'e'))> */
		func() bool {
			position243, tokenIndex243, depth243 := position, tokenIndex, depth
			{
				position244 := position
				depth++
				{
					position245, tokenIndex245, depth245 := position, tokenIndex, depth
					if buffer[position] != rune('t') {
						goto l246
					}
					position++
					if buffer[position] != rune('r') {
						goto l246
					}
					position++
					if buffer[position] != rune('u') {
						goto l246
					}
					position++
					if buffer[position] != rune('e') {
						goto l246
					}
					position++
					goto l245
				l246:
					position, tokenIndex, depth = position245, tokenIndex245, depth245
					if buffer[position] != rune('f') {
						goto l243
					}
					position++
					if buffer[position] != rune('a') {
						goto l243
					}
					position++
					if buffer[position] != rune('l') {
						goto l243
					}
					position++
					if buffer[position] != rune('s') {
						goto l243
					}
					position++
					if buffer[position] != rune('e') {
						goto l243
					}
					position++
				}
			l245:
				depth--
				add(ruleBoolean, position244)
			}
			return true
		l243:
			position, tokenIndex, depth = position243, tokenIndex243, depth243
			return false
		},
		/* 56 Nil <- <(('n' 'i' 'l') / '~')> */
		func() bool {
			position247, tokenIndex247, depth247 := position, tokenIndex, depth
			{
				position248 := position
				depth++
				{
					position249, tokenIndex249, depth249 := position, tokenIndex, depth
					if buffer[position] != rune('n') {
						goto l250
					}
					position++
					if buffer[position] != rune('i') {
						goto l250
					}
					position++
					if buffer[position] != rune('l') {
						goto l250
					}
					position++
					goto l249
				l250:
					position, tokenIndex, depth = position249, tokenIndex249, depth249
					if buffer[position] != rune('~') {
						goto l247
					}
					position++
				}
			l249:
				depth--
				add(ruleNil, position248)
			}
			return true
		l247:
			position, tokenIndex, depth = position247, tokenIndex247, depth247
			return false
		},
		/* 57 Undefined <- <('~' '~')> */
		func() bool {
			position251, tokenIndex251, depth251 := position, tokenIndex, depth
			{
				position252 := position
				depth++
				if buffer[position] != rune('~') {
					goto l251
				}
				position++
				if buffer[position] != rune('~') {
					goto l251
				}
				position++
				depth--
				add(ruleUndefined, position252)
			}
			return true
		l251:
			position, tokenIndex, depth = position251, tokenIndex251, depth251
			return false
		},
		/* 58 Symbol <- <('$' Name)> */
		func() bool {
			position253, tokenIndex253, depth253 := position, tokenIndex, depth
			{
				position254 := position
				depth++
				if buffer[position] != rune('$') {
					goto l253
				}
				position++
				if !_rules[ruleName]() {
					goto l253
				}
				depth--
				add(ruleSymbol, position254)
			}
			return true
		l253:
			position, tokenIndex, depth = position253, tokenIndex253, depth253
			return false
		},
		/* 59 List <- <(StartList ExpressionList? ']')> */
		func() bool {
			position255, tokenIndex255, depth255 := position, tokenIndex, depth
			{
				position256 := position
				depth++
				if !_rules[ruleStartList]() {
					goto l255
				}
				{
					position257, tokenIndex257, depth257 := position, tokenIndex, depth
					if !_rules[ruleExpressionList]() {
						goto l257
					}
					goto l258
				l257:
					position, tokenIndex, depth = position257, tokenIndex257, depth257
				}
			l258:
				if buffer[position] != rune(']') {
					goto l255
				}
				position++
				depth--
				add(ruleList, position256)
			}
			return true
		l255:
			position, tokenIndex, depth = position255, tokenIndex255, depth255
			return false
		},
		/* 60 StartList <- <('[' ws)> */
		func() bool {
			position259, tokenIndex259, depth259 := position, tokenIndex, depth
			{
				position260 := position
				depth++
				if buffer[position] != rune('[') {
					goto l259
				}
				position++
				if !_rules[rulews]() {
					goto l259
				}
				depth--
				add(ruleStartList, position260)
			}
			return true
		l259:
			position, tokenIndex, depth = position259, tokenIndex259, depth259
			return false
		},
		/* 61 Map <- <(CreateMap ws Assignments? '}')> */
		func() bool {
			position261, tokenIndex261, depth261 := position, tokenIndex, depth
			{
				position262 := position
				depth++
				if !_rules[ruleCreateMap]() {
					goto l261
				}
				if !_rules[rulews]() {
					goto l261
				}
				{
					position263, tokenIndex263, depth263 := position, tokenIndex, depth
					if !_rules[ruleAssignments]() {
						goto l263
					}
					goto l264
				l263:
					position, tokenIndex, depth = position263, tokenIndex263, depth263
				}
			l264:
				if buffer[position] != rune('}') {
					goto l261
				}
				position++
				depth--
				add(ruleMap, position262)
			}
			return true
		l261:
			position, tokenIndex, depth = position261, tokenIndex261, depth261
			return false
		},
		/* 62 CreateMap <- <'{'> */
		func() bool {
			position265, tokenIndex265, depth265 := position, tokenIndex, depth
			{
				position266 := position
				depth++
				if buffer[position] != rune('{') {
					goto l265
				}
				position++
				depth--
				add(ruleCreateMap, position266)
			}
			return true
		l265:
			position, tokenIndex, depth = position265, tokenIndex265, depth265
			return false
		},
		/* 63 Assignments <- <(Assignment (',' Assignment)*)> */
		func() bool {
			position267, tokenIndex267, depth267 := position, tokenIndex, depth
			{
				position268 := position
				depth++
				if !_rules[ruleAssignment]() {
					goto l267
				}
			l269:
				{
					position270, tokenIndex270, depth270 := position, tokenIndex, depth
					if buffer[position] != rune(',') {
						goto l270
					}
					position++
					if !_rules[ruleAssignment]() {
						goto l270
					}
					goto l269
				l270:
					position, tokenIndex, depth = position270, tokenIndex270, depth270
				}
				depth--
				add(ruleAssignments, position268)
			}
			return true
		l267:
			position, tokenIndex, depth = position267, tokenIndex267, depth267
			return false
		},
		/* 64 Assignment <- <(Expression '=' Expression)> */
		func() bool {
			position271, tokenIndex271, depth271 := position, tokenIndex, depth
			{
				position272 := position
				depth++
				if !_rules[ruleExpression]() {
					goto l271
				}
				if buffer[position] != rune('=') {
					goto l271
				}
				position++
				if !_rules[ruleExpression]() {
					goto l271
				}
				depth--
				add(ruleAssignment, position272)
			}
			return true
		l271:
			position, tokenIndex, depth = position271, tokenIndex271, depth271
			return false
		},
		/* 65 Merge <- <(RefMerge / SimpleMerge)> */
		func() bool {
			position273, tokenIndex273, depth273 := position, tokenIndex, depth
			{
				position274 := position
				depth++
				{
					position275, tokenIndex275, depth275 := position, tokenIndex, depth
					if !_rules[ruleRefMerge]() {
						goto l276
					}
					goto l275
				l276:
					position, tokenIndex, depth = position275, tokenIndex275, depth275
					if !_rules[ruleSimpleMerge]() {
						goto l273
					}
				}
			l275:
				depth--
				add(ruleMerge, position274)
			}
			return true
		l273:
			position, tokenIndex, depth = position273, tokenIndex273, depth273
			return false
		},
		/* 66 RefMerge <- <('m' 'e' 'r' 'g' 'e' !(req_ws Required) (req_ws (Replace / On))? req_ws Reference)> */
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
					if !_rules[rulereq_ws]() {
						goto l279
					}
					if !_rules[ruleRequired]() {
						goto l279
					}
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
				if !_rules[rulereq_ws]() {
					goto l277
				}
				if !_rules[ruleReference]() {
					goto l277
				}
				depth--
				add(ruleRefMerge, position278)
			}
			return true
		l277:
			position, tokenIndex, depth = position277, tokenIndex277, depth277
			return false
		},
		/* 67 SimpleMerge <- <('m' 'e' 'r' 'g' 'e' !'(' (req_ws (Replace / Required / On))?)> */
		func() bool {
			position284, tokenIndex284, depth284 := position, tokenIndex, depth
			{
				position285 := position
				depth++
				if buffer[position] != rune('m') {
					goto l284
				}
				position++
				if buffer[position] != rune('e') {
					goto l284
				}
				position++
				if buffer[position] != rune('r') {
					goto l284
				}
				position++
				if buffer[position] != rune('g') {
					goto l284
				}
				position++
				if buffer[position] != rune('e') {
					goto l284
				}
				position++
				{
					position286, tokenIndex286, depth286 := position, tokenIndex, depth
					if buffer[position] != rune('(') {
						goto l286
					}
					position++
					goto l284
				l286:
					position, tokenIndex, depth = position286, tokenIndex286, depth286
				}
				{
					position287, tokenIndex287, depth287 := position, tokenIndex, depth
					if !_rules[rulereq_ws]() {
						goto l287
					}
					{
						position289, tokenIndex289, depth289 := position, tokenIndex, depth
						if !_rules[ruleReplace]() {
							goto l290
						}
						goto l289
					l290:
						position, tokenIndex, depth = position289, tokenIndex289, depth289
						if !_rules[ruleRequired]() {
							goto l291
						}
						goto l289
					l291:
						position, tokenIndex, depth = position289, tokenIndex289, depth289
						if !_rules[ruleOn]() {
							goto l287
						}
					}
				l289:
					goto l288
				l287:
					position, tokenIndex, depth = position287, tokenIndex287, depth287
				}
			l288:
				depth--
				add(ruleSimpleMerge, position285)
			}
			return true
		l284:
			position, tokenIndex, depth = position284, tokenIndex284, depth284
			return false
		},
		/* 68 Replace <- <('r' 'e' 'p' 'l' 'a' 'c' 'e')> */
		func() bool {
			position292, tokenIndex292, depth292 := position, tokenIndex, depth
			{
				position293 := position
				depth++
				if buffer[position] != rune('r') {
					goto l292
				}
				position++
				if buffer[position] != rune('e') {
					goto l292
				}
				position++
				if buffer[position] != rune('p') {
					goto l292
				}
				position++
				if buffer[position] != rune('l') {
					goto l292
				}
				position++
				if buffer[position] != rune('a') {
					goto l292
				}
				position++
				if buffer[position] != rune('c') {
					goto l292
				}
				position++
				if buffer[position] != rune('e') {
					goto l292
				}
				position++
				depth--
				add(ruleReplace, position293)
			}
			return true
		l292:
			position, tokenIndex, depth = position292, tokenIndex292, depth292
			return false
		},
		/* 69 Required <- <('r' 'e' 'q' 'u' 'i' 'r' 'e' 'd')> */
		func() bool {
			position294, tokenIndex294, depth294 := position, tokenIndex, depth
			{
				position295 := position
				depth++
				if buffer[position] != rune('r') {
					goto l294
				}
				position++
				if buffer[position] != rune('e') {
					goto l294
				}
				position++
				if buffer[position] != rune('q') {
					goto l294
				}
				position++
				if buffer[position] != rune('u') {
					goto l294
				}
				position++
				if buffer[position] != rune('i') {
					goto l294
				}
				position++
				if buffer[position] != rune('r') {
					goto l294
				}
				position++
				if buffer[position] != rune('e') {
					goto l294
				}
				position++
				if buffer[position] != rune('d') {
					goto l294
				}
				position++
				depth--
				add(ruleRequired, position295)
			}
			return true
		l294:
			position, tokenIndex, depth = position294, tokenIndex294, depth294
			return false
		},
		/* 70 On <- <('o' 'n' req_ws Name)> */
		func() bool {
			position296, tokenIndex296, depth296 := position, tokenIndex, depth
			{
				position297 := position
				depth++
				if buffer[position] != rune('o') {
					goto l296
				}
				position++
				if buffer[position] != rune('n') {
					goto l296
				}
				position++
				if !_rules[rulereq_ws]() {
					goto l296
				}
				if !_rules[ruleName]() {
					goto l296
				}
				depth--
				add(ruleOn, position297)
			}
			return true
		l296:
			position, tokenIndex, depth = position296, tokenIndex296, depth296
			return false
		},
		/* 71 Auto <- <('a' 'u' 't' 'o')> */
		func() bool {
			position298, tokenIndex298, depth298 := position, tokenIndex, depth
			{
				position299 := position
				depth++
				if buffer[position] != rune('a') {
					goto l298
				}
				position++
				if buffer[position] != rune('u') {
					goto l298
				}
				position++
				if buffer[position] != rune('t') {
					goto l298
				}
				position++
				if buffer[position] != rune('o') {
					goto l298
				}
				position++
				depth--
				add(ruleAuto, position299)
			}
			return true
		l298:
			position, tokenIndex, depth = position298, tokenIndex298, depth298
			return false
		},
		/* 72 Default <- <Action1> */
		func() bool {
			position300, tokenIndex300, depth300 := position, tokenIndex, depth
			{
				position301 := position
				depth++
				if !_rules[ruleAction1]() {
					goto l300
				}
				depth--
				add(ruleDefault, position301)
			}
			return true
		l300:
			position, tokenIndex, depth = position300, tokenIndex300, depth300
			return false
		},
		/* 73 Sync <- <('s' 'y' 'n' 'c' '[' Level7 ((((LambdaExpr LambdaExt) / (LambdaOrExpr LambdaOrExpr)) (('|' Expression) / Default)) / (LambdaOrExpr Default Default)) ']')> */
		func() bool {
			position302, tokenIndex302, depth302 := position, tokenIndex, depth
			{
				position303 := position
				depth++
				if buffer[position] != rune('s') {
					goto l302
				}
				position++
				if buffer[position] != rune('y') {
					goto l302
				}
				position++
				if buffer[position] != rune('n') {
					goto l302
				}
				position++
				if buffer[position] != rune('c') {
					goto l302
				}
				position++
				if buffer[position] != rune('[') {
					goto l302
				}
				position++
				if !_rules[ruleLevel7]() {
					goto l302
				}
				{
					position304, tokenIndex304, depth304 := position, tokenIndex, depth
					{
						position306, tokenIndex306, depth306 := position, tokenIndex, depth
						if !_rules[ruleLambdaExpr]() {
							goto l307
						}
						if !_rules[ruleLambdaExt]() {
							goto l307
						}
						goto l306
					l307:
						position, tokenIndex, depth = position306, tokenIndex306, depth306
						if !_rules[ruleLambdaOrExpr]() {
							goto l305
						}
						if !_rules[ruleLambdaOrExpr]() {
							goto l305
						}
					}
				l306:
					{
						position308, tokenIndex308, depth308 := position, tokenIndex, depth
						if buffer[position] != rune('|') {
							goto l309
						}
						position++
						if !_rules[ruleExpression]() {
							goto l309
						}
						goto l308
					l309:
						position, tokenIndex, depth = position308, tokenIndex308, depth308
						if !_rules[ruleDefault]() {
							goto l305
						}
					}
				l308:
					goto l304
				l305:
					position, tokenIndex, depth = position304, tokenIndex304, depth304
					if !_rules[ruleLambdaOrExpr]() {
						goto l302
					}
					if !_rules[ruleDefault]() {
						goto l302
					}
					if !_rules[ruleDefault]() {
						goto l302
					}
				}
			l304:
				if buffer[position] != rune(']') {
					goto l302
				}
				position++
				depth--
				add(ruleSync, position303)
			}
			return true
		l302:
			position, tokenIndex, depth = position302, tokenIndex302, depth302
			return false
		},
		/* 74 LambdaExt <- <(',' Expression)> */
		func() bool {
			position310, tokenIndex310, depth310 := position, tokenIndex, depth
			{
				position311 := position
				depth++
				if buffer[position] != rune(',') {
					goto l310
				}
				position++
				if !_rules[ruleExpression]() {
					goto l310
				}
				depth--
				add(ruleLambdaExt, position311)
			}
			return true
		l310:
			position, tokenIndex, depth = position310, tokenIndex310, depth310
			return false
		},
		/* 75 LambdaOrExpr <- <(LambdaExpr / ('|' Expression))> */
		func() bool {
			position312, tokenIndex312, depth312 := position, tokenIndex, depth
			{
				position313 := position
				depth++
				{
					position314, tokenIndex314, depth314 := position, tokenIndex, depth
					if !_rules[ruleLambdaExpr]() {
						goto l315
					}
					goto l314
				l315:
					position, tokenIndex, depth = position314, tokenIndex314, depth314
					if buffer[position] != rune('|') {
						goto l312
					}
					position++
					if !_rules[ruleExpression]() {
						goto l312
					}
				}
			l314:
				depth--
				add(ruleLambdaOrExpr, position313)
			}
			return true
		l312:
			position, tokenIndex, depth = position312, tokenIndex312, depth312
			return false
		},
		/* 76 Catch <- <('c' 'a' 't' 'c' 'h' '[' Level7 LambdaOrExpr ']')> */
		func() bool {
			position316, tokenIndex316, depth316 := position, tokenIndex, depth
			{
				position317 := position
				depth++
				if buffer[position] != rune('c') {
					goto l316
				}
				position++
				if buffer[position] != rune('a') {
					goto l316
				}
				position++
				if buffer[position] != rune('t') {
					goto l316
				}
				position++
				if buffer[position] != rune('c') {
					goto l316
				}
				position++
				if buffer[position] != rune('h') {
					goto l316
				}
				position++
				if buffer[position] != rune('[') {
					goto l316
				}
				position++
				if !_rules[ruleLevel7]() {
					goto l316
				}
				if !_rules[ruleLambdaOrExpr]() {
					goto l316
				}
				if buffer[position] != rune(']') {
					goto l316
				}
				position++
				depth--
				add(ruleCatch, position317)
			}
			return true
		l316:
			position, tokenIndex, depth = position316, tokenIndex316, depth316
			return false
		},
		/* 77 MapMapping <- <('m' 'a' 'p' '{' Level7 LambdaOrExpr '}')> */
		func() bool {
			position318, tokenIndex318, depth318 := position, tokenIndex, depth
			{
				position319 := position
				depth++
				if buffer[position] != rune('m') {
					goto l318
				}
				position++
				if buffer[position] != rune('a') {
					goto l318
				}
				position++
				if buffer[position] != rune('p') {
					goto l318
				}
				position++
				if buffer[position] != rune('{') {
					goto l318
				}
				position++
				if !_rules[ruleLevel7]() {
					goto l318
				}
				if !_rules[ruleLambdaOrExpr]() {
					goto l318
				}
				if buffer[position] != rune('}') {
					goto l318
				}
				position++
				depth--
				add(ruleMapMapping, position319)
			}
			return true
		l318:
			position, tokenIndex, depth = position318, tokenIndex318, depth318
			return false
		},
		/* 78 Mapping <- <('m' 'a' 'p' '[' Level7 LambdaOrExpr ']')> */
		func() bool {
			position320, tokenIndex320, depth320 := position, tokenIndex, depth
			{
				position321 := position
				depth++
				if buffer[position] != rune('m') {
					goto l320
				}
				position++
				if buffer[position] != rune('a') {
					goto l320
				}
				position++
				if buffer[position] != rune('p') {
					goto l320
				}
				position++
				if buffer[position] != rune('[') {
					goto l320
				}
				position++
				if !_rules[ruleLevel7]() {
					goto l320
				}
				if !_rules[ruleLambdaOrExpr]() {
					goto l320
				}
				if buffer[position] != rune(']') {
					goto l320
				}
				position++
				depth--
				add(ruleMapping, position321)
			}
			return true
		l320:
			position, tokenIndex, depth = position320, tokenIndex320, depth320
			return false
		},
		/* 79 MapSelection <- <('s' 'e' 'l' 'e' 'c' 't' '{' Level7 LambdaOrExpr '}')> */
		func() bool {
			position322, tokenIndex322, depth322 := position, tokenIndex, depth
			{
				position323 := position
				depth++
				if buffer[position] != rune('s') {
					goto l322
				}
				position++
				if buffer[position] != rune('e') {
					goto l322
				}
				position++
				if buffer[position] != rune('l') {
					goto l322
				}
				position++
				if buffer[position] != rune('e') {
					goto l322
				}
				position++
				if buffer[position] != rune('c') {
					goto l322
				}
				position++
				if buffer[position] != rune('t') {
					goto l322
				}
				position++
				if buffer[position] != rune('{') {
					goto l322
				}
				position++
				if !_rules[ruleLevel7]() {
					goto l322
				}
				if !_rules[ruleLambdaOrExpr]() {
					goto l322
				}
				if buffer[position] != rune('}') {
					goto l322
				}
				position++
				depth--
				add(ruleMapSelection, position323)
			}
			return true
		l322:
			position, tokenIndex, depth = position322, tokenIndex322, depth322
			return false
		},
		/* 80 Selection <- <('s' 'e' 'l' 'e' 'c' 't' '[' Level7 LambdaOrExpr ']')> */
		func() bool {
			position324, tokenIndex324, depth324 := position, tokenIndex, depth
			{
				position325 := position
				depth++
				if buffer[position] != rune('s') {
					goto l324
				}
				position++
				if buffer[position] != rune('e') {
					goto l324
				}
				position++
				if buffer[position] != rune('l') {
					goto l324
				}
				position++
				if buffer[position] != rune('e') {
					goto l324
				}
				position++
				if buffer[position] != rune('c') {
					goto l324
				}
				position++
				if buffer[position] != rune('t') {
					goto l324
				}
				position++
				if buffer[position] != rune('[') {
					goto l324
				}
				position++
				if !_rules[ruleLevel7]() {
					goto l324
				}
				if !_rules[ruleLambdaOrExpr]() {
					goto l324
				}
				if buffer[position] != rune(']') {
					goto l324
				}
				position++
				depth--
				add(ruleSelection, position325)
			}
			return true
		l324:
			position, tokenIndex, depth = position324, tokenIndex324, depth324
			return false
		},
		/* 81 Sum <- <('s' 'u' 'm' '[' Level7 '|' Level7 LambdaOrExpr ']')> */
		func() bool {
			position326, tokenIndex326, depth326 := position, tokenIndex, depth
			{
				position327 := position
				depth++
				if buffer[position] != rune('s') {
					goto l326
				}
				position++
				if buffer[position] != rune('u') {
					goto l326
				}
				position++
				if buffer[position] != rune('m') {
					goto l326
				}
				position++
				if buffer[position] != rune('[') {
					goto l326
				}
				position++
				if !_rules[ruleLevel7]() {
					goto l326
				}
				if buffer[position] != rune('|') {
					goto l326
				}
				position++
				if !_rules[ruleLevel7]() {
					goto l326
				}
				if !_rules[ruleLambdaOrExpr]() {
					goto l326
				}
				if buffer[position] != rune(']') {
					goto l326
				}
				position++
				depth--
				add(ruleSum, position327)
			}
			return true
		l326:
			position, tokenIndex, depth = position326, tokenIndex326, depth326
			return false
		},
		/* 82 Lambda <- <('l' 'a' 'm' 'b' 'd' 'a' (LambdaRef / LambdaExpr))> */
		func() bool {
			position328, tokenIndex328, depth328 := position, tokenIndex, depth
			{
				position329 := position
				depth++
				if buffer[position] != rune('l') {
					goto l328
				}
				position++
				if buffer[position] != rune('a') {
					goto l328
				}
				position++
				if buffer[position] != rune('m') {
					goto l328
				}
				position++
				if buffer[position] != rune('b') {
					goto l328
				}
				position++
				if buffer[position] != rune('d') {
					goto l328
				}
				position++
				if buffer[position] != rune('a') {
					goto l328
				}
				position++
				{
					position330, tokenIndex330, depth330 := position, tokenIndex, depth
					if !_rules[ruleLambdaRef]() {
						goto l331
					}
					goto l330
				l331:
					position, tokenIndex, depth = position330, tokenIndex330, depth330
					if !_rules[ruleLambdaExpr]() {
						goto l328
					}
				}
			l330:
				depth--
				add(ruleLambda, position329)
			}
			return true
		l328:
			position, tokenIndex, depth = position328, tokenIndex328, depth328
			return false
		},
		/* 83 LambdaRef <- <(req_ws Expression)> */
		func() bool {
			position332, tokenIndex332, depth332 := position, tokenIndex, depth
			{
				position333 := position
				depth++
				if !_rules[rulereq_ws]() {
					goto l332
				}
				if !_rules[ruleExpression]() {
					goto l332
				}
				depth--
				add(ruleLambdaRef, position333)
			}
			return true
		l332:
			position, tokenIndex, depth = position332, tokenIndex332, depth332
			return false
		},
		/* 84 LambdaExpr <- <(ws Params ws ('-' '>') Expression)> */
		func() bool {
			position334, tokenIndex334, depth334 := position, tokenIndex, depth
			{
				position335 := position
				depth++
				if !_rules[rulews]() {
					goto l334
				}
				if !_rules[ruleParams]() {
					goto l334
				}
				if !_rules[rulews]() {
					goto l334
				}
				if buffer[position] != rune('-') {
					goto l334
				}
				position++
				if buffer[position] != rune('>') {
					goto l334
				}
				position++
				if !_rules[ruleExpression]() {
					goto l334
				}
				depth--
				add(ruleLambdaExpr, position335)
			}
			return true
		l334:
			position, tokenIndex, depth = position334, tokenIndex334, depth334
			return false
		},
		/* 85 Params <- <('|' StartParams ws Names? '|')> */
		func() bool {
			position336, tokenIndex336, depth336 := position, tokenIndex, depth
			{
				position337 := position
				depth++
				if buffer[position] != rune('|') {
					goto l336
				}
				position++
				if !_rules[ruleStartParams]() {
					goto l336
				}
				if !_rules[rulews]() {
					goto l336
				}
				{
					position338, tokenIndex338, depth338 := position, tokenIndex, depth
					if !_rules[ruleNames]() {
						goto l338
					}
					goto l339
				l338:
					position, tokenIndex, depth = position338, tokenIndex338, depth338
				}
			l339:
				if buffer[position] != rune('|') {
					goto l336
				}
				position++
				depth--
				add(ruleParams, position337)
			}
			return true
		l336:
			position, tokenIndex, depth = position336, tokenIndex336, depth336
			return false
		},
		/* 86 StartParams <- <Action2> */
		func() bool {
			position340, tokenIndex340, depth340 := position, tokenIndex, depth
			{
				position341 := position
				depth++
				if !_rules[ruleAction2]() {
					goto l340
				}
				depth--
				add(ruleStartParams, position341)
			}
			return true
		l340:
			position, tokenIndex, depth = position340, tokenIndex340, depth340
			return false
		},
		/* 87 Names <- <(NextName (',' NextName)* DefaultValue? (',' NextName DefaultValue)* VarParams?)> */
		func() bool {
			position342, tokenIndex342, depth342 := position, tokenIndex, depth
			{
				position343 := position
				depth++
				if !_rules[ruleNextName]() {
					goto l342
				}
			l344:
				{
					position345, tokenIndex345, depth345 := position, tokenIndex, depth
					if buffer[position] != rune(',') {
						goto l345
					}
					position++
					if !_rules[ruleNextName]() {
						goto l345
					}
					goto l344
				l345:
					position, tokenIndex, depth = position345, tokenIndex345, depth345
				}
				{
					position346, tokenIndex346, depth346 := position, tokenIndex, depth
					if !_rules[ruleDefaultValue]() {
						goto l346
					}
					goto l347
				l346:
					position, tokenIndex, depth = position346, tokenIndex346, depth346
				}
			l347:
			l348:
				{
					position349, tokenIndex349, depth349 := position, tokenIndex, depth
					if buffer[position] != rune(',') {
						goto l349
					}
					position++
					if !_rules[ruleNextName]() {
						goto l349
					}
					if !_rules[ruleDefaultValue]() {
						goto l349
					}
					goto l348
				l349:
					position, tokenIndex, depth = position349, tokenIndex349, depth349
				}
				{
					position350, tokenIndex350, depth350 := position, tokenIndex, depth
					if !_rules[ruleVarParams]() {
						goto l350
					}
					goto l351
				l350:
					position, tokenIndex, depth = position350, tokenIndex350, depth350
				}
			l351:
				depth--
				add(ruleNames, position343)
			}
			return true
		l342:
			position, tokenIndex, depth = position342, tokenIndex342, depth342
			return false
		},
		/* 88 NextName <- <(ws Name ws)> */
		func() bool {
			position352, tokenIndex352, depth352 := position, tokenIndex, depth
			{
				position353 := position
				depth++
				if !_rules[rulews]() {
					goto l352
				}
				if !_rules[ruleName]() {
					goto l352
				}
				if !_rules[rulews]() {
					goto l352
				}
				depth--
				add(ruleNextName, position353)
			}
			return true
		l352:
			position, tokenIndex, depth = position352, tokenIndex352, depth352
			return false
		},
		/* 89 Name <- <([a-z] / [A-Z] / [0-9] / '_')+> */
		func() bool {
			position354, tokenIndex354, depth354 := position, tokenIndex, depth
			{
				position355 := position
				depth++
				{
					position358, tokenIndex358, depth358 := position, tokenIndex, depth
					if c := buffer[position]; c < rune('a') || c > rune('z') {
						goto l359
					}
					position++
					goto l358
				l359:
					position, tokenIndex, depth = position358, tokenIndex358, depth358
					if c := buffer[position]; c < rune('A') || c > rune('Z') {
						goto l360
					}
					position++
					goto l358
				l360:
					position, tokenIndex, depth = position358, tokenIndex358, depth358
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l361
					}
					position++
					goto l358
				l361:
					position, tokenIndex, depth = position358, tokenIndex358, depth358
					if buffer[position] != rune('_') {
						goto l354
					}
					position++
				}
			l358:
			l356:
				{
					position357, tokenIndex357, depth357 := position, tokenIndex, depth
					{
						position362, tokenIndex362, depth362 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l363
						}
						position++
						goto l362
					l363:
						position, tokenIndex, depth = position362, tokenIndex362, depth362
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l364
						}
						position++
						goto l362
					l364:
						position, tokenIndex, depth = position362, tokenIndex362, depth362
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l365
						}
						position++
						goto l362
					l365:
						position, tokenIndex, depth = position362, tokenIndex362, depth362
						if buffer[position] != rune('_') {
							goto l357
						}
						position++
					}
				l362:
					goto l356
				l357:
					position, tokenIndex, depth = position357, tokenIndex357, depth357
				}
				depth--
				add(ruleName, position355)
			}
			return true
		l354:
			position, tokenIndex, depth = position354, tokenIndex354, depth354
			return false
		},
		/* 90 DefaultValue <- <('=' Expression)> */
		func() bool {
			position366, tokenIndex366, depth366 := position, tokenIndex, depth
			{
				position367 := position
				depth++
				if buffer[position] != rune('=') {
					goto l366
				}
				position++
				if !_rules[ruleExpression]() {
					goto l366
				}
				depth--
				add(ruleDefaultValue, position367)
			}
			return true
		l366:
			position, tokenIndex, depth = position366, tokenIndex366, depth366
			return false
		},
		/* 91 VarParams <- <('.' '.' '.' ws)> */
		func() bool {
			position368, tokenIndex368, depth368 := position, tokenIndex, depth
			{
				position369 := position
				depth++
				if buffer[position] != rune('.') {
					goto l368
				}
				position++
				if buffer[position] != rune('.') {
					goto l368
				}
				position++
				if buffer[position] != rune('.') {
					goto l368
				}
				position++
				if !_rules[rulews]() {
					goto l368
				}
				depth--
				add(ruleVarParams, position369)
			}
			return true
		l368:
			position, tokenIndex, depth = position368, tokenIndex368, depth368
			return false
		},
		/* 92 Reference <- <(((TagPrefix ('.' / Key)) / ('.'? Key)) FollowUpRef)> */
		func() bool {
			position370, tokenIndex370, depth370 := position, tokenIndex, depth
			{
				position371 := position
				depth++
				{
					position372, tokenIndex372, depth372 := position, tokenIndex, depth
					if !_rules[ruleTagPrefix]() {
						goto l373
					}
					{
						position374, tokenIndex374, depth374 := position, tokenIndex, depth
						if buffer[position] != rune('.') {
							goto l375
						}
						position++
						goto l374
					l375:
						position, tokenIndex, depth = position374, tokenIndex374, depth374
						if !_rules[ruleKey]() {
							goto l373
						}
					}
				l374:
					goto l372
				l373:
					position, tokenIndex, depth = position372, tokenIndex372, depth372
					{
						position376, tokenIndex376, depth376 := position, tokenIndex, depth
						if buffer[position] != rune('.') {
							goto l376
						}
						position++
						goto l377
					l376:
						position, tokenIndex, depth = position376, tokenIndex376, depth376
					}
				l377:
					if !_rules[ruleKey]() {
						goto l370
					}
				}
			l372:
				if !_rules[ruleFollowUpRef]() {
					goto l370
				}
				depth--
				add(ruleReference, position371)
			}
			return true
		l370:
			position, tokenIndex, depth = position370, tokenIndex370, depth370
			return false
		},
		/* 93 TagPrefix <- <((('d' 'o' 'c' ('.' / ':') '-'? [0-9]+) / Tag) (':' ':'))> */
		func() bool {
			position378, tokenIndex378, depth378 := position, tokenIndex, depth
			{
				position379 := position
				depth++
				{
					position380, tokenIndex380, depth380 := position, tokenIndex, depth
					if buffer[position] != rune('d') {
						goto l381
					}
					position++
					if buffer[position] != rune('o') {
						goto l381
					}
					position++
					if buffer[position] != rune('c') {
						goto l381
					}
					position++
					{
						position382, tokenIndex382, depth382 := position, tokenIndex, depth
						if buffer[position] != rune('.') {
							goto l383
						}
						position++
						goto l382
					l383:
						position, tokenIndex, depth = position382, tokenIndex382, depth382
						if buffer[position] != rune(':') {
							goto l381
						}
						position++
					}
				l382:
					{
						position384, tokenIndex384, depth384 := position, tokenIndex, depth
						if buffer[position] != rune('-') {
							goto l384
						}
						position++
						goto l385
					l384:
						position, tokenIndex, depth = position384, tokenIndex384, depth384
					}
				l385:
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l381
					}
					position++
				l386:
					{
						position387, tokenIndex387, depth387 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l387
						}
						position++
						goto l386
					l387:
						position, tokenIndex, depth = position387, tokenIndex387, depth387
					}
					goto l380
				l381:
					position, tokenIndex, depth = position380, tokenIndex380, depth380
					if !_rules[ruleTag]() {
						goto l378
					}
				}
			l380:
				if buffer[position] != rune(':') {
					goto l378
				}
				position++
				if buffer[position] != rune(':') {
					goto l378
				}
				position++
				depth--
				add(ruleTagPrefix, position379)
			}
			return true
		l378:
			position, tokenIndex, depth = position378, tokenIndex378, depth378
			return false
		},
		/* 94 Tag <- <(TagComponent (('.' / ':') TagComponent)*)> */
		func() bool {
			position388, tokenIndex388, depth388 := position, tokenIndex, depth
			{
				position389 := position
				depth++
				if !_rules[ruleTagComponent]() {
					goto l388
				}
			l390:
				{
					position391, tokenIndex391, depth391 := position, tokenIndex, depth
					{
						position392, tokenIndex392, depth392 := position, tokenIndex, depth
						if buffer[position] != rune('.') {
							goto l393
						}
						position++
						goto l392
					l393:
						position, tokenIndex, depth = position392, tokenIndex392, depth392
						if buffer[position] != rune(':') {
							goto l391
						}
						position++
					}
				l392:
					if !_rules[ruleTagComponent]() {
						goto l391
					}
					goto l390
				l391:
					position, tokenIndex, depth = position391, tokenIndex391, depth391
				}
				depth--
				add(ruleTag, position389)
			}
			return true
		l388:
			position, tokenIndex, depth = position388, tokenIndex388, depth388
			return false
		},
		/* 95 TagComponent <- <(([a-z] / [A-Z] / '_') ([a-z] / [A-Z] / [0-9] / '_')*)> */
		func() bool {
			position394, tokenIndex394, depth394 := position, tokenIndex, depth
			{
				position395 := position
				depth++
				{
					position396, tokenIndex396, depth396 := position, tokenIndex, depth
					if c := buffer[position]; c < rune('a') || c > rune('z') {
						goto l397
					}
					position++
					goto l396
				l397:
					position, tokenIndex, depth = position396, tokenIndex396, depth396
					if c := buffer[position]; c < rune('A') || c > rune('Z') {
						goto l398
					}
					position++
					goto l396
				l398:
					position, tokenIndex, depth = position396, tokenIndex396, depth396
					if buffer[position] != rune('_') {
						goto l394
					}
					position++
				}
			l396:
			l399:
				{
					position400, tokenIndex400, depth400 := position, tokenIndex, depth
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
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l404
						}
						position++
						goto l401
					l404:
						position, tokenIndex, depth = position401, tokenIndex401, depth401
						if buffer[position] != rune('_') {
							goto l400
						}
						position++
					}
				l401:
					goto l399
				l400:
					position, tokenIndex, depth = position400, tokenIndex400, depth400
				}
				depth--
				add(ruleTagComponent, position395)
			}
			return true
		l394:
			position, tokenIndex, depth = position394, tokenIndex394, depth394
			return false
		},
		/* 96 FollowUpRef <- <PathComponent*> */
		func() bool {
			{
				position406 := position
				depth++
			l407:
				{
					position408, tokenIndex408, depth408 := position, tokenIndex, depth
					if !_rules[rulePathComponent]() {
						goto l408
					}
					goto l407
				l408:
					position, tokenIndex, depth = position408, tokenIndex408, depth408
				}
				depth--
				add(ruleFollowUpRef, position406)
			}
			return true
		},
		/* 97 PathComponent <- <(('.' Key) / ('.'? Index))> */
		func() bool {
			position409, tokenIndex409, depth409 := position, tokenIndex, depth
			{
				position410 := position
				depth++
				{
					position411, tokenIndex411, depth411 := position, tokenIndex, depth
					if buffer[position] != rune('.') {
						goto l412
					}
					position++
					if !_rules[ruleKey]() {
						goto l412
					}
					goto l411
				l412:
					position, tokenIndex, depth = position411, tokenIndex411, depth411
					{
						position413, tokenIndex413, depth413 := position, tokenIndex, depth
						if buffer[position] != rune('.') {
							goto l413
						}
						position++
						goto l414
					l413:
						position, tokenIndex, depth = position413, tokenIndex413, depth413
					}
				l414:
					if !_rules[ruleIndex]() {
						goto l409
					}
				}
			l411:
				depth--
				add(rulePathComponent, position410)
			}
			return true
		l409:
			position, tokenIndex, depth = position409, tokenIndex409, depth409
			return false
		},
		/* 98 Key <- <(([a-z] / [A-Z] / [0-9] / '_') ([a-z] / [A-Z] / [0-9] / '_' / '-')* (':' ([a-z] / [A-Z] / [0-9] / '_') ([a-z] / [A-Z] / [0-9] / '_' / '-')*)?)> */
		func() bool {
			position415, tokenIndex415, depth415 := position, tokenIndex, depth
			{
				position416 := position
				depth++
				{
					position417, tokenIndex417, depth417 := position, tokenIndex, depth
					if c := buffer[position]; c < rune('a') || c > rune('z') {
						goto l418
					}
					position++
					goto l417
				l418:
					position, tokenIndex, depth = position417, tokenIndex417, depth417
					if c := buffer[position]; c < rune('A') || c > rune('Z') {
						goto l419
					}
					position++
					goto l417
				l419:
					position, tokenIndex, depth = position417, tokenIndex417, depth417
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l420
					}
					position++
					goto l417
				l420:
					position, tokenIndex, depth = position417, tokenIndex417, depth417
					if buffer[position] != rune('_') {
						goto l415
					}
					position++
				}
			l417:
			l421:
				{
					position422, tokenIndex422, depth422 := position, tokenIndex, depth
					{
						position423, tokenIndex423, depth423 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l424
						}
						position++
						goto l423
					l424:
						position, tokenIndex, depth = position423, tokenIndex423, depth423
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l425
						}
						position++
						goto l423
					l425:
						position, tokenIndex, depth = position423, tokenIndex423, depth423
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l426
						}
						position++
						goto l423
					l426:
						position, tokenIndex, depth = position423, tokenIndex423, depth423
						if buffer[position] != rune('_') {
							goto l427
						}
						position++
						goto l423
					l427:
						position, tokenIndex, depth = position423, tokenIndex423, depth423
						if buffer[position] != rune('-') {
							goto l422
						}
						position++
					}
				l423:
					goto l421
				l422:
					position, tokenIndex, depth = position422, tokenIndex422, depth422
				}
				{
					position428, tokenIndex428, depth428 := position, tokenIndex, depth
					if buffer[position] != rune(':') {
						goto l428
					}
					position++
					{
						position430, tokenIndex430, depth430 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l431
						}
						position++
						goto l430
					l431:
						position, tokenIndex, depth = position430, tokenIndex430, depth430
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l432
						}
						position++
						goto l430
					l432:
						position, tokenIndex, depth = position430, tokenIndex430, depth430
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l433
						}
						position++
						goto l430
					l433:
						position, tokenIndex, depth = position430, tokenIndex430, depth430
						if buffer[position] != rune('_') {
							goto l428
						}
						position++
					}
				l430:
				l434:
					{
						position435, tokenIndex435, depth435 := position, tokenIndex, depth
						{
							position436, tokenIndex436, depth436 := position, tokenIndex, depth
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l437
							}
							position++
							goto l436
						l437:
							position, tokenIndex, depth = position436, tokenIndex436, depth436
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l438
							}
							position++
							goto l436
						l438:
							position, tokenIndex, depth = position436, tokenIndex436, depth436
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l439
							}
							position++
							goto l436
						l439:
							position, tokenIndex, depth = position436, tokenIndex436, depth436
							if buffer[position] != rune('_') {
								goto l440
							}
							position++
							goto l436
						l440:
							position, tokenIndex, depth = position436, tokenIndex436, depth436
							if buffer[position] != rune('-') {
								goto l435
							}
							position++
						}
					l436:
						goto l434
					l435:
						position, tokenIndex, depth = position435, tokenIndex435, depth435
					}
					goto l429
				l428:
					position, tokenIndex, depth = position428, tokenIndex428, depth428
				}
			l429:
				depth--
				add(ruleKey, position416)
			}
			return true
		l415:
			position, tokenIndex, depth = position415, tokenIndex415, depth415
			return false
		},
		/* 99 Index <- <('[' '-'? [0-9]+ ']')> */
		func() bool {
			position441, tokenIndex441, depth441 := position, tokenIndex, depth
			{
				position442 := position
				depth++
				if buffer[position] != rune('[') {
					goto l441
				}
				position++
				{
					position443, tokenIndex443, depth443 := position, tokenIndex, depth
					if buffer[position] != rune('-') {
						goto l443
					}
					position++
					goto l444
				l443:
					position, tokenIndex, depth = position443, tokenIndex443, depth443
				}
			l444:
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l441
				}
				position++
			l445:
				{
					position446, tokenIndex446, depth446 := position, tokenIndex, depth
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l446
					}
					position++
					goto l445
				l446:
					position, tokenIndex, depth = position446, tokenIndex446, depth446
				}
				if buffer[position] != rune(']') {
					goto l441
				}
				position++
				depth--
				add(ruleIndex, position442)
			}
			return true
		l441:
			position, tokenIndex, depth = position441, tokenIndex441, depth441
			return false
		},
		/* 100 IP <- <([0-9]+ '.' [0-9]+ '.' [0-9]+ '.' [0-9]+)> */
		func() bool {
			position447, tokenIndex447, depth447 := position, tokenIndex, depth
			{
				position448 := position
				depth++
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l447
				}
				position++
			l449:
				{
					position450, tokenIndex450, depth450 := position, tokenIndex, depth
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l450
					}
					position++
					goto l449
				l450:
					position, tokenIndex, depth = position450, tokenIndex450, depth450
				}
				if buffer[position] != rune('.') {
					goto l447
				}
				position++
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l447
				}
				position++
			l451:
				{
					position452, tokenIndex452, depth452 := position, tokenIndex, depth
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l452
					}
					position++
					goto l451
				l452:
					position, tokenIndex, depth = position452, tokenIndex452, depth452
				}
				if buffer[position] != rune('.') {
					goto l447
				}
				position++
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l447
				}
				position++
			l453:
				{
					position454, tokenIndex454, depth454 := position, tokenIndex, depth
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l454
					}
					position++
					goto l453
				l454:
					position, tokenIndex, depth = position454, tokenIndex454, depth454
				}
				if buffer[position] != rune('.') {
					goto l447
				}
				position++
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l447
				}
				position++
			l455:
				{
					position456, tokenIndex456, depth456 := position, tokenIndex, depth
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l456
					}
					position++
					goto l455
				l456:
					position, tokenIndex, depth = position456, tokenIndex456, depth456
				}
				depth--
				add(ruleIP, position448)
			}
			return true
		l447:
			position, tokenIndex, depth = position447, tokenIndex447, depth447
			return false
		},
		/* 101 ws <- <(' ' / '\t' / '\n' / '\r')*> */
		func() bool {
			{
				position458 := position
				depth++
			l459:
				{
					position460, tokenIndex460, depth460 := position, tokenIndex, depth
					{
						position461, tokenIndex461, depth461 := position, tokenIndex, depth
						if buffer[position] != rune(' ') {
							goto l462
						}
						position++
						goto l461
					l462:
						position, tokenIndex, depth = position461, tokenIndex461, depth461
						if buffer[position] != rune('\t') {
							goto l463
						}
						position++
						goto l461
					l463:
						position, tokenIndex, depth = position461, tokenIndex461, depth461
						if buffer[position] != rune('\n') {
							goto l464
						}
						position++
						goto l461
					l464:
						position, tokenIndex, depth = position461, tokenIndex461, depth461
						if buffer[position] != rune('\r') {
							goto l460
						}
						position++
					}
				l461:
					goto l459
				l460:
					position, tokenIndex, depth = position460, tokenIndex460, depth460
				}
				depth--
				add(rulews, position458)
			}
			return true
		},
		/* 102 req_ws <- <(' ' / '\t' / '\n' / '\r')+> */
		func() bool {
			position465, tokenIndex465, depth465 := position, tokenIndex, depth
			{
				position466 := position
				depth++
				{
					position469, tokenIndex469, depth469 := position, tokenIndex, depth
					if buffer[position] != rune(' ') {
						goto l470
					}
					position++
					goto l469
				l470:
					position, tokenIndex, depth = position469, tokenIndex469, depth469
					if buffer[position] != rune('\t') {
						goto l471
					}
					position++
					goto l469
				l471:
					position, tokenIndex, depth = position469, tokenIndex469, depth469
					if buffer[position] != rune('\n') {
						goto l472
					}
					position++
					goto l469
				l472:
					position, tokenIndex, depth = position469, tokenIndex469, depth469
					if buffer[position] != rune('\r') {
						goto l465
					}
					position++
				}
			l469:
			l467:
				{
					position468, tokenIndex468, depth468 := position, tokenIndex, depth
					{
						position473, tokenIndex473, depth473 := position, tokenIndex, depth
						if buffer[position] != rune(' ') {
							goto l474
						}
						position++
						goto l473
					l474:
						position, tokenIndex, depth = position473, tokenIndex473, depth473
						if buffer[position] != rune('\t') {
							goto l475
						}
						position++
						goto l473
					l475:
						position, tokenIndex, depth = position473, tokenIndex473, depth473
						if buffer[position] != rune('\n') {
							goto l476
						}
						position++
						goto l473
					l476:
						position, tokenIndex, depth = position473, tokenIndex473, depth473
						if buffer[position] != rune('\r') {
							goto l468
						}
						position++
					}
				l473:
					goto l467
				l468:
					position, tokenIndex, depth = position468, tokenIndex468, depth468
				}
				depth--
				add(rulereq_ws, position466)
			}
			return true
		l465:
			position, tokenIndex, depth = position465, tokenIndex465, depth465
			return false
		},
		/* 104 Action0 <- <{}> */
		func() bool {
			{
				add(ruleAction0, position)
			}
			return true
		},
		/* 105 Action1 <- <{}> */
		func() bool {
			{
				add(ruleAction1, position)
			}
			return true
		},
		/* 106 Action2 <- <{}> */
		func() bool {
			{
				add(ruleAction2, position)
			}
			return true
		},
	}
	p.rules = _rules
}
