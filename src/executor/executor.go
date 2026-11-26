package executor

import (
	"fmt"
	"math"
	"mydb/src/catalog"
	"mydb/src/sql"
	"mydb/src/storageLayer"
	"mydb/src/tuple"
)

func literalToValue(lit sql.Literal, typ tuple.TypeID) (tuple.Value, error) {
	if lit.Kind == sql.LiteralNull {
		return tuple.Value{}, fmt.Errorf("NULL literal is not supported yet")
	}
	switch typ {
	case tuple.TypeInt:
		if lit.Kind != sql.LiteralInt {
			return tuple.Value{}, fmt.Errorf("Expected INT literal, got %v", lit.Kind)
		}
		if lit.IntVal < math.MinInt32 || lit.IntVal > math.MaxInt32 {
			return tuple.Value{}, fmt.Errorf("INT literal out of range: %d", lit.IntVal)
		}
		return tuple.NewIntValue(int32(lit.IntVal)), nil
	case tuple.TypeVarchar:
		if lit.Kind != sql.LiteralString {
			return tuple.Value{}, fmt.Errorf("Expected VARCHAR literal, got %v", lit.Kind)
		}
		return tuple.NewVarcharValue(lit.StrVal), nil
	case tuple.TypeBoolean:
		if lit.Kind != sql.LiteralBool {
			return tuple.Value{}, fmt.Errorf("Expected BOOLEAN literal, got %v", lit.Kind)
		}
		return tuple.NewBooleanValue(lit.BoolVal), nil
	case tuple.TypeBigInt:
		if lit.Kind != sql.LiteralInt {
			return tuple.Value{}, fmt.Errorf("Expected BIGINT literal, got %v", lit.Kind)
		}
		return tuple.NewBigIntValue(lit.IntVal), nil
	default:
		return tuple.Value{}, fmt.Errorf("Unsupported column type: %v", typ)
	}
}

type Executor struct {
	catalog *catalog.Catalog
}

func NewExecutor(catalog *catalog.Catalog) *Executor {
	return &Executor{catalog: catalog}
}

type Result struct {
	Columns []string
	Rows    []*tuple.Tuple
	Message string
}

// execute function that take a statment and execute it
func (e *Executor) Execute(stmt sql.Statement) (*Result, error) {
	switch s := stmt.(type) {
	case *sql.CreateTableStatement:
		return e.executeCreateTable(s)
	case *sql.InsertStatement:
		return e.executeInsert(s)
	case *sql.SelectStatement:
		return e.executeSelect(s)
	case *sql.DeleteStatement:
		return e.executeDelete(s)
	case *sql.UpdateStatement:
		return e.executeUpdate(s)
	default:
		return nil, fmt.Errorf("Unsupported statement type: %T", stmt)
	}
}

// execute create table statement
func (e *Executor) executeCreateTable(stmt *sql.CreateTableStatement) (*Result, error) {
	columns := make([]tuple.Column, 0, len(stmt.Columns))
	for _, col := range stmt.Columns {
		if col.Type == tuple.TypeUnknown {
			return nil, fmt.Errorf("Unknown column type for column %s", col.Name)
		}
		c, err := tuple.NewColumn(col.Name, col.Type)
		if err != nil {
			return nil, fmt.Errorf("Failed to create column %s: %w", col.Name, err)
		}
		columns = append(columns, c)
	}

	schema, err := tuple.NewSchema(columns)
	if err != nil {
		return nil, fmt.Errorf("Failed to create schema: %w", err)
	}

	_, err = e.catalog.CreateTable(stmt.TableName, schema)
	if err != nil {
		return nil, fmt.Errorf("Failed to create table: %w", err)
	}
	return &Result{Message: "Table created successfully"}, nil
}

// execute insert statement
func (e *Executor) executeInsert(stmt *sql.InsertStatement) (*Result, error) {
	table, schema, err := e.catalog.GetTable(stmt.TableName)
	if err != nil {
		return nil, fmt.Errorf("Failed to get table: %w", err)
	}

	if len(stmt.Values) != schema.GetNumberOfColumns() {
		return nil, fmt.Errorf("%w: number of values (%d) does not match number of columns (%d)",
			tuple.ErrValueCountMismatch, len(stmt.Values), schema.GetNumberOfColumns())
	}

	values := make([]tuple.Value, 0, len(stmt.Values))
	for i, lit := range stmt.Values {
		col, err := schema.GetColumn(i)
		if err != nil {
			return nil, fmt.Errorf("Failed to get column %d: %w", i, err)
		}
		colType := col.Type
		val, err := literalToValue(lit, colType)
		if err != nil {
			return nil, fmt.Errorf("Failed to convert literal to value for column %s: %w", col.Name, err)
		}
		values = append(values, val)
	}
	row := tuple.NewTuple(values)
	serializer, err := tuple.NewSerializer(schema)
	if err != nil {
		return nil, fmt.Errorf("Failed to create serializer: %w", err)
	}
	serializedRow, err := serializer.Serialize(row)
	if err != nil {
		return nil, fmt.Errorf("Failed to serialize row: %w", err)
	}
	_, err = table.Insert(serializedRow)
	if err != nil {
		return nil, fmt.Errorf("Failed to insert values into table: %w", err)
	}

	return &Result{Message: "Row inserted successfully"}, nil
}

