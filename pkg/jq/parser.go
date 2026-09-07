package jq

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// ParseError is a syntax error with a span indicating the problematic region.
type ParseError struct {
	Message string
	Span    Span
}

func (e *ParseError) Error() string {
	return e.Message
}

// parser is the internal recursive-descent parser state.
type parser struct {
	input       string
	pos         int
	lastContent int // position right after last non-whitespace/comment content
}

// Parse parses a jq filter expression string into an AST.
func Parse(input string) (*Expression, error) {
	p := &parser{input: input}
	p.skipWhitespaceAndComments()
	start := p.pos
	expr, err := p.parseQuery()
	if err != nil {
		return nil, err
	}
	p.skipWhitespaceAndComments()
	if p.pos < len(p.input) {
		return nil, p.syntaxError("unexpected token")
	}
	contentEnd := min(p.lastContent, len(p.input))
	if expr.Span.End > contentEnd || expr.Span.Start < start {
		expr.Span = Span{Start: start, End: contentEnd}
	}
	return expr, nil
}

// IsIdentifier checks if the text is a valid jq identifier.
func IsIdentifier(text string) bool {
	if len(text) == 0 {
		return false
	}
	for i, ch := range text {
		if i == 0 {
			if !isIdentifierStart(ch) {
				return false
			}
		} else {
			if !isIdentifierPart(ch) {
				return false
			}
		}
	}
	return true
}

func (p *parser) syntaxError(msg string) *ParseError {
	return &ParseError{
		Message: msg,
		Span:    Span{Start: p.pos, End: min(p.pos+1, len(p.input))},
	}
}

func (p *parser) syntaxErrorf(format string, args ...any) *ParseError {
	return &ParseError{
		Message: fmt.Sprintf(format, args...),
		Span:    Span{Start: p.pos, End: min(p.pos+1, len(p.input))},
	}
}

