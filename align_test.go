package tepra_test

import (
	"errors"
	"testing"

	"github.com/mpyw/tepra.tm2"
)

func TestParseAlign(t *testing.T) {
	for a := tepra.AlignLeft; a <= tepra.AlignRight; a++ {
		if got, err := tepra.ParseAlign(a.String()); err != nil || got != a {
			t.Errorf("ParseAlign(%q) = %v, %v", a.String(), got, err)
		}
	}
	for a := tepra.VerticalAlignTop; a <= tepra.VerticalAlignBottom; a++ {
		if got, err := tepra.ParseVerticalAlign(a.String()); err != nil || got != a {
			t.Errorf("ParseVerticalAlign(%q) = %v, %v", a.String(), got, err)
		}
	}
	if _, err := tepra.ParseAlign("justify"); !errors.Is(err, tepra.ErrUnknownName) {
		t.Errorf("ParseAlign(justify): %v, want ErrUnknownName", err)
	}
	if _, err := tepra.ParseVerticalAlign("center"); !errors.Is(err, tepra.ErrUnknownName) {
		t.Errorf("ParseVerticalAlign(center): %v, want ErrUnknownName", err)
	}
	if got := tepra.Align(3).String() + " " + tepra.VerticalAlign(3).String(); got != "Align(3) VerticalAlign(3)" {
		t.Errorf("String() = %q", got)
	}
}