// execute select statement
func (e *Executor) executeSelect(stmt *sql.SelectStatement) (*Result, error) {
	table, schema, err := e.catalog.GetTable(stmt.TableName)
	if err != nil {
		return nil, fmt.Errorf("Failed to get table: %w", err)
	}

	serializer, err := tuple.NewSerializer(schema)
	if err != nil {
		return nil, fmt.Errorf("Failed to create serializer: %w", err)
	}

	columns := make([]string, 0, schema.GetNumberOfColumns())
	for i := 0; i < schema.GetNumberOfColumns(); i++ {
		col, err := schema.GetColumn(i)
		if err != nil {
			return nil, fmt.Errorf("Failed to get column %d: %w", i, err)
		}
		columns = append(columns, col.Name)
	}

	// Resolve the WHERE clause column and value once, before scanning: the answer is
	// the same for every row, and a bad query fails even on an empty table.
	var whereIdx int
	var whereValue tuple.Value
	if stmt.Where != nil {
		whereIdx, err = schema.GetColumnIndex(stmt.Where.ColumnName)
		if err != nil {
			return nil, fmt.Errorf("column %s: %w", stmt.Where.ColumnName, err)
		}
		whereValue, err = literalToValue(stmt.Where.Value, schema.Columns[whereIdx].Type)
		if err != nil {
			return nil, fmt.Errorf("WHERE %s: %w", stmt.Where.ColumnName, err)
		}
	}

	rows := make([]*tuple.Tuple, 0)
	scanner := table.NewScanner()
	defer scanner.Close() // releases the pinned page on every return path
	for {
		_, data, ok, err := scanner.Next()
		if err != nil {
			return nil, fmt.Errorf("Failed to get next record: %w", err)
		}
		if !ok {
			break
		}
		rowData, err := serializer.Deserialize(data)
		if err != nil {
			return nil, fmt.Errorf("Failed to deserialize record: %w", err)
		}
		if stmt.Where != nil {
			c, err := rowData.Value(whereIdx).Compare(whereValue)
			if err != nil {
				return nil, fmt.Errorf("WHERE %s: %w", stmt.Where.ColumnName, err)
			}
			if !matches(c, stmt.Where.Op) {
				continue
			}
		}
		rows = append(rows, rowData)
	}

	return &Result{Columns: columns, Rows: rows}, nil
}

// matches turns a Compare result into the answer for one WHERE operator.
func matches(c int, op sql.CompareOp) bool {
	switch op {
	case sql.OpEq:
		return c == 0
	case sql.OpNeq:
		return c != 0
	case sql.OpLt:
		return c < 0
	case sql.OpGt:
		return c > 0
	case sql.OpLte:
		return c <= 0
	case sql.OpGte:
		return c >= 0
	}
	return false
}

// execute delete statment
func (e *Executor) executeDelete(stmt *sql.DeleteStatement) (*Result, error) {
	table, schema, err := e.catalog.GetTable(stmt.TableName)
	if err != nil {
		return nil, fmt.Errorf("DELETE %s: %w", stmt.TableName, err)
	}
	serializer, err := tuple.NewSerializer(schema)
	if err != nil {
		return nil, fmt.Errorf("DELETE %s: failed to create serializer: %w", stmt.TableName, err)
	}
	// resolve where clause
	var whereColumnIdx int
	var whereColumnValue tuple.Value

	if stmt.Where != nil {
		whereIdx, err := schema.GetColumnIndex(stmt.Where.ColumnName)
		if err != nil {
			return nil, fmt.Errorf("WHERE %s: %w", stmt.Where.ColumnName, err)
		}
		whereValue, err := literalToValue(stmt.Where.Value, schema.Columns[whereIdx].Type)
		if err != nil {
			return nil, fmt.Errorf("WHERE %s: %w", stmt.Where.ColumnName, err)
		}

		whereColumnIdx = whereIdx
		whereColumnValue = whereValue

	}

	scanner := table.NewScanner()
	defer scanner.Close()
	var affectedRows []storageLayer.RecordID

	for {
		rid, data, ok, err := scanner.Next()
		if err != nil {
			return nil, fmt.Errorf("DELETE %s: failed to read next row: %w", stmt.TableName, err)
		}

		if !ok {
			break
		}

		rowData, err := serializer.Deserialize(data)

		if err != nil {
			return nil, fmt.Errorf("DELETE %s: failed to decode row at %v: %w", stmt.TableName, rid, err)
		}

		if stmt.Where != nil {
			c, err := rowData.Value(whereColumnIdx).Compare(whereColumnValue)
			if err != nil {
				return nil, fmt.Errorf("WHERE %s: %w", stmt.Where.ColumnName, err)
			}

			if !matches(c, stmt.Where.Op) {
				continue
			}
		}
		affectedRows = append(affectedRows, rid)
	}
	scanner.Close()

	for i, rid := range affectedRows {
		if err := table.Delete(rid); err != nil {
			return nil, fmt.Errorf("DELETE %s: failed to delete row at %v (%d of %d done): %w",
				stmt.TableName, rid, i, len(affectedRows), err)
		}
	}

	return &Result{Message: fmt.Sprintf("%d row(s) deleted", len(affectedRows))}, nil

}

