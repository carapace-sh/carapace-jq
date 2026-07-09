package jq

// ExpressionKind identifies the kind of AST node stored in an Expression.
type ExpressionKind int

const (
	KindIdentity ExpressionKind = iota
	KindRecursiveDescent
	KindField
	KindIndex
	KindSlice
	KindIterator
	KindOptional
	KindPipe
	KindComma
	KindAlternative
	KindAssign
	KindUpdateAssign
	KindBinary
	KindNegate
	KindNumber
	KindString
	KindFormat
	KindBool
	KindNull
	KindVariable
	KindArray
	KindObject
	KindFunctionCall
	KindIf
	KindTry
	KindReduce
	KindForeach
	KindAsBinding
	KindLabel
	KindBreak
	KindDef
	KindParenthesized
	KindPatternAlternative
)

func (k ExpressionKind) String() string {
	switch k {
	case KindIdentity:
		return "Identity"
	case KindRecursiveDescent:
		return "RecursiveDescent"
	case KindField:
		return "Field"
	case KindIndex:
		return "Index"
	case KindSlice:
		return "Slice"
	case KindIterator:
		return "Iterator"
	case KindOptional:
		return "Optional"
	case KindPipe:
		return "Pipe"
	case KindComma:
		return "Comma"
	case KindAlternative:
		return "Alternative"
	case KindAssign:
		return "Assign"
	case KindUpdateAssign:
		return "UpdateAssign"
	case KindBinary:
		return "Binary"
	case KindNegate:
		return "Negate"
	case KindNumber:
		return "Number"
	case KindString:
		return "String"
	case KindFormat:
		return "Format"
	case KindBool:
		return "Bool"
	case KindNull:
		return "Null"
	case KindVariable:
		return "Variable"
	case KindArray:
		return "Array"
	case KindObject:
		return "Object"
	case KindFunctionCall:
		return "FunctionCall"
	case KindIf:
		return "If"
	case KindTry:
		return "Try"
	case KindReduce:
		return "Reduce"
	case KindForeach:
		return "Foreach"
	case KindAsBinding:
		return "AsBinding"
	case KindLabel:
		return "Label"
	case KindBreak:
		return "Break"
	case KindDef:
		return "Def"
	case KindParenthesized:
		return "Parenthesized"
	case KindPatternAlternative:
		return "PatternAlternative"
	}
	return "Unknown"
}

// BinaryOp identifies a binary operator.
type BinaryOp int

const (
	OpAdd BinaryOp = iota
	OpSub
	OpMul
	OpDiv
	OpMod
	OpEq
	OpNe
	OpLt
	OpGt
	OpLe
	OpGe
	OpAnd
	OpOr
)

func (op BinaryOp) String() string {
	switch op {
	case OpAdd:
		return "+"
	case OpSub:
		return "-"
	case OpMul:
		return "*"
	case OpDiv:
		return "/"
	case OpMod:
		return "%"
	case OpEq:
		return "=="
	case OpNe:
		return "!="
	case OpLt:
		return "<"
	case OpGt:
		return ">"
	case OpLe:
		return "<="
	case OpGe:
		return ">="
	case OpAnd:
		return "and"
	case OpOr:
		return "or"
	}
	return ""
}

// AssignOp identifies an assignment operator.
type AssignOp int

const (
	AssignPlain AssignOp = iota
	AssignUpdate
	AssignAdd
	AssignSub
	AssignMul
	AssignDiv
	AssignMod
	AssignAlt
)

func (op AssignOp) String() string {
	switch op {
	case AssignPlain:
		return "="
	case AssignUpdate:
		return "|="
	case AssignAdd:
		return "+="
	case AssignSub:
		return "-="
	case AssignMul:
		return "*="
	case AssignDiv:
		return "/="
	case AssignMod:
		return "%="
	case AssignAlt:
		return "//="
	}
	return ""
}

// Expression is the universal AST node. The payload field holds kind-specific
// data accessed via type-asserting accessors.
type Expression struct {
	Kind    ExpressionKind `json:"kind"`
	Span    Span           `json:"span"`
	payload any
}

// --- Payload types ---

type IdentityExpr struct{}

type RecursiveDescentExpr struct{}

type FieldExpr struct {
	Name      string // key name (e.g. "foo" in .foo)
	Base      *Expression // the value being accessed (nil for top-level .foo)
	Optional  bool   // .foo? — error suppression
}

type IndexExpr struct {
	Index   *Expression // expression inside .[...]
	Base    *Expression // the value being indexed (nil for top-level .[...])
	Optional bool        // .[expr]? — error suppression
}

