package tepra_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mpyw/tepra.tm2"
)

func TestNewPages(t *testing.T) {
	f, err := tepra.NewPages(tepra.Tape18mm, []string{"カイロ", "ポリ袋\nチャック袋", "割り箸\nスプーン\nフォーク"})
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Documents) != 3 || f.CurrentPage != 0 || len(f.TableData) != 0 {
		t.Fatalf("%d documents, current page %d, %d rows", len(f.Documents), f.CurrentPage, len(f.TableData))
	}
	// Each page fits its own text, as the app showed for these three.
	for i, want := range []float64{35, 17.5, 11.5} {
		doc := f.Documents[i]
		if !doc.MultiplePage || doc.PourMode {
			t.Errorf("page %d: MultiplePage %v, PourMode %v", i, doc.MultiplePage, doc.PourMode)
		}
		if d := fixtureAs[*tepra.TextDrawer](t, doc.Drawers[0]); d.FontSize != want {
			t.Errorf("page %d: size %v, want %v", i, d.FontSize, want)
		}
	}
	if f.TextWords != "カイロ,ポリ袋\nチャック袋,割り箸\nスプーン\nフォーク" {
		t.Errorf("textWords %q", f.TextWords)
	}
}

func TestNewPages_rejects(t *testing.T) {
	if _, err := tepra.NewPages(tepra.Tape18mm, nil); !errors.Is(err, tepra.ErrNoText) {
		t.Errorf("no pages: %v, want ErrNoText", err)
	}
	// Pages count from 1, as in the app, and a bad option comes with them.
	_, err := tepra.NewPages(tepra.Tape18mm, []string{"a", ""}, tepra.WithMarginMM(0))
	if !errors.Is(err, tepra.ErrNoText) || !errors.Is(err, tepra.ErrInvalidOptions) || !strings.Contains(err.Error(), "page 2:") {
		t.Errorf("empty page: %v, want ErrNoText on page 2, and ErrInvalidOptions", err)
	}
}

func TestNewPages_reportsAllLayoutErrors(t *testing.T) {
	texts := []string{
		"A",
		"Very long text on page 2",
		"Very long text on page 3",
	}
	_, err := tepra.NewPages(tepra.Tape18mm, texts, tepra.WithLengthMM(20), tepra.WithFontSize(20))
	if err == nil {
		t.Fatal("no error")
	}
	if !errors.Is(err, tepra.ErrNoRoom) {
		t.Errorf("got %v, want ErrNoRoom", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "page 2:") || !strings.Contains(msg, "page 3:") {
		t.Errorf("error %q should mention both page 2 and page 3", msg)
	}
	if strings.Contains(msg, "page 1:") {
		t.Errorf("page 1 should fit, got %q", msg)
	}
}
