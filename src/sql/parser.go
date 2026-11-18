package sql

import (
	"fmt"
	"strconv"
)

type ParseError struct {
	Message string
	Pos     int
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("ParseError at position %d: %s", e.Pos, e.Message)
}

type Parser struct {
	tokens []Token
	pos    int
}

// peak functions returns the current token without advancing the position.
func (p *Parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF, Literal: "", Pos: p.pos}
	}
	return p.tokens[p.pos]
}

// next function returns the current token and advances the position.
func (p *Parser) next() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF, Literal: "", Pos: p.pos}
	}
	token := p.tokens[p.pos]
	p.pos++
	return token
}

// expect keyword, returns error if the current token is not the expected keyword.
func (p *Parser) expectKeyword(kw string) error {
	token := p.next()
	if token.Type != TokenKeyword || token.Literal != kw {
		return &ParseError{Message: fmt.Sprintf("Expected keyword %s, got %s", kw, token.Literal), Pos: token.Pos}
	}
	return nil
}

// expect symbol, returns error if the current token is not the expected symbol.
func (p *Parser) expectSymbol(sym string) error {
	token := p.next()
	if token.Type != TokenSymbol || token.Literal != sym {
		return &ParseError{Message: fmt.Sprintf("Expected symbol %s, got %s", sym, token.Literal), Pos: token.Pos}
	}
	return nil
}

// expect identifier, returns error if the current token is not an identifier.
func (p *Parser) expectIdentifier() (string, error) {
	token := p.next()
	if token.Type != TokenIdent {
		return "", &ParseError{Message: fmt.Sprintf("Expected identifier, got %s", token.Literal), Pos: token.Pos}
	}
	return token.Literal, nil
}

// expect end, returns error if the current token is not EOF or ;.
func (p *Parser) expectEnd() error {
	if tok := p.peek(); tok.Type == TokenSymbol && tok.Literal == ";" {
		p.next()
	}
	tok := p.next()
	if tok.Type != TokenEOF {
		return &ParseError{Message: fmt.Sprintf("unexpected %q after end of statement", tok.Literal), Pos: tok.Pos}
	}
	return nil
}

// parse function parses the tokens into a Statement.
func Parse(input string) (Statement, error) {
	tokens, err := Tokenize(input)
	if err != nil {
		return nil, err
	}

	p := &Parser{tokens: tokens}

	var stmt Statement
	token := p.peek()

	switch token.Type {
	case TokenKeyword:
		switch token.Literal {
		case "SELECT":
			s, err := p.parseSelect()
			if err != nil {
				return nil, err
			}
			stmt = s
		default:
			return nil, &ParseError{Message: fmt.Sprintf("Unexpected keyword: %s", token.Literal), Pos: token.Pos}
		}
		if err := p.expectEnd(); err != nil {
			return nil, err
		}
		return stmt, nil
	default:
		return nil, &ParseError{Message: fmt.Sprintf("Unexpected token: %s", token.Literal), Pos: token.Pos}
	}
}

// parse select statement, returns error if the current token is not a SELECT keyword.
// grammer: selet * from Table name
func (p *Parser) parseSelect() (*SelectStatement, error) {
	if err := p.expectKeyword("SELECT"); err != nil {
		return nil, err
	}
	if err := p.expectSymbol("*"); err != nil {
		return nil, err
	}
	if err := p.expectKeyword("FROM"); err != nil {
		return nil, err
	}
	tableName, err := p.expectIdentifier()
	if err != nil {
		return nil, err
	}

	whereClause, err := parseWhereClause(p)
	if err != nil {
		return nil, err
	}

	if whereClause != nil {
		return &SelectStatement{TableName: tableName, Where: whereClause}, nil
	}
	return &SelectStatement{TableName: tableName}, nil
}

// parse select helpers
func parseLiteral(p *Parser) (Literal, error) {
	token := p.next()
	switch token.Type {
	case TokenNumber:
		intVal, err := strconv.ParseInt(token.Literal, 10, 64)
		if err != nil {
			return Literal{}, &ParseError{Message: fmt.Sprintf("Failed to parse integer: %s", token.Literal), Pos: token.Pos}
		}
		return Literal{Kind: LiteralInt, IntVal: intVal}, nil
	case TokenString:
		return Literal{Kind: LiteralString, StrVal: token.Literal}, nil
	case TokenKeyword:
		if token.Literal == "TRUE" || token.Literal == "FALSE" {
			boolVal, err := strconv.ParseBool(token.Literal)
			if err != nil {
				return Literal{}, &ParseError{Message: fmt.Sprintf("Failed to parse boolean: %s", token.Literal), Pos: token.Pos}
			}
			return Literal{Kind: LiteralBool, BoolVal: boolVal}, nil
		} else if token.Literal == "NULL" {
			return Literal{Kind: LiteralNull}, nil
		}
	}
	return Literal{}, &ParseError{Message: fmt.Sprintf("Unexpected literal: %s", token.Literal), Pos: token.Pos}
}

// parse compare operator, returns error if the current token is not a compare operator.
var operationsMap = map[string]CompareOp{
	"=":  OpEq,
	"!=": OpNeq,
	"<":  OpLt,
	">":  OpGt,
	"<=": OpLte,
	">=": OpGte,
}

func parseCompareOp(p *Parser) (CompareOp, error) {
	token := p.next()
	if token.Type != TokenSymbol {
		return 0, &ParseError{Message: fmt.Sprintf("Expected compare operator, got %s", token.Literal), Pos: token.Pos}
	}
	if op, ok := operationsMap[token.Literal]; ok {
		return op, nil
	}
	return 0, &ParseError{Message: fmt.Sprintf("Unexpected compare operator: %s", token.Literal), Pos: token.Pos}
}

// parse where clause, returns error if the current token is not a WHERE keyword.
func parseWhereClause(p *Parser) (*WhereClause, error) {
	token := p.peek()
	if token.Type != TokenKeyword || token.Literal != "WHERE" {
		return nil, nil
	}

	if err := p.expectKeyword("WHERE"); err != nil {
		return nil, err
	}
	columnName, err := p.expectIdentifier()
	if err != nil {
		return nil, err
	}
	op, err := parseCompareOp(p)
	if err != nil {
		return nil, err
	}
	value, err := parseLiteral(p)
	if err != nil {
		return nil, err
	}
	return &WhereClause{ColumnName: columnName, Op: op, Value: value}, nil
}
