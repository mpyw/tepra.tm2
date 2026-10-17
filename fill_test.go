package tepra_test

import (
	"errors"
	"testing"

	"github.com/mpyw/tepra.tm2"
)

func TestParseFillMode(t *testing.T) {
	for _, m := range []tepra.FillMode{tepra.FillNone, tepra.FillInverted, tepra.FillShaded} {
		got, err := tepra.ParseFillMode(m.String())
		if err != nil || got != m {
			t.Errorf("ParseFillMode(%q) = %v, %v", m.String(), got, err)
		}
	}
	if _, err := tepra.ParseFillMode("black"); !errors.Is(err, tepra.ErrUnknownName) {
		t.Errorf("ParseFillMode(black): %v, want ErrUnknownName", err)
	}
	if got := tepra.FillMode(9).String(); got != "FillMode(9)" {
		t.Errorf("String() = %q", got)
	}
}
