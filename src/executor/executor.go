package executor

import (
	"fmt"
	"math"
	"mydb/src/sql"
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
