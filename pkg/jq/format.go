package jq

import (
	"fmt"
	"strings"
)

// Format returns a normalized string representation of the expression.
// It produces a canonical form that parses to the same AST.
func Format(expr *Expression) string {
	return formatExpr(expr, precPipe)
}

func formatExpr(e *Expression, ctxPrec int) string {
	if e == nil {
		return ""
	}
	result := formatExprInner(e)
	if needsParens(e, ctxPrec) {
		return "(" + result + ")"
	}
	return result
}

func exprPrec(e *Expression) int {
	switch e.Kind {
	case KindPipe:
		return precPipe
	case KindComma:
		return precComma
	case KindAlternative:
		return precAlternative
	case KindAssign, KindUpdateAssign:
		return precAssign
	case KindBinary:
		b := e.payload.(*BinaryExpr)
		switch b.Op {
		case OpOr:
			return precOr
		case OpAnd:
			return precAnd
		case OpEq, OpNe, OpLt, OpGt, OpLe, OpGe:
			return precCompare
		case OpAdd, OpSub:
			return precAddSub
		case OpMul, OpDiv, OpMod:
			return precMulDiv
		}
	case KindNegate:
		return precPrefix
	case KindParenthesized:
		return precPrimary
	default:
		return precPrimary
	}
	return precPrimary
}

func needsParens(e *Expression, ctxPrec int) bool {
	return exprPrec(e) < ctxPrec
}