func (p *parser) peek() rune {
	if p.pos >= len(p.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(p.input[p.pos:])
	return r
}

func (p *parser) advance() rune {
	if p.pos >= len(p.input) {
		return 0
	}
	r, w := utf8.DecodeRuneInString(p.input[p.pos:])
	p.pos += w
	p.lastContent = p.pos
	return r
}

func (p *parser) atEnd() bool {
	return p.pos >= len(p.input)
}

func (p *parser) skipWhitespace() {
	for p.pos < len(p.input) {
		r, w := utf8.DecodeRuneInString(p.input[p.pos:])
		if !isWhitespace(r) {
			break
		}
		p.pos += w
	}
}

func (p *parser) skipComment() bool {
	if p.pos < len(p.input) && p.input[p.pos] == '#' {
		for p.pos < len(p.input) && p.input[p.pos] != '\n' {
			p.pos++
		}
		return true
	}
	return false
}

func (p *parser) skipWhitespaceAndComments() {
	for {
		p.skipWhitespace()
		if !p.skipComment() {
			break
		}
	}
}

func (p *parser) matchString(s string) bool {
	if p.pos+len(s) > len(p.input) {
		return false
	}
	return p.input[p.pos:p.pos+len(s)] == s
}

// --- Operator precedence constants ---
// Higher = tighter binding, matching jq's parser.y precedence.
const (
	precPipe        = 1  // | (right-assoc)
	precComma       = 2  // , (left-assoc)
	precAlternative = 3  // // (right-assoc)
	precAssign      = 4  // = |= += -= *= /= %= //= (non-assoc)
	precOr          = 5  // or (left-assoc)
	precAnd         = 6  // and (left-assoc)
	precCompare     = 7  // == != < > <= >= (non-assoc)
	precAddSub      = 8  // + - (left-assoc)
	precMulDiv      = 9  // * / % (left-assoc)
	precPostfix     = 10 // ? . [ (postfix)
	precPrefix      = 11 // - (unary)
	precPrimary     = 12
)

// --- Parsing methods ---

// parseQuery handles top-level constructs that interact with |:
// def, import, include, module, as-binding, label, and the pipe level.
func (p *parser) parseQuery() (*Expression, error) {
	p.skipWhitespaceAndComments()
	start := p.pos

	// Check for def
	if p.matchKeyword("def") {
		return p.parseDef(start)
	}

	// Check for import/include/module (deferred — parse as error for now)
	if p.matchKeyword("import") || p.matchKeyword("include") || p.matchKeyword("module") {
		return nil, p.syntaxError("module system not yet supported")
	}

	// Check for label
	if p.matchKeyword("label") {
		return p.parseLabel(start)
	}

	// Parse the pipe-level expression
	expr, err := p.parsePipe()
	if err != nil {
		return nil, err
	}

	// Check for as-binding: exp as $pattern | body
	p.skipWhitespaceAndComments()
	if p.matchKeyword("as") {
		return p.parseAsBinding(expr, start)
	}

	return expr, nil
}

// matchKeyword checks if the input at the current position matches the given
// keyword followed by a word boundary (whitespace, punctuation, or EOF).
func (p *parser) matchKeyword(kw string) bool {
	if !p.matchString(kw) {
		return false
	}
	end := p.pos + len(kw)
	if end >= len(p.input) {
		return true
	}
	next := p.input[end]
	// Word boundary: not an identifier character
	return !isIdentifierPart(rune(next))
}

// parseDef parses: def name[(args)]: body; rest
func (p *parser) parseDef(start int) (*Expression, error) {
	p.pos += 3 // consume "def"
	p.skipWhitespaceAndComments()

	name, ok := p.scanIdentifier()
	if !ok {
		return nil, p.syntaxError("expected function name after 'def'")
	}

	var args []FunctionArg
	p.skipWhitespaceAndComments()
	if !p.atEnd() && p.peek() == '(' {
		p.advance() // consume (
		for {
			p.skipWhitespaceAndComments()
			if !p.atEnd() && p.peek() == ')' {
				p.advance()
				break
			}
			// Argument can be: name (filter arg) or $name (value arg)
			isValue := false
			if !p.atEnd() && p.peek() == '$' {
				isValue = true
				p.advance() // consume $
			}

			argName, ok := p.scanIdentifier()
			if !ok {
				return nil, p.syntaxError("expected argument name")
			}
			args = append(args, FunctionArg{Name: argName, IsValue: isValue})

			p.skipWhitespaceAndComments()
			if !p.atEnd() && p.peek() == ';' {
				p.advance()
				continue
			}
			p.skipWhitespaceAndComments()
			if !p.atEnd() && p.peek() == ')' {
				p.advance()
				break
			}
			return nil, p.syntaxError("expected ';' or ')' in def arguments")
		}
	}

	p.skipWhitespaceAndComments()
	if p.atEnd() || p.peek() != ':' {
		return nil, p.syntaxError("expected ':' after function name/args")
	}
	p.advance() // consume :

	p.skipWhitespaceAndComments()
	body, err := p.parseQuery()
	if err != nil {
		return nil, err
	}

	p.skipWhitespaceAndComments()
	if p.atEnd() || p.peek() != ';' {
		return nil, p.syntaxError("expected ';' after function definition body")
	}
	p.advance() // consume ;

	p.skipWhitespaceAndComments()
	var rest *Expression
	if !p.atEnd() {
		rest, err = p.parseQuery()
		if err != nil {
			return nil, err
		}
	} else {
		return nil, p.syntaxError("expected expression after 'def ...;'")
	}

	return &Expression{
		Kind:    KindDef,
		Span:    Span{Start: start, End: p.pos},
		payload: &DefExpr{Name: name, Args: args, Body: body, Rest: rest},
	}, nil
}

// parseLabel parses: label $name | body
func (p *parser) parseLabel(start int) (*Expression, error) {
	p.pos += 5 // consume "label"
	p.skipWhitespaceAndComments()
	if p.atEnd() || p.peek() != '$' {
		return nil, p.syntaxError("expected '$' after 'label'")
	}
	p.advance() // consume $
	name, ok := p.scanIdentifier()
	if !ok {
		return nil, p.syntaxError("expected label name after '$'")
	}
	p.skipWhitespaceAndComments()
	if p.atEnd() || p.peek() != '|' {
		return nil, p.syntaxError("expected '|' after label name")
	}
	p.advance() // consume |
	p.skipWhitespaceAndComments()
	body, err := p.parseQuery()
	if err != nil {
		return nil, err
	}
	return &Expression{
		Kind:    KindLabel,
		Span:    Span{Start: start, End: p.pos},
		payload: &LabelExpr{Name: name, Body: body},
	}, nil
}

// parseAsBinding parses: source as pattern | body
func (p *parser) parseAsBinding(source *Expression, start int) (*Expression, error) {
	p.pos += 2 // consume "as"
	p.skipWhitespaceAndComments()
	pattern, err := p.parsePattern()
	if err != nil {
		return nil, err
	}
	p.skipWhitespaceAndComments()
	if p.atEnd() || p.peek() != '|' {
		return nil, p.syntaxError("expected '|' after 'as' pattern")
	}
	p.advance() // consume |
	p.skipWhitespaceAndComments()
	body, err := p.parseQuery()
	if err != nil {
		return nil, err
	}
	return &Expression{
		Kind:    KindAsBinding,
		Span:    Span{Start: start, End: p.pos},
		payload: &AsBindingExpr{Source: source, Pattern: pattern, Body: body},
	}, nil
}

// parsePattern parses a destructuring pattern: $name, [$a, $b], {$x, $y: $z}
// or a ?// alternative.
func (p *parser) parsePattern() (*Expression, error) {
	pattern, err := p.parsePatternSingle()
	if err != nil {
		return nil, err
	}
	// Check for ?// alternative
	for {
		p.skipWhitespaceAndComments()
		if !p.matchString("?//") {
			break
		}
		p.pos += 3
		p.skipWhitespaceAndComments()
		next, err := p.parsePatternSingle()
		if err != nil {
			return nil, err
		}
		if pattern.Kind != KindPatternAlternative {
			pattern = &Expression{
				Kind:    KindPatternAlternative,
				Span:    Span{Start: pattern.Span.Start, End: next.Span.End},
				payload: &PatternAlternativeExpr{Patterns: []*Expression{pattern, next}},
			}
		} else {
			pa := pattern.payload.(*PatternAlternativeExpr)
			pa.Patterns = append(pa.Patterns, next)
			pattern.Span.End = next.Span.End
		}
	}
	return pattern, nil
}

func (p *parser) parsePatternSingle() (*Expression, error) {
	p.skipWhitespaceAndComments()
	start := p.pos

	if p.peek() == '$' {
		p.advance()
		name, ok := p.scanIdentifier()
		if !ok {
			return nil, p.syntaxError("expected variable name after '$'")
		}
		return &Expression{
			Kind:    KindVariable,
			Span:    Span{Start: start, End: p.pos},
			payload: &VariableExpr{Name: name},
		}, nil
	}

	if p.peek() == '[' {
		return p.parseArrayPattern(start)
	}

	if p.peek() == '{' {
		return p.parseObjectPattern(start)
	}

	return nil, p.syntaxError("expected pattern: $name, [...], or {...}")
}

func (p *parser) parseArrayPattern(start int) (*Expression, error) {
	p.advance() // consume [
	var elements []*Expression
	p.skipWhitespaceAndComments()
	if !p.atEnd() && p.peek() == ']' {
		return nil, p.syntaxError("empty array pattern is not allowed")
	}
	for {
		p.skipWhitespaceAndComments()
		if !p.atEnd() && p.peek() == ']' {
			p.advance()
			break
		}
		elem, err := p.parsePatternSingle()
		if err != nil {
			return nil, err
		}
		elements = append(elements, elem)
		p.skipWhitespaceAndComments()
		if !p.atEnd() && p.peek() == ',' {
			p.advance()
			continue
		}
		p.skipWhitespaceAndComments()
		if !p.atEnd() && p.peek() == ']' {
			p.advance()
			break
		}
		return nil, p.syntaxError("expected ',' or ']' in array pattern")
	}
	return &Expression{
		Kind:    KindArray,
		Span:    Span{Start: start, End: p.pos},
		payload: &ArrayExpr{Elements: elements},
	}, nil
}

func (p *parser) parseObjectPattern(start int) (*Expression, error) {
	p.advance() // consume {
	var entries []ObjectEntry
	p.skipWhitespaceAndComments()
	if !p.atEnd() && p.peek() == '}' {
		return nil, p.syntaxError("empty object pattern is not allowed")
	}
	for {
		p.skipWhitespaceAndComments()
		if !p.atEnd() && p.peek() == '}' {
			p.advance()
			break
		}
		// Object pattern entries are: key: $var  or  $var (shorthand)
		if p.peek() == '$' {
			p.advance()
			name, ok := p.scanIdentifier()
			if !ok {
				return nil, p.syntaxError("expected variable name in object pattern")
			}
			entries = append(entries, ObjectEntry{
				KeyKind: ObjectKeyShorthand,
				KeyName: name,
				Value:   &Expression{Kind: KindVariable, Span: Span{Start: p.pos - len(name) - 1, End: p.pos}, payload: &VariableExpr{Name: name}},
			})
		} else {
			// key: $var
			key, ok := p.scanIdentifier()
			if !ok {
				return nil, p.syntaxError("expected key or $var in object pattern")
			}
			p.skipWhitespaceAndComments()
			if p.atEnd() || p.peek() != ':' {
				return nil, p.syntaxError("expected ':' in object pattern")
			}
			p.advance()
			p.skipWhitespaceAndComments()
			val, err := p.parsePatternSingle()
			if err != nil {
				return nil, err
			}
			entries = append(entries, ObjectEntry{
				KeyKind: ObjectKeyBare,
				KeyName: key,
				Value:   val,
			})
		}
		p.skipWhitespaceAndComments()
		if !p.atEnd() && p.peek() == ',' {
			p.advance()
			continue
		}
		p.skipWhitespaceAndComments()
		if !p.atEnd() && p.peek() == '}' {
			p.advance()
			break
		}
		return nil, p.syntaxError("expected ',' or '}' in object pattern")
	}
	return &Expression{
		Kind:    KindObject,
		Span:    Span{Start: start, End: p.pos},
		payload: &ObjectExpr{Entries: entries},
	}, nil
}

// parsePipe handles the pipe operator (right-associative).
// This is the full expression level including commas.
func (p *parser) parsePipe() (*Expression, error) {
	left, err := p.parsePratt(precPipe)
	if err != nil {
		return nil, err
	}
	// Handle as-binding at the pipe level
	p.skipWhitespaceAndComments()
	if p.matchKeyword("as") {
		start := left.Span.Start
		return p.parseAsBinding(left, start)
	}
	return left, nil
}

// parseExp parses a pipe-level expression without commas.
// Used for object values, array elements, if/then/else,
// reduce/foreach init/update, and other contexts where comma is a separator.
// Pipe is right-associative: a | b | c parses as a | (b | c).
func (p *parser) parseExp() (*Expression, error) {
	left, err := p.parsePratt(precComma + 1)
	if err != nil {
		return nil, err
	}
	p.skipWhitespaceAndComments()

	// Check for as-binding: exp as $pattern | body
	if p.matchKeyword("as") {
		return p.parseAsBinding(left, left.Span.Start)
	}

	if p.atEnd() || p.peek() != '|' {
		return left, nil
	}
	// Don't consume |= as pipe
	if p.pos+1 < len(p.input) && p.input[p.pos+1] == '=' {
		return left, nil
	}
	p.advance() // consume |
	p.skipWhitespaceAndComments()
	right, err := p.parseExp()
	if err != nil {
		return nil, err
	}
	return &Expression{
		Kind:    KindPipe,
		Span:    Span{Start: left.Span.Start, End: right.Span.End},
		payload: &PipeExpr{LHS: left, RHS: right},
	}, nil
}

// parsePratt is the Pratt parser for all precedence levels.
func (p *parser) parsePratt(minPrec int) (*Expression, error) {
	left, err := p.parsePrefix()
	if err != nil {
		return nil, err
	}
	// lastNonAssocPrec tracks the precedence of the last consumed non-associative
	// operator, so we can reject chaining at the same level (e.g. a == b == c).
	lastNonAssocPrec := -1
	for {
		// Handle postfix operators (? .foo .[...] .[])
		left, err = p.parsePostfix(left)
		if err != nil {
			return nil, err
		}

		p.skipWhitespaceAndComments()
		if p.atEnd() {
			break
		}

		op, prec, rightAssoc := p.peekInfixOp()
		if op == "" || prec < minPrec {
			break
		}

		// Non-associative operators cannot chain at the same precedence
		// (e.g. a == b == c is a syntax error in jq).
		if lastNonAssocPrec == prec && isNonAssociative(op) {
			return nil, p.syntaxErrorf("operator '%s' is non-associative", op)
		}

		// For non-associative operators, don't allow same-precedence nesting
		nextMinPrec := prec + 1
		if rightAssoc {
			nextMinPrec = prec
		}

		p.consumeInfixOp(op)
		p.skipWhitespaceAndComments()
		right, err := p.parsePratt(nextMinPrec)
		if err != nil {
			return nil, err
		}

		left = p.makeBinaryNode(op, left, right)

		if isNonAssociative(op) {
			lastNonAssocPrec = prec
		} else {
			lastNonAssocPrec = -1
		}
	}
	return left, nil
}

// peekInfixOp checks what infix operator is at the current position.
// Returns the operator string, its precedence, and whether it's right-associative.
func (p *parser) peekInfixOp() (op string, prec int, rightAssoc bool) {
	// Multi-char operators (check longest first)
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
	case '|':
		// | but not |= (already checked above)
		return "|", precPipe, true
	case ',':
		return ",", precComma, false
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

	// Keyword operators: and, or
	if p.matchKeyword("and") {
		return "and", precAnd, false
	}
	if p.matchKeyword("or") {
		return "or", precOr, false
	}

	return "", 0, false
}

// isNonAssociative reports whether the operator is non-associative,
// meaning it cannot be chained at the same precedence level (e.g. a == b == c).
func isNonAssociative(op string) bool {
	switch op {
	case "=", "|=", "+=", "-=", "*=", "/=", "%=", "//=",
		"==", "!=", "<", ">", "<=", ">=":
		return true
	}
	return false
}

func (p *parser) consumeInfixOp(op string) {
	p.pos += len(op)
	p.lastContent = p.pos
}

// makeBinaryNode creates the appropriate AST node for an infix operator.
func (p *parser) makeBinaryNode(op string, lhs, rhs *Expression) *Expression {
	span := Span{Start: lhs.Span.Start, End: rhs.Span.End}
	switch op {
	case "|":
		return &Expression{Kind: KindPipe, Span: span, payload: &PipeExpr{LHS: lhs, RHS: rhs}}
	case ",":
		return &Expression{Kind: KindComma, Span: span, payload: &CommaExpr{LHS: lhs, RHS: rhs}}
	case "//":
		return &Expression{Kind: KindAlternative, Span: span, payload: &AlternativeExpr{LHS: lhs, RHS: rhs}}
	case "=":
		return &Expression{Kind: KindAssign, Span: span, payload: &AssignExpr{Op: AssignPlain, LHS: lhs, RHS: rhs}}
	case "|=":
		return &Expression{Kind: KindUpdateAssign, Span: span, payload: &AssignExpr{Op: AssignUpdate, LHS: lhs, RHS: rhs}}
	case "+=":
		return &Expression{Kind: KindUpdateAssign, Span: span, payload: &AssignExpr{Op: AssignAdd, LHS: lhs, RHS: rhs}}
	case "-=":
		return &Expression{Kind: KindUpdateAssign, Span: span, payload: &AssignExpr{Op: AssignSub, LHS: lhs, RHS: rhs}}
	case "*=":
		return &Expression{Kind: KindUpdateAssign, Span: span, payload: &AssignExpr{Op: AssignMul, LHS: lhs, RHS: rhs}}
	case "/=":
		return &Expression{Kind: KindUpdateAssign, Span: span, payload: &AssignExpr{Op: AssignDiv, LHS: lhs, RHS: rhs}}
	case "%=":
		return &Expression{Kind: KindUpdateAssign, Span: span, payload: &AssignExpr{Op: AssignMod, LHS: lhs, RHS: rhs}}
	case "//=":
		return &Expression{Kind: KindUpdateAssign, Span: span, payload: &AssignExpr{Op: AssignAlt, LHS: lhs, RHS: rhs}}
	case "+":
		return &Expression{Kind: KindBinary, Span: span, payload: &BinaryExpr{Op: OpAdd, LHS: lhs, RHS: rhs}}
	case "-":
		return &Expression{Kind: KindBinary, Span: span, payload: &BinaryExpr{Op: OpSub, LHS: lhs, RHS: rhs}}
	case "*":
		return &Expression{Kind: KindBinary, Span: span, payload: &BinaryExpr{Op: OpMul, LHS: lhs, RHS: rhs}}
	case "/":
		return &Expression{Kind: KindBinary, Span: span, payload: &BinaryExpr{Op: OpDiv, LHS: lhs, RHS: rhs}}
	case "%":
		return &Expression{Kind: KindBinary, Span: span, payload: &BinaryExpr{Op: OpMod, LHS: lhs, RHS: rhs}}
	case "==":
		return &Expression{Kind: KindBinary, Span: span, payload: &BinaryExpr{Op: OpEq, LHS: lhs, RHS: rhs}}
	case "!=":
		return &Expression{Kind: KindBinary, Span: span, payload: &BinaryExpr{Op: OpNe, LHS: lhs, RHS: rhs}}
	case "<":
		return &Expression{Kind: KindBinary, Span: span, payload: &BinaryExpr{Op: OpLt, LHS: lhs, RHS: rhs}}
	case ">":
		return &Expression{Kind: KindBinary, Span: span, payload: &BinaryExpr{Op: OpGt, LHS: lhs, RHS: rhs}}
	case "<=":
		return &Expression{Kind: KindBinary, Span: span, payload: &BinaryExpr{Op: OpLe, LHS: lhs, RHS: rhs}}
	case ">=":
		return &Expression{Kind: KindBinary, Span: span, payload: &BinaryExpr{Op: OpGe, LHS: lhs, RHS: rhs}}
	case "and":
		return &Expression{Kind: KindBinary, Span: span, payload: &BinaryExpr{Op: OpAnd, LHS: lhs, RHS: rhs}}
	case "or":
		return &Expression{Kind: KindBinary, Span: span, payload: &BinaryExpr{Op: OpOr, LHS: lhs, RHS: rhs}}
	}
	return &Expression{Kind: KindBinary, Span: span, payload: &BinaryExpr{Op: OpAdd, LHS: lhs, RHS: rhs}}
}

// parsePrefix handles unary minus.
func (p *parser) parsePrefix() (*Expression, error) {
	p.skipWhitespaceAndComments()
	start := p.pos
	if p.peek() == '-' {
		p.advance()
		p.skipWhitespaceAndComments()
		arg, err := p.parsePratt(precPrefix)
		if err != nil {
			return nil, err
		}
		return &Expression{
			Kind:    KindNegate,
			Span:    Span{Start: start, End: arg.Span.End},
			payload: &NegateExpr{Arg: arg},
		}, nil
	}
	return p.parsePostfixChain()
}

// parsePostfixChain parses a primary followed by postfix operators,
// without handling any infix operators. Used by parsePrefix.
func (p *parser) parsePostfixChain() (*Expression, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	return p.parsePostfix(left)
}

// parsePostfix handles postfix operators: ? .foo .["foo"] .[0] .[2:4] .[]
// These can chain: .foo.bar?[].baz
func (p *parser) parsePostfix(left *Expression) (*Expression, error) {
	for {
		// ? optional/error-suppression operator
		if !p.atEnd() && p.peek() == '?' {
			p.advance()
			left = &Expression{
				Kind:    KindOptional,
				Span:    Span{Start: left.Span.Start, End: p.pos},
				payload: &OptionalExpr{Arg: left},
			}
			continue
		}

		// . field access (but not .. which is recursive descent handled in primary)
		if !p.atEnd() && p.peek() == '.' {
			// Check it's not .. (recursive descent — that's a primary)
			if p.pos+1 < len(p.input) && p.input[p.pos+1] == '.' {
				break
			}
			p.advance() // consume .
			var err error
			left, err = p.parseFieldAccess(left)
			if err != nil {
				return nil, err
			}
			continue
		}

		// [ indexing/slice/iterator
		if !p.atEnd() && p.peek() == '[' {
			p.advance() // consume [
			left = p.parseBracketAccess(left)
			if left == nil {
				return nil, p.syntaxError("invalid bracket access")
			}
			continue
		}

		break
	}
	return left, nil
}

// parseFieldAccess parses the part after . in field access.
// The . has already been consumed.
func (p *parser) parseFieldAccess(base *Expression) (*Expression, error) {
	start := base.Span.Start

	// .foo — identifier key
	if name, ok := p.scanIdentifier(); ok {
		return &Expression{
			Kind:    KindField,
			Span:    Span{Start: start, End: p.pos},
			payload: &FieldExpr{Name: name, Base: base},
		}, nil
	}

	// ."foo" or .'foo' — quoted key (jq uses ."..." syntax)
	if !p.atEnd() && p.peek() == '"' {
		parts, err := p.parseStringLiteralValue()
		if err != nil {
			return nil, err
		}
		// For field access, the string must be a simple text (no interpolation)
		var name strings.Builder
		for _, part := range parts {
			if t, ok := part.(StringText); ok {
				name.WriteString(t.Text)
			} else {
				return nil, p.syntaxError("string interpolation not allowed in field name")
			}
		}
		return &Expression{
			Kind:    KindField,
			Span:    Span{Start: start, End: p.pos},
			payload: &FieldExpr{Name: name.String(), Base: base},
		}, nil
	}

	return nil, p.syntaxError("expected field name after '.'")
}

// parseBracketAccess parses the content inside .[...]
// The [ has already been consumed. Returns the AST node.
func (p *parser) parseBracketAccess(base *Expression) *Expression {
	start := base.Span.Start
	p.skipWhitespaceAndComments()

	// .[] — iterator
	if !p.atEnd() && p.peek() == ']' {
		p.advance() // consume ]
		return &Expression{
			Kind:    KindIterator,
			Span:    Span{Start: start, End: p.pos},
			payload: &IteratorExpr{Base: base},
		}
	}

	// .[expr] or .[start:end] — index or slice
	// First, try to parse an expression
	var firstExpr *Expression
	if !p.atEnd() && p.peek() != ':' {
		var err error
		firstExpr, err = p.parseExp()
		if err != nil {
			return nil
		}
	}

	p.skipWhitespaceAndComments()

	// .[start:end] — slice
	if !p.atEnd() && p.peek() == ':' {
		p.advance() // consume :
		p.skipWhitespaceAndComments()

		var endExpr *Expression
		if !p.atEnd() && p.peek() != ']' {
			var err error
			endExpr, err = p.parseExp()
			if err != nil {
				return nil
			}
		}

		p.skipWhitespaceAndComments()
		if p.atEnd() || p.peek() != ']' {
			return nil
		}
		p.advance() // consume ]
		return &Expression{
			Kind:    KindSlice,
			Span:    Span{Start: start, End: p.pos},
			payload: &SliceExpr{Start: firstExpr, End: endExpr, Base: base},
		}
	}

	// .[expr] — index
	if p.atEnd() || p.peek() != ']' {
		return nil
	}
	p.advance() // consume ]
	if firstExpr == nil {
		return nil
	}
	return &Expression{
		Kind:    KindIndex,
		Span:    Span{Start: start, End: p.pos},
		payload: &IndexExpr{Index: firstExpr, Base: base},
	}
}

// parsePrimary handles all primary expressions.
func (p *parser) parsePrimary() (*Expression, error) {
	p.skipWhitespaceAndComments()
	start := p.pos

	if p.atEnd() {
		return nil, p.syntaxError("unexpected end of input")
	}

	ch := p.peek()

	// . — identity or field access or recursive descent
	if ch == '.' {
		// Check for .. (recursive descent)
		if p.pos+1 < len(p.input) && p.input[p.pos+1] == '.' {
			p.advance() // consume first .
			p.advance() // consume second .
			return &Expression{
				Kind:    KindRecursiveDescent,
				Span:    Span{Start: start, End: p.pos},
				payload: &RecursiveDescentExpr{},
			}, nil
		}
		p.advance() // consume .
		identity := &Expression{
			Kind:    KindIdentity,
			Span:    Span{Start: start, End: start + 1},
			payload: &IdentityExpr{},
		}
		// Check if followed by field name, string, or bracket
		if name, ok := p.scanIdentifier(); ok {
			return &Expression{
				Kind:    KindField,
				Span:    Span{Start: start, End: p.pos},
				payload: &FieldExpr{Name: name, Base: identity},
			}, nil
		}
		if !p.atEnd() && p.peek() == '"' {
			parts, err := p.parseStringLiteralValue()
			if err != nil {
				return nil, err
			}
			var name strings.Builder
			for _, part := range parts {
				if t, ok := part.(StringText); ok {
					name.WriteString(t.Text)
				} else {
					return nil, p.syntaxError("string interpolation not allowed in field name")
				}
			}
			return &Expression{
				Kind:    KindField,
				Span:    Span{Start: start, End: p.pos},
				payload: &FieldExpr{Name: name.String(), Base: identity},
			}, nil
		}
		if !p.atEnd() && p.peek() == '[' {
			p.advance() // consume [
			result := p.parseBracketAccess(identity)
			if result == nil {
				return nil, p.syntaxError("invalid bracket access")
			}
			return result, nil
		}
		// Just . (identity)
		return &Expression{
			Kind:    KindIdentity,
			Span:    Span{Start: start, End: p.pos},
			payload: &IdentityExpr{},
		}, nil
	}

	// ( expr )
	if ch == '(' {
		p.advance()
		p.skipWhitespaceAndComments()
		inner, err := p.parseQuery()
		if err != nil {
			return nil, err
		}
		p.skipWhitespaceAndComments()
		if p.atEnd() || p.peek() != ')' {
			return nil, p.syntaxError("expected ')'")
		}
		p.advance()
		return &Expression{
			Kind:    KindParenthesized,
			Span:    Span{Start: start, End: p.pos},
			payload: &ParenthesizedExpr{Inner: inner},
		}, nil
	}

	// [ array construction ]
	if ch == '[' {
		return p.parseArray(start)
	}

	// { object construction }
	if ch == '{' {
		return p.parseObject(start)
	}

	// " string "
	if ch == '"' {
		parts, err := p.parseStringLiteralValue()
		if err != nil {
			return nil, err
		}
		return &Expression{
			Kind:    KindString,
			Span:    Span{Start: start, End: p.pos},
			payload: &StringExpr{Parts: parts},
		}, nil
	}

	// @ format string
	if ch == '@' {
		return p.parseFormat(start)
	}

	// $ variable
	if ch == '$' {
		return p.parseVariable(start)
	}

	// Numbers
	if isDigit(ch) {
		text, ok := p.scanNumber()
		if !ok {
			return nil, p.syntaxError("invalid number literal")
		}
		return &Expression{
			Kind:    KindNumber,
			Span:    Span{Start: start, End: p.pos},
			payload: &NumberExpr{Text: text},
		}, nil
	}

	// Keywords and identifiers
	if isIdentifierStart(ch) {
		return p.parseKeywordOrFunction(start)
	}

	return nil, p.syntaxErrorf("unexpected character %q", ch)
}

// parseArray parses [expr, expr, ...]
func (p *parser) parseArray(start int) (*Expression, error) {
	p.advance() // consume [
	var elements []*Expression
	p.skipWhitespaceAndComments()
	if !p.atEnd() && p.peek() == ']' {
		p.advance()
		return &Expression{
			Kind:    KindArray,
			Span:    Span{Start: start, End: p.pos},
			payload: &ArrayExpr{Elements: elements},
		}, nil
	}
	for {
		p.skipWhitespaceAndComments()
		elem, err := p.parseExp()
		if err != nil {
			return nil, err
		}
		elements = append(elements, elem)
		p.skipWhitespaceAndComments()
		if !p.atEnd() && p.peek() == ',' {
			p.advance()
			continue
		}
		p.skipWhitespaceAndComments()
		if !p.atEnd() && p.peek() == ']' {
			p.advance()
			break
		}
		return nil, p.syntaxError("expected ',' or ']' in array")
	}
	return &Expression{
		Kind:    KindArray,
		Span:    Span{Start: start, End: p.pos},
		payload: &ArrayExpr{Elements: elements},
	}, nil
}

// parseObject parses {key: value, ...} with shorthands
func (p *parser) parseObject(start int) (*Expression, error) {
	p.advance() // consume {
	var entries []ObjectEntry
	p.skipWhitespaceAndComments()
	if !p.atEnd() && p.peek() == '}' {
		p.advance()
		return &Expression{
			Kind:    KindObject,
			Span:    Span{Start: start, End: p.pos},
			payload: &ObjectExpr{Entries: entries},
		}, nil
	}
	for {
		p.skipWhitespaceAndComments()
		entry, err := p.parseObjectEntry()
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
		p.skipWhitespaceAndComments()
		if !p.atEnd() && p.peek() == ',' {
			p.advance()
			continue
		}
		p.skipWhitespaceAndComments()
		if !p.atEnd() && p.peek() == '}' {
			p.advance()
			break
		}
		return nil, p.syntaxError("expected ',' or '}' in object")
	}
	return &Expression{
		Kind:    KindObject,
		Span:    Span{Start: start, End: p.pos},
		payload: &ObjectExpr{Entries: entries},
	}, nil
}

// parseObjectEntry parses one entry in an object literal.
func (p *parser) parseObjectEntry() (ObjectEntry, error) {
	start := p.pos

	// {$var} or {$var: value} — variable key
	if p.peek() == '$' {
		p.advance() // consume $
		name, ok := p.scanIdentifier()
		if !ok {
			return ObjectEntry{}, p.syntaxError("expected variable name after '$'")
		}
		varEnd := p.pos
		p.skipWhitespaceAndComments()
		// {$var} shorthand — key is var name, value is the variable
		if !p.atEnd() && (p.peek() == ',' || p.peek() == '}') {
			return ObjectEntry{
				KeyKind: ObjectKeyShorthand,
				KeyName: name,
				Value: &Expression{
					Kind:    KindVariable,
					Span:    Span{Start: start, End: varEnd},
					payload: &VariableExpr{Name: name},
				},
			}, nil
		}
		// {$var: value} — variable value as key
		if !p.atEnd() && p.peek() == ':' {
			p.advance()
			p.skipWhitespaceAndComments()
			val, err := p.parseExp()
			if err != nil {
				return ObjectEntry{}, err
			}
			return ObjectEntry{
				KeyKind: ObjectKeyVariable,
				KeyName: name,
				Value:   val,
			}, nil
		}
		return ObjectEntry{}, p.syntaxError("expected ':' or '}' after variable in object")
	}

	// {(expr): value} — expression key
	if p.peek() == '(' {
		p.advance()
		p.skipWhitespaceAndComments()
		keyExpr, err := p.parseQuery()
		if err != nil {
			return ObjectEntry{}, err
		}
		p.skipWhitespaceAndComments()
		if p.atEnd() || p.peek() != ')' {
			return ObjectEntry{}, p.syntaxError("expected ')' in object key")
		}
		p.advance()
		p.skipWhitespaceAndComments()
		if p.atEnd() || p.peek() != ':' {
			return ObjectEntry{}, p.syntaxError("expected ':' after object key")
		}
		p.advance()
		p.skipWhitespaceAndComments()
		val, err := p.parseExp()
		if err != nil {
			return ObjectEntry{}, err
		}
		return ObjectEntry{
			KeyKind: ObjectKeyExpression,
			Key:     keyExpr,
			Value:   val,
		}, nil
	}

	// {"string": value} — string key
	if p.peek() == '"' {
		parts, err := p.parseStringLiteralValue()
		if err != nil {
			return ObjectEntry{}, err
		}
		var name strings.Builder
		for _, part := range parts {
			if t, ok := part.(StringText); ok {
				name.WriteString(t.Text)
			} else {
				return ObjectEntry{}, p.syntaxError("string interpolation not allowed in object key")
			}
		}
		p.skipWhitespaceAndComments()
		if p.atEnd() || p.peek() != ':' {
			return ObjectEntry{}, p.syntaxError("expected ':' after string key")
		}
		p.advance()
		p.skipWhitespaceAndComments()
		val, err := p.parseExp()
		if err != nil {
			return ObjectEntry{}, err
		}
		return ObjectEntry{
			KeyKind: ObjectKeyString,
			KeyName: name.String(),
			Value:   val,
		}, nil
	}

	// {foo: value} or {foo} — bare identifier key
	name, ok := p.scanIdentifier()
	if !ok {
		return ObjectEntry{}, p.syntaxError("expected key in object")
	}
	p.skipWhitespaceAndComments()
	// {foo} shorthand — key is foo, value is .foo
	if !p.atEnd() && (p.peek() == ',' || p.peek() == '}') {
		return ObjectEntry{
			KeyKind: ObjectKeyShorthand,
			KeyName: name,
			Value: &Expression{
				Kind:    KindField,
				Span:    Span{Start: start, End: p.pos},
				payload: &FieldExpr{Name: name},
			},
		}, nil
	}
	// {foo: value}
	if !p.atEnd() && p.peek() == ':' {
		p.advance()
		p.skipWhitespaceAndComments()
		val, err := p.parseExp()
		if err != nil {
			return ObjectEntry{}, err
		}
		return ObjectEntry{
			KeyKind: ObjectKeyBare,
			KeyName: name,
			Value:   val,
		}, nil
	}
	return ObjectEntry{}, p.syntaxError("expected ':' or '}' after key in object")
}

// parseFormat parses @format or @format "string"
func (p *parser) parseFormat(start int) (*Expression, error) {
	name, ok := p.scanFormatName()
	if !ok {
		return nil, p.syntaxError("expected format name after '@'")
	}
	p.skipWhitespaceAndComments()
	// @format "string" — format with string
	if !p.atEnd() && p.peek() == '"' {
		parts, err := p.parseStringLiteralValue()
		if err != nil {
			return nil, err
		}
		strExpr := &Expression{
			Kind:    KindString,
			Span:    Span{Start: start, End: p.pos},
			payload: &StringExpr{Parts: parts},
		}
		return &Expression{
			Kind:    KindFormat,
			Span:    Span{Start: start, End: p.pos},
			payload: &FormatExpr{Name: name, String: strExpr},
		}, nil
	}
	// Bare @format (applies to .)
	return &Expression{
		Kind:    KindFormat,
		Span:    Span{Start: start, End: p.pos},
		payload: &FormatExpr{Name: name},
	}, nil
}

// parseVariable parses $name
func (p *parser) parseVariable(start int) (*Expression, error) {
	p.advance() // consume $
	// Check for $__loc__ or other special variables
	name, ok := p.scanIdentifier()
	if !ok {
		return nil, p.syntaxError("expected variable name after '$'")
	}
	return &Expression{
		Kind:    KindVariable,
		Span:    Span{Start: start, End: p.pos},
		payload: &VariableExpr{Name: name},
	}, nil
}

// parseKeywordOrFunction handles keywords (if, try, reduce, foreach, true, false, null, etc.)
// and function calls.
func (p *parser) parseKeywordOrFunction(start int) (*Expression, error) {
	name, ok := p.scanIdentifier()
	if !ok {
		return nil, p.syntaxError("expected identifier")
	}

	switch name {
	case "true":
		return &Expression{
			Kind:    KindBool,
			Span:    Span{Start: start, End: p.pos},
			payload: &BoolExpr{Value: true},
		}, nil
	case "false":
		return &Expression{
			Kind:    KindBool,
			Span:    Span{Start: start, End: p.pos},
			payload: &BoolExpr{Value: false},
		}, nil
	case "null":
		return &Expression{
			Kind:    KindNull,
			Span:    Span{Start: start, End: p.pos},
			payload: &NullExpr{},
		}, nil
	case "if":
		return p.parseIf(start)
	case "try":
		return p.parseTry(start)
	case "reduce":
		return p.parseReduce(start)
	case "foreach":
		return p.parseForeach(start)
	case "break":
		return p.parseBreak(start)
	}

	// Function call: name(args) or bare name (0-arg function)
	p.skipWhitespaceAndComments()
	if !p.atEnd() && p.peek() == '(' {
		p.advance() // consume (
		var args []*Expression
		p.skipWhitespaceAndComments()
		if !p.atEnd() && p.peek() == ')' {
			p.advance()
		} else {
			for {
				p.skipWhitespaceAndComments()
				// Parse each argument as a pipe-level expression so that
				// commas form comma-expressions within a single argument.
				// Only ';' separates arguments.
				arg, err := p.parsePipe()
				if err != nil {
					return nil, err
				}
				args = append(args, arg)
				p.skipWhitespaceAndComments()
				if !p.atEnd() && p.peek() == ';' {
					p.advance()
					continue
				}
				p.skipWhitespaceAndComments()
				if !p.atEnd() && p.peek() == ')' {
					p.advance()
					break
				}
				return nil, p.syntaxError("expected ';' or ')' in function arguments")
			}
		}
		return &Expression{
			Kind:    KindFunctionCall,
			Span:    Span{Start: start, End: p.pos},
			payload: &FunctionCallExpr{Name: name, Args: args},
		}, nil
	}

	// 0-arg function call
	return &Expression{
		Kind:    KindFunctionCall,
		Span:    Span{Start: start, End: p.pos},
		payload: &FunctionCallExpr{Name: name},
	}, nil
}

// parseIf parses: if cond then a elif cond2 then b else c end
func (p *parser) parseIf(start int) (*Expression, error) {
	p.skipWhitespaceAndComments()
	cond, err := p.parsePipe()
	if err != nil {
		return nil, err
	}
	p.skipWhitespaceAndComments()
	if !p.matchKeyword("then") {
		return nil, p.syntaxError("expected 'then' after if condition")
	}
	p.pos += 4 // consume "then"
	p.skipWhitespaceAndComments()
	thenExpr, err := p.parsePipe()
	if err != nil {
		return nil, err
	}

	var elifs []ElifBranch
	for {
		p.skipWhitespaceAndComments()
		if !p.matchKeyword("elif") {
			break
		}
		p.pos += 4 // consume "elif"
		p.skipWhitespaceAndComments()
		elifCond, err := p.parsePipe()
		if err != nil {
			return nil, err
		}
		p.skipWhitespaceAndComments()
		if !p.matchKeyword("then") {
			return nil, p.syntaxError("expected 'then' after elif condition")
		}
		p.pos += 4
		p.skipWhitespaceAndComments()
		elifThen, err := p.parsePipe()
		if err != nil {
			return nil, err
		}
		elifs = append(elifs, ElifBranch{Cond: elifCond, Then: elifThen})
	}

	var elseExpr *Expression
	p.skipWhitespaceAndComments()
	if p.matchKeyword("else") {
		p.pos += 4 // consume "else"
		p.skipWhitespaceAndComments()
		elseExpr, err = p.parsePipe()
		if err != nil {
			return nil, err
		}
	}

	p.skipWhitespaceAndComments()
	if !p.matchKeyword("end") {
		return nil, p.syntaxError("expected 'end'")
	}
	p.pos += 3 // consume "end"

	return &Expression{
		Kind:    KindIf,
		Span:    Span{Start: start, End: p.pos},
		payload: &IfExpr{Cond: cond, Then: thenExpr, Elifs: elifs, Else: elseExpr},
	}, nil
}

// parseTry parses: try body catch handler  or  try body
func (p *parser) parseTry(start int) (*Expression, error) {
	p.skipWhitespaceAndComments()
	body, err := p.parsePratt(precPostfix)
	if err != nil {
		return nil, err
	}
	// Also handle postfix on the try body
	body, err = p.parsePostfix(body)
	if err != nil {
		return nil, err
	}
	p.skipWhitespaceAndComments()
	var catchExpr *Expression
	if p.matchKeyword("catch") {
		p.pos += 5 // consume "catch"
		p.skipWhitespaceAndComments()
		catchExpr, err = p.parsePratt(precPostfix)
		if err != nil {
			return nil, err
		}
		catchExpr, err = p.parsePostfix(catchExpr)
		if err != nil {
			return nil, err
		}
	}
	return &Expression{
		Kind:    KindTry,
		Span:    Span{Start: start, End: p.pos},
		payload: &TryExpr{Body: body, Catch: catchExpr},
	}, nil
}

// parseReduce parses: reduce exp as $var (init; update)
func (p *parser) parseReduce(start int) (*Expression, error) {
	return p.parseReduceForeach(start, true)
}

// parseForeach parses: foreach exp as $var (init; update; extract?)
func (p *parser) parseForeach(start int) (*Expression, error) {
	return p.parseReduceForeach(start, false)
}

func (p *parser) parseReduceForeach(start int, isReduce bool) (*Expression, error) {
	p.skipWhitespaceAndComments()
	source, err := p.parsePratt(precPipe)
	if err != nil {
		return nil, err
	}
	p.skipWhitespaceAndComments()
	if !p.matchKeyword("as") {
		return nil, p.syntaxError("expected 'as' in reduce/foreach")
	}
	p.pos += 2 // consume "as"
	p.skipWhitespaceAndComments()
	pattern, err := p.parsePattern()
	if err != nil {
		return nil, err
	}
	p.skipWhitespaceAndComments()
	if p.atEnd() || p.peek() != '(' {
		return nil, p.syntaxError("expected '(' in reduce/foreach")
	}
	p.advance()
	p.skipWhitespaceAndComments()
	init, err := p.parseQuery()
	if err != nil {
		return nil, err
	}
	p.skipWhitespaceAndComments()
	if p.atEnd() || p.peek() != ';' {
		return nil, p.syntaxError("expected ';' in reduce/foreach")
	}
	p.advance()
	p.skipWhitespaceAndComments()
	update, err := p.parseQuery()
	if err != nil {
		return nil, err
	}
	p.skipWhitespaceAndComments()

	var extract *Expression
	if !isReduce {
		// foreach can have an optional third expression
		if !p.atEnd() && p.peek() == ';' {
			p.advance()
			p.skipWhitespaceAndComments()
			extract, err = p.parseQuery()
			if err != nil {
				return nil, err
			}
			p.skipWhitespaceAndComments()
		}
	}

	if p.atEnd() || p.peek() != ')' {
		return nil, p.syntaxError("expected ')' in reduce/foreach")
	}
	p.advance()

	if isReduce {
		return &Expression{
			Kind:    KindReduce,
			Span:    Span{Start: start, End: p.pos},
			payload: &ReduceExpr{Source: source, Pattern: pattern, Init: init, Update: update},
		}, nil
	}
	return &Expression{
		Kind:    KindForeach,
		Span:    Span{Start: start, End: p.pos},
		payload: &ForeachExpr{Source: source, Pattern: pattern, Init: init, Update: update, Extract: extract},
	}, nil
}

// parseBreak parses: break $name
func (p *parser) parseBreak(start int) (*Expression, error) {
	p.skipWhitespaceAndComments()
	if p.atEnd() || p.peek() != '$' {
		return nil, p.syntaxError("expected '$' after 'break'")
	}
	p.advance()
	name, ok := p.scanIdentifier()
	if !ok {
		return nil, p.syntaxError("expected label name after 'break $'")
	}
	return &Expression{
		Kind:    KindBreak,
		Span:    Span{Start: start, End: p.pos},
		payload: &BreakExpr{Name: name},
	}, nil
}
