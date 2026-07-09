package jq

type Span struct {
	Start int
	End   int
}

type Pos struct {
	Offset int
	Line   int
	Column int
}
