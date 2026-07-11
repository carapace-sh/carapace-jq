package jq

// ExpectedToken represents a type of token expected at a completion position.
type ExpectedToken int

const (
	ExpectedExpression ExpectedToken = iota
	ExpectedOperator
	ExpectedPipe
	ExpectedColon
	ExpectedSemicolon
	ExpectedClosingParen
	ExpectedClosingBracket
	ExpectedClosingBrace
	ExpectedComma
	ExpectedStringClose
	ExpectedKeyword // then, else, elif, end, catch, as
	ExpectedFormatName
	ExpectedDollar
	ExpectedDefColon
	ExpectedDefSemicolon
	ExpectedOpeningParen
)

func (t ExpectedToken) String() string {
	switch t {
	case ExpectedExpression:
		return "Expression"
	case ExpectedOperator:
		return "Operator"
	case ExpectedPipe:
		return "|"
	case ExpectedColon:
		return ":"
	case ExpectedSemicolon:
		return ";"
	case ExpectedClosingParen:
		return ")"
	case ExpectedClosingBracket:
		return "]"
	case ExpectedClosingBrace:
		return "}"
	case ExpectedComma:
		return ","
	case ExpectedStringClose:
		return "quote"
	case ExpectedKeyword:
		return "Keyword"
	case ExpectedFormatName:
		return "FormatName"
	case ExpectedDollar:
		return "$"
	case ExpectedDefColon:
		return ":"
	case ExpectedDefSemicolon:
		return ";"
	case ExpectedOpeningParen:
		return "("
	}
	return "Unknown"
}

func (t ExpectedToken) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// ValidOperator represents an operator that could be valid at a completion position.
type ValidOperator struct {
	Op          string `json:"op"`
	Description string `json:"description"`
}

// FunctionContext provides details about an ongoing function call at the completion position.
type FunctionContext struct {
	Name     string        `json:"name"`
	Args     []*Expression `json:"args,omitempty"`
	ArgIndex int           `json:"argIndex"`
}

// ObjectContext provides details about object construction at the completion position.
type ObjectContext struct {
	// InKey is true when completing an object key
	InKey bool `json:"inKey"`
	// InValue is true when completing an object value (after colon)
	InValue bool `json:"inValue"`
	// KeyName is the key name for the current entry (if known)
	KeyName string `json:"keyName,omitempty"`
}

// ReduceContext provides details about reduce/foreach at the completion position.
type ReduceContext struct {
	// IsForeach is true for foreach, false for reduce
	IsForeach bool `json:"isForeach"`
	// Section indicates which part of the (init; update; extract) is being completed
	Section string `json:"section"` // "init", "update", "extract"
}

// CompletionContext describes what is expected at the completion position.
type CompletionContext struct {
	ExpectedTokens []ExpectedToken `json:"expectedTokens"`

	ValidOperators []ValidOperator `json:"validOperators,omitempty"`

	// PartialIdent is the partial identifier being typed (e.g. "ma" in "ma")
	PartialIdent string `json:"partialIdent,omitempty"`

	// PartialString is the partial string literal content being typed
	// without the surrounding quotes
	PartialString string `json:"partialString,omitempty"`

	// StringQuote is the quote character used for the partial string
	StringQuote rune `json:"stringQuote,omitempty"`

	// Function is non-nil when the cursor is inside a function call
	Function *FunctionContext `json:"function,omitempty"`

	// Object is non-nil when the cursor is inside object construction
	Object *ObjectContext `json:"object,omitempty"`

	// Reduce is non-nil when the cursor is inside reduce/foreach parens
	Reduce *ReduceContext `json:"reduce,omitempty"`

	// InFormat is true when completing a @format name
	InFormat bool `json:"inFormat"`

	// PartialFormat is the partial @format name being typed
	PartialFormat string `json:"partialFormat,omitempty"`

	// InAsPattern is true when completing a destructuring pattern
	InAsPattern bool `json:"inAsPattern"`

	// InDef is true when completing inside a def
	InDef bool `json:"inDef"`

	// DefName is the partial function name being defined
	DefName string `json:"defName,omitempty"`

	// AfterDot is true when the cursor is after a dot (field access)
	AfterDot bool `json:"afterDot"`

	// InStringInterp is true when the cursor is inside \(...) interpolation
	InStringInterp bool `json:"inStringInterp"`
}
