package tm2

// FillMode is how a text box fills the box behind its text. Each mode was
// seen in TEPRA LINK 2 for Android, opening a file this package wrote. The
// app crashes on a file with any other value, from 3 up.
type FillMode int

// The fill modes.
const (
	// FillNone is black text on the tape.
	FillNone FillMode = 0
	// FillInverted is white text on a black box the size of the frame. The
	// app shows only the last line of such a box, so NewPour and NewText
	// give each line a box of its own.
	FillInverted FillMode = 1
	// FillShaded is black text on a light, dotted box the size of the frame.
	FillShaded FillMode = 2
)

// ParseFillMode returns the fill mode of a name: none, inverted or shaded.
func ParseFillMode(name string) (FillMode, error) {
	return parseName[FillMode]("fill", fillModeNames, name)
}

// Valid reports whether m is one of the fill modes the app opens.
func (m FillMode) Valid() bool {
	return m >= 0 && int(m) < len(fillModeNames)
}

// String returns the name of m, as ParseFillMode takes it.
func (m FillMode) String() string {
	return nameOf("FillMode", fillModeNames, m)
}

// fillModeNames are the names of String and ParseFillMode.
var fillModeNames = []string{FillNone: "none", FillInverted: "inverted", FillShaded: "shaded"}
