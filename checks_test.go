package tepra_test

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"math"
	"strings"
	"testing"

	"github.com/mpyw/tepra.tm2"
)

// Numbers that are not numbers fail as options, not later when the file is
// written (#5).
func TestOptions_rejectNaNAndInf(t *testing.T) {
	nan, inf := math.NaN(), math.Inf(1)
	tests := map[string]tepra.Option{
		"font size NaN":    tepra.WithFontSize(nan),
		"font size Inf":    tepra.WithFontSize(inf),
		"border width NaN": tepra.WithBorderWidth(nan),
		"border width Inf": tepra.WithBorderWidth(inf),
		"padding NaN":      tepra.WithPaddingMM(nan),
		"padding Inf":      tepra.WithPaddingMM(inf),
		"margin NaN":       tepra.WithMarginMM(nan),
		"length NaN":       tepra.WithLengthMM(nan),
	}
	for name, opt := range tests {
		if _, err := tepra.NewText(tepra.Tape18mm, "x", opt); !errors.Is(err, tepra.ErrInvalidOptions) {
			t.Errorf("%s: %v, want ErrInvalidOptions", name, err)
		}
	}
}

// Text a file cannot hold fails at once, with the place it is in (#5).
func TestBuilders_rejectTextThatIsNotUTF8(t *testing.T) {
	bad := "\xff"
	if _, err := tepra.NewText(tepra.Tape18mm, bad); !errors.Is(err, tepra.ErrBadText) {
		t.Errorf("NewText: %v, want ErrBadText", err)
	}
	_, err := tepra.NewPages(tepra.Tape18mm, []string{"a", bad})
	if !errors.Is(err, tepra.ErrBadText) || !strings.Contains(err.Error(), "page 2:") {
		t.Errorf("NewPages: %v, want ErrBadText on page 2", err)
	}
	_, err = tepra.NewPour(tepra.Tape18mm, bad, []string{"a", bad})
	if !errors.Is(err, tepra.ErrBadText) || !strings.Contains(err.Error(), "value 2:") || !strings.Contains(err.Error(), "column title:") {
		t.Errorf("NewPour: %v, want ErrBadText on value 2 and the column title", err)
	}
}

// An empty value is no label, in pour as in pages (#5).
func TestNewPour_rejectsEmptyValues(t *testing.T) {
	_, err := tepra.NewPour(tepra.Tape18mm, "c", []string{"a", ""})
	if !errors.Is(err, tepra.ErrNoText) || !strings.Contains(err.Error(), "value 2:") {
		t.Errorf("%v, want ErrNoText on value 2", err)
	}
}

// "\r\n" and a lone "\r" break lines as "\n" does (#5).
func TestBuilders_carriageReturns(t *testing.T) {
	f, err := tepra.NewText(tepra.Tape18mm, "a\r\nb\rc")
	if err != nil {
		t.Fatal(err)
	}
	if got := fixtureAs[*tepra.TextDrawer](t, f.Documents[0].Drawers[0]).Text; got != "a\nb\nc" {
		t.Errorf("NewText: %q", got)
	}
	values := []string{"a\r\nb"}
	f, err = tepra.NewPour(tepra.Tape18mm, "c", values)
	if err != nil {
		t.Fatal(err)
	}
	if got := f.TableData[0][0]; got != "a\nb" {
		t.Errorf("NewPour: %q", got)
	}
	if values[0] != "a\r\nb" {
		t.Error("NewPour changed the caller's values")
	}
	f, err = tepra.NewPages(tepra.Tape18mm, []string{"a\rb"})
	if err != nil {
		t.Fatal(err)
	}
	if got := fixtureAs[*tepra.TextDrawer](t, f.Documents[0].Drawers[0]).Text; got != "a\nb" {
		t.Errorf("NewPages: %q", got)
	}
}

// A fixed length too short for the text is ErrNoRoom, not font size 0
// (#5).
func TestLengthMM_tooShortForTheText(t *testing.T) {
	long := strings.Repeat("あ", 300)
	if _, err := tepra.NewPour(tepra.Tape18mm, "c", []string{long}, tepra.WithLengthMM(10)); !errors.Is(err, tepra.ErrNoRoom) {
		t.Errorf("NewPour: %v, want ErrNoRoom", err)
	}
	if _, err := tepra.NewText(tepra.Tape18mm, long, tepra.WithLengthMM(10)); !errors.Is(err, tepra.ErrNoRoom) {
		t.Errorf("NewText: %v, want ErrNoRoom", err)
	}
}

