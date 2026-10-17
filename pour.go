package tepra

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
	errs := []error{err}
	if len(values) > PourMaxRows {
		errs = append(errs, fmt.Errorf("%w: %d values to pour; the app takes up to %d", ErrOverLimit, len(values), PourMaxRows))
	}
	if !utf8.ValidString(column) {
		errs = append(errs, fmt.Errorf("column title: %w", ErrBadText))
	}
	values = slices.Clone(values)
	for i, v := range values {
		v = textLines(v)
		values[i] = v
		if err := textCheck(v); err != nil {
			errs = append(errs, fmt.Errorf("value %d: %w", i+1, err))
			continue
		}
		if n := utf8.RuneCountInString(v); n > PourMaxCellRunes {
			errs = append(errs, fmt.Errorf("%w: value %d has %d characters; the app takes up to %d", ErrOverLimit, i+1, n, PourMaxCellRunes))
		}
		if err := c.code.checkCode(v); err != nil {
			errs = append(errs, fmt.Errorf("value %d: %w", i+1, err))
		}
	}
	if err := errors.Join(errs...); err != nil {
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
		return layoutLineOf(v, col, len(l.texts))
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
