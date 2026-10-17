package tepra

import (
	"fmt"
	"slices"
	"strings"
)

// parseName returns the value whose name is name, of names indexed by
// value. kind names what is parsed, for the error.
//
//declscope:shared // fill.go, border.go and align.go parse names with it
func parseName[T ~int](kind string, names []string, name string) (T, error) {
	i := slices.Index(names, name)
	if i < 0 {
		return 0, fmt.Errorf("%w: %s %q; known: %s", ErrUnknownName, kind, name, strings.Join(names, ", "))
	}
	return T(i), nil
}

// nameOf returns the name of v, of names indexed by value, or typ(v) for a
// value with no name.
//
//declscope:shared // fill.go, border.go and align.go name their values with it
func nameOf[T ~int](typ string, names []string, v T) string {
	if v < 0 || int(v) >= len(names) {
		return fmt.Sprintf("%s(%d)", typ, int(v))
	}
	return names[v]
}
