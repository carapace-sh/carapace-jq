package jq

import (
	"slices"
	"unicode/utf8"
)

// ParseForCompletion parses a partial jq filter expression and returns a
// CompletionContext describing what is expected at the end of the input.
// Partial expressions are allowed — the parser recovers from errors to
// report what tokens would be valid at the cursor position.
func ParseForCompletion(input string) *CompletionContext {
	cursor := len(input)
	p := &compParser{
		input:  input,
		pos:    0,
		cursor: cursor,
		ctx:    &CompletionContext{},
	}
	p.skipWS()
	p.parseQuery()
	if len(p.ctx.ExpectedTokens) == 0 {
		p.ctx.ExpectedTokens = append(p.ctx.ExpectedTokens, ExpectedExpression)
	}
	p.ctx.ExpectedTokens = dedupTokens(p.ctx.ExpectedTokens)
	p.ctx.ValidOperators = dedupOperators(p.ctx.ValidOperators)
	return p.ctx
}

type compParser struct {
	input  string
	pos    int
	cursor int
	ctx    *CompletionContext

	// consumed is true when at least one token was consumed before reaching cursor
	consumed bool

	// exprStarted is true when we've started parsing a new expression at the cursor
	// (set by beforeExpression, prevents afterExpression from overriding)
	exprStarted bool

	// funcStack tracks nested function calls
	funcStack []*funcState

	// objStack tracks nested object constructions
	objStack []*objState

	// reduceStack tracks nested reduce/foreach
	reduceStack []*reduceState

	// parenDepth tracks nested parentheses
	parenDepth int

	// lastExpr is the last parsed expression
	lastExpr *Expression
}

type funcState struct {
	name     string
	args     []*Expression
	argIndex int
}

type objState struct {
	inKey   bool
	inValue bool
	keyName string
}

type reduceState struct {
	isForeach bool
	section   string
}

// --- Cursor-bounded methods ---

func (p *compParser) atCursorOrEnd() bool {
	return p.pos >= len(p.input) || p.pos >= p.cursor
}

