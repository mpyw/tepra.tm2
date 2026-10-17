package tepra_test

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/mpyw/tepra.tm2"
)

func TestNewPour_writesTheKeysTheAppWrites(t *testing.T) {
	app := decodeFixture(t, readFixture(t, fixturePourPath))
	f, err := tepra.NewPour(tepra.Tape18mm, "品名", []string{"a\nb", "c"})
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	got := decodeFixture(t, buf.Bytes())

	pairs := []struct {
		name     string
		app, got *fixtureObject
	}{
		{"file", fixtureAs[*fixtureObject](t, app), fixtureAs[*fixtureObject](t, got)},
		{"document", fixtureDocument(t, app), fixtureDocument(t, got)},
		{"pour drawer", fixtureDrawer(t, app, "pour"), fixtureDrawer(t, got, "pour")},
	}
	// A drawer starts with the head the app writes.
	if got := fixtureDrawer(t, got, "pour").keys[:3]; !slices.Equal(got, []string{"type", "identifier", ""}) {
		t.Errorf("pour drawer starts with %q", got)
	}
	for _, p := range pairs {
		if !fixtureSameKeys(p.app.keys, p.got.keys) {
			t.Errorf("%s keys:\napp %q\ngot %q", p.name, p.app.keys, p.got.keys)
		}
	}
}

func TestNewPour(t *testing.T) {
	at := time.Date(2026, 10, 17, 16, 39, 0, 0, time.Local)
	values := []string{"ウエットティッシュ\nポケットティッシュ", "割り箸\nスプーン\nフォーク", "カイロ"}
	f, err := tepra.NewPour(tepra.Tape18mm, "品名", values, tepra.WithName("日用品"), tepra.WithTime(at), tepra.WithBold())
	if err != nil {
		t.Fatal(err)
	}

	if f.FileName != "日用品.tm2" || f.Date != "2026/10/17 16:39" || f.Version != tepra.FileVersion {
		t.Errorf("fileName %q, date %q, version %q", f.FileName, f.Date, f.Version)
	}
	if f.TextWords != values[0] {
		t.Errorf("textWords %q, want the first value", f.TextWords)
	}
	if len(f.TableHeader) != tepra.FileTableColumns || f.TableHeader[0] != "品名" {
		t.Errorf("tableHeader %q", f.TableHeader)
	}
	if len(f.TableData) != len(values) {
		t.Fatalf("got %d rows, want %d", len(f.TableData), len(values))
	}
	for i, row := range f.TableData {
		if len(row) != tepra.FileTableColumns || row[0] != values[i] || strings.Join(row[1:], "") != "" {
			t.Errorf("row %d: %q", i, row)
		}
	}

	doc := f.Documents[0]
	if !doc.PourMode || !doc.MultiplePage || len(doc.Drawers) != 1 {
		t.Fatalf("PourMode %v, MultiplePage %v, %d drawers", doc.PourMode, doc.MultiplePage, len(doc.Drawers))
	}
	d := fixtureAs[*tepra.PourDrawer](t, doc.Drawers[0])
	// Three lines, the most of any value, fill the 43.2 points of the tape.
	if d.FontSize != 11.5 || d.FontName != tepra.FontGothicBold || d.ColumnNumber != 0 || d.Text != values[0] {
		t.Errorf("drawer: size %v, font %q, column %d, text %q", d.FontSize, d.FontName, d.ColumnNumber, d.Text)
	}
	// Fitted to the frame, a value of fewer lines would run past its sides.
	if !d.SizePriority {
		t.Error("the font size does not hold on every label")
	}
	// The widest line is "ウエットティッシュ" and "ポケットティッシュ", 9 ems.
	if want := (tepra.Rect{0, 4.4, 104, 43.2}); !fixtureFrameNear(d.Frame, want) {
		t.Errorf("drawer frame %v, want %v", d.Frame, want)
	}
	if want := (tepra.Rect{0, 0, 104, 52}); doc.Frame != want || doc.PrintableFrame != want {
		t.Errorf("document frame %v, printable %v, want %v", doc.Frame, doc.PrintableFrame, want)
	}
	if *f.SavedLength != doc.LengthMM() {
		t.Errorf("savedLength %v, length %v", *f.SavedLength, doc.LengthMM())
	}
	if d.Identifier == "" || doc.Identifier == "" || d.Identifier == doc.Identifier {
		t.Errorf("identifiers %q and %q", d.Identifier, doc.Identifier)
	}
}

func TestNewPour_takesTheFontSizeGiven(t *testing.T) {
	f, err := tepra.NewPour(tepra.Tape18mm, "c", []string{"ab"}, tepra.WithFontSize(20))
	if err != nil {
		t.Fatal(err)
	}
	d := fixtureAs[*tepra.PourDrawer](t, f.Documents[0].Drawers[0])
	// Two narrow characters are one em.
	if d.FontSize != 20 || d.Frame[2] != 20 {
		t.Errorf("size %v, width %v", d.FontSize, d.Frame[2])
	}
}

func TestNewPour_rejects(t *testing.T) {
	tests := map[string]struct {
		values []string
		want   error
	}{
		"no values":      {nil, tepra.ErrNoText},
		"too many rows":  {make([]string, tepra.PourMaxRows+1), tepra.ErrOverLimit},
		"too long value": {[]string{strings.Repeat("あ", tepra.PourMaxCellRunes+1)}, tepra.ErrOverLimit},
	}
	for name, tt := range tests {
		if _, err := tepra.NewPour(tepra.Tape18mm, "c", tt.values); !errors.Is(err, tt.want) {
			t.Errorf("%s: %v, want %v", name, err, tt.want)
		}
	}
}

func TestNewPour_reportsEveryError(t *testing.T) {
	// A bad option, and two bad values counted from 1, as rows are.
	_, err := tepra.NewPour(tepra.Tape18mm, "c", []string{"123", "123456789012", "x"},
		tepra.WithCode(tepra.CodeJAN13), tepra.WithMarginMM(0))
	if !errors.Is(err, tepra.ErrInvalidOptions) || !errors.Is(err, tepra.ErrBadCode) {
		t.Fatalf("%v, want ErrInvalidOptions and ErrBadCode", err)
	}
	for _, want := range []string{"margin", "value 1:", "value 3:"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("%v, want %q in it", err, want)
		}
	}
	if strings.Contains(err.Error(), "value 2:") {
		t.Errorf("%v: value 2 is good", err)
	}
}
