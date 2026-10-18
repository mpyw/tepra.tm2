package tm2_test

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mpyw/tepra.tm2"
)

func TestNewText_iconOfEachFormat(t *testing.T) {
	var gifData bytes.Buffer
	pal := image.NewPaletted(image.Rect(0, 0, 6, 3), color.Palette{color.Black, color.White})
	if err := gif.Encode(&gifData, pal, nil); err != nil {
		t.Fatal(err)
	}
	pictures := map[string][]byte{
		"png":  fixturePicture(t, 6, 3, false),
		"jpeg": fixturePicture(t, 6, 3, true),
		"gif":  gifData.Bytes(),
	}
	for format, pic := range pictures {
		f, err := tm2.NewText(tm2.Tape18mm, "x", tm2.WithIcon(pic))
		if err != nil {
			t.Fatalf("%s: %v", format, err)
		}
		// The app stores pictures as PNG, so the others are converted.
		img := fixtureAs[*tm2.ImageDrawer](t, f.Documents[0].Drawers[0])
		cfg, err := png.DecodeConfig(bytes.NewReader(img.Image))
		if err != nil || cfg.Width != 6 || cfg.Height != 3 {
			t.Errorf("%s: %+v, %v; want a 6 by 3 PNG", format, cfg, err)
		}
		// The icon is as high as the tape prints, and keeps its shape.
		if w, h := img.Frame[2], img.Frame[3]; math.Abs(w-2*h) > 1e-9 {
			t.Errorf("%s: icon %v by %v, want twice as wide as high", format, w, h)
		}
	}
	if _, err := tm2.NewText(tm2.Tape18mm, "x", tm2.WithBackground([]byte("not a picture"))); !errors.Is(err, tm2.ErrBadImage) {
		t.Errorf("bad background: %v, want ErrBadImage", err)
	}
}

func TestNewText_iconWithFixedLength(t *testing.T) {
	pic := fixturePicture(t, 10, 10, false)
	f, err := tm2.NewText(tm2.Tape18mm, "x", tm2.WithIcon(pic), tm2.WithLengthMM(80))
	if err != nil {
		t.Fatal(err)
	}
	doc := f.Documents[0]
	icon := fixtureAs[*tm2.ImageDrawer](t, doc.Drawers[0])
	text := fixtureAs[*tm2.TextDrawer](t, doc.Drawers[1])
	// The icon and the text move together, centred: the icon is still
	// before the text, with the same gap, and the blank on each side is the
	// same.
	gap := text.Frame[0] - (icon.Frame[0] + icon.Frame[2])
	if math.Abs(gap-tm2.MMToPoints(1.5)) > 1e-9 {
		t.Errorf("gap %v between icon and text", gap)
	}
	left, right := icon.Frame[0], doc.Frame[2]-(text.Frame[0]+text.Frame[2])
	if math.Abs(left-right) > 1e-9 {
		t.Errorf("blank %v before, %v after", left, right)
	}
}

func TestNewPages_layoutError(t *testing.T) {
	_, err := tm2.NewPages(tm2.Tape18mm, []string{"a", "b"}, tm2.WithPaddingMM(10))
	// The same error on both pages is reported once, with both (#9).
	if !errors.Is(err, tm2.ErrNoRoom) || !strings.HasPrefix(err.Error(), "pages 1, 2: ") || strings.Contains(err.Error(), "\n") {
		t.Errorf("%v, want one ErrNoRoom on pages 1 and 2", err)
	}
	// Different errors are each reported.
	_, err = tm2.NewPages(tm2.Tape18mm, []string{"a", strings.Repeat("あ", 50), strings.Repeat("あ", 60)}, tm2.WithFontSize(20), tm2.WithLengthMM(20))
	if !errors.Is(err, tm2.ErrNoRoom) || !strings.Contains(err.Error(), "pages 2, 3: ") {
		t.Errorf("%v, want ErrNoRoom on pages 2 and 3", err)
	}
}

func TestNewPour_tooManyInvertedLines(t *testing.T) {
	value := strings.Repeat("a\n", tm2.FileTableColumns) + "a"
	_, err := tm2.NewPour(tm2.Tape50mm, "c", []string{value}, tm2.WithFill(tm2.FillInverted), tm2.WithFontSize(1))
	if !errors.Is(err, tm2.ErrOverLimit) {
		t.Errorf("%v, want ErrOverLimit", err)
	}
}

func TestFile_WriteFile_rejectsBadPage(t *testing.T) {
	f, err := tm2.NewText(tm2.Tape18mm, "x")
	if err != nil {
		t.Fatal(err)
	}
	f.CurrentPage = 1
	path := filepath.Join(t.TempDir(), "x.tm2")
	if err := f.WriteFile(path); !errors.Is(err, tm2.ErrCrashesApp) {
		t.Errorf("page 1 of 1: %v, want ErrCrashesApp", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a file was written: %v", err)
	}
}

// encoding/json calls MarshalJSONTo and UnmarshalJSONFrom when a type has
// them. MarshalJSON and UnmarshalJSON are for callers that use them by
// name, and must give the same.
func TestMarshalJSON_byName(t *testing.T) {
	f, err := tm2.ReadFile(fixturePourPath)
	if err != nil {
		t.Fatal(err)
	}
	var want bytes.Buffer
	if _, err := f.WriteTo(&want); err != nil {
		t.Fatal(err)
	}
	got, err := f.MarshalJSON()
	if err != nil || !bytes.Equal(got, want.Bytes()) {
		t.Errorf("File.MarshalJSON: %v; differs from WriteTo: %v", err, !bytes.Equal(got, want.Bytes()))
	}
	ds := f.Documents[0].Drawers
	b, err := ds.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var back tm2.Drawers
	if err := back.UnmarshalJSON(b); err != nil || len(back) != len(ds) {
		t.Errorf("Drawers.UnmarshalJSON: %v, %d drawers", err, len(back))
	}
}

// The bytes given to WithIcon and WithBackground are copied: the caller may
// reuse its buffer (#9).
func TestWithIcon_copiesThePicture(t *testing.T) {
	pic := fixturePicture(t, 4, 4, false)
	opt := tm2.WithIcon(pic)
	clear(pic)
	f, err := tm2.NewText(tm2.Tape18mm, "x", opt)
	if err != nil {
		t.Fatalf("%v: the option read the cleared buffer", err)
	}
	if fixtureAs[*tm2.ImageDrawer](t, f.Documents[0].Drawers[0]).Image[1] != 'P' {
		t.Error("the icon is not the PNG given")
	}
}

// A nil drawer, or a raw one with no JSON, is ErrBadFile, not a panic (#9).
func TestDrawers_nil(t *testing.T) {
	var text *tm2.TextDrawer
	for name, d := range map[string]tm2.Drawer{"nil": nil, "nil text": text, "raw with no JSON": &tm2.RawDrawer{Type: "shape"}} {
		f, err := tm2.NewText(tm2.Tape18mm, "x")
		if err != nil {
			t.Fatal(err)
		}
		f.Documents[0].Drawers = append(f.Documents[0].Drawers, d)
		var buf bytes.Buffer
		if _, err := f.WriteTo(&buf); !errors.Is(err, tm2.ErrBadFile) {
			t.Errorf("%s: %v, want ErrBadFile", name, err)
		}
	}
}
