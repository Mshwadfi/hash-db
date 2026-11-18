package sql

import (
	"reflect"
	"testing"
)

func NewTestParser(t *testing.T, input string) *Parser {
	tokens, err := Tokenize(input)
	if err != nil {
		t.Fatalf("Failed to tokenize input: %v", err)
	}
	return &Parser{tokens: tokens, pos: 0}
}

func TestExpecteHelpers(t *testing.T) {
	parser := NewTestParser(t, "FROM users;")
	if err := parser.expectKeyword("FROM"); err != nil {
		t.Fatal(err)
	}

	tableName, err := parser.expectIdentifier()
	if err != nil {
		t.Fatalf("Expected identifier for table name, got error: %v", err)
	}
	if tableName != "users" {
		t.Fatalf("Expected table name 'users', got '%s'", tableName)
	}

	if err := parser.expectSymbol(";"); err != nil {
		t.Fatalf("Expected ; symbol, got error: %v", err)
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
		want    *SelectStatement
	}{
		{"SELECT * FROM users", false, &SelectStatement{TableName: "users"}},
		{"select * from users;", false, &SelectStatement{TableName: "users"}},
		{"SELECT * users", true, nil},
		{"SELECT * FROM", true, nil},
		{"SELECT * FROM users extra", true, nil},
		{"SELECT * FROM users )", true, nil},
		{"SELECT * FROM users ; junk", true, nil},
		{"", true, nil},
		{"SELECT * FROM users WHERE id = 1", false, &SelectStatement{TableName: "users", Where: &WhereClause{ColumnName: "id", Op: OpEq, Value: Literal{Kind: LiteralInt, IntVal: 1}}}},
		{"SELECT * FROM users WHERE name = 'Alice'", false, &SelectStatement{TableName: "users", Where: &WhereClause{ColumnName: "name", Op: OpEq, Value: Literal{Kind: LiteralString, StrVal: "Alice"}}}},
		{"SELECT * FROM users WHERE active = true", false, &SelectStatement{TableName: "users", Where: &WhereClause{ColumnName: "active", Op: OpEq, Value: Literal{Kind: LiteralBool, BoolVal: true}}}},
		// every comparison operator
		{"SELECT * FROM users WHERE age != 5", false, selectWhere("age", OpNeq, intLit(5))},
		{"SELECT * FROM users WHERE age < 5", false, selectWhere("age", OpLt, intLit(5))},
		{"SELECT * FROM users WHERE age > 5", false, selectWhere("age", OpGt, intLit(5))},
		{"SELECT * FROM users WHERE age <= 5", false, selectWhere("age", OpLte, intLit(5))},
		{"SELECT * FROM users WHERE age >= 5", false, selectWhere("age", OpGte, intLit(5))},
		// remaining literal kinds
		{"SELECT * FROM users WHERE active = FALSE", false, selectWhere("active", OpEq, Literal{Kind: LiteralBool, BoolVal: false})},
		{"SELECT * FROM users WHERE deleted = NULL", false, selectWhere("deleted", OpEq, Literal{Kind: LiteralNull})},
		{"SELECT * FROM users WHERE name = ''", false, selectWhere("name", OpEq, Literal{Kind: LiteralString, StrVal: ""})},
		{"SELECT * FROM users WHERE age >= 18;", false, selectWhere("age", OpGte, intLit(18))},
		// malformed WHERE clauses
		{"SELECT * FROM users WHERE", true, nil},
		{"SELECT * FROM users WHERE age", true, nil},
		{"SELECT * FROM users WHERE age >", true, nil},
		{"SELECT * FROM users WHERE age > users", true, nil},
		{"SELECT * FROM users WHERE age > 1.5", true, nil},
		{"SELECT * FROM users WHERE 5 = age", true, nil},
		{"SELECT * FROM users WHERE age '=' 5", true, nil},
		{"SELECT * FROM users WHERE age = 5 extra", true, nil},
		{"SELECT * FROM users WHERE age = 99999999999999999999", true, nil},
		// {"SELECT * FROM users WHERE deleted IS NULL", false, &SelectStatement{TableName: "users", Where: &WhereClause{ColumnName: "deleted", Op: OpIs, Value: Literal{Kind: LiteralNull, NullVal: true}}}},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			stmt, err := Parse(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %+v", stmt)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			// sel, ok := stmt.(*SelectStatement)
			if !reflect.DeepEqual(stmt, tc.want) {
				t.Fatalf("got %+v, want %+v", stmt, tc.want)
			}
		})
	}
}

func intLit(n int64) Literal {
	return Literal{Kind: LiteralInt, IntVal: n}
}

func selectWhere(column string, op CompareOp, value Literal) *SelectStatement {
	return &SelectStatement{TableName: "users", Where: &WhereClause{ColumnName: column, Op: op, Value: value}}
}
