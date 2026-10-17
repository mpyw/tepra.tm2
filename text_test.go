package tepra_test

import (
	"bytes"
	"errors"
	"slices"
	"testing"

	"github.com/mpyw/tepra.tm2"
)

func TestNewText(t *testing.T) {
	f, err := tepra.NewText(tepra.Tape18mm, "会議室A\n3F", tepra.WithMarginMM(5))
	if err != nil {
		t.Fatal(err)
	}
	doc := f.Documents[0]
	if doc.PourMode || doc.MultiplePage || len(f.TableData) != 0 {
		t.Errorf("PourMode %v, MultiplePage %v, %d rows", doc.PourMode, doc.MultiplePage, len(f.TableData))
	}
	d := fixtureAs[*tepra.TextDrawer](t, doc.Drawers[0])
	// Two lines; "会議室A" is 3.5 ems.
	if d.FontSize != 17.5 || d.FontName != tepra.FontGothic || d.Frame[2] != 62 {
		t.Errorf("size %v, font %q, width %v", d.FontSize, d.FontName, d.Frame[2])
	}
	if f.TextWords != "会議室A\n3F" || f.FileName != "label.tm2" {
		t.Errorf("textWords %q, fileName %q", f.TextWords, f.FileName)
	}
	// 62 points and 5 mm on each side.
	if got, want := doc.LengthMM(), 62/72.0*25.4+10; got-want > 1e-9 || want-got > 1e-9 {
		t.Errorf("length %v mm, want %v", got, want)
	}
}

func TestNewText_writesTheKeysTheAppWrites(t *testing.T) {
	app := decodeFixture(t, readFixture(t, fixturePourPath))
	f, err := tepra.NewText(tepra.Tape18mm, "x")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	want, got := fixtureDrawer(t, app, "text"), fixtureDrawer(t, decodeFixture(t, buf.Bytes()), "text")
	if !fixtureSameKeys(want.keys, got.keys) || !slices.Equal(got.keys[:3], []string{"type", "identifier", ""}) {
		t.Errorf("text drawer keys:\napp %q\ngot %q", want.keys, got.keys)
	}
}

func TestNewText_rejectsEmptyText(t *testing.T) {
	if _, err := tepra.NewText(tepra.Tape18mm, ""); !errors.Is(err, tepra.ErrNoText) {
		t.Errorf("%v, want ErrNoText", err)
	}
}
