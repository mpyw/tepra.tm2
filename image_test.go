package tepra_test

import (
	"errors"
	"testing"

	"github.com/mpyw/tepra.tm2"
)

func TestParseImageMode(t *testing.T) {
	for _, m := range []tepra.ImageMode{tepra.ImageModeDiffusion, tepra.ImageModeThreshold, tepra.ImageModeScreen} {
		if got, err := tepra.ParseImageMode(m.String()); err != nil || got != m {
			t.Errorf("ParseImageMode(%q) = %v, %v", m.String(), got, err)
		}
	}
	for _, name := range []string{"", "dither"} {
		if _, err := tepra.ParseImageMode(name); !errors.Is(err, tepra.ErrUnknownName) {
			t.Errorf("ParseImageMode(%q): %v", name, err)
		}
	}
	if got := tepra.ImageMode(0).String(); got != "ImageMode(0)" {
		t.Errorf("String() = %q", got)
	}
}

func TestOptions_imageMode(t *testing.T) {
	pic := fixturePicture(t, 10, 10, false)
	f, err := tepra.NewText(tepra.Tape18mm, "x", tepra.WithIcon(pic), tepra.WithBackground(pic), tepra.WithImageMode(tepra.ImageModeScreen), tepra.WithImageThreshold(64))
	if err != nil {
		t.Fatal(err)
	}
	for i, d := range f.Documents[0].Drawers[:2] {
		img := fixtureAs[*tepra.ImageDrawer](t, d)
		if img.ImageMode != tepra.ImageModeScreen || img.ImageThreshold != 64 {
			t.Errorf("picture %d: mode %v, threshold %d", i, img.ImageMode, img.ImageThreshold)
		}
	}
	f, err = tepra.NewText(tepra.Tape18mm, "x", tepra.WithIcon(pic))
	if err != nil {
		t.Fatal(err)
	}
	if img := fixtureAs[*tepra.ImageDrawer](t, f.Documents[0].Drawers[0]); img.ImageMode != tepra.ImageModeThreshold || img.ImageThreshold != tepra.DefaultImageThreshold {
		t.Errorf("default: mode %v, threshold %d", img.ImageMode, img.ImageThreshold)
	}
}
