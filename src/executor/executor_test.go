package executor

import (
	"errors"
	"math"
	"path/filepath"
	"testing"

	"mydb/src/catalog"
	"mydb/src/sql"
	"mydb/src/storageLayer"
	"mydb/src/tuple"
)

func TestLiteralToValue(t *testing.T) {
	tests := []struct {
		name    string
		lit     sql.Literal
		typ     tuple.TypeID
		want    tuple.Value
		wantErr bool
	}{
		// INT: range is checked before narrowing int64 -> int32
		{"int ok", sql.Literal{Kind: sql.LiteralInt, IntVal: 5}, tuple.TypeInt, tuple.NewIntValue(5), false},
		{"max int32", sql.Literal{Kind: sql.LiteralInt, IntVal: math.MaxInt32}, tuple.TypeInt, tuple.NewIntValue(math.MaxInt32), false},
		{"max int32 + 1", sql.Literal{Kind: sql.LiteralInt, IntVal: math.MaxInt32 + 1}, tuple.TypeInt, tuple.Value{}, true},
		{"min int32", sql.Literal{Kind: sql.LiteralInt, IntVal: math.MinInt32}, tuple.TypeInt, tuple.NewIntValue(math.MinInt32), false},
		{"min int32 - 1", sql.Literal{Kind: sql.LiteralInt, IntVal: math.MinInt32 - 1}, tuple.TypeInt, tuple.Value{}, true},
		{"string into int", sql.Literal{Kind: sql.LiteralString, StrVal: "5"}, tuple.TypeInt, tuple.Value{}, true},

		// BIGINT: holds values INT can't
		{"bigint fits big", sql.Literal{Kind: sql.LiteralInt, IntVal: 3000000000}, tuple.TypeBigInt, tuple.NewBigIntValue(3000000000), false},

		// VARCHAR
		{"varchar ok", sql.Literal{Kind: sql.LiteralString, StrVal: "a"}, tuple.TypeVarchar, tuple.NewVarcharValue("a"), false},
		{"int into varchar", sql.Literal{Kind: sql.LiteralInt, IntVal: 5}, tuple.TypeVarchar, tuple.Value{}, true},

		// BOOLEAN
		{"bool ok", sql.Literal{Kind: sql.LiteralBool, BoolVal: true}, tuple.TypeBoolean, tuple.NewBooleanValue(true), false},
		{"string into bool", sql.Literal{Kind: sql.LiteralString, StrVal: "true"}, tuple.TypeBoolean, tuple.Value{}, true},

		// unsupported inputs
		{"null", sql.Literal{Kind: sql.LiteralNull}, tuple.TypeInt, tuple.Value{}, true},
		{"unknown column type", sql.Literal{Kind: sql.LiteralInt, IntVal: 5}, tuple.TypeUnknown, tuple.Value{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := literalToValue(tt.lit, tt.typ)
			if (err != nil) != tt.wantErr {
				t.Errorf("literalToValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("literalToValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

// newTestExecutor opens a fresh database in a per-test temp dir, so tests
// never share tables.
func newTestExecutor(t *testing.T) *Executor {
	t.Helper()
	dm, err := storageLayer.OpenDiskManager(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	bm, err := storageLayer.NewBufferManager(10, dm)
	if err != nil {
		t.Fatal(err)
	}
	cat, err := catalog.OpenCatalog(bm, dm)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		bm.FlushAll()
		dm.Close()
	})
	return NewExecutor(cat)
}

// exec parses and executes one statement.
func exec(e *Executor, query string) (*Result, error) {
	stmt, err := sql.Parse(query)
	if err != nil {
		return nil, err
	}
	return e.Execute(stmt)
}

// mustExec is exec for setup steps that must succeed.
func mustExec(t *testing.T, e *Executor, query string) *Result {
	t.Helper()
	res, err := exec(e, query)
	if err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return res
}

func TestExecuteCreateTable(t *testing.T) {
	e := newTestExecutor(t)
	mustExec(t, e, "CREATE TABLE users (id INT, name VARCHAR)")

	// check the effect, not just the reply: the table must be in the catalog
	_, schema, err := e.catalog.GetTable("users")
	if err != nil {
		t.Fatal(err)
	}
	want := []tuple.Column{{Name: "id", Type: tuple.TypeInt}, {Name: "name", Type: tuple.TypeVarchar}}
	if len(schema.Columns) != len(want) {
		t.Fatalf("got %d columns, want %d", len(schema.Columns), len(want))
	}
	for i, c := range want {
		if schema.Columns[i] != c {
			t.Errorf("column %d = %+v, want %+v", i, schema.Columns[i], c)
		}
	}

	if _, err := exec(e, "CREATE TABLE users (id INT)"); !errors.Is(err, catalog.ErrTableExists) {
		t.Errorf("create twice: got %v, want ErrTableExists", err)
	}
	if _, err := exec(e, "CREATE TABLE dup (id INT, id INT)"); !errors.Is(err, tuple.ErrDuplicateColumn) {
		t.Errorf("duplicate column: got %v, want ErrDuplicateColumn", err)
	}
	if _, err := e.Execute(&sql.UpdateStatement{}); err == nil {
		t.Error("unsupported statement: expected error")
	}
}

func TestExecuteInsertErrors(t *testing.T) {
	e := newTestExecutor(t)
	mustExec(t, e, "CREATE TABLE users (id INT, name VARCHAR)")

	tests := []struct {
		query string
		want  error // nil means: any error is fine
	}{
		{"INSERT INTO nope VALUES (1, 'a')", catalog.ErrTableNotFound},
		{"INSERT INTO users VALUES (1, 'a', TRUE)", tuple.ErrValueCountMismatch},
		{"INSERT INTO users VALUES (1)", tuple.ErrValueCountMismatch},
		{"INSERT INTO users VALUES ('x', 'a')", nil},
		{"INSERT INTO users VALUES (3000000000, 'a')", nil},
	}
	for _, tc := range tests {
		t.Run(tc.query, func(t *testing.T) {
			_, err := exec(e, tc.query)
			if err == nil {
				t.Fatal("expected error")
			}
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}

	// none of the failed inserts may have stored anything
	if res := mustExec(t, e, "SELECT * FROM users"); len(res.Rows) != 0 {
		t.Errorf("failed inserts stored %d rows", len(res.Rows))
	}
}

// TestInsertSelectRoundTrip is the first end-to-end check: rows written by
// INSERT come back from SELECT unchanged and in insertion order.
func TestInsertSelectRoundTrip(t *testing.T) {
	e := newTestExecutor(t)
	mustExec(t, e, "CREATE TABLE users (id INT, name VARCHAR)")

	if res := mustExec(t, e, "SELECT * FROM users"); len(res.Rows) != 0 {
		t.Fatalf("empty table returned %d rows", len(res.Rows))
	}

	mustExec(t, e, "INSERT INTO users VALUES (1, 'muhammad')")
	mustExec(t, e, "INSERT INTO users VALUES (2, 'ali')")

	res := mustExec(t, e, "SELECT * FROM users")
	if len(res.Columns) != 2 || res.Columns[0] != "id" || res.Columns[1] != "name" {
		t.Fatalf("columns = %v, want [id name]", res.Columns)
	}
	want := [][]tuple.Value{
		{tuple.NewIntValue(1), tuple.NewVarcharValue("muhammad")},
		{tuple.NewIntValue(2), tuple.NewVarcharValue("ali")},
	}
	if len(res.Rows) != len(want) {
		t.Fatalf("got %d rows, want %d", len(res.Rows), len(want))
	}
	for i, row := range res.Rows {
		for j, v := range want[i] {
			if got := row.Value(j); got != v {
				t.Errorf("row %d col %d = %v, want %v", i, j, got, v)
			}
		}
	}
}
