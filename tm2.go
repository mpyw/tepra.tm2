//declscope:core

// Package tm2 reads and writes .tm2 files, the labels of TEPRA LINK 2.
//
// TEPRA LINK 2 is KING JIM's app for the "TEPRA" PRO label printers. A .tm2
// file is one line of JSON. This package builds such files, so a label can be
// made elsewhere, opened in the app and printed from there. It does not talk
// to a printer.
//
// The format is not documented. This package follows files the app wrote, and
// writes every key they have. The keys of an object come in the order of the
// fields of its type, not of those files.
//
// Lengths are in points (1/72 inch), as in the file, unless a name says mm.
package tm2

// Rect is a frame: x, y, width and height, in points. y grows down the tape.
type Rect [4]float64

// Color is red, green, blue and alpha, each from 0 to 1.
type Color [4]float64

// The colors the app writes for black text on a white tape.
var (
	Black = Color{0, 0, 0, 1}
	White = Color{1, 1, 1, 1}
)

// DefaultMarginMM is the blank tape before and after a label as the app sets
// it, in mm.
const DefaultMarginMM = 1.0

// The fonts of TEPRA LINK 2, by the names the app writes.
const (
	FontGothic     = "OTBGoStdR-Normal"
	FontGothicBold = "OTBGoStdB-Normal"
)
