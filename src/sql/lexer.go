package sql

import (
	"fmt"
	"strings"
)

type TokenType int

const (
	// Special tokens
	TokenEOF TokenType = iota
	TokenIdent
	TokenNumber
	TokenString
	TokenSymbol
	TokenKeyword
)

type Token struct {
	Type    TokenType
	Literal string
	Pos     int
}

var keywords = map[string]TokenType{
	"SELECT":  TokenKeyword,
	"FROM":    TokenKeyword,
	"WHERE":   TokenKeyword,
	"INSERT":  TokenKeyword,
	"UPDATE":  TokenKeyword,
	"DELETE":  TokenKeyword,
	"CREATE":  TokenKeyword,
	"TABLE":   TokenKeyword,
	"DROP":    TokenKeyword,
	"INTO":    TokenKeyword,
	"VALUES":  TokenKeyword,
	"SET":     TokenKeyword,
	"NOT":     TokenKeyword,
	"TRUE":    TokenKeyword,
	"FALSE":   TokenKeyword,
	"INT":     TokenKeyword,
	"VARCHAR": TokenKeyword,
	"BIGINT":  TokenKeyword,
	"BOOLEAN": TokenKeyword,
	"NULL":    TokenKeyword,
}

type LexError struct {
	Message string
	Pos     int
}

func (e *LexError) Error() string {
	return fmt.Sprintf("LexError at position %d: %s", e.Pos, e.Message)
}

func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isSymbol(ch byte) bool {
	operators := "()*=<>!,;"
	return strings.ContainsRune(operators, rune(ch))
}
func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func Tokenize(input string) ([]Token, error) {
	var tokens = []Token{}
	pos := 0
	runes := []rune(input)
	n := len(runes)

	for pos < n {
		ch := runes[pos]

		switch {
		case isWhitespace(byte(ch)):
			pos++
		case isSymbol(byte(ch)):
			switch {
			case ch == '!':
				if pos+1 < n && runes[pos+1] == '=' {
					tokens = append(tokens, Token{Type: TokenSymbol, Literal: "!=", Pos: pos})
					pos += 2
				} else {
					return nil, &LexError{Message: fmt.Sprintf("Unexpected character: %c", ch), Pos: pos}
				}
			case ch == '<' || ch == '>':
				if pos+1 < n && runes[pos+1] == '=' {
					tokens = append(tokens, Token{Type: TokenSymbol, Literal: string(ch) + "=", Pos: pos})
					pos += 2
				} else {
					tokens = append(tokens, Token{Type: TokenSymbol, Literal: string(ch), Pos: pos})
					pos++
				}
			default:
				tokens = append(tokens, Token{Type: TokenSymbol, Literal: string(ch), Pos: pos})
				pos++
			}

		case ch == '\'':
			token, newPos, err := LexString(runes, pos)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, token)
			pos = newPos

		case isDigit(byte(ch)):
			token, newPos, err := LexNumber(runes, pos)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, token)
			pos = newPos

		case isLetter(byte(ch)):
			token, newPos := lexIdentOrKeyword(runes, pos)
			tokens = append(tokens, token)
			pos = newPos

		default:
			return nil, &LexError{Message: fmt.Sprintf("Unexpected character: %c", ch), Pos: pos}
		}
	}

	return append(tokens, Token{Type: TokenEOF, Literal: "", Pos: pos}), nil
}

func LexString(runes []rune, start int) (Token, int, error) {
	pos := start
	n := len(runes)
	if pos >= n || runes[pos] != '\'' {
		return Token{}, pos, &LexError{Message: "Expected opening quote for string", Pos: pos}
	}
	pos++ // Skip the opening quote

	var sb strings.Builder
	for pos < n {
		ch := runes[pos]
		if ch == '\'' {
			pos++ // Skip the closing quote
			return Token{Type: TokenString, Literal: sb.String(), Pos: start}, pos, nil
		}
		sb.WriteRune(ch)
		pos++
	}
	return Token{}, start, &LexError{Message: "Unterminated string literal", Pos: start}
}

func LexNumber(runes []rune, start int) (Token, int, error) {
	pos := start
	n := len(runes)
	var sb strings.Builder

	for pos < n && isDigit(byte(runes[pos])) {
		sb.WriteRune(runes[pos])
		pos++
	}

	// later we may support floating point numbers
	if pos < n && runes[pos] == '.' {
		sb.WriteRune('.')
		pos++
		for pos < n && isDigit(byte(runes[pos])) {
			sb.WriteRune(runes[pos])
			pos++
		}
	}

	return Token{Type: TokenNumber, Literal: sb.String(), Pos: start}, pos, nil
}

func lexIdentOrKeyword(runes []rune, start int) (Token, int) {
	pos := start
	n := len(runes)
	for pos < n && (isLetter(byte(runes[pos])) || isDigit(byte(runes[pos])) || runes[pos] == '_') {
		pos++
	}

	word := string(runes[start:pos])
	upper := strings.ToUpper(word)

	if keywords[upper] == TokenKeyword {
		return Token{Type: TokenKeyword, Literal: upper, Pos: start}, pos
	}
	return Token{Type: TokenIdent, Literal: word, Pos: start}, pos
}
