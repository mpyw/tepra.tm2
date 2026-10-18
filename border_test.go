package tm2_test

import (
	"errors"
	"testing"

	"github.com/mpyw/tepra.tm2"
)

func TestParseBorderType(t *testing.T) {
	for b := tm2.BorderNone; b <= tm2.BorderBottom; b++ {
		got, err := tm2.ParseBorderType(b.String())
		if err != nil || got != b {
			t.Errorf("ParseBorderType(%q) = %v, %v", b.String(), got, err)
		}
	}
	if _, err := tm2.ParseBorderType("double"); !errors.Is(err, tm2.ErrUnknownName) {
		t.Errorf("ParseBorderType(double): %v, want ErrUnknownName", err)
	}
	if got := tm2.BorderType(6).String(); got != "BorderType(6)" {
		t.Errorf("String() = %q", got)
	}
}
