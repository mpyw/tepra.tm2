package tm2_test

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/mpyw/tepra.tm2"
)

func TestLookupTape(t *testing.T) {
	got, err := tm2.LookupTape(18)
	if err != nil || got != tm2.Tape18mm {
		t.Errorf("LookupTape(18) = %v, %v", got, err)
	}
	if _, err := tm2.LookupTape(5); !errors.Is(err, tm2.ErrTapeNotSupported) {
		t.Errorf("LookupTape(5): %v, want ErrTapeNotSupported", err)
	}
}

// TestTapes_asTheAppWrote checks each tape against the empty label the app
// wrote on it: the frame, the printable band, and the text box in the band.
func TestTapes_asTheAppWrote(t *testing.T) {
	for _, tape := range tm2.Tapes {
		path := fmt.Sprintf("testdata/blank-%dmm.tm2", tape.WidthMM)
		f, err := tm2.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		doc := f.Documents[0]
		box := fixtureAs[*tm2.TextDrawer](t, doc.Drawers[0]).Frame
		near := func(a, b float64) bool { return math.Abs(a-b) < 1e-4 }
		if doc.TapeWidthMM != tape.WidthMM || doc.PrintableArea != tape.PrintableHeight || doc.Frame[3] != tape.FrameHeight {
			t.Errorf("%s: width %d, printable %v, frame %v; want %+v", path, doc.TapeWidthMM, doc.PrintableArea, doc.Frame, tape)
		}
		if !near(box[1], tape.PrintableTop()) || !near(box[3], tape.PrintableHeight) {
			t.Errorf("%s: text box %v, want y %v, height %v", path, box, tape.PrintableTop(), tape.PrintableHeight)
		}
	}
}

func TestTapes_noRoomForText(t *testing.T) {
	// The padding of a border leaves 0.1 mm of the 3.1 mm a 4 mm tape prints.
	_, err := tm2.NewText(tm2.Tape4mm, "a\nb", tm2.WithBorder(tm2.BorderBox))
	if !errors.Is(err, tm2.ErrNoRoom) {
		t.Errorf("%v, want ErrNoRoom", err)
	}
	if _, err := tm2.NewText(tm2.Tape4mm, "a\nb", tm2.WithBorder(tm2.BorderBox), tm2.WithPaddingMM(0.3)); err != nil {
		t.Error(err)
	}
}

func TestTapes_minLength(t *testing.T) {
	// The app shows a label shorter than 12 mm as 12 mm long, as the empty
	// label it wrote on 4 mm tape is.
	f, err := tm2.NewText(tm2.Tape4mm, "a")
	if err != nil {
		t.Fatal(err)
	}
	doc := f.Documents[0]
	if got := doc.LengthMM(); math.Abs(got-12) > 1e-9 {
		t.Errorf("length %v mm, want 12", got)
	}
	d := fixtureAs[*tm2.TextDrawer](t, doc.Drawers[0])
	if left, right := d.Frame[0], doc.Frame[2]-d.Frame[0]-d.Frame[2]; math.Abs(left-right) > 1e-9 {
		t.Errorf("text at %v in %v, want it centred", d.Frame, doc.Frame[2])
	}
	app, err := tm2.ReadFile("testdata/blank-4mm.tm2")
	if err != nil {
		t.Fatal(err)
	}
	if got := *app.SavedLength; math.Abs(got-12) > 1e-6 {
		t.Errorf("the app's 4 mm label is %v mm", got)
	}
}
