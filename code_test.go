package tepra_test

import (
	"errors"
	"testing"

	"github.com/mpyw/tepra.tm2"
)

func TestParseCodeKind(t *testing.T) {
	for k := tepra.CodeText; k <= tepra.CodeJAN13; k++ {
		if got, err := tepra.ParseCodeKind(k.String()); err != nil || got != k {
			t.Errorf("ParseCodeKind(%q) = %v, %v", k.String(), got, err)
		}
	}
	if _, err := tepra.ParseCodeKind("ean8"); !errors.Is(err, tepra.ErrUnknownName) {
		t.Errorf("ParseCodeKind(ean8): %v", err)
	}
}

func TestNewPour_codes(t *testing.T) {
	ok := map[tepra.CodeKind][]string{
		tepra.CodeQR:      {"https://example.com/a?b=1", "日本語も"},
		tepra.CodeCODE39:  {"ABC-123", "X Y.$/+%"},
		tepra.CodeCODE128: {"abc-123", "~!@#"},
		tepra.CodeJAN13:   {"490123456789", "451234567890"},
	}
	for k, values := range ok {
		f, err := tepra.NewPour(tepra.Tape18mm, "c", values, tepra.WithCode(k))
		if err != nil {
			t.Errorf("%v: %v", k, err)
			continue
		}
		d := fixtureAs[*tepra.PourDrawer](t, f.Documents[0].Drawers[0])
		if d.Attribute != k {
			t.Errorf("%v: attribute %v", k, d.Attribute)
		}
		// A QR code gets a square frame; a barcode one 2.5 times as long.
		want := 43.2
		if k != tepra.CodeQR {
			want *= 2.5
		}
		if d.Frame[2] != want {
			t.Errorf("%v: frame %v, want %v long", k, d.Frame, want)
		}
	}
	bad := map[tepra.CodeKind]string{
		tepra.CodeQR:      "two\nlines",
		tepra.CodeCODE39:  "lower",
		tepra.CodeCODE128: "日本語",
		tepra.CodeJAN13:   "4901234567894",
	}
	for k, v := range bad {
		if _, err := tepra.NewPour(tepra.Tape18mm, "c", []string{v}, tepra.WithCode(k)); !errors.Is(err, tepra.ErrBadCode) {
			t.Errorf("%v, %q: %v, want ErrBadCode", k, v, err)
		}
	}
	if _, err := tepra.NewPour(tepra.Tape18mm, "c", []string{"x"}, tepra.WithCode(9)); !errors.Is(err, tepra.ErrInvalidOptions) {
		t.Errorf("unknown code: %v, want ErrInvalidOptions", err)
	}
}
