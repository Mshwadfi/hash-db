package tuple

import (
	"errors"
	"testing"
)

func TestValueCompare(t *testing.T) {
	tests := []struct {
		name    string
		v, o    Value
		want    int
		wantErr error
	}{
		{"int less", NewIntValue(1), NewIntValue(2), -1, nil},
		{"int equal", NewIntValue(2), NewIntValue(2), 0, nil},
		{"int greater", NewIntValue(3), NewIntValue(2), 1, nil},
		{"int negative", NewIntValue(-5), NewIntValue(3), -1, nil},
		{"bigint greater", NewBigIntValue(3000000000), NewBigIntValue(1), 1, nil},
		{"bigint equal", NewBigIntValue(-7), NewBigIntValue(-7), 0, nil},
		{"varchar less", NewVarcharValue("a"), NewVarcharValue("b"), -1, nil},
		{"varchar equal", NewVarcharValue("bob"), NewVarcharValue("bob"), 0, nil},
		{"varchar prefix is less", NewVarcharValue("ab"), NewVarcharValue("abc"), -1, nil},
		{"varchar byte order", NewVarcharValue("Z"), NewVarcharValue("a"), -1, nil},
		{"bool false < true", NewBooleanValue(false), NewBooleanValue(true), -1, nil},
		{"bool equal", NewBooleanValue(true), NewBooleanValue(true), 0, nil},
		{"bool true > false", NewBooleanValue(true), NewBooleanValue(false), 1, nil},
		{"type mismatch", NewIntValue(1), NewVarcharValue("1"), 0, ErrTypeMismatch},
		{"int vs bigint", NewIntValue(1), NewBigIntValue(1), 0, ErrTypeMismatch},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.v.Compare(tc.o)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
}