func formatExprInner(e *Expression) string {
	switch e.Kind {
	case KindIdentity:
		return "."
	case KindRecursiveDescent:
		return ".."
	case KindField:
		f := e.payload.(*FieldExpr)
		var s string
		if f.Base != nil && f.Base.Kind != KindIdentity {
			s = formatExpr(f.Base, precPostfix) + formatField(f.Name)
		} else {
			s = formatField(f.Name)
		}
		if f.Optional {
			s += "?"
		}
		return s
	case KindIndex:
		ix := e.payload.(*IndexExpr)
		var s string
		if ix.Base != nil && ix.Base.Kind != KindIdentity {
			s = formatExpr(ix.Base, precPostfix) + fmt.Sprintf("[%s]", formatExpr(ix.Index, precPipe))
		} else {
			s = fmt.Sprintf(".[%s]", formatExpr(ix.Index, precPipe))
		}
		return s
	case KindSlice:
		sl := e.payload.(*SliceExpr)
		var s string
		var prefix string
		if sl.Base != nil && sl.Base.Kind != KindIdentity {
			prefix = formatExpr(sl.Base, precPostfix)
			s = prefix + "["
		} else {
			s = ".["
		}
		if sl.Start != nil {
			s += formatExpr(sl.Start, precPipe)
		}
		s += ":"
		if sl.End != nil {
			s += formatExpr(sl.End, precPipe)
		}
		s += "]"
		return s
	case KindIterator:
		ie := e.payload.(*IteratorExpr)
		var s string
		if ie.Base != nil && ie.Base.Kind != KindIdentity {
			s = formatExpr(ie.Base, precPostfix) + "[]"
		} else {
			s = ".[]"
		}
		return s
	case KindOptional:
		return formatExpr(e.payload.(*OptionalExpr).Arg, precPostfix) + "?"
	case KindPipe:
		pe := e.payload.(*PipeExpr)
		return fmt.Sprintf("%s | %s", formatExpr(pe.LHS, precPipe+1), formatExpr(pe.RHS, precPipe))
	case KindComma:
		ce := e.payload.(*CommaExpr)
		return fmt.Sprintf("%s, %s", formatExpr(ce.LHS, precComma), formatExpr(ce.RHS, precComma+1))
	case KindAlternative:
		ae := e.payload.(*AlternativeExpr)
		return fmt.Sprintf("%s // %s", formatExpr(ae.LHS, precAlternative+1), formatExpr(ae.RHS, precAlternative))
	case KindAssign:
		ae := e.payload.(*AssignExpr)
		return fmt.Sprintf("%s %s %s", formatExpr(ae.LHS, precAssign+1), ae.Op.String(), formatExpr(ae.RHS, precAssign+1))
	case KindUpdateAssign:
		ae := e.payload.(*AssignExpr)
		return fmt.Sprintf("%s %s %s", formatExpr(ae.LHS, precAssign+1), ae.Op.String(), formatExpr(ae.RHS, precAssign+1))
	case KindBinary:
		b := e.payload.(*BinaryExpr)
		opStr := b.Op.String()
		prec := exprPrec(e)
		return fmt.Sprintf("%s %s %s", formatExpr(b.LHS, prec), opStr, formatExpr(b.RHS, prec+1))
	case KindNegate:
		ne := e.payload.(*NegateExpr)
		return fmt.Sprintf("-%s", formatExpr(ne.Arg, precPrefix))
	case KindNumber:
		return e.payload.(*NumberExpr).Text
	case KindString:
		return formatString(e.payload.(*StringExpr))
	case KindFormat:
		fe := e.payload.(*FormatExpr)
		s := "@" + fe.Name
		if fe.String != nil {
			s += " " + formatString(fe.String.payload.(*StringExpr))
		}
		return s
	case KindBool:
		if e.payload.(*BoolExpr).Value {
			return "true"
		}
		return "false"
	case KindNull:
		return "null"
	case KindVariable:
		return "$" + e.payload.(*VariableExpr).Name
	case KindArray:
		ae := e.payload.(*ArrayExpr)
		parts := make([]string, len(ae.Elements))
		for i, el := range ae.Elements {
			parts[i] = formatExpr(el, precPipe)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case KindObject:
		return formatObject(e.payload.(*ObjectExpr))
	case KindFunctionCall:
		fc := e.payload.(*FunctionCallExpr)
		if len(fc.Args) == 0 {
			return fc.Name
		}
		parts := make([]string, len(fc.Args))
		for i, a := range fc.Args {
			parts[i] = formatExpr(a, precPipe)
		}
		return fmt.Sprintf("%s(%s)", fc.Name, strings.Join(parts, ", "))
	case KindIf:
		return formatIf(e.payload.(*IfExpr))
	case KindTry:
		te := e.payload.(*TryExpr)
		s := "try " + formatExpr(te.Body, precPostfix)
		if te.Catch != nil {
			s += " catch " + formatExpr(te.Catch, precPostfix)
		}
		return s
	case KindReduce:
		re := e.payload.(*ReduceExpr)
		return fmt.Sprintf("reduce %s as %s (%s; %s)",
			formatExpr(re.Source, precPipe),
			formatPattern(re.Pattern),
			formatExpr(re.Init, precPipe),
			formatExpr(re.Update, precPipe))
	case KindForeach:
		fe := e.payload.(*ForeachExpr)
		s := fmt.Sprintf("foreach %s as %s (%s; %s",
			formatExpr(fe.Source, precPipe),
			formatPattern(fe.Pattern),
			formatExpr(fe.Init, precPipe),
			formatExpr(fe.Update, precPipe))
		if fe.Extract != nil {
			s += "; " + formatExpr(fe.Extract, precPipe)
		}
		s += ")"
		return s
	case KindAsBinding:
		ae := e.payload.(*AsBindingExpr)
		return fmt.Sprintf("%s as %s | %s",
			formatExpr(ae.Source, precPipe),
			formatPattern(ae.Pattern),
			formatExpr(ae.Body, precPipe))
	case KindLabel:
		le := e.payload.(*LabelExpr)
		return fmt.Sprintf("label $%s | %s", le.Name, formatExpr(le.Body, precPipe))
	case KindBreak:
		return "break $" + e.payload.(*BreakExpr).Name
	case KindDef:
		de := e.payload.(*DefExpr)
		s := "def " + de.Name
		if len(de.Args) > 0 {
			argParts := make([]string, len(de.Args))
			for i, a := range de.Args {
				if a.IsValue {
					argParts[i] = "$" + a.Name
				} else {
					argParts[i] = a.Name
				}
			}
			s += "(" + strings.Join(argParts, "; ") + ")"
		}
		s += ": " + formatExpr(de.Body, precPipe) + "; " + formatExpr(de.Rest, precPipe)
		return s
	case KindParenthesized:
		return "(" + formatExpr(e.payload.(*ParenthesizedExpr).Inner, precPipe) + ")"
	case KindPatternAlternative:
		pa := e.payload.(*PatternAlternativeExpr)
		parts := make([]string, len(pa.Patterns))
		for i, p := range pa.Patterns {
			parts[i] = formatPattern(p)
		}
		return strings.Join(parts, " ?// ")
	}
	return ""
}

func formatField(name string) string {
	// Use .foo for simple identifiers, ."foo" for complex
	if isIdentifier(name) && !isKeyword(name) {
		return "." + name
	}
	return fmt.Sprintf(".%q", name)
}
func formatString(se *StringExpr) string {
	var sb strings.Builder
	sb.WriteByte('"')
	for _, part := range se.Parts {
		switch pt := part.(type) {
		case StringText:
			sb.WriteString(escapeStringText(pt.Text))
		case StringInterp:
			sb.WriteString("\\(")
			sb.WriteString(formatExpr(pt.Expr, precPipe))
			sb.WriteString(")")
		}
	}
	sb.WriteByte('"')
	return sb.String()
}

func escapeStringText(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch r {
		case '"':
			sb.WriteString("\\\"")
		case '\\':
			sb.WriteString("\\\\")
		case '\n':
			sb.WriteString("\\n")
		case '\r':
			sb.WriteString("\\r")
		case '\t':
			sb.WriteString("\\t")
		case '\b':
			sb.WriteString("\\b")
		case '\f':
			sb.WriteString("\\f")
		default:
			if r < 0x20 {
				sb.WriteString(fmt.Sprintf("\\u%04x", r))
			} else {
				sb.WriteRune(r)
			}
		}
	}
	return sb.String()
}