// A label of automatic length is at most 3000 mm, as a fixed one (#5).
func TestLayout_tooLong(t *testing.T) {
	if _, err := tepra.NewText(tepra.Tape18mm, "x", tepra.WithFontSize(1e300)); !errors.Is(err, tepra.ErrOverLimit) {
		t.Errorf("font size 1e300: %v, want ErrOverLimit", err)
	}
	if _, err := tepra.NewText(tepra.Tape18mm, strings.Repeat("あ", 500)); !errors.Is(err, tepra.ErrOverLimit) {
		t.Errorf("500 characters: %v, want ErrOverLimit", err)
	}
	if _, err := tepra.NewText(tepra.Tape18mm, strings.Repeat("あ", 100)); err != nil {
		t.Errorf("100 characters: %v", err)
	}
}

// An error on many values or pages is reported once, and the message stays
// short however long the list is (#18).
func TestErrors_bounded(t *testing.T) {
	_, err := tepra.NewPour(tepra.Tape18mm, "c", make([]string, 100000))
	if !errors.Is(err, tepra.ErrOverLimit) || len(err.Error()) > 200 {
		t.Errorf("100000 values: %d bytes: %.200v", len(err.Error()), err)
	}
	_, err = tepra.NewPour(tepra.Tape18mm, "c", make([]string, 1000))
	if !errors.Is(err, tepra.ErrNoText) || err.Error() != "values 1, 2, 3, 4, 5 and 995 more: tepra: no text" {
		t.Errorf("1000 empty values: %v", err)
	}
	_, err = tepra.NewPages(tepra.Tape18mm, make([]string, 1000))
	if !errors.Is(err, tepra.ErrNoText) || strings.Count(err.Error(), "\n") != 0 {
		t.Errorf("1000 empty pages: %v", err)
	}
	// Many different errors: the first ten, then a count.
	values := make([]string, 30)
	for i := range values {
		values[i] = strings.Repeat("1", i+1)
	}
	_, err = tepra.NewPour(tepra.Tape18mm, "c", values, tepra.WithCode(tepra.CodeJAN13))
	if !errors.Is(err, tepra.ErrBadCode) || !strings.HasSuffix(err.Error(), "and 19 more errors") || strings.Count(err.Error(), "\n") != 10 {
		t.Errorf("29 bad codes: %v", err)
	}
}

// A name that is not UTF-8 fails as an option, not in WriteTo (#19).
func TestWithName_rejectsTextThatIsNotUTF8(t *testing.T) {
	_, err := tepra.NewText(tepra.Tape18mm, "x", tepra.WithName("\xff"))
	if !errors.Is(err, tepra.ErrInvalidOptions) || !errors.Is(err, tepra.ErrBadText) {
		t.Errorf("%v, want ErrInvalidOptions and ErrBadText", err)
	}
}

// Text of blanks only prints nothing, so it is no text (#19).
func TestBuilders_rejectBlankText(t *testing.T) {
	for _, text := range []string{" ", "\n", "\r", "　"} {
		if _, err := tepra.NewText(tepra.Tape18mm, text); !errors.Is(err, tepra.ErrNoText) {
			t.Errorf("NewText(%q): %v, want ErrNoText", text, err)
		}
		if _, err := tepra.NewPour(tepra.Tape18mm, "c", []string{"a", text}); !errors.Is(err, tepra.ErrNoText) {
			t.Errorf("NewPour(%q): %v, want ErrNoText", text, err)
		}
	}
}

// A picture declaring more pixels than are taken fails from its header,
// before it is decoded (#19).
func TestPictures_tooLarge(t *testing.T) {
	// The header of a PNG of 20000 by 20000 grey pixels, with no pixel data.
	ihdr := binary.BigEndian.AppendUint32([]byte("IHDR"), 20000)
	ihdr = binary.BigEndian.AppendUint32(ihdr, 20000)
	ihdr = append(ihdr, 8, 0, 0, 0, 0)
	png := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\x0d")
	png = binary.BigEndian.AppendUint32(append(png, ihdr...), crc32.ChecksumIEEE(ihdr))
	for name, opt := range map[string]tepra.Option{"icon": tepra.WithIcon(png), "background": tepra.WithBackground(png)} {
		_, err := tepra.NewText(tepra.Tape18mm, "x", opt)
		if !errors.Is(err, tepra.ErrBadImage) || !strings.Contains(err.Error(), "20000 by 20000") {
			t.Errorf("%s: %v, want ErrBadImage for its size", name, err)
		}
	}
}

// customDrawer is a Drawer of the caller's own, with no BaseDrawer.
type customDrawer struct{}

func (customDrawer) DrawerType() string      { return "custom" }
func (customDrawer) Base() *tepra.BaseDrawer { return nil }

// A drawer with no BaseDrawer is ErrBadFile, not a panic (#19).
func TestDrawers_noBase(t *testing.T) {
	f, err := tepra.NewText(tepra.Tape18mm, "x")
	if err != nil {
		t.Fatal(err)
	}
	f.Documents[0].Drawers = append(f.Documents[0].Drawers, customDrawer{})
	if err := f.WriteFile(t.TempDir() + "/x.tm2"); !errors.Is(err, tepra.ErrBadFile) {
		t.Errorf("%v, want ErrBadFile", err)
	}
}
