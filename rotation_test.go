package tepra_test

import (
	"errors"
	"testing"

	"github.com/mpyw/tepra.tm2"
)

func TestParseRotation(t *testing.T) {
	for name, want := range map[string]tepra.Rotation{"0": tepra.RotationNone, "90": tepra.Rotation90, "180": tepra.Rotation180, "270": tepra.Rotation270} {
		got, err := tepra.ParseRotation(name)
		if err != nil || got != want || got.String() != name {
			t.Errorf("ParseRotation(%q) = %v, %v", name, got, err)
		}
	}
	if _, err := tepra.ParseRotation("45"); !errors.Is(err, tepra.ErrUnknownName) {
		t.Errorf("ParseRotation(45): %v", err)
	}
	if got := tepra.Rotation(90).String(); got != "Rotation(90)" {
		t.Errorf("String() = %q", got)
	}
}
