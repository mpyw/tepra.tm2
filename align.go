package tm2

// Align is where a text box puts each line across its frame. Each value was
// seen in TEPRA LINK 2 for Android, opening a file this package wrote. The
// app crashes on a file with 3.
type Align int

// The alignments across.
const (
	AlignLeft   Align = 0
	AlignCenter Align = 1
	AlignRight  Align = 2
)

// ParseAlign returns the alignment of a name: left, center or right.
func ParseAlign(name string) (Align, error) {
	return parseName[Align]("alignment", alignNames, name)
}

// Valid reports whether a is one of the alignments the app opens.
func (a Align) Valid() bool {
	return a >= 0 && int(a) < len(alignNames)
}

// String returns the name of a, as ParseAlign takes it.
func (a Align) String() string {
	return nameOf("Align", alignNames, a)
}

// VerticalAlign is where a text box puts its lines down its frame. Each
// value was seen in the app, opening a file this package wrote. The app
// opened 3, and showed it as VerticalAlignMiddle, so it is not offered.
type VerticalAlign int

// The alignments down.
const (
	VerticalAlignTop    VerticalAlign = 0
	VerticalAlignMiddle VerticalAlign = 1
	VerticalAlignBottom VerticalAlign = 2
)

// ParseVerticalAlign returns the alignment of a name: top, middle or bottom.
func ParseVerticalAlign(name string) (VerticalAlign, error) {
	return parseName[VerticalAlign]("vertical alignment", verticalAlignNames, name)
}

// Valid reports whether a is one of the alignments seen in the app.
func (a VerticalAlign) Valid() bool {
	return a >= 0 && int(a) < len(verticalAlignNames)
}

// String returns the name of a, as ParseVerticalAlign takes it.
func (a VerticalAlign) String() string {
	return nameOf("VerticalAlign", verticalAlignNames, a)
}

// alignNames are the names of Align's String and ParseAlign.
var alignNames = []string{AlignLeft: "left", AlignCenter: "center", AlignRight: "right"}

// verticalAlignNames are the names of VerticalAlign's String and
// ParseVerticalAlign.
var verticalAlignNames = []string{
	VerticalAlignTop:    "top",
	VerticalAlignMiddle: "middle",
	VerticalAlignBottom: "bottom",
}