type SliceExpr struct {
	Start *Expression // nil if omitted
	End   *Expression // nil if omitted
	Base    *Expression // the value being sliced (nil for top-level .[...])
	Optional bool
}

type IteratorExpr struct {
	Base     *Expression // the value being iterated (nil for top-level .[])
	Optional bool
}

type OptionalExpr struct {
	Arg *Expression // the expression being made optional
}

type PipeExpr struct {
	LHS *Expression
	RHS *Expression
}

type CommaExpr struct {
	LHS *Expression
	RHS *Expression
}

type AlternativeExpr struct {
	LHS *Expression
	RHS *Expression
}

type AssignExpr struct {
	Op  AssignOp
	LHS *Expression
	RHS *Expression
}

type BinaryExpr struct {
	Op  BinaryOp
	LHS *Expression
	RHS *Expression
}

type NegateExpr struct {
	Arg *Expression
}

type NumberExpr struct {
	Text string // original literal text
}

// StringPart is one segment of a string literal.
type StringPart interface{ stringPart() }

type StringText struct {
	Text string
}

func (StringText) stringPart() {}

type StringInterp struct {
	Expr *Expression
}

func (StringInterp) stringPart() {}

type StringExpr struct {
	Parts []StringPart
}

type FormatExpr struct {
	Name   string      // format name without @ (e.g. "base64")
	String *Expression // the string literal, or nil for bare @foo
}

type BoolExpr struct {
	Value bool
}

type NullExpr struct{}

type VariableExpr struct {
	Name string // without $ prefix
}

type ArrayExpr struct {
	Elements []*Expression
}

// ObjectKeyKind identifies how an object key was written.
type ObjectKeyKind int

const (
	ObjectKeyBare        ObjectKeyKind = iota // {foo: ...}
	ObjectKeyString                           // {"foo": ...}
	ObjectKeyVariable                         // {$foo: ...} — variable value as key
	ObjectKeyShorthand                        // {foo} or {$foo}
	ObjectKeyExpression                       // {(expr): ...}
)

type ObjectEntry struct {
	KeyKind ObjectKeyKind
	// KeyName is the key string for bare/string/shorthand keys.
	// For ObjectKeyVariable shorthand {$foo}, KeyName is "foo" (without $).
	// For ObjectKeyVariable with value {$bar: ...}, KeyName is "bar".
	KeyName string
	// Key is the key expression for ObjectKeyExpression.
	Key *Expression
	// Value is the value expression. Nil for shorthands (expanded at eval time).
	Value *Expression
}

type ObjectExpr struct {
	Entries []ObjectEntry
}

type FunctionArg struct {
	Name    string // argument name (e.g. "f" in def map(f): ...)
	IsValue bool   // true for ($f) value-argument shorthand
}

type FunctionCallExpr struct {
	Name string
	Args []*Expression
}

type IfExpr struct {
	Cond     *Expression
	Then     *Expression
	Elifs    []ElifBranch
	Else     *Expression // nil if no else clause
}

type ElifBranch struct {
	Cond *Expression
	Then *Expression
}

type TryExpr struct {
	Body   *Expression
	Catch  *Expression // nil if no catch clause
}

type ReduceExpr struct {
	Source    *Expression // the exp in "reduce exp as ..."
	Pattern   *Expression // the $var or destructuring pattern
	Init      *Expression
	Update    *Expression
}

type ForeachExpr struct {
	Source    *Expression
	Pattern   *Expression
	Init      *Expression
	Update    *Expression
	Extract   *Expression // nil if omitted
}

type AsBindingExpr struct {
	Source  *Expression
	Pattern *Expression // $var or destructuring pattern
	Body    *Expression // the rest after the pipe
}

type LabelExpr struct {
	Name string // without $ prefix
	Body *Expression
}

type BreakExpr struct {
	Name string // without $ prefix
}

type DefExpr struct {
	Name string
	Args []FunctionArg
	Body *Expression
	Rest *Expression // expression after the def (after the ;)
}

type ParenthesizedExpr struct {
	Inner *Expression
}

// PatternAlternativeExpr is used in destructuring: [$a] ?// {$a}
type PatternAlternativeExpr struct {
	Patterns []*Expression
}

// --- Accessor methods ---

func (e *Expression) Payload() any { return e.payload }

func (e *Expression) FieldName() string {
	if e.Kind != KindField {
		return ""
	}
	return e.payload.(*FieldExpr).Name
}

func (e *Expression) FieldBase() *Expression {
	if e.Kind != KindField {
		return nil
	}
	return e.payload.(*FieldExpr).Base
}

func (e *Expression) FieldOptional() bool {
	if e.Kind != KindField {
		return false
	}
	return e.payload.(*FieldExpr).Optional
}