func (p *compParser) peek() rune {
	if p.pos >= len(p.input) || p.pos >= p.cursor {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(p.input[p.pos:])
	return r
}

func (p *compParser) advance() rune {
	if p.pos >= len(p.input) || p.pos >= p.cursor {
		return 0
	}
	r, w := utf8.DecodeRuneInString(p.input[p.pos:])
	p.pos += w
	p.consumed = true
	return r
}

func (p *compParser) skipWS() {
	for p.pos < len(p.input) && p.pos < p.cursor {
		r, w := utf8.DecodeRuneInString(p.input[p.pos:])
		if !isWhitespace(r) && r != '#' {
			break
		}
		if r == '#' {
			for p.pos < len(p.input) && p.pos < p.cursor {
				r2, w2 := utf8.DecodeRuneInString(p.input[p.pos:])
				if r2 == '\n' {
					break
				}
				p.pos += w2
			}
			continue
		}
		p.pos += w
	}
}

func (p *compParser) matchString(s string) bool {
	end := p.pos + len(s)
	if end > len(p.input) || end > p.cursor {
		return false
	}
	return p.input[p.pos:end] == s
}

func (p *compParser) matchKeyword(kw string) bool {
	if !p.matchString(kw) {
		return false
	}
	end := p.pos + len(kw)
	if end >= p.cursor {
		return true
	}
	next := p.input[end]
	return !isIdentifierPart(rune(next))
}

func (p *compParser) addExpected(t ExpectedToken) {
	p.ctx.ExpectedTokens = append(p.ctx.ExpectedTokens, t)
}

func (p *compParser) addOperator(op, desc string) {
	p.ctx.ValidOperators = append(p.ctx.ValidOperators, ValidOperator{Op: op, Description: desc})
}

// afterExpression adds tokens valid after a complete expression.
func (p *compParser) afterExpression() {
	if p.ctx.StringQuote != 0 {
		return
	}
	p.addExpected(ExpectedOperator)
	p.addOperator("|", "pipe")
	p.addOperator(",", "comma")
	p.addOperator("//", "alternative")
	p.addOperator("=", "assign")
	p.addOperator("|=", "update assign")
	p.addOperator("+=", "add assign")
	p.addOperator("-=", "sub assign")
	p.addOperator("*=", "mul assign")
	p.addOperator("/=", "div assign")
	p.addOperator("%=", "mod assign")
	p.addOperator("//=", "alt assign")
	p.addOperator("==", "equal")
	p.addOperator("!=", "not equal")
	p.addOperator("<", "less than")
	p.addOperator(">", "greater than")
	p.addOperator("<=", "less or equal")
	p.addOperator(">=", "greater or equal")
	p.addOperator("+", "add")
	p.addOperator("-", "subtract")
	p.addOperator("*", "multiply")
	p.addOperator("/", "divide")
	p.addOperator("%", "modulo")

	if len(p.funcStack) > 0 {
		p.addExpected(ExpectedClosingParen)
		p.addExpected(ExpectedComma)
	}
	if p.parenDepth > 0 {
		p.addExpected(ExpectedClosingParen)
	}
	if len(p.objStack) > 0 {
		top := p.objStack[len(p.objStack)-1]
		if top.inValue {
			p.addExpected(ExpectedComma)
			p.addExpected(ExpectedClosingBrace)
		}
	}
	if len(p.reduceStack) > 0 {
		p.addExpected(ExpectedSemicolon)
		p.addExpected(ExpectedClosingParen)
	}
}

func (p *compParser) beforeExpression() {
	p.addExpected(ExpectedExpression)
	p.exprStarted = true
}

// --- Parsing methods (mirror parser.go but tolerant) ---

func (p *compParser) parseQuery() {
	p.skipWS()
	if p.atCursorOrEnd() {
		if !p.consumed {
			p.beforeExpression()
		}
		return
	}

	// def
	if p.matchKeyword("def") {
		p.parseDef()
		return
	}

	// label
	if p.matchKeyword("label") {
		p.parseLabel()
		return
	}

	// Parse pipe-level expression
	p.parsePipeLevel()

	// Check for as-binding
	p.skipWS()
	if p.matchKeyword("as") {
		p.parseAsBinding()
	}
}

func (p *compParser) parseDef() {
	p.ctx.InDef = true
	p.pos += 3
	p.skipWS()
	if p.atCursorOrEnd() {
		p.ctx.DefName = ""
		return
	}
	name, ok := p.scanIdent()
	if !ok {
		p.ctx.DefName = p.partialIdent()
		return
	}
	p.ctx.DefName = name
	p.skipWS()
	if p.atCursorOrEnd() {
		p.addExpected(ExpectedDefColon)
		return
	}
	// Args
	if p.peek() == '(' {
		p.advance()
		p.parseDefArgs()
	}
	p.skipWS()
	if p.atCursorOrEnd() {
		p.addExpected(ExpectedDefColon)
		return
	}
	if p.peek() == ':' {
		p.advance()
	} else {
		p.addExpected(ExpectedDefColon)
		return
	}
	p.skipWS()
	if p.atCursorOrEnd() {
		p.beforeExpression()
		return
	}
	p.parseQuery()
	p.skipWS()
	if p.atCursorOrEnd() {
		p.addExpected(ExpectedDefSemicolon)
		return
	}
	if p.peek() == ';' {
		p.advance()
	}
	p.skipWS()
	if !p.atCursorOrEnd() {
		p.parseQuery()
	}
	p.ctx.InDef = false
}

func (p *compParser) parseDefArgs() {
	for {
		p.skipWS()
		if p.atCursorOrEnd() {
			p.addExpected(ExpectedDollar)
			return
		}
		if p.peek() == ')' {
			p.advance()
			return
		}
		if p.peek() == '$' {
			p.advance()
		}
		p.skipWS()
		if p.atCursorOrEnd() {
			return
		}
		p.scanIdent()
		p.skipWS()
		if p.atCursorOrEnd() {
			p.addExpected(ExpectedComma)
			p.addExpected(ExpectedClosingParen)
			return
		}
		if p.peek() == ',' || p.peek() == ';' {
			p.advance()
			continue
		}
		if p.peek() == ')' {
			p.advance()
			return
		}
	}
}

func (p *compParser) parseLabel() {
	p.pos += 5
	p.skipWS()
	if p.atCursorOrEnd() {
		p.addExpected(ExpectedDollar)
		return
	}
	if p.peek() == '$' {
		p.advance()
	}
	p.skipWS()
	if p.atCursorOrEnd() {
		return
	}
	p.scanIdent()
	p.skipWS()
	if p.atCursorOrEnd() {
		p.addExpected(ExpectedPipe)
		return
	}
	if p.peek() == '|' {
		p.advance()
	}
	p.skipWS()
	if !p.atCursorOrEnd() {
		p.parseQuery()
	}
}

func (p *compParser) parseAsBinding() {
	p.pos += 2 // consume "as"
	p.skipWS()
	p.ctx.InAsPattern = true
	p.parsePattern()
	p.ctx.InAsPattern = false
	p.skipWS()
	if p.atCursorOrEnd() {
		p.addExpected(ExpectedPipe)
		return
	}
	if p.peek() == '|' {
		p.advance()
	}
	p.skipWS()
	if !p.atCursorOrEnd() {
		p.parseQuery()
	}
}

func (p *compParser) parsePattern() {
	p.skipWS()
	if p.atCursorOrEnd() {
		p.addExpected(ExpectedDollar)
		return
	}
	p.parsePatternSingle()
	// Check for ?//
	p.skipWS()
	for p.matchString("?//") {
		p.pos += 3
		p.skipWS()
		p.parsePatternSingle()
	}
}

func (p *compParser) parsePatternSingle() {
	if p.peek() == '$' {
		p.advance()
		p.skipWS()
		if !p.atCursorOrEnd() {
			p.scanIdent()
		}
		return
	}
	if p.peek() == '[' {
		p.advance()
		for {
			p.skipWS()
			if p.atCursorOrEnd() {
				p.addExpected(ExpectedClosingBracket)
				return
			}
			if p.peek() == ']' {
				p.advance()
				return
			}
			p.parsePatternSingle()
			p.skipWS()
			if p.atCursorOrEnd() {
				p.addExpected(ExpectedComma)
				p.addExpected(ExpectedClosingBracket)
				return
			}
			if p.peek() == ',' {
				p.advance()
				continue
			}
			if p.peek() == ']' {
				p.advance()
				return
			}
		}
	}
	if p.peek() == '{' {
		p.advance()
		for {
			p.skipWS()
			if p.atCursorOrEnd() {
				p.addExpected(ExpectedClosingBrace)
				return
			}
			if p.peek() == '}' {
				p.advance()
				return
			}
			if p.peek() == '$' {
				p.advance()
				p.scanIdent()
			} else {
				p.scanIdent()
				p.skipWS()
				if !p.atCursorOrEnd() && p.peek() == ':' {
					p.advance()
					p.parsePatternSingle()
				}
			}
			p.skipWS()
			if p.atCursorOrEnd() {
				p.addExpected(ExpectedComma)
				p.addExpected(ExpectedClosingBrace)
				return
			}
			if p.peek() == ',' {
				p.advance()
				continue
			}
			if p.peek() == '}' {
				p.advance()
				return
			}
		}
	}
}

// parsePipeLevel handles pipe and comma operators
func (p *compParser) parsePipeLevel() {
	p.parseExpLevel()
	for {
		p.skipWS()
		if p.atCursorOrEnd() {
			break
		}
		// Check for pipe
		if p.peek() == '|' && (p.pos+1 >= p.cursor || p.input[p.pos+1] != '=') {
			p.advance()
			p.skipWS()
			if p.atCursorOrEnd() {
				p.beforeExpression()
				return
			}
			p.parseExpLevel()
			continue
		}
		// Check for comma
		if p.peek() == ',' {
			p.advance()
			p.skipWS()
			if p.atCursorOrEnd() {
				p.beforeExpression()
				return
			}
			p.parseExpLevel()
			continue
		}
		break
	}
}

// parseExpLevel handles everything below comma (alternative, assign, or, and, etc.)
func (p *compParser) parseExpLevel() {
	p.parsePratt(precPipe + 1) // exclude pipe and comma, handle them in parsePipeLevel
}

// parsePratt handles the Pratt parser levels
func (p *compParser) parsePratt(minPrec int) {
	p.skipWS()
	p.exprStarted = false
	if p.atCursorOrEnd() {
		if !p.consumed {
			p.beforeExpression()
		}
		return
	}
	p.parsePrefix()
	for {
		p.skipWS()
		if p.atCursorOrEnd() {
			break
		}
		op, prec, _ := p.peekInfixOp()
		if op == "" || prec < minPrec {
			break
		}
		p.consumeInfixOp(op)
		p.skipWS()
		if p.atCursorOrEnd() {
			p.beforeExpression()
			return
		}
		p.parsePratt(prec + 1)
	}
	// After complete expression, add what's valid
	p.skipWS()
	if p.atCursorOrEnd() && p.consumed && !p.exprStarted {
		p.afterExpression()
	}
}

func (p *compParser) peekInfixOp() (op string, prec int, rightAssoc bool) {
	if p.matchString("//=") {
		return "//=", precAssign, false
	}
	if p.matchString("//") {
		return "//", precAlternative, true
	}
	if p.matchString("|=") {
		return "|=", precAssign, false
	}
	if p.matchString("==") {
		return "==", precCompare, false
	}
	if p.matchString("!=") {
		return "!=", precCompare, false
	}
	if p.matchString(">=") {
		return ">=", precCompare, false
	}
	if p.matchString("<=") {
		return "<=", precCompare, false
	}
	if p.matchString("+=") {
		return "+=", precAssign, false
	}
	if p.matchString("-=") {
		return "-=", precAssign, false
	}
	if p.matchString("*=") {
		return "*=", precAssign, false
	}
	if p.matchString("/=") {
		return "/=", precAssign, false
	}
	if p.matchString("%=") {
		return "%=", precAssign, false
	}
	ch := p.peek()
	switch ch {
	case '=':
		return "=", precAssign, false
	case '+':
		return "+", precAddSub, false
	case '-':
		return "-", precAddSub, false
	case '*':
		return "*", precMulDiv, false
	case '/':
		return "/", precMulDiv, false
	case '%':
		return "%", precMulDiv, false
	case '<':
		return "<", precCompare, false
	case '>':
		return ">", precCompare, false
	}
	if p.matchKeyword("and") {
		return "and", precAnd, false
	}
	if p.matchKeyword("or") {
		return "or", precOr, false
	}
	return "", 0, false
}

func (p *compParser) consumeInfixOp(op string) {
	p.pos += len(op)
	p.consumed = true
}

func (p *compParser) parsePrefix() {
	p.skipWS()
	if p.atCursorOrEnd() {
		p.beforeExpression()
		return
	}
	if p.peek() == '-' {
		p.advance()
		p.skipWS()
		if p.atCursorOrEnd() {
			p.beforeExpression()
			return
		}
		p.parsePratt(precPrefix)
		return
	}
	p.parsePostfixChain()
}

func (p *compParser) parsePostfixChain() {
	p.parsePrimary()
	p.parsePostfix()
}

func (p *compParser) parsePostfix() {
	for {
		if p.atCursorOrEnd() {
			break
		}
		if p.peek() == '?' {
			p.advance()
			continue
		}
		if p.peek() == '.' {
			if p.pos+1 < p.cursor && p.input[p.pos+1] == '.' {
				break
			}
			p.advance()
			p.ctx.AfterDot = true
			p.parseFieldAccess()
			continue
		}
		if p.peek() == '[' {
			p.advance()
			p.parseBracketAccess()
			continue
		}
		break
	}
}

func (p *compParser) parseFieldAccess() {
	p.skipWS()
	if p.atCursorOrEnd() {
		p.addExpected(ExpectedExpression)
		return
	}
	// Identifier field
	if name, ok := p.scanIdent(); ok {
		p.ctx.PartialIdent = name
		p.ctx.AfterDot = false
		return
	}
	// Quoted field
	if p.peek() == '"' {
		p.parseStringLiteral()
		return
	}
	// Just a dot with nothing after — offer field names
	p.addExpected(ExpectedExpression)
}

func (p *compParser) parseBracketAccess() {
	p.skipWS()
	if p.atCursorOrEnd() {
		p.addExpected(ExpectedClosingBracket)
		p.beforeExpression()
		return
	}
	// .[] iterator
	if p.peek() == ']' {
		p.advance()
		return
	}
	// .[expr] or .[start:end]
	if p.peek() != ':' {
		p.parseExpLevel()
	}
	p.skipWS()
	if p.atCursorOrEnd() {
		p.addExpected(ExpectedColon)
		p.addExpected(ExpectedClosingBracket)
		return
	}
	if p.peek() == ':' {
		p.advance()
		p.skipWS()
		if p.atCursorOrEnd() {
			p.addExpected(ExpectedClosingBracket)
			return
		}
		if p.peek() != ']' {
			p.parseExpLevel()
		}
	}
	p.skipWS()
	if p.atCursorOrEnd() {
		p.addExpected(ExpectedClosingBracket)
		return
	}
	if p.peek() == ']' {
		p.advance()
	}
}

func (p *compParser) parsePrimary() {
	p.skipWS()
	if p.atCursorOrEnd() {
		p.beforeExpression()
		return
	}

	ch := p.peek()

	// . identity/field/recursive descent
	if ch == '.' {
		if p.pos+1 < p.cursor && p.input[p.pos+1] == '.' {
			p.advance()
			p.advance()
			return
		}
		p.advance()
		p.ctx.AfterDot = true
		p.skipWS()
		if p.atCursorOrEnd() {
			p.addExpected(ExpectedExpression)
			return
		}
		if name, ok := p.scanIdent(); ok {
			p.ctx.PartialIdent = name
			p.ctx.AfterDot = false
			return
		}
		if p.peek() == '"' {
			p.parseStringLiteral()
			return
		}
		if p.peek() == '[' {
			p.advance()
			p.parseBracketAccess()
			return
		}
		p.ctx.AfterDot = false
		return
	}

	// ( expr )
	if ch == '(' {
		p.advance()
		p.parenDepth++
		p.skipWS()
		if p.atCursorOrEnd() {
			p.beforeExpression()
			p.addExpected(ExpectedClosingParen)
			return
		}
		p.parseQuery()
		p.skipWS()
		if p.atCursorOrEnd() {
			p.addExpected(ExpectedClosingParen)
			return
		}
		if p.peek() == ')' {
			p.advance()
		}
		p.parenDepth--
		return
	}

	// [ array ]
	if ch == '[' {
		p.advance()
		p.skipWS()
		if p.atCursorOrEnd() {
			p.beforeExpression()
			p.addExpected(ExpectedClosingBracket)
			return
		}
		if p.peek() == ']' {
			p.advance()
			return
		}
		p.parseExpLevel()
		p.skipWS()
		for !p.atCursorOrEnd() && p.peek() == ',' {
			p.advance()
			p.skipWS()
			if p.atCursorOrEnd() {
				p.beforeExpression()
				p.addExpected(ExpectedClosingBracket)
				return
			}
			p.parseExpLevel()
			p.skipWS()
		}
		if !p.atCursorOrEnd() && p.peek() == ']' {
			p.advance()
		}
		return
	}

	// { object }
	if ch == '{' {
		p.advance()
		p.objStack = append(p.objStack, &objState{inKey: true})
		p.parseObjectEntries()
		if len(p.objStack) > 0 {
			p.objStack = p.objStack[:len(p.objStack)-1]
		}
		return
	}

	// " string "
	if ch == '"' {
		p.parseStringLiteral()
		return
	}

	// @ format
	if ch == '@' {
		p.parseFormat()
		return
	}

	// $ variable
	if ch == '$' {
		p.advance()
		p.skipWS()
		if !p.atCursorOrEnd() {
			p.scanIdent()
		}
		return
	}

	// Numbers
	if isDigit(ch) {
		p.scanNumber()
		return
	}

	// Keywords and identifiers
	if isIdentifierStart(ch) {
		p.parseKeywordOrFunction()
		return
	}

	// Unknown character — just offer expression
	p.beforeExpression()
}

func (p *compParser) parseObjectEntries() {
	for {
		p.skipWS()
		if p.atCursorOrEnd() {
			top := p.currentObj()
			if top != nil {
				p.ctx.Object = &ObjectContext{
					InKey:   top.inKey,
					InValue: top.inValue,
					KeyName: top.keyName,
				}
			}
			p.addExpected(ExpectedClosingBrace)
			return
		}
		if p.peek() == '}' {
			p.advance()
			return
		}
		p.parseObjectEntry()
		p.skipWS()
		if p.atCursorOrEnd() {
			p.addExpected(ExpectedComma)
			p.addExpected(ExpectedClosingBrace)
			return
		}
		if p.peek() == ',' {
			p.advance()
			continue
		}
		if p.peek() == '}' {
			p.advance()
			return
		}
	}
}

func (p *compParser) parseObjectEntry() {
	top := p.currentObj()
	if top != nil {
		top.inKey = true
		top.inValue = false
	}
	p.skipWS()
	if p.atCursorOrEnd() {
		p.ctx.Object = &ObjectContext{InKey: true}
		return
	}

	// {$var} or {$var: value}
	if p.peek() == '$' {
		p.advance()
		name, _ := p.scanIdent()
		if top != nil {
			top.keyName = name
		}
		p.skipWS()
		if p.atCursorOrEnd() {
			p.addExpected(ExpectedColon)
			p.addExpected(ExpectedComma)
			p.addExpected(ExpectedClosingBrace)
			return
		}
		if p.peek() == ':' {
			p.advance()
			if top != nil {
				top.inKey = false
				top.inValue = true
			}
			p.skipWS()
			if p.atCursorOrEnd() {
				p.ctx.Object = &ObjectContext{InValue: true, KeyName: name}
				p.beforeExpression()
				return
			}
			p.parseExpLevel()
			if top != nil {
				top.inValue = false
			}
		}
		return
	}

	// {(expr): value}
	if p.peek() == '(' {
		p.advance()
		p.parenDepth++
		p.skipWS()
		if !p.atCursorOrEnd() {
			p.parseQuery()
		}
		p.skipWS()
		if !p.atCursorOrEnd() && p.peek() == ')' {
			p.advance()
		}
		p.parenDepth--
		p.skipWS()
		if !p.atCursorOrEnd() && p.peek() == ':' {
			p.advance()
		}
		p.skipWS()
		if p.atCursorOrEnd() {
			if top != nil {
				top.inKey = false
				top.inValue = true
			}
			p.ctx.Object = &ObjectContext{InValue: true}
			p.beforeExpression()
			return
		}
		p.parseExpLevel()
		return
	}

	// {"string": value}
	if p.peek() == '"' {
		p.parseStringLiteral()
		p.skipWS()
		if !p.atCursorOrEnd() && p.peek() == ':' {
			p.advance()
		}
		p.skipWS()
		if p.atCursorOrEnd() {
			if top != nil {
				top.inKey = false
				top.inValue = true
			}
			p.ctx.Object = &ObjectContext{InValue: true}
			p.beforeExpression()
			return
		}
		p.parseExpLevel()
		return
	}

	// {foo: value} or {foo}
	name, ok := p.scanIdent()
	if top != nil {
		top.keyName = name
	}
	p.skipWS()
	if p.atCursorOrEnd() {
		// Could be shorthand {foo} or {foo: value}
		p.ctx.Object = &ObjectContext{InKey: false, KeyName: name}
		p.addExpected(ExpectedColon)
		p.addExpected(ExpectedComma)
		p.addExpected(ExpectedClosingBrace)
		return
	}
	_ = ok
	if p.peek() == ':' {
		p.advance()
		if top != nil {
			top.inKey = false
			top.inValue = true
		}
		p.skipWS()
		if p.atCursorOrEnd() {
			p.ctx.Object = &ObjectContext{InValue: true, KeyName: name}
			p.beforeExpression()
			return
		}
		p.parseExpLevel()
		if top != nil {
			top.inValue = false
		}
	}
}

func (p *compParser) currentObj() *objState {
	if len(p.objStack) == 0 {
		return nil
	}
	return p.objStack[len(p.objStack)-1]
}

func (p *compParser) parseStringLiteral() {
	if p.peek() != '"' {
		return
	}
	p.advance() // consume opening "
	p.ctx.StringQuote = '"'

	for {
		if p.atCursorOrEnd() {
			// Partial string — capture content
			p.ctx.PartialString = p.partialStringContent()
			p.addExpected(ExpectedStringClose)
			return
		}
		ch := p.peek()
		if ch == '"' {
			p.advance()
			p.ctx.StringQuote = 0
			return
		}
		if ch == '\\' {
			p.advance()
			if p.atCursorOrEnd() {
				p.ctx.PartialString = p.partialStringContent()
				p.addExpected(ExpectedStringClose)
				return
			}
			next := p.peek()
			if next == '(' {
				// String interpolation
				p.ctx.InStringInterp = true
				p.advance() // consume (
				p.skipWS()
				if p.atCursorOrEnd() {
					p.beforeExpression()
					return
				}
				p.parseExpLevel()
				p.skipWS()
				if !p.atCursorOrEnd() && p.peek() == ')' {
					p.advance()
				}
				p.ctx.InStringInterp = false
				continue
			}
			p.advance() // consume escaped char
			continue
		}
		p.advance()
	}
}

func (p *compParser) partialStringContent() string {
	// Find the opening quote and extract content between it and cursor
	start := p.pos
	// Search backwards for opening quote
	for i := p.pos - 1; i >= 0; i-- {
		if p.input[i] == '"' && (i == 0 || p.input[i-1] != '\\') {
			return p.input[i+1 : start]
		}
	}
	return p.input[:start]
}

func (p *compParser) parseFormat() {
	p.ctx.InFormat = true
	p.advance() // consume @
	p.skipWS()
	if p.atCursorOrEnd() {
		p.ctx.PartialFormat = p.partialIdent()
		p.addExpected(ExpectedFormatName)
		return
	}
	name, ok := p.scanIdent()
	if ok {
		p.ctx.InFormat = false
		_ = name
	}
	p.skipWS()
	if p.atCursorOrEnd() {
		// Could be followed by a string
		p.addExpected(ExpectedExpression)
		return
	}
	if p.peek() == '"' {
		p.parseStringLiteral()
	}
	p.ctx.InFormat = false
}

func (p *compParser) parseKeywordOrFunction() {
	name, ok := p.scanIdent()
	if !ok {
		p.ctx.PartialIdent = p.partialIdent()
		p.beforeExpression()
		return
	}

	switch name {
	case "if":
		p.parseIf()
		return
	case "try":
		p.parseTry()
		return
	case "reduce":
		p.parseReduceForeach(true)
		return
	case "foreach":
		p.parseReduceForeach(false)
		return
	case "break":
		p.skipWS()
		if p.atCursorOrEnd() {
			p.addExpected(ExpectedDollar)
			return
		}
		if p.peek() == '$' {
			p.advance()
			p.scanIdent()
		}
		return
	case "true", "false", "null":
		return
	case "then", "else", "elif", "end", "catch", "as":
		// These are keywords that should have been consumed by their
		// respective constructs. If we encounter them here, backtrack
		// and let the caller handle them.
		p.pos -= len(name)
		p.consumed = true // still consumed something before
		return
	}

	// Function call
	p.skipWS()
	if p.atCursorOrEnd() {
		p.ctx.PartialIdent = name
		p.beforeExpression()
		p.addExpected(ExpectedClosingParen)
		return
	}
	if p.peek() == '(' {
		p.advance()
		p.funcStack = append(p.funcStack, &funcState{name: name, argIndex: 0})
		p.parseFunctionArgs()
		if len(p.funcStack) > 0 {
			fs := p.funcStack[len(p.funcStack)-1]
			p.ctx.Function = &FunctionContext{
				Name:     fs.name,
				Args:     fs.args,
				ArgIndex: fs.argIndex,
			}
			p.funcStack = p.funcStack[:len(p.funcStack)-1]
		}
		return
	}
	// 0-arg function
	p.ctx.PartialIdent = name
}

func (p *compParser) parseFunctionArgs() {
	for {
		p.skipWS()
		if p.atCursorOrEnd() {
			p.beforeExpression()
			p.addExpected(ExpectedClosingParen)
			return
		}
		if p.peek() == ')' {
			p.advance()
			return
		}
		p.parseExpLevel()
		p.skipWS()
		if p.atCursorOrEnd() {
			p.addExpected(ExpectedComma)
			p.addExpected(ExpectedClosingParen)
			return
		}
		if p.peek() == ',' || p.peek() == ';' {
			p.advance()
			if len(p.funcStack) > 0 {
				p.funcStack[len(p.funcStack)-1].argIndex++
			}
			continue
		}
		if p.peek() == ')' {
			p.advance()
			return
		}
	}
}

func (p *compParser) parseIf() {
	p.skipWS()
	if p.atCursorOrEnd() {
		p.beforeExpression()
		return
	}
	p.parseExpLevel() // condition
	p.skipWS()
	if p.atCursorOrEnd() {
		p.addExpected(ExpectedKeyword) // then
		return
	}
	if !p.matchKeyword("then") {
		return
	}
	p.pos += 4
	p.skipWS()
	if p.atCursorOrEnd() {
		p.beforeExpression()
		return
	}
	p.parseExpLevel() // then body
	for {
		p.skipWS()
		if p.atCursorOrEnd() {
			p.addExpected(ExpectedKeyword) // elif, else, end
			return
		}
		if p.matchKeyword("elif") {
			p.pos += 4
			p.skipWS()
			if p.atCursorOrEnd() {
				p.beforeExpression()
				return
			}
			p.parseExpLevel()
			p.skipWS()
			if p.atCursorOrEnd() {
				p.addExpected(ExpectedKeyword) // then
				return
			}
			if p.matchKeyword("then") {
				p.pos += 4
			}
			p.skipWS()
			if p.atCursorOrEnd() {
				p.beforeExpression()
				return
			}
			p.parseExpLevel()
			continue
		}
		break
	}
	p.skipWS()
	if p.atCursorOrEnd() {
		p.addExpected(ExpectedKeyword) // else, end
		return
	}
	if p.matchKeyword("else") {
		p.pos += 4
		p.skipWS()
		if p.atCursorOrEnd() {
			p.beforeExpression()
			return
		}
		p.parseExpLevel()
	}
	p.skipWS()
	if p.atCursorOrEnd() {
		p.addExpected(ExpectedKeyword) // end
		return
	}
	if p.matchKeyword("end") {
		p.pos += 3
	}
}

func (p *compParser) parseTry() {
	p.skipWS()
	if p.atCursorOrEnd() {
		p.beforeExpression()
		return
	}
	p.parsePratt(precPostfix)
	p.parsePostfix()
	p.skipWS()
	if p.atCursorOrEnd() {
		p.addExpected(ExpectedKeyword) // catch
		p.afterExpression()
		return
	}
	if p.matchKeyword("catch") {
		p.pos += 5
		p.skipWS()
		if p.atCursorOrEnd() {
			p.beforeExpression()
			return
		}
		p.parsePratt(precPostfix)
		p.parsePostfix()
	}
}

func (p *compParser) parseReduceForeach(isReduce bool) {
	p.skipWS()
	if p.atCursorOrEnd() {
		p.beforeExpression()
		return
	}
	p.parsePratt(precPipe) // source (no comma, no pipe)
	p.skipWS()
	if p.atCursorOrEnd() {
		p.addExpected(ExpectedKeyword) // as
		return
	}
	if p.matchKeyword("as") {
		p.pos += 2
	}
	p.skipWS()
	if p.atCursorOrEnd() {
		p.ctx.InAsPattern = true
		p.addExpected(ExpectedDollar)
		return
	}
	p.ctx.InAsPattern = true
	p.parsePattern()
	p.ctx.InAsPattern = false
	p.skipWS()
	if p.atCursorOrEnd() {
		p.addExpected(ExpectedClosingParen)
		return
	}
	if p.peek() == '(' {
		p.advance()
	}
	p.reduceStack = append(p.reduceStack, &reduceState{isForeach: !isReduce, section: "init"})
	// init
	p.skipWS()
	if p.atCursorOrEnd() {
		p.ctx.Reduce = &ReduceContext{IsForeach: !isReduce, Section: "init"}
		p.beforeExpression()
		p.addExpected(ExpectedSemicolon)
		return
	}
	p.parseQuery()
	p.skipWS()
	if p.atCursorOrEnd() {
		p.ctx.Reduce = &ReduceContext{IsForeach: !isReduce, Section: "init"}
		p.addExpected(ExpectedSemicolon)
		return
	}
	if p.peek() == ';' {
		p.advance()
	}
	// update
	if len(p.reduceStack) > 0 {
		p.reduceStack[len(p.reduceStack)-1].section = "update"
	}
	p.skipWS()
	if p.atCursorOrEnd() {
		p.ctx.Reduce = &ReduceContext{IsForeach: !isReduce, Section: "update"}
		p.beforeExpression()
		p.addExpected(ExpectedSemicolon)
		p.addExpected(ExpectedClosingParen)
		return
	}
	p.parseQuery()
	p.skipWS()
	if p.atCursorOrEnd() {
		p.ctx.Reduce = &ReduceContext{IsForeach: !isReduce, Section: "update"}
		p.addExpected(ExpectedSemicolon)
		p.addExpected(ExpectedClosingParen)
		return
	}
	// extract (foreach only)
	if !isReduce && p.peek() == ';' {
		p.advance()
		if len(p.reduceStack) > 0 {
			p.reduceStack[len(p.reduceStack)-1].section = "extract"
		}
		p.skipWS()
		if p.atCursorOrEnd() {
			p.ctx.Reduce = &ReduceContext{IsForeach: true, Section: "extract"}
			p.beforeExpression()
			p.addExpected(ExpectedClosingParen)
			return
		}
		p.parseQuery()
		p.skipWS()
	}
	if !p.atCursorOrEnd() && p.peek() == ')' {
		p.advance()
	}
	if len(p.reduceStack) > 0 {
		p.reduceStack = p.reduceStack[:len(p.reduceStack)-1]
	}
}

// --- Scanner helpers ---

func (p *compParser) scanIdent() (string, bool) {
	if p.atCursorOrEnd() {
		return "", false
	}
	if !isIdentifierStart(p.peek()) {
		return "", false
	}
	start := p.pos
	for !p.atCursorOrEnd() && isIdentifierPart(p.peek()) {
		p.advance()
	}
	return p.input[start:p.pos], true
}

func (p *compParser) scanNumber() {
	if p.atCursorOrEnd() || !isDigit(p.peek()) {
		return
	}
	for !p.atCursorOrEnd() && isDigit(p.peek()) {
		p.advance()
	}
	if !p.atCursorOrEnd() && p.peek() == '.' {
		if p.pos+1 < p.cursor && isDigit(rune(p.input[p.pos+1])) {
			p.advance()
			for !p.atCursorOrEnd() && isDigit(p.peek()) {
				p.advance()
			}
		}
	}
	if !p.atCursorOrEnd() && (p.peek() == 'e' || p.peek() == 'E') {
		p.advance()
		if !p.atCursorOrEnd() && (p.peek() == '+' || p.peek() == '-') {
			p.advance()
		}
		for !p.atCursorOrEnd() && isDigit(p.peek()) {
			p.advance()
		}
	}
}

func (p *compParser) partialIdent() string {
	if p.pos == 0 {
		return ""
	}
	// Find the start of the partial identifier
	end := p.pos
	for i := p.pos - 1; i >= 0; i-- {
		ch := rune(p.input[i])
		if i == p.pos-1 {
			if !isIdentifierPart(ch) {
				break
			}
		} else {
			if !isIdentifierPart(ch) {
				i++
				return p.input[i:end]
			}
		}
	}
	_ = end
	// If we get here, the identifier starts at position 0
	start := 0
	for i := p.pos - 1; i >= 0; i-- {
		if !isIdentifierPart(rune(p.input[i])) {
			start = i + 1
			break
		}
	}
	return p.input[start:p.pos]
}

// --- Dedup helpers ---

func dedupTokens(tokens []ExpectedToken) []ExpectedToken {
	seen := make(map[ExpectedToken]bool)
	result := make([]ExpectedToken, 0, len(tokens))
	for _, t := range tokens {
		if !seen[t] {
			seen[t] = true
			result = append(result, t)
		}
	}
	return result
}

func dedupOperators(ops []ValidOperator) []ValidOperator {
	seen := make(map[string]bool)
	result := make([]ValidOperator, 0, len(ops))
	for _, op := range ops {
		if !seen[op.Op] {
			seen[op.Op] = true
			result = append(result, op)
		}
	}
	return slices.Clip(result)
}
