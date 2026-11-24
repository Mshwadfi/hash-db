package executor

import (
	"math"
	"testing"

	"mydb/src/sql"
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
