package tm2_test

import (
	"errors"
	"testing"

	"github.com/mpyw/tepra.tm2"
)

func TestParseAlign(t *testing.T) {
	for a := tm2.AlignLeft; a <= tm2.AlignRight; a++ {
		if got, err := tm2.ParseAlign(a.String()); err != nil || got != a {
			t.Errorf("ParseAlign(%q) = %v, %v", a.String(), got, err)
		}
	}
	for a := tm2.VerticalAlignTop; a <= tm2.VerticalAlignBottom; a++ {
		if got, err := tm2.ParseVerticalAlign(a.String()); err != nil || got != a {
			t.Errorf("ParseVerticalAlign(%q) = %v, %v", a.String(), got, err)
		}
	}
	if _, err := tm2.ParseAlign("justify"); !errors.Is(err, tm2.ErrUnknownName) {
		t.Errorf("ParseAlign(justify): %v, want ErrUnknownName", err)
	}
	if _, err := tm2.ParseVerticalAlign("center"); !errors.Is(err, tm2.ErrUnknownName) {
		t.Errorf("ParseVerticalAlign(center): %v, want ErrUnknownName", err)
	}
	if got := tm2.Align(3).String() + " " + tm2.VerticalAlign(3).String(); got != "Align(3) VerticalAlign(3)" {
		t.Errorf("String() = %q", got)
	}
}
