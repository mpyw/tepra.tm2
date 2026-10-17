package tepra

import "errors"

// NewText returns a file of one label that prints text. text may hold line
// breaks ("\n").
func NewText(tape Tape, text string, opts ...TextOption) (*File, error) {
	c, err := newTextOptionConfig(opts)
	if text == "" {
		err = errors.Join(err, ErrNoText)
	}
	if err != nil {
		return nil, err
	}
	doc, err := newTextDocument(tape, text, c)
	if err != nil {
		return nil, err
	}
	return newFile(c, doc), nil
}

// newTextDocument lays out a label of text, its font size fitted to it.
// text is not empty.
//
//declscope:shared // pages.go makes each page with it
func newTextDocument(tape Tape, text string, c *optionConfig) (Document, error) {
	l, err := layoutLabel(tape, []string{text}, c)
	if err != nil {
		return Document{}, err
	}
	drawers := l.pictures
	for i, frame := range l.texts {
		line := text
		if len(l.texts) > 1 {
			line = layoutLineOf(text, i, len(l.texts))
		}
		d := NewTextDrawer(line, frame)
		d.FontSize = l.size
		c.styleText(d)
		drawers = append(drawers, d)
	}
	doc := newDocument(tape, l.length, c.marginMM, drawers...)
	c.styleDocument(&doc)
	return doc, nil
}
