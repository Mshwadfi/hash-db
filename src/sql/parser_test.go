package sql

import (
	"errors"
	"math"
	"mydb/src/tuple"
	"reflect"
	"strings"
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
		{"SELECT * FROM users WHERE age > -1", false, selectWhere("age", OpGt, intLit(-1))},
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

func strLit(s string) Literal {
	return Literal{Kind: LiteralString, StrVal: s}
}

func boolLit(b bool) Literal {
	return Literal{Kind: LiteralBool, BoolVal: b}
}

func nullLit() Literal {
	return Literal{Kind: LiteralNull}
}

// eof marks an error that should be reported at the end of the input.
const eof = "<EOF>"

func TestParseInsert(t *testing.T) {
	tests := []struct {
		input string
		want  *InsertStatement
		// errAt is the piece of input the error should point at (its first
		// occurrence), or eof. Empty means the parse must succeed.
		errAt string
	}{
		// valid statements
		{input: "INSERT INTO users VALUES (1, 'Alice')", want: &InsertStatement{TableName: "users", Values: []Literal{intLit(1), strLit("Alice")}}},
		{input: "INSERT INTO users VALUES (2, 'Bob');", want: &InsertStatement{TableName: "users", Values: []Literal{intLit(2), strLit("Bob")}}},
		{input: "INSERT INTO users VALUES (1)", want: &InsertStatement{TableName: "users", Values: []Literal{intLit(1)}}},
		{input: "insert into users values (1)", want: &InsertStatement{TableName: "users", Values: []Literal{intLit(1)}}},
		{input: "INSERT INTO users VALUES (NULL, TRUE, FALSE, '')", want: &InsertStatement{TableName: "users", Values: []Literal{nullLit(), boolLit(true), boolLit(false), strLit("")}}},
		{input: "INSERT INTO users VALUES ( 1 ,\n\t'x' )", want: &InsertStatement{TableName: "users", Values: []Literal{intLit(1), strLit("x")}}},
		{input: "INSERT INTO users VALUES (1, 2, 3, 4, 5, 6, 7, 8)", want: &InsertStatement{TableName: "users", Values: []Literal{intLit(1), intLit(2), intLit(3), intLit(4), intLit(5), intLit(6), intLit(7), intLit(8)}}},

		// negative numbers
		{input: "INSERT INTO t VALUES (-5)", want: &InsertStatement{TableName: "t", Values: []Literal{intLit(-5)}}},
		{input: "INSERT INTO t VALUES (- 5)", want: &InsertStatement{TableName: "t", Values: []Literal{intLit(-5)}}},
		{input: "INSERT INTO t VALUES (-1, 2, -3)", want: &InsertStatement{TableName: "t", Values: []Literal{intLit(-1), intLit(2), intLit(-3)}}},
		{input: "INSERT INTO t VALUES (-9223372036854775808)", want: &InsertStatement{TableName: "t", Values: []Literal{intLit(math.MinInt64)}}},
		{input: "INSERT INTO t VALUES ('-')", want: &InsertStatement{TableName: "t", Values: []Literal{strLit("-")}}},
		{input: "INSERT INTO t VALUES (-9223372036854775809)", errAt: "9223372036854775809"},
		{input: "INSERT INTO t VALUES (--5)", errAt: "-5"},
		{input: "INSERT INTO t VALUES (-'x')", errAt: "'x'"},
		{input: "INSERT INTO t VALUES (-TRUE)", errAt: "TRUE"},
		{input: "INSERT INTO t VALUES (-)", errAt: ")"},
		{input: "INSERT INTO t VALUES (5-)", errAt: "-"},

		// missing or wrong keywords / names
		{input: "INSERT users VALUES (1)", errAt: "users"},
		{input: "INSERT INTO VALUES (1)", errAt: "VALUES"},
		{input: "INSERT INTO users (1)", errAt: "("},
		{input: "INSERT INTO users VALUES 1", errAt: "1"},
		{input: "INSERT INTO", errAt: eof},
		{input: "INSERT INTO users VALUES", errAt: eof},

		// malformed value lists
		{input: "INSERT INTO users VALUES ()", errAt: ")"},
		{input: "INSERT INTO users VALUES (1,)", errAt: ")"},
		{input: "INSERT INTO users VALUES (,1)", errAt: ","},
		{input: "INSERT INTO users VALUES (1,,2)", errAt: ",2"},
		{input: "INSERT INTO users VALUES (1 2)", errAt: "2"},
		{input: "INSERT INTO users VALUES (1", errAt: eof},
		{input: "INSERT INTO users VALUES (1,", errAt: eof},
		{input: "INSERT INTO t VALUES (name)", errAt: "name"},
		{input: "INSERT INTO t VALUES (1)(2)", errAt: "(2)"},

		// trailing garbage
		{input: "INSERT INTO users VALUES (3, 'Charlie') extra", errAt: "extra"},
		{input: "INSERT INTO users VALUES (1); junk", errAt: "junk"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			stmt, err := Parse(tc.input)

			if tc.errAt == "" {
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(stmt, tc.want) {
					t.Fatalf("got %+v, want %+v", stmt, tc.want)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected error at %q, got %+v", tc.errAt, stmt)
			}
			var pe *ParseError
			if !errors.As(err, &pe) {
				t.Fatalf("expected *ParseError, got %T: %v", err, err)
			}
			wantPos := len(tc.input)
			if tc.errAt != eof {
				wantPos = strings.Index(tc.input, tc.errAt)
			}
			if pe.Pos != wantPos {
				t.Fatalf("error at position %d, want %d (%q): %v", pe.Pos, wantPos, tc.errAt, err)
			}
		})
	}
}

func TestParseCreate(t *testing.T) {
	tests := []struct {
		input string
		want  *CreateTableStatement
		errAt string
	}{
		{input: "CREATE TABLE users (id INT, name varchar, active BOOLEAN)", want: &CreateTableStatement{TableName: "users", Columns: []ColumnDef{{Name: "id", Type: tuple.TypeInt}, {Name: "name", Type: tuple.TypeVarchar}, {Name: "active", Type: tuple.TypeBoolean}}}},
		{input: "CREATE TABLE users (id INT, name varchar)", want: &CreateTableStatement{TableName: "users", Columns: []ColumnDef{{Name: "id", Type: tuple.TypeInt}, {Name: "name", Type: tuple.TypeVarchar}}}},
		{input: "CREATE TABLE users (id INT)", want: &CreateTableStatement{TableName: "users", Columns: []ColumnDef{{Name: "id", Type: tuple.TypeInt}}}},
		{input: "CREATE TABLE users (id INT, name varchar, active BOOLEAN);", want: &CreateTableStatement{TableName: "users", Columns: []ColumnDef{{Name: "id", Type: tuple.TypeInt}, {Name: "name", Type: tuple.TypeVarchar}, {Name: "active", Type: tuple.TypeBoolean}}}},
		{input: "CREATE TABLE users (id INT, name varchar, active BOOLEAN) extra", errAt: "extra"},
		{input: "CREATE TABLE users id INT, name varchar, active BOOLEAN)", errAt: "id"},
		{input: "CREATE TABLE users (id INT, name varchar, active BOOLEAN", errAt: eof},
		{input: "CREATE TABLE users (id INT, name varchar, active UNKNOWN)", errAt: "UNKNOWN"},
		{input: "CREATE TABLE t (id SELECT)", errAt: "SELECT"},
		{input: "CREATE TABLE t ()", errAt: ")"},
		{input: "CREATE TABLE t (id)", errAt: ")"},
		{input: "CREATE TABLE t (id INT,)", errAt: ")"},
		{input: "CREATE t (id INT)", errAt: "t"},
		{input: "CREATE TABLE t (id BIGINT)", want: &CreateTableStatement{TableName: "t", Columns: []ColumnDef{{Name: "id", Type: tuple.TypeBigInt}}}},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			stmt, err := Parse(tc.input)
			if tc.errAt == "" {
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(stmt, tc.want) {
					t.Fatalf("got %+v, want %+v", stmt, tc.want)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected error at %q, got %+v", tc.errAt, stmt)
			}
			var pe *ParseError
			if !errors.As(err, &pe) {
				t.Fatalf("expected *ParseError, got %T: %v", err, err)
			}
			wantPos := len(tc.input)
			if tc.errAt != eof {
				wantPos = strings.Index(tc.input, tc.errAt)
			}
			if pe.Pos != wantPos {
				t.Fatalf("error at position %d, want %d (%q): %v", pe.Pos, wantPos, tc.errAt, err)
			}
		})
	}
}
