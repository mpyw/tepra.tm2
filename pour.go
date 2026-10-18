package tm2

import (
	"errors"
	"fmt"
	"slices"
	"unicode/utf8"
)

// The limits of the app's pour data.
const (
	PourMaxRows      = 1000
	PourMaxCellRunes = 1000
)

// NewPour returns a file that prints one label per value, with the column
// titled column. A value may hold line breaks ("\n"). All the labels share
// one frame, sized for the longest value.
//
// Use NewPour for codes ([WithCode]). For a list of text, [NewPages] with
// [WithLengthMM] lays the labels out better: the app shows poured text at the
// left of its frame whatever [WithAlign] says, and inverted fill only behind
// the letters.
//
// Inverted text of more than one line is poured from a column per line:
// column, then column 2, column 3 and on.
//
// The text keeps its font size on every label. The app would otherwise fit
// each value to the shared frame: a value of fewer lines would grow to its
// height, and run past its sides.
func NewPour(tape Tape, column string, values []string, opts ...PourOption) (*File, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("%w: no values to pour", ErrNoText)
	}
	// Every bad option and value is reported at once. Values count from 1,
	// as rows do in the app.
	c, err := newPourOptionConfig(opts)
	if len(values) > PourMaxRows {
		// The values are not checked one by one: the list is the error.
		return nil, errors.Join(err, fmt.Errorf("%w: %d values to pour; the app takes up to %d", ErrOverLimit, len(values), PourMaxRows))
	}
	errs := []error{err}
	if !utf8.ValidString(column) {
		errs = append(errs, fmt.Errorf("column title: %w", ErrBadText))
	}
	bad := errIndexed{noun: "value"}
	values = slices.Clone(values)
	for i, v := range values {
		v = textLines(v)
		values[i] = v
		if err := textCheck(v); err != nil {
			bad.add(i+1, err)
			continue
		}
		if n := utf8.RuneCountInString(v); n > PourMaxCellRunes {
			bad.add(i+1, fmt.Errorf("%w: %d characters; the app takes up to %d", ErrOverLimit, n, PourMaxCellRunes))
			continue
		}
		bad.add(i+1, c.code.checkCode(v))
	}
	if err := errors.Join(append(errs, bad.err())...); err != nil {
		return nil, err
	}

	l, err := layoutLabel(tape, values, c)
	if err != nil {
		return nil, err
	}
	if len(l.texts) > FileTableColumns {
		return nil, fmt.Errorf("%w: %d lines of inverted text; the app takes up to %d columns", ErrOverLimit, len(l.texts), FileTableColumns)
	}
	// cell is what column col of a row of v holds.
	cell := func(v string, col int) string {
		if len(l.texts) == 1 {
			return v
		}
		return layoutLineOf(v, col, len(l.texts), c.verticalAlign)
	}
	drawers := l.pictures
	for col, frame := range l.texts {
		d := NewPourDrawer(col, cell(values[0], col), frame)
		d.FontSize = l.size
		d.SizePriority = true
		d.Attribute = c.code
		c.styleText(&d.TextDrawer)
		drawers = append(drawers, d)
	}

	doc := newDocument(tape, l.length, c.marginMM, drawers...)
	doc.MultiplePage = true
	doc.PourMode = true
	c.styleDocument(&doc)

	f := newFile(c, doc)
	f.TableHeader[0] = column
	for col := 1; col < len(l.texts); col++ {
		f.TableHeader[col] = fmt.Sprintf("%s %d", column, col+1)
	}
	f.TableData = make([][]string, len(values))
	for i, v := range values {
		row := make([]string, FileTableColumns)
		for col := range l.texts {
			row[col] = cell(v, col)
		}
		f.TableData[i] = row
	}
	return f, nil
}
