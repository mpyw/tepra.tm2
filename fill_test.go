package tm2_test

import (
	"errors"
	"testing"

	"github.com/mpyw/tepra.tm2"
)

func TestParseFillMode(t *testing.T) {
	for _, m := range []tm2.FillMode{tm2.FillNone, tm2.FillInverted, tm2.FillShaded} {
		got, err := tm2.ParseFillMode(m.String())
		if err != nil || got != m {
			t.Errorf("ParseFillMode(%q) = %v, %v", m.String(), got, err)
		}
	}
	if _, err := tm2.ParseFillMode("black"); !errors.Is(err, tm2.ErrUnknownName) {
		t.Errorf("ParseFillMode(black): %v, want ErrUnknownName", err)
	}
	if got := tm2.FillMode(9).String(); got != "FillMode(9)" {
		t.Errorf("String() = %q", got)
	}
}
