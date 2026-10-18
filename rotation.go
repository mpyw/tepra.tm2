package tm2

// Rotation turns a drawer in its frame, by a quarter turn or a half turn.
// Each value was seen in TEPRA LINK 2 for Android, opening a file this
// package wrote. The app crashes on a file with 90, and so on any value it
// does not know.
type Rotation int

// The rotations, clockwise.
const (
	RotationNone Rotation = 0
	Rotation180  Rotation = 1
	// Rotation270 turns the text a quarter turn left: it reads bottom to
	// top.
	Rotation270 Rotation = 2
	// Rotation90 turns the text a quarter turn right: it reads top to
	// bottom.
	Rotation90 Rotation = 3
)

// ParseRotation returns the rotation of a name, in degrees clockwise: 0, 90,
// 180 or 270.
func ParseRotation(name string) (Rotation, error) {
	return parseName("rotation", rotationNames, name, RotationNone, Rotation90, Rotation180, Rotation270)
}

// Valid reports whether r is one of the rotations the app opens.
func (r Rotation) Valid() bool {
	return r >= 0 && int(r) < len(rotationNames)
}

// String returns the name of r, as ParseRotation takes it.
func (r Rotation) String() string {
	return nameOf("Rotation", rotationNames, r)
}

// sideways reports whether r turns lines to run across the tape.
//
//declscope:shared // layout.go lays such lines out across the tape
func (r Rotation) sideways() bool {
	return r == Rotation90 || r == Rotation270
}

// rotationNames are the names of String and ParseRotation.
var rotationNames = []string{
	RotationNone: "0",
	Rotation180:  "180",
	Rotation270:  "270",
	Rotation90:   "90",
}
