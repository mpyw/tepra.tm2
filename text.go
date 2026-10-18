package tepra

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// NewText returns a file of one label that prints text. text may hold line
// breaks ("\n").
func NewText(tape Tape, text string, opts ...TextOption) (*File, error) {
	c, err := newTextOptionConfig(opts)
	text = textLines(text)
	if err := errors.Join(err, textCheck(text)); err != nil {
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
			line = layoutLineOf(text, i, len(l.texts), c.verticalAlign)
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

// textLines turns the line breaks of text into "\n": "\r\n" of Windows and
// a lone "\r" break lines too.
//
//declscope:shared // pages.go and pour.go read their texts with it
func textLines(text string) string {
	return strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
}

// textCheck reports what a file cannot hold as the text of a label: no
// text, text of blanks only, which prints nothing, or text that is not
// UTF-8.
//
//declscope:shared // pages.go and pour.go check their texts with it
func textCheck(text string) error {
	switch {
	case strings.TrimSpace(text) == "":
		return ErrNoText
	case !utf8.ValidString(text):
		return ErrBadText
	}
	return nil
}
