package tm2

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// CodeKind is what a PourDrawer shows its column as: text, or a code made of
// it. 1 showed a QR code in the app, and 2 and 3 barcodes. That 2 and 3 are
// CODE39 and CODE128, and 4 JAN-13, follows the order of the app's manual.
type CodeKind int

// The kinds of code.
const (
	CodeText    CodeKind = 0
	CodeQR      CodeKind = 1
	CodeCODE39  CodeKind = 2
	CodeCODE128 CodeKind = 3
	// CodeJAN13 takes 12 digits. The app adds the check digit.
	CodeJAN13 CodeKind = 4
)

// ParseCodeKind returns the kind of code of a name: text, qr, code39,
// code128 or jan13.
func ParseCodeKind(name string) (CodeKind, error) {
	return parseName[CodeKind]("code", codeKindNames, name)
}

// Valid reports whether k is one of the kinds of code known.
func (k CodeKind) Valid() bool {
	return k >= 0 && int(k) < len(codeKindNames)
}

// String returns the name of k, as ParseCodeKind takes it.
func (k CodeKind) String() string {
	return nameOf("CodeKind", codeKindNames, k)
}

// checkCode reports why value cannot be shown as a code of kind k, or nil.
//
//declscope:shared // pour.go checks each value it pours as a code
func (k CodeKind) checkCode(value string) error {
	var bad bool
	switch k {
	case CodeText:
		return nil
	case CodeQR:
		bad = value == "" || strings.Contains(value, "\n")
	case CodeCODE39:
		bad = value == "" || strings.Trim(value, code39Chars) != ""
	case CodeCODE128:
		bad = value == "" || strings.IndexFunc(value, func(r rune) bool { return r < ' ' || r > '~' }) >= 0
	case CodeJAN13:
		bad = len(value) != 12 || strings.Trim(value, "0123456789") != ""
	}
	if bad {
		return fmt.Errorf("%w: %q as %v; %s", ErrBadCode, codeShort(value), k, codeRules[k])
	}
	return nil
}

// code39Chars are the characters CODE39 encodes.
const code39Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ -.$/+%"

// codeRules say what each kind of code takes, for an error.
var codeRules = map[CodeKind]string{
	CodeQR:      "it takes one line",
	CodeCODE39:  "it takes digits, capital letters, space and -.$/+%",
	CodeCODE128: "it takes ASCII characters",
	CodeJAN13:   "it takes 12 digits; the app adds the check digit",
}

// codeKindNames are the names of String and ParseCodeKind.
var codeKindNames = []string{
	CodeText:    "text",
	CodeQR:      "qr",
	CodeCODE39:  "code39",
	CodeCODE128: "code128",
	CodeJAN13:   "jan13",
}

// codeShort is value for a message: its first 40 characters, and "…" when
// it has more.
func codeShort(value string) string {
	const most = 40
	if utf8.RuneCountInString(value) <= most {
		return value
	}
	return string([]rune(value)[:most]) + "…"
}
