package tm2_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/mpyw/tepra.tm2"
)

func TestNewText_icon(t *testing.T) {
	icon := fixturePicture(t, 20, 10, false)
	f, err := tm2.NewText(tm2.Tape18mm, "カイロ", tm2.WithIcon(icon), tm2.WithFontSize(30))
	if err != nil {
		t.Fatal(err)
	}
	doc := f.Documents[0]
	if len(doc.Drawers) != 2 {
		t.Fatalf("got %d drawers, want the icon and the text", len(doc.Drawers))
	}
	img := fixtureAs[*tm2.ImageDrawer](t, doc.Drawers[0])
	text := fixtureAs[*tm2.TextDrawer](t, doc.Drawers[1])
	// As high as the tape prints, and twice as wide, as the picture is.
	if want := (tm2.Rect{0, 4.4, 86.4, 43.2}); !fixtureFrameNear(img.Frame, want) {
		t.Errorf("icon frame %v, want %v", img.Frame, want)
	}
	if !bytes.Equal(img.Image, icon) {
		t.Error("a PNG is not kept as it is")
	}
	// 1.5 mm after the icon; three ems of 30 points.
	gap := tm2.MMToPoints(1.5)
	if want := (tm2.Rect{86.4 + gap, 4.4, 90, 43.2}); !fixtureFrameNear(text.Frame, want) {
		t.Errorf("text frame %v, want %v", text.Frame, want)
	}
	if got, want := doc.Frame[2], 86.4+gap+90; got-want > 1e-9 || want-got > 1e-9 {
		t.Errorf("label %v points long, want %v", got, want)
	}
	if f.TextWords != "カイロ" {
		t.Errorf("textWords %q", f.TextWords)
	}
}

func TestNewPour_backgroundAndPadding(t *testing.T) {
	bg := fixturePicture(t, 30, 10, true)
	icon := fixturePicture(t, 10, 10, false)
	f, err := tm2.NewPour(tm2.Tape18mm, "c", []string{"a", "b"}, tm2.WithBackground(bg), tm2.WithIcon(icon), tm2.WithPaddingMM(2), tm2.WithFontSize(10))
	if err != nil {
		t.Fatal(err)
	}
	doc := f.Documents[0]
	if len(doc.Drawers) != 3 {
		t.Fatalf("got %d drawers, want the background, the icon and the text", len(doc.Drawers))
	}
	back := fixtureAs[*tm2.ImageDrawer](t, doc.Drawers[0])
	ic := fixtureAs[*tm2.ImageDrawer](t, doc.Drawers[1])
	pour := fixtureAs[*tm2.PourDrawer](t, doc.Drawers[2])
	pad := tm2.MMToPoints(2)
	inner := 43.2 - 2*pad
	if want := (tm2.Rect{pad, 4.4 + pad, inner, inner}); !fixtureFrameNear(ic.Frame, want) {
		t.Errorf("icon frame %v, want %v", ic.Frame, want)
	}
	textX := pad + inner + tm2.MMToPoints(1.5)
	if want := (tm2.Rect{textX, 4.4 + pad, 10, inner}); !fixtureFrameNear(pour.Frame, want) {
		t.Errorf("text frame %v, want %v", pour.Frame, want)
	}
	// The background covers the whole label.
	if want := (tm2.Rect{0, 4.4, textX + 10 + pad, 43.2}); !fixtureFrameNear(back.Frame, want) || back.Frame[2] != doc.Frame[2] {
		t.Errorf("background frame %v, want %v, as long as the label (%v)", back.Frame, want, doc.Frame[2])
	}
	// The JPEG is stored as a PNG, of the label's shape.
	if !bytes.HasPrefix(back.Image, []byte("\x89PNG")) {
		t.Fatal("background is not a PNG")
	}
	if !fixtureShapeNear(t, back.Image, back.Frame) {
		t.Error("background is not of the label's shape")
	}
}

func TestNewText_rejectsBadPictures(t *testing.T) {
	tests := map[string]struct {
		opt  tm2.Option
		want error
	}{
		"not a picture icon":       {tm2.WithIcon([]byte("not a picture")), tm2.ErrBadImage},
		"not a picture background": {tm2.WithBackground([]byte("not a picture")), tm2.ErrBadImage},
		"padding past the tape":    {tm2.WithPaddingMM(8), tm2.ErrNoRoom},
	}
	for name, tt := range tests {
		if _, err := tm2.NewText(tm2.Tape18mm, "x", tt.opt); !errors.Is(err, tt.want) {
			t.Errorf("%s: %v, want %v", name, err, tt.want)
		}
		if _, err := tm2.NewPour(tm2.Tape18mm, "c", []string{"x"}, tt.opt); !errors.Is(err, tt.want) {
			t.Errorf("%s, pour: %v, want %v", name, err, tt.want)
		}
	}
}
