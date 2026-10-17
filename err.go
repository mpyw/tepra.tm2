package tepra

import "errors"

// The errors of this package. Each error it returns wraps one or more of
// these, to test with errors.Is. Errors of the file system, and of a reader
// or a writer, are returned as they are, to test with errors.Is too.
var (
	// ErrTapeNotSupported is a tape width whose sizes are not known.
	ErrTapeNotSupported = errors.New("tepra: tape width not supported")
	// ErrUnknownName is a name ParseFillMode, ParseBorderType, ParseAlign or
	// ParseVerticalAlign does not know.
	ErrUnknownName = errors.New("tepra: unknown name")
	// ErrInvalidOptions is an Option, or options together, that NewPour or
	// NewText cannot use.
	ErrInvalidOptions = errors.New("tepra: invalid options")
	// ErrCrashesApp is a value TEPRA LINK 2 crashes on. It is not written.
	ErrCrashesApp = errors.New("tepra: the app crashes on it")
	// ErrBadFile is a file that is not a .tm2 file this package can read, or
	// one it cannot encode. It wraps the JSON error.
	ErrBadFile = errors.New("tepra: bad .tm2 file")
	// ErrBadText is text that is not UTF-8, which a file cannot hold.
	ErrBadText = errors.New("tepra: text is not UTF-8")
	// ErrNoText is a label with no text: an empty text, or no values to
	// pour.
	ErrNoText = errors.New("tepra: no text")
	// ErrOverLimit is more than the app takes: rows, characters in a cell,
	// or columns.
	ErrOverLimit = errors.New("tepra: over the app's limit")
	// ErrNoRoom is padding that leaves no room on the tape.
	ErrNoRoom = errors.New("tepra: no room on the tape")
	// ErrBadImage is a picture that cannot be read as PNG, JPEG or GIF.
	ErrBadImage = errors.New("tepra: bad picture")
	// ErrBadCode is a value that cannot be shown as the kind of code asked
	// for.
	ErrBadCode = errors.New("tepra: bad value for a code")
)