func (e *Expression) IndexExpr() *Expression {
	if e.Kind != KindIndex {
		return nil
	}
	return e.payload.(*IndexExpr).Index
}

func (e *Expression) IndexBase() *Expression {
	if e.Kind != KindIndex {
		return nil
	}
	return e.payload.(*IndexExpr).Base
}

func (e *Expression) SliceStart() *Expression {
	if e.Kind != KindSlice {
		return nil
	}
	return e.payload.(*SliceExpr).Start
}

func (e *Expression) SliceEnd() *Expression {
	if e.Kind != KindSlice {
		return nil
	}
	return e.payload.(*SliceExpr).End
}

func (e *Expression) IteratorOptional() bool {
	if e.Kind != KindIterator {
		return false
	}
	return e.payload.(*IteratorExpr).Optional
}

func (e *Expression) OptionalArg() *Expression {
	if e.Kind != KindOptional {
		return nil
	}
	return e.payload.(*OptionalExpr).Arg
}

func (e *Expression) PipeLHS() *Expression {
	if e.Kind != KindPipe {
		return nil
	}
	return e.payload.(*PipeExpr).LHS
}

func (e *Expression) PipeRHS() *Expression {
	if e.Kind != KindPipe {
		return nil
	}
	return e.payload.(*PipeExpr).RHS
}

func (e *Expression) CommaLHS() *Expression {
	if e.Kind != KindComma {
		return nil
	}
	return e.payload.(*CommaExpr).LHS
}

func (e *Expression) CommaRHS() *Expression {
	if e.Kind != KindComma {
		return nil
	}
	return e.payload.(*CommaExpr).RHS
}

func (e *Expression) AltLHS() *Expression {
	if e.Kind != KindAlternative {
		return nil
	}
	return e.payload.(*AlternativeExpr).LHS
}

func (e *Expression) AltRHS() *Expression {
	if e.Kind != KindAlternative {
		return nil
	}
	return e.payload.(*AlternativeExpr).RHS
}

func (e *Expression) AssignOp() AssignOp {
	if e.Kind != KindAssign && e.Kind != KindUpdateAssign {
		return -1
	}
	return e.payload.(*AssignExpr).Op
}

func (e *Expression) AssignLHS() *Expression {
	if e.Kind != KindAssign && e.Kind != KindUpdateAssign {
		return nil
	}
	return e.payload.(*AssignExpr).LHS
}

func (e *Expression) AssignRHS() *Expression {
	if e.Kind != KindAssign && e.Kind != KindUpdateAssign {
		return nil
	}
	return e.payload.(*AssignExpr).RHS
}

func (e *Expression) BinaryOp() BinaryOp {
	if e.Kind != KindBinary {
		return -1
	}
	return e.payload.(*BinaryExpr).Op
}

func (e *Expression) BinaryLHS() *Expression {
	if e.Kind != KindBinary {
		return nil
	}
	return e.payload.(*BinaryExpr).LHS
}

func (e *Expression) BinaryRHS() *Expression {
	if e.Kind != KindBinary {
		return nil
	}
	return e.payload.(*BinaryExpr).RHS
}

func (e *Expression) NegateArg() *Expression {
	if e.Kind != KindNegate {
		return nil
	}
	return e.payload.(*NegateExpr).Arg
}

func (e *Expression) NumberText() string {
	if e.Kind != KindNumber {
		return ""
	}
	return e.payload.(*NumberExpr).Text
}

func (e *Expression) StringParts() []StringPart {
	if e.Kind != KindString {
		return nil
	}
	return e.payload.(*StringExpr).Parts
}

func (e *Expression) FormatName() string {
	if e.Kind != KindFormat {
		return ""
	}
	return e.payload.(*FormatExpr).Name
}

func (e *Expression) FormatString() *Expression {
	if e.Kind != KindFormat {
		return nil
	}
	return e.payload.(*FormatExpr).String
}

func (e *Expression) BoolValue() bool {
	if e.Kind != KindBool {
		return false
	}
	return e.payload.(*BoolExpr).Value
}

func (e *Expression) VariableName() string {
	if e.Kind != KindVariable {
		return ""
	}
	return e.payload.(*VariableExpr).Name
}

func (e *Expression) ArrayElements() []*Expression {
	if e.Kind != KindArray {
		return nil
	}
	return e.payload.(*ArrayExpr).Elements
}

func (e *Expression) ObjectEntries() []ObjectEntry {
	if e.Kind != KindObject {
		return nil
	}
	return e.payload.(*ObjectExpr).Entries
}

func (e *Expression) FunctionName() string {
	if e.Kind != KindFunctionCall {
		return ""
	}
	return e.payload.(*FunctionCallExpr).Name
}

