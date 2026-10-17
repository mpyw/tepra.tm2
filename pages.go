package tepra

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
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
	var failed pagesErrors
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
	if err := failed.join(); err != nil {
		return nil, err
	}
	return newFile(c, docs...), nil
}

// pagesErrors are the errors of laying pages out, by message. The options
// often make the same error on many pages, which is then reported once,
// with the pages it is on.
type pagesErrors struct {
	errs  []error
	pages [][]int
}

// add records err of page.
func (p *pagesErrors) add(page int, err error) {
	for i, e := range p.errs {
		if e.Error() == err.Error() {
			p.pages[i] = append(p.pages[i], page)
			return
		}
	}
	p.errs = append(p.errs, err)
	p.pages = append(p.pages, []int{page})
}

// join returns every error, each with its pages, or nil.
func (p *pagesErrors) join() error {
	out := make([]error, len(p.errs))
	for i, err := range p.errs {
		pages := make([]string, len(p.pages[i]))
		for j, n := range p.pages[i] {
			pages[j] = strconv.Itoa(n)
		}
		label := "page"
		if len(pages) > 1 {
			label = "pages"
		}
		out[i] = fmt.Errorf("%s %s: %w", label, strings.Join(pages, ", "), err)
	}
	return errors.Join(out...)
}
