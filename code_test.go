package tm2_test

import (
	"errors"
	"testing"

	"github.com/mpyw/tepra.tm2"
)

func TestParseCodeKind(t *testing.T) {
	for k := tm2.CodeText; k <= tm2.CodeJAN13; k++ {
		if got, err := tm2.ParseCodeKind(k.String()); err != nil || got != k {
			t.Errorf("ParseCodeKind(%q) = %v, %v", k.String(), got, err)
		}
	}
	if _, err := tm2.ParseCodeKind("ean8"); !errors.Is(err, tm2.ErrUnknownName) {
		t.Errorf("ParseCodeKind(ean8): %v", err)
	}
}

func TestNewPour_codes(t *testing.T) {
	ok := map[tm2.CodeKind][]string{
		tm2.CodeQR:      {"https://example.com/a?b=1", "日本語も"},
		tm2.CodeCODE39:  {"ABC-123", "X Y.$/+%"},
		tm2.CodeCODE128: {"abc-123", "~!@#"},
		tm2.CodeJAN13:   {"490123456789", "451234567890"},
	}
	for k, values := range ok {
		f, err := tm2.NewPour(tm2.Tape18mm, "c", values, tm2.WithCode(k))
		if err != nil {
			t.Errorf("%v: %v", k, err)
			continue
		}
		d := fixtureAs[*tm2.PourDrawer](t, f.Documents[0].Drawers[0])
		if d.Attribute != k {
			t.Errorf("%v: attribute %v", k, d.Attribute)
		}
		// A QR code gets a square frame; a barcode one 2.5 times as long.
		want := 43.2
		if k != tm2.CodeQR {
			want *= 2.5
		}
		if d.Frame[2] != want {
			t.Errorf("%v: frame %v, want %v long", k, d.Frame, want)
		}
	}
	bad := map[tm2.CodeKind]string{
		tm2.CodeQR:      "two\nlines",
		tm2.CodeCODE39:  "lower",
		tm2.CodeCODE128: "日本語",
		tm2.CodeJAN13:   "4901234567894",
	}
	for k, v := range bad {
		if _, err := tm2.NewPour(tm2.Tape18mm, "c", []string{v}, tm2.WithCode(k)); !errors.Is(err, tm2.ErrBadCode) {
			t.Errorf("%v, %q: %v, want ErrBadCode", k, v, err)
		}
	}
	if _, err := tm2.NewPour(tm2.Tape18mm, "c", []string{"x"}, tm2.WithCode(9)); !errors.Is(err, tm2.ErrInvalidOptions) {
		t.Errorf("unknown code: %v, want ErrInvalidOptions", err)
	}
}
