package tm2

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// The errors of this package. Each error it returns wraps one or more of
// these, to test with errors.Is. Errors of the file system, and of a reader
// or a writer, are returned as they are, to test with errors.Is too.
var (
	// ErrTapeNotSupported is a tape width whose sizes are not known.
	ErrTapeNotSupported = errors.New("tm2: tape width not supported")
	// ErrUnknownName is a name ParseFillMode, ParseBorderType, ParseAlign,
	// ParseVerticalAlign, ParseRotation, ParseCodeKind or ParseImageMode
	// does not know.
	ErrUnknownName = errors.New("tm2: unknown name")
	// ErrInvalidOptions is an Option, or options together, that NewText,
	// NewPages or NewPour cannot use.
	ErrInvalidOptions = errors.New("tm2: invalid options")
	// ErrCrashesApp is a value TEPRA LINK 2 crashes on. It is not written.
	ErrCrashesApp = errors.New("tm2: the app crashes on it")
	// ErrBadFile is a file that is not a .tm2 file this package can read, or
	// one it cannot encode. It wraps the JSON error.
	ErrBadFile = errors.New("tm2: bad .tm2 file")
	// ErrBadText is text that is not UTF-8, which a file cannot hold.
	ErrBadText = errors.New("tm2: text is not UTF-8")
	// ErrNoText is a label with no text: an empty text, or no values to
	// pour.
	ErrNoText = errors.New("tm2: no text")
	// ErrOverLimit is more than the app takes: rows, characters in a cell,
	// columns, or a label over 3000 mm.
	ErrOverLimit = errors.New("tm2: over the app's limit")
	// ErrNoRoom is no room on the tape for what is on a label: padding that
	// takes it, too many lines, a line too long to run across the tape, a
	// font size too large, or a fixed length too short.
	ErrNoRoom = errors.New("tm2: no room on the tape")
	// ErrBadImage is a picture that cannot be read as PNG, JPEG or GIF.
	ErrBadImage = errors.New("tm2: bad picture")
	// ErrBadCode is a value that cannot be shown as the kind of code asked
	// for.
	ErrBadCode = errors.New("tm2: bad value for a code")
	// ErrBadPatch is an operation File.Patch cannot apply, or a file the
	// operations make that is no .tm2 file.
	ErrBadPatch = errors.New("tm2: bad patch")
)

// errIndexed collects the errors of the items of a list, numbered from 1,
// by message: the options often make one error on many items, reported
// once with the items it is on. The message is bounded however long the
// list is.
//
//declscope:shared // pages.go and pour.go check their lists with it
type errIndexed struct {
	noun string // what an item is called, such as "page"
	// errs are the errors, each once, and items the items of each.
	//
	//declscope:private
	errs []error
	//declscope:private
	items [][]int
}

// The most errIndexed lists: of the items of one error, and of errors.
const (
	errIndexedItems = 5
	errIndexedErrs  = 10
)

// add records err, if any, of item.
//
//declscope:shared // pages.go and pour.go record each item's error
func (e *errIndexed) add(item int, err error) {
	if err == nil {
		return
	}
	for i, prev := range e.errs {
		if prev.Error() == err.Error() {
			e.items[i] = append(e.items[i], item)
			return
		}
	}
	e.errs = append(e.errs, err)
	e.items = append(e.items, []int{item})
}

// err returns every error, each with its items, or nil.
//
//declscope:shared // pages.go and pour.go return what they recorded
func (e *errIndexed) err() error {
	var out []error
	for i, err := range e.errs {
		if i == errIndexedErrs {
			out = append(out, errMore(e.errs[i:]))
			break
		}
		items := e.items[i]
		names := make([]string, min(len(items), errIndexedItems))
		for j := range names {
			names[j] = strconv.Itoa(items[j])
		}
		list := strings.Join(names, ", ")
		if more := len(items) - len(names); more > 0 {
			list += fmt.Sprintf(" and %d more", more)
		}
		noun := e.noun
		if len(items) > 1 {
			noun += "s"
		}
		out = append(out, fmt.Errorf("%s %s: %w", noun, list, err))
	}
	return errors.Join(out...)
}

// errMore stands for errors not listed, by their count. It wraps them, so
// errors.Is still finds their sentinels.
type errMore []error

// Error says how many errors are not listed.
func (e errMore) Error() string { return fmt.Sprintf("and %d more errors", len(e)) }

// Unwrap returns the errors not listed.
func (e errMore) Unwrap() []error { return e }
