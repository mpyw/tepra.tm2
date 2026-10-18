package tm2

// BorderType is the line a drawer draws around its frame. Each type was
// seen in the app, opening a file this package wrote. Values from 6 up drew
// no line.
type BorderType int

// The border types.
const (
	BorderNone      BorderType = 0
	BorderBox       BorderType = 1
	BorderRounded   BorderType = 2
	BorderTopBottom BorderType = 3
	BorderTop       BorderType = 4
	BorderBottom    BorderType = 5
)

// DefaultBorderWidth is the width of a border as the app writes it. In the
// app, 0.35 showed thinner, and 1.4 and 2.8 wider.
const DefaultBorderWidth = 0.7

// DefaultBorderPaddingMM is the padding of a label with a border and no
// WithPaddingMM, in mm.
const DefaultBorderPaddingMM = 1.5

// ParseBorderType returns the border type of a name: none, box, rounded,
// top-bottom, top or bottom.
func ParseBorderType(name string) (BorderType, error) {
	return parseName[BorderType]("border", borderTypeNames, name)
}

// Valid reports whether b is one of the border types known to draw.
func (b BorderType) Valid() bool {
	return b >= 0 && int(b) < len(borderTypeNames)
}

// String returns the name of b, as ParseBorderType takes it.
func (b BorderType) String() string {
	return nameOf("BorderType", borderTypeNames, b)
}

// borderTypeNames are the names of String and ParseBorderType.
var borderTypeNames = []string{
	BorderNone:      "none",
	BorderBox:       "box",
	BorderRounded:   "rounded",
	BorderTopBottom: "top-bottom",
	BorderTop:       "top",
	BorderBottom:    "bottom",
}