// execute update statement
func (e *Executor) executeUpdate(stmt *sql.UpdateStatement) (*Result, error) {
	table, schema, err := e.catalog.GetTable(stmt.TableName)
	if err != nil {
		return nil, fmt.Errorf("UPDATE %s: %w", stmt.TableName, err)
	}
	serializer, err := tuple.NewSerializer(schema)
	if err != nil {
		return nil, fmt.Errorf("UPDATE %s: failed to create serializer: %w", stmt.TableName, err)
	}

	// resolve set clauses once, so a bad column or value fails even on an empty table
	setIdx := make([]int, len(stmt.Set))
	setValues := make([]tuple.Value, len(stmt.Set))
	for i, set := range stmt.Set {
		idx, err := schema.GetColumnIndex(set.ColumnName)
		if err != nil {
			return nil, fmt.Errorf("SET %s: %w", set.ColumnName, err)
		}
		value, err := literalToValue(set.Value, schema.Columns[idx].Type)
		if err != nil {
			return nil, fmt.Errorf("SET %s: %w", set.ColumnName, err)
		}
		setIdx[i] = idx
		setValues[i] = value
	}

	// resolve where clause
	var whereColumnIdx int
	var whereColumnValue tuple.Value

	if stmt.Where != nil {
		whereIdx, err := schema.GetColumnIndex(stmt.Where.ColumnName)
		if err != nil {
			return nil, fmt.Errorf("WHERE %s: %w", stmt.Where.ColumnName, err)
		}
		whereValue, err := literalToValue(stmt.Where.Value, schema.Columns[whereIdx].Type)
		if err != nil {
			return nil, fmt.Errorf("WHERE %s: %w", stmt.Where.ColumnName, err)
		}

		whereColumnIdx = whereIdx
		whereColumnValue = whereValue
	}

	// first: get matching rows
	// we don't update while scanning: Update can move a row to a later page,
	// and the scanner would then visit it again
	scanner := table.NewScanner()
	defer scanner.Close()
	var affectedRows []storageLayer.RecordID
	var newData [][]byte

	for {
		rid, data, ok, err := scanner.Next()
		if err != nil {
			return nil, fmt.Errorf("UPDATE %s: failed to read next row: %w", stmt.TableName, err)
		}

		if !ok {
			break
		}

		rowData, err := serializer.Deserialize(data)
		if err != nil {
			return nil, fmt.Errorf("UPDATE %s: failed to decode row at %v: %w", stmt.TableName, rid, err)
		}

		if stmt.Where != nil {
			c, err := rowData.Value(whereColumnIdx).Compare(whereColumnValue)
			if err != nil {
				return nil, fmt.Errorf("WHERE %s: %w", stmt.Where.ColumnName, err)
			}

			if !matches(c, stmt.Where.Op) {
				continue
			}
		}

		// build the new row: copy old values then overwrite the SET columns
		values := make([]tuple.Value, rowData.NumValues())
		for i := range values {
			values[i] = rowData.Value(i)
		}
		for i, idx := range setIdx {
			values[idx] = setValues[i]
		}

		serializedRow, err := serializer.Serialize(tuple.NewTuple(values))
		if err != nil {
			return nil, fmt.Errorf("UPDATE %s: failed to encode row at %v: %w", stmt.TableName, rid, err)
		}

		affectedRows = append(affectedRows, rid)
		newData = append(newData, serializedRow)
	}
	scanner.Close()

	// second: write the updated rows
	for i, rid := range affectedRows {
		if _, _, err := table.Update(rid, newData[i]); err != nil {
			return nil, fmt.Errorf("UPDATE %s: failed to update row at %v (%d of %d done): %w",
				stmt.TableName, rid, i, len(affectedRows), err)
		}
	}

	return &Result{Message: fmt.Sprintf("%d row(s) updated", len(affectedRows))}, nil
}
