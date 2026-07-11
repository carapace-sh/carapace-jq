package jq

import (
	"unicode/utf8"
)

func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\r' || r == '\n' || r == '\x0c'
}

func isIdentifierStart(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
}

func isIdentifierPart(r rune) bool {
	return isIdentifierStart(r) || (r >= '0' && r <= '9')
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isHexDigit(r rune) bool {
	return isDigit(r) || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

func hexVal(r rune) rune {
	if r >= '0' && r <= '9' {
		return r - '0'
	}
	if r >= 'a' && r <= 'f' {
		return r - 'a' + 10
	}
	if r >= 'A' && r <= 'F' {
		return r - 'A' + 10
	}
	return 0
}

// jq keywords — these are NOT valid function names
var jqKeywords = map[string]bool{
	"if":      true,
	"then":    true,
	"else":    true,
	"elif":    true,
	"end":     true,
	"try":     true,
	"catch":   true,
	"reduce":  true,
	"foreach": true,
	"as":      true,
	"def":     true,
	"import":  true,
	"include": true,
	"module":  true,
	"label":   true,
	"break":   true,
	"and":     true,
	"or":      true,
	"not":     true,
	"true":    true,
	"false":   true,
	"null":    true,
	"__loc__": true,
}

func isKeyword(s string) bool {
	return jqKeywords[s]
}

func isFunctionName(s string) bool {
	if len(s) == 0 {
		return false
	}
	if isKeyword(s) {
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

// scanIdentifier scans an identifier starting at the current position.
// Returns the identifier text and true if found.
func (p *parser) scanIdentifier() (string, bool) {
	if p.atEnd() {
		return "", false
	}
	if !isIdentifierStart(p.peek()) {
		return "", false
	}
	start := p.pos
	for !p.atEnd() && isIdentifierPart(p.peek()) {
		p.advance()
	}
	return p.input[start:p.pos], true
}

// scanNumber scans a jq number literal. Returns the literal text and true if found.
// jq number syntax: [0-9]+('.'[0-9]+)?([eE][+-]?[0-9]+)?
// Also handles leading '.' for decimals like .5 (though jq requires a digit before .)
func (p *parser) scanNumber() (string, bool) {
	if p.atEnd() || !isDigit(p.peek()) {
		return "", false
	}
	start := p.pos
	// integer part
	for !p.atEnd() && isDigit(p.peek()) {
		p.advance()
	}
	// fractional part
	if !p.atEnd() && p.peek() == '.' {
		// peek ahead to make sure next char is a digit
		if p.pos+1 < len(p.input) && isDigit(rune(p.input[p.pos+1])) {
			p.advance() // consume .
			for !p.atEnd() && isDigit(p.peek()) {
				p.advance()
			}
		}
	}
	// exponent part
	if !p.atEnd() && (p.peek() == 'e' || p.peek() == 'E') {
		saved := p.pos
		p.advance() // consume e/E
		if !p.atEnd() && (p.peek() == '+' || p.peek() == '-') {
			p.advance()
		}
		if p.atEnd() || !isDigit(p.peek()) {
			// not a valid exponent, backtrack
			p.pos = saved
		} else {
			for !p.atEnd() && isDigit(p.peek()) {
				p.advance()
			}
		}
	}
	return p.input[start:p.pos], true
}

// scanFormatName scans a @format name. Returns the name (without @) and true if found.
func (p *parser) scanFormatName() (string, bool) {
	if p.atEnd() || p.peek() != '@' {
		return "", false
	}
	saved := p.pos
	p.advance() // consume @
	start := p.pos
	for !p.atEnd() && isIdentifierPart(p.peek()) {
		p.advance()
	}
	if p.pos == start {
		// @ with no name — backtrack
		p.pos = saved
		return "", false
	}
	return p.input[start:p.pos], true
}

// parseStringLiteralValue parses a double-quoted string literal with escape
// sequences and string interpolation. Returns a list of StringPart.
// The caller must have verified that p.peek() == '"'.
func (p *parser) parseStringLiteralValue() ([]StringPart, error) {
	if p.peek() != '"' {
		return nil, p.syntaxError("expected string literal")
	}
	p.advance() // consume opening "

	var parts []StringPart
	var text []rune

	for {
		if p.atEnd() {
			return nil, p.syntaxError("unterminated string literal")
		}
		ch := p.peek()
		if ch == '"' {
			p.advance() // consume closing "
			if len(text) > 0 {
				parts = append(parts, StringText{Text: string(text)})
			}
			return parts, nil
		}
		if ch == '\\' {
			p.advance() // consume backslash
			if p.atEnd() {
				return nil, p.syntaxError("unterminated escape sequence")
			}
			next := p.peek()
			if next == '(' {
				// String interpolation \(expr)
				if len(text) > 0 {
					parts = append(parts, StringText{Text: string(text)})
					text = text[:0]
				}
				p.advance() // consume (
				// Parse the inner expression as a pipe-level expression (no commas)
				p.skipWhitespaceAndComments()
				inner, err := p.parseExp()
				if err != nil {
					return nil, err
				}
				p.skipWhitespaceAndComments()
				if p.atEnd() || p.peek() != ')' {
					return nil, p.syntaxError("expected ')' in string interpolation")
				}
				p.advance() // consume )
				parts = append(parts, StringInterp{Expr: inner})
				continue
			}
			escaped := p.advance()
			switch escaped {
			case '"':
				text = append(text, '"')
			case '\\':
				text = append(text, '\\')
			case '/':
				text = append(text, '/')
			case 'b':
				text = append(text, '\b')
			case 'f':
				text = append(text, '\f')
			case 'n':
				text = append(text, '\n')
			case 'r':
				text = append(text, '\r')
			case 't':
				text = append(text, '\t')
			case 'u':
				// Unicode escape: \uXXXX
				var runes []rune
				for range 4 {
					if p.atEnd() || !isHexDigit(p.peek()) {
						return nil, p.syntaxError("invalid unicode escape sequence")
					}
					runes = append(runes, p.advance())
				}
				var codepoint rune
				for _, r := range runes {
					codepoint = codepoint<<4 | hexVal(r)
				}
				// Handle surrogate pairs
				if codepoint >= 0xD800 && codepoint <= 0xDBFF {
					// High surrogate — look for low surrogate
					if !p.atEnd() && p.peek() == '\\' {
						saved := p.pos
						p.advance()
						if !p.atEnd() && p.peek() == 'u' {
							p.advance()
							var lo []rune
							valid := true
							for range 4 {
								if p.atEnd() || !isHexDigit(p.peek()) {
									valid = false
									break
								}
								lo = append(lo, p.advance())
							}
							if valid {
								var loCodepoint rune
								for _, r := range lo {
									loCodepoint = loCodepoint<<4 | hexVal(r)
								}
								if loCodepoint >= 0xDC00 && loCodepoint <= 0xDFFF {
									codepoint = 0x10000 + (codepoint-0xD800)<<10 + (loCodepoint - 0xDC00)
								} else {
									// Not a low surrogate, backtrack
									p.pos = saved
								}
							} else {
								p.pos = saved
							}
						} else {
							p.pos = saved
						}
					}
				}
				text = append(text, codepoint)
			default:
				return nil, p.syntaxErrorf("invalid escape sequence \\%c", escaped)
			}
		} else {
			r, w := utf8.DecodeRuneInString(p.input[p.pos:])
			text = append(text, r)
			p.pos += w
			p.lastContent = p.pos
		}
	}
}
