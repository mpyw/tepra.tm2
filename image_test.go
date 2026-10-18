package tm2_test

import (
	"bytes"
	"errors"
	"image/png"
	"testing"

	"github.com/mpyw/tepra.tm2"
)

func TestParseImageMode(t *testing.T) {
	for _, m := range []tm2.ImageMode{tm2.ImageModeDiffusion, tm2.ImageModeThreshold, tm2.ImageModeScreen} {
		if got, err := tm2.ParseImageMode(m.String()); err != nil || got != m {
			t.Errorf("ParseImageMode(%q) = %v, %v", m.String(), got, err)
		}
	}
	for _, name := range []string{"", "dither"} {
		if _, err := tm2.ParseImageMode(name); !errors.Is(err, tm2.ErrUnknownName) {
			t.Errorf("ParseImageMode(%q): %v", name, err)
		}
	}
	if got := tm2.ImageMode(0).String(); got != "ImageMode(0)" {
		t.Errorf("String() = %q", got)
	}
}

func TestOptions_imageMode(t *testing.T) {
	pic := fixturePicture(t, 10, 10, false)
	f, err := tm2.NewText(tm2.Tape18mm, "x", tm2.WithIcon(pic), tm2.WithBackground(pic), tm2.WithImageMode(tm2.ImageModeScreen), tm2.WithImageThreshold(64))
	if err != nil {
		t.Fatal(err)
	}
	for i, d := range f.Documents[0].Drawers[:2] {
		img := fixtureAs[*tm2.ImageDrawer](t, d)
		if img.ImageMode != tm2.ImageModeScreen || img.ImageThreshold != 64 {
			t.Errorf("picture %d: mode %v, threshold %d", i, img.ImageMode, img.ImageThreshold)
		}
	}
	f, err = tm2.NewText(tm2.Tape18mm, "x", tm2.WithIcon(pic))
	if err != nil {
		t.Fatal(err)
	}
	if img := fixtureAs[*tm2.ImageDrawer](t, f.Documents[0].Drawers[0]); img.ImageMode != tm2.ImageModeThreshold || img.ImageThreshold != tm2.DefaultImageThreshold {
		t.Errorf("default: mode %v, threshold %d", img.ImageMode, img.ImageThreshold)
	}
}

// A stretched background has no more pixels than the printer has dots for
// the label: 360 per inch (#8).
func TestBackground_atMostThePrintersDots(t *testing.T) {
	f, err := tm2.NewText(tm2.Tape18mm, "x", tm2.WithBackground(fixturePicture(t, 10, 2000, false)))
	if err != nil {
		t.Fatal(err)
	}
	bg := fixtureAs[*tm2.ImageDrawer](t, f.Documents[0].Drawers[0])
	cfg, err := png.DecodeConfig(bytes.NewReader(bg.Image))
	if err != nil {
		t.Fatal(err)
	}
	// 43.2 points high at 5 dots a point.
	if cfg.Height != 216 || float64(cfg.Width) > bg.Frame[2]*5+1 {
		t.Errorf("%d by %d pixels for a %v by %v frame", cfg.Width, cfg.Height, bg.Frame[2], bg.Frame[3])
	}
}
