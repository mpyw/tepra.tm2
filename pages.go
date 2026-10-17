package tepra

import (
	"errors"
	"fmt"
	"slices"
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
	texts = slices.Clone(texts)
	for i, text := range texts {
		texts[i] = textLines(text)
		if err := textCheck(texts[i]); err != nil {
			errs = append(errs, fmt.Errorf("page %d: %w", i+1, err))
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	docs := make([]Document, len(texts))
	for i, text := range texts {
		doc, err := newTextDocument(tape, text, c)
		if err != nil {
			// The options make the same error on every page: one is enough.
			return nil, fmt.Errorf("page %d: %w", i+1, err)
		}
		// Without it the app opens the first page alone, as a label of one
		// page.
		doc.MultiplePage = true
		docs[i] = doc
	}
	return newFile(c, docs...), nil
}