func formatObject(oe *ObjectExpr) string {
	parts := make([]string, len(oe.Entries))
	for i, entry := range oe.Entries {
		switch entry.KeyKind {
		case ObjectKeyBare:
			parts[i] = fmt.Sprintf("%s: %s", entry.KeyName, formatExpr(entry.Value, precPipe))
		case ObjectKeyString:
			parts[i] = fmt.Sprintf("%q: %s", entry.KeyName, formatExpr(entry.Value, precPipe))
		case ObjectKeyVariable:
			parts[i] = fmt.Sprintf("$%s: %s", entry.KeyName, formatExpr(entry.Value, precPipe))
		case ObjectKeyShorthand:
			if entry.Value != nil && entry.Value.Kind == KindVariable {
				parts[i] = "$" + entry.KeyName
			} else {
				parts[i] = entry.KeyName
			}
		case ObjectKeyExpression:
			parts[i] = fmt.Sprintf("(%s): %s", formatExpr(entry.Key, precPipe), formatExpr(entry.Value, precPipe))
		}
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func formatIf(ie *IfExpr) string {
	var s strings.Builder
	s.WriteString("if " + formatExpr(ie.Cond, precPipe) + " then " + formatExpr(ie.Then, precPipe))
	for _, elif := range ie.Elifs {
		s.WriteString(" elif " + formatExpr(elif.Cond, precPipe) + " then " + formatExpr(elif.Then, precPipe))
	}
	if ie.Else != nil {
		s.WriteString(" else " + formatExpr(ie.Else, precPipe))
	}
	s.WriteString(" end")
	return s.String()
}

func formatPattern(e *Expression) string {
	if e == nil {
		return ""
	}
	switch e.Kind {
	case KindVariable:
		return "$" + e.payload.(*VariableExpr).Name
	case KindArray:
		ae := e.payload.(*ArrayExpr)
		parts := make([]string, len(ae.Elements))
		for i, el := range ae.Elements {
			parts[i] = formatPattern(el)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case KindObject:
		oe := e.payload.(*ObjectExpr)
		parts := make([]string, len(oe.Entries))
		for i, entry := range oe.Entries {
			if entry.KeyKind == ObjectKeyShorthand && entry.Value != nil && entry.Value.Kind == KindVariable {
				parts[i] = "$" + entry.KeyName
			} else {
				parts[i] = entry.KeyName + ": " + formatPattern(entry.Value)
			}
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case KindPatternAlternative:
		pa := e.payload.(*PatternAlternativeExpr)
		parts := make([]string, len(pa.Patterns))
		for i, p := range pa.Patterns {
			parts[i] = formatPattern(p)
		}
		return strings.Join(parts, " ?// ")
	}
	return formatExpr(e, precPrimary)
}

func isIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i, ch := range s {
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
