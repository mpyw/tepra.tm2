package tepra_test

import (
	"errors"
	"testing"

	"github.com/mpyw/tepra.tm2"
)

func TestParseBorderType(t *testing.T) {
	for b := tepra.BorderNone; b <= tepra.BorderBottom; b++ {
		got, err := tepra.ParseBorderType(b.String())
		if err != nil || got != b {
			t.Errorf("ParseBorderType(%q) = %v, %v", b.String(), got, err)
		}
	}
	if _, err := tepra.ParseBorderType("double"); !errors.Is(err, tepra.ErrUnknownName) {
		t.Errorf("ParseBorderType(double): %v, want ErrUnknownName", err)
	}
	if got := tepra.BorderType(6).String(); got != "BorderType(6)" {
		t.Errorf("String() = %q", got)
	}
}
