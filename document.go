package tm2

import "encoding/json/jsontext"

// Document is one label of a file: the tape settings and its drawers.
type Document struct {
	// Drawers are drawn in order, each over the ones before.
	Drawers Drawers `json:"drawers"`
	// Frame is the label without its margins. Its width is the label's
	// length, as the app showed it.
	Frame Rect `json:"frame"`
	// TapeWidthMM is the tape's width in mm. The app shows the label on a
	// tape of this width.
	TapeWidthMM int `json:"SettingsTapeWidth"`
	// PrintableArea is the height of the band the app prints on, in points.
	// It is the tape's PrintableHeight. A smaller value showed a smaller
	// band.
	PrintableArea float64 `json:"SettingsPrintableArea"`
	// AutoLength makes the label as long as Frame. Unset, the label is
	// FixedLength long.
	AutoLength  bool    `json:"SettingsAutoLength"`
	FixedLength float64 `json:"SettingsFixedLength"`
	// Margin is the blank tape before and after Frame, in points.
	Margin float64 `json:"SettingsMargin"`
	// ForegroundColor and BackgroundColor are the colors of the tape's ink
	// and of the tape, as the app shows the label. Swapped, the whole label
	// showed black.
	ForegroundColor Color `json:"SettingsForegroundColor"`
	BackgroundColor Color `json:"SettingsBackgroundColor"`
	// TapeVertical turns the whole label to run down the tape.
	TapeVertical bool `json:"SettingsTapeVertical"`
	// MultiplePage shows the app's controls for more than one page. The app
	// sets it on a pour label.
	MultiplePage bool `json:"MultiplePage"`
	// PourMode shows the label in the app's pour form (流し込み), filled
	// from the file's TableData. Unset, it is a plain label (一般).
	PourMode bool `json:"PourMode"`

	// Below: written as the app writes them. Values tried showed no change
	// in the app, or none were tried.
	// Name and PrintableFrame: another name, and half the width.
	Name                string `json:"name"`
	PrintableFrame      Rect   `json:"printableFrame"`
	Identifier          string `json:"identifier"`
	ForegroundColorName string `json:"SettingsForegroundColorName"`
	BackgroundColorName string `json:"SettingsBackgroundColorName"`

	// Unknown holds the keys this package does not know, as they were read,
	// and writes them back after the others. A newer app may write keys a
	// file read and written here must not lose.
	Unknown jsontext.Value `json:",embed"`
}

// newDocument returns a label on tape, length points long without margins,
// set as the app sets a new one.
//
//declscope:shared // pour.go and text.go put their drawers on it
func newDocument(tape Tape, length, marginMM float64, drawers ...Drawer) Document {
	frame := Rect{0, 0, length, tape.FrameHeight}
	return Document{
		Drawers:             drawers,
		Name:                "Tape",
		Frame:               frame,
		PrintableFrame:      frame,
		Identifier:          newUUID(),
		TapeWidthMM:         tape.WidthMM,
		AutoLength:          true,
		FixedLength:         MMToPoints(100),
		Margin:              MMToPoints(marginMM),
		ForegroundColor:     Black,
		BackgroundColor:     White,
		ForegroundColorName: "Black",
		BackgroundColorName: "White",
		PrintableArea:       tape.PrintableHeight,
	}
}

// LengthMM is the length of tape the label takes, margins included, in mm.
func (d *Document) LengthMM() float64 {
	return PointsToMM(d.Frame[2] + 2*d.Margin)
}
