package sql

import "mydb/src/tuple"

// Statement is implemented by every top-level SQL statement type our v1
// grammar supports. It's a marker interface — there's no shared behavior
// across statement types (a CreateTable and a Select have nothing in
// common structurally), so this exists purely so the parser can return
// "some statement" and the executor can type-switch on the concrete type.
type Statement interface {
	isStatement()
}

// ---- CREATE TABLE ----

type ColumnDef struct {
	Name string
	Type tuple.TypeID
}

type CreateTableStatement struct {
	TableName string
	Columns   []ColumnDef
}

func (*CreateTableStatement) isStatement() {}

// ---- INSERT ----

type InsertStatement struct {
	TableName string
	Values    []Literal
}

func (*InsertStatement) isStatement() {}

// ---- SELECT ----

type SelectStatement struct {
	TableName string
	Where     *WhereClause
}

func (*SelectStatement) isStatement() {}

// ---- UPDATE ----

type SetClause struct {
	ColumnName string
	Value      Literal
}

type UpdateStatement struct {
	TableName string
	Set       []SetClause
	Where     *WhereClause
}

func (*UpdateStatement) isStatement() {}

// ---- DELETE ----

type DeleteStatement struct {
	TableName string
	Where     *WhereClause
}

func (*DeleteStatement) isStatement() {}

// ---- shared pieces ----

// WhereClause is a single comparison: <column> <op> <literal>. Our v1
// grammar supports exactly one condition — no AND/OR — so this is flat,
// not a tree of nested boolean expressions.
type WhereClause struct {
	ColumnName string
	Op         CompareOp
	Value      Literal
}

type CompareOp int

const (
	OpEq CompareOp = iota
	OpNeq
	OpLt
	OpGt
	OpLte
	OpGte
	OpIs
)

// Literal is a parsed constant value from SQL source — a number, string,
// or boolean token turned into a typed Go value, but NOT yet a
// tuple.Value, because the AST doesn't know a column's declared type.
// "25" could become TypeInt or TypeBigInt depending on the schema — only
// the executor, which has the Schema in hand, can make that call.
type Literal struct {
	Kind LiteralKind

	IntVal  int64
	StrVal  string
	BoolVal bool
}

type LiteralKind int

const (
	LiteralInt LiteralKind = iota
	LiteralString
	LiteralBool
	LiteralNull
)
