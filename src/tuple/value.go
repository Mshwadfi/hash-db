package tuple

import (
	"cmp"
	"errors"
	"fmt"
)

var (
	// ErrTypeMismatch is returned when a value is of the wrong type
	ErrTypeMismatch = errors.New("value: type mismatch")
	errInvalidType  = errors.New("value: invalid type")
)

type Value struct {
	typ         TypeID
	intValue    int32
	boolValue   bool
	stringValue string
	bigIntValue int64
}

func NewIntValue(value int32) Value {
	return Value{
		typ:      TypeInt,
		intValue: value,
	}
}

func NewBigIntValue(value int64) Value {
	return Value{
		typ:         TypeBigInt,
		bigIntValue: value,
	}
}

func NewBooleanValue(value bool) Value {
	return Value{
		typ:       TypeBoolean,
		boolValue: value,
	}
}

func NewVarcharValue(value string) Value {
	return Value{
		typ:         TypeVarchar,
		stringValue: value,
	}
}

func (v Value) Type() TypeID {
	return v.typ
}

func (v Value) AsInt() (int32, error) {
	if v.typ != TypeInt {
		return 0, ErrTypeMismatch
	}
	return v.intValue, nil
}

func (v Value) AsBigInt() (int64, error) {
	if v.typ != TypeBigInt {
		return 0, ErrTypeMismatch
	}
	return v.bigIntValue, nil
}

func (v Value) AsBoolean() (bool, error) {
	if v.typ != TypeBoolean {
		return false, ErrTypeMismatch
	}
	return v.boolValue, nil
}

func (v Value) AsVarchar() (string, error) {
	if v.typ != TypeVarchar {
		return "", ErrTypeMismatch
	}
	return v.stringValue, nil
}

// Compare orders v against other: -1 if v < other, 0 if equal, +1 if v > other.
// Both values must have the same type, otherwise ErrTypeMismatch.
//
// Ordering rules:
//   - INT, BIGINT: numeric order
//   - VARCHAR: byte order (no collation), so 'Z' < 'a'
//   - BOOLEAN: false < true, same as Postgres
func (v Value) Compare(other Value) (int, error) {
	if v.typ != other.typ {
		return 0, fmt.Errorf("%w: cannot compare %s with %s", ErrTypeMismatch, v.typ, other.typ)
	}
	switch v.typ {
	case TypeInt:
		return cmp.Compare(v.intValue, other.intValue), nil
	case TypeBigInt:
		return cmp.Compare(v.bigIntValue, other.bigIntValue), nil
	case TypeVarchar:
		return cmp.Compare(v.stringValue, other.stringValue), nil
	case TypeBoolean:
		return cmp.Compare(boolToInt(v.boolValue), boolToInt(other.boolValue)), nil
	}
	return 0, fmt.Errorf("%w: %s", errInvalidType, v.typ)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
