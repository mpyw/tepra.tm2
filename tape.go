package tepra

import (
	"fmt"
	"slices"
)

// Tape is a tape width and the sizes the app writes for it.
type Tape struct {
	// WidthMM is the tape's width in mm, as in SettingsTapeWidth.
	WidthMM int
	// PrintableHeight is the height the printer can print on, in points.
	PrintableHeight float64
	// FrameHeight is the height of the label's frame, in points. The
	// printable band sits in its middle.
	FrameHeight float64
}

// PrintableTop is the y of the printable band in the frame, in points.
func (t Tape) PrintableTop() float64 {
	return (t.FrameHeight - t.PrintableHeight) / 2
}

// The tapes whose sizes are known. Each one's sizes come from a file the
// app wrote for it, in testdata.
var (
	Tape4mm  = Tape{WidthMM: 4, PrintableHeight: 8.8, FrameHeight: 12}
	Tape6mm  = Tape{WidthMM: 6, PrintableHeight: 14.400001, FrameHeight: 18}
	Tape9mm  = Tape{WidthMM: 9, PrintableHeight: 21.6, FrameHeight: 26}
	Tape12mm = Tape{WidthMM: 12, PrintableHeight: 28.800001, FrameHeight: 35}
	Tape18mm = Tape{WidthMM: 18, PrintableHeight: 43.2, FrameHeight: 52}
	Tape24mm = Tape{WidthMM: 24, PrintableHeight: 51.2, FrameHeight: 69}
	Tape36mm = Tape{WidthMM: 36, PrintableHeight: 76.8, FrameHeight: 103}
	Tape50mm = Tape{WidthMM: 50, PrintableHeight: 135.8, FrameHeight: 142}
)

// Tapes are the tapes whose sizes are known, narrowest first. Other widths
// need a file the app wrote for them.
var Tapes = []Tape{Tape4mm, Tape6mm, Tape9mm, Tape12mm, Tape18mm, Tape24mm, Tape36mm, Tape50mm}

// LookupTape returns the tape of a width in mm.
func LookupTape(widthMM int) (Tape, error) {
	i := slices.IndexFunc(Tapes, func(t Tape) bool { return t.WidthMM == widthMM })
	if i < 0 {
		return Tape{}, fmt.Errorf("%w: %d mm; supported: %v mm", ErrTapeNotSupported, widthMM, tapeWidths())
	}
	return Tapes[i], nil
}

// tapeWidths lists the widths of Tapes, in mm.
func tapeWidths() []int {
	widths := make([]int, len(Tapes))
	for i, t := range Tapes {
		widths[i] = t.WidthMM
	}
	return widths
}
