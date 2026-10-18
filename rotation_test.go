package tm2_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mpyw/tepra.tm2"
)

func TestParseRotation(t *testing.T) {
	for name, want := range map[string]tm2.Rotation{"0": tm2.RotationNone, "90": tm2.Rotation90, "180": tm2.Rotation180, "270": tm2.Rotation270} {
		got, err := tm2.ParseRotation(name)
		if err != nil || got != want || got.String() != name {
			t.Errorf("ParseRotation(%q) = %v, %v", name, got, err)
		}
	}
	// The error lists the rotations clockwise, not in the app's order.
	if _, err := tm2.ParseRotation("45"); !errors.Is(err, tm2.ErrUnknownName) || !strings.HasSuffix(err.Error(), "known: 0, 90, 180, 270") {
		t.Errorf("ParseRotation(45): %v", err)
	}
	if got := tm2.Rotation(90).String(); got != "Rotation(90)" {
		t.Errorf("String() = %q", got)
	}
}