func (e *Expression) FunctionArgs() []*Expression {
	if e.Kind != KindFunctionCall {
		return nil
	}
	return e.payload.(*FunctionCallExpr).Args
}

func (e *Expression) IfCond() *Expression {
	if e.Kind != KindIf {
		return nil
	}
	return e.payload.(*IfExpr).Cond
}

func (e *Expression) IfThen() *Expression {
	if e.Kind != KindIf {
		return nil
	}
	return e.payload.(*IfExpr).Then
}

func (e *Expression) IfElifs() []ElifBranch {
	if e.Kind != KindIf {
		return nil
	}
	return e.payload.(*IfExpr).Elifs
}

func (e *Expression) IfElse() *Expression {
	if e.Kind != KindIf {
		return nil
	}
	return e.payload.(*IfExpr).Else
}

func (e *Expression) TryBody() *Expression {
	if e.Kind != KindTry {
		return nil
	}
	return e.payload.(*TryExpr).Body
}

func (e *Expression) TryCatch() *Expression {
	if e.Kind != KindTry {
		return nil
	}
	return e.payload.(*TryExpr).Catch
}

func (e *Expression) ReduceSource() *Expression {
	if e.Kind != KindReduce {
		return nil
	}
	return e.payload.(*ReduceExpr).Source
}

func (e *Expression) ReducePattern() *Expression {
	if e.Kind != KindReduce {
		return nil
	}
	return e.payload.(*ReduceExpr).Pattern
}

func (e *Expression) ReduceInit() *Expression {
	if e.Kind != KindReduce {
		return nil
	}
	return e.payload.(*ReduceExpr).Init
}

func (e *Expression) ReduceUpdate() *Expression {
	if e.Kind != KindReduce {
		return nil
	}
	return e.payload.(*ReduceExpr).Update
}

func (e *Expression) ForeachSource() *Expression {
	if e.Kind != KindForeach {
		return nil
	}
	return e.payload.(*ForeachExpr).Source
}

func (e *Expression) ForeachPattern() *Expression {
	if e.Kind != KindForeach {
		return nil
	}
	return e.payload.(*ForeachExpr).Pattern
}

func (e *Expression) ForeachInit() *Expression {
	if e.Kind != KindForeach {
		return nil
	}
	return e.payload.(*ForeachExpr).Init
}

func (e *Expression) ForeachUpdate() *Expression {
	if e.Kind != KindForeach {
		return nil
	}
	return e.payload.(*ForeachExpr).Update
}

func (e *Expression) ForeachExtract() *Expression {
	if e.Kind != KindForeach {
		return nil
	}
	return e.payload.(*ForeachExpr).Extract
}

func (e *Expression) AsSource() *Expression {
	if e.Kind != KindAsBinding {
		return nil
	}
	return e.payload.(*AsBindingExpr).Source
}

func (e *Expression) AsPattern() *Expression {
	if e.Kind != KindAsBinding {
		return nil
	}
	return e.payload.(*AsBindingExpr).Pattern
}

func (e *Expression) AsBody() *Expression {
	if e.Kind != KindAsBinding {
		return nil
	}
	return e.payload.(*AsBindingExpr).Body
}

func (e *Expression) LabelName() string {
	if e.Kind != KindLabel {
		return ""
	}
	return e.payload.(*LabelExpr).Name
}

func (e *Expression) LabelBody() *Expression {
	if e.Kind != KindLabel {
		return nil
	}
	return e.payload.(*LabelExpr).Body
}

func (e *Expression) BreakName() string {
	if e.Kind != KindBreak {
		return ""
	}
	return e.payload.(*BreakExpr).Name
}

func (e *Expression) DefName() string {
	if e.Kind != KindDef {
		return ""
	}
	return e.payload.(*DefExpr).Name
}

func (e *Expression) DefArgs() []FunctionArg {
	if e.Kind != KindDef {
		return nil
	}
	return e.payload.(*DefExpr).Args
}

func (e *Expression) DefBody() *Expression {
	if e.Kind != KindDef {
		return nil
	}
	return e.payload.(*DefExpr).Body
}

func (e *Expression) DefRest() *Expression {
	if e.Kind != KindDef {
		return nil
	}
	return e.payload.(*DefExpr).Rest
}

func (e *Expression) ParenthesizedInner() *Expression {
	if e.Kind != KindParenthesized {
		return nil
	}
	return e.payload.(*ParenthesizedExpr).Inner
}

func (e *Expression) PatternAlternatives() []*Expression {
	if e.Kind != KindPatternAlternative {
		return nil
	}
	return e.payload.(*PatternAlternativeExpr).Patterns
}
