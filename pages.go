package tm2

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
	bad := errIndexed{noun: "page"}
	texts = slices.Clone(texts)
	for i, text := range texts {
		texts[i] = textLines(text)
		bad.add(i+1, textCheck(texts[i]))
	}
	if err := errors.Join(err, bad.err()); err != nil {
		return nil, err
	}
	docs := make([]Document, len(texts))
	failed := errIndexed{noun: "page"}
	for i, text := range texts {
		doc, err := newTextDocument(tape, text, c)
		if err != nil {
			failed.add(i+1, err)
			continue
		}
		// Without it the app opens the first page alone, as a label of one
		// page.
		doc.MultiplePage = true
		docs[i] = doc
	}
	if err := failed.err(); err != nil {
		return nil, err
	}
	return newFile(c, docs...), nil
}
