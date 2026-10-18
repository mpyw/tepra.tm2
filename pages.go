package tepra

import (
	"errors"
	"fmt"
)

// NewPages returns a file of one page per text, each laid out as NewText
// lays out its label. So each label has a font size and a length of its
// own, unlike the labels of NewPour. The app lists the pages, and prints
// them together. A text may hold line breaks ("\n").
func NewPages(tape Tape, texts []string, opts ...TextOption) (*File, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("%w: no pages", ErrNoText)
	}
	c, err := newTextOptionConfig(opts)
	errs := []error{err}
	for i, text := range texts {
		if text == "" {
			errs = append(errs, fmt.Errorf("page %d: %w", i+1, ErrNoText))
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	docs := make([]Document, len(texts))
	var layoutErrs []error
	for i, text := range texts {
		doc, err := newTextDocument(tape, text, c)
		if err != nil {
			layoutErrs = append(layoutErrs, fmt.Errorf("page %d: %w", i+1, err))
			continue
		}
		// Without it the app opens the first page alone, as a label of one
		// page.
		doc.MultiplePage = true
		docs[i] = doc
	}
	if err := errors.Join(layoutErrs...); err != nil {
		return nil, err
	}
	return newFile(c, docs...), nil
}
