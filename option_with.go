//declscope:namespace with

package tepra

import (
	"bytes"
	"time"
)

// WithName names the file in the app, without ".tm2". It is "label" by
// default.
func WithName(name string) Option {
	return func(c *optionConfig) { c.name = name }
}

// WithTime sets the file's date. It is the time of NewPour or NewText by
// default.
func WithTime(t time.Time) Option {
	return func(c *optionConfig) { c.at = t }
}

// WithFontSize sets the size of the text, in points. By default it is the
// size at which the text fills the tape.
func WithFontSize(pt float64) Option {
	return func(c *optionConfig) {
		if pt <= 0 {
			c.optionError("font size %v is not above 0", pt)
		}
		c.fontSize = pt
	}
}

// WithBold writes the text in FontGothicBold.
func WithBold() Option {
	return func(c *optionConfig) { c.bold = true }
}

// WithFill sets the box behind the text.
func WithFill(m FillMode) Option {
	return func(c *optionConfig) {
		if !m.Valid() {
			c.optionError("%v is not a fill mode: %w", m, ErrCrashesApp)
		}
		c.fill = m
	}
}

// WithAlign sets where each line of text goes across its frame. It is
// AlignLeft by default, as the app sets it.
func WithAlign(a Align) Option {
	return func(c *optionConfig) {
		if !a.Valid() {
			c.optionError("%v is not an alignment: %w", a, ErrCrashesApp)
		}
		c.align = a
	}
}

// WithVerticalAlign sets where the text goes down its frame. It is
// VerticalAlignMiddle by default, as the app sets it.
func WithVerticalAlign(a VerticalAlign) Option {
	return func(c *optionConfig) {
		if !a.Valid() {
			c.optionError("%v is not a vertical alignment", a)
		}
		c.verticalAlign = a
	}
}

// WithRotation turns the text. Rotation90 and Rotation270 run its lines
// across the tape.
func WithRotation(r Rotation) Option {
	return func(c *optionConfig) {
		if !r.Valid() {
			c.optionError("%v is not a rotation: %w", r, ErrCrashesApp)
		}
		c.rotation = r
	}
}

// WithVertical writes the text top to bottom, each line to the left of the
// one before. Its lines run across the tape, as with Rotation90.
func WithVertical() Option {
	return func(c *optionConfig) { c.vertical = true }
}

// WithCode shows each value of NewPour as a code instead of text. The app
// shows codes in pour data only, so NewText and NewPages do not take it.
func WithCode(k CodeKind) PourOnlyOption {
	return func(c *optionConfig) {
		if !k.Valid() {
			c.optionError("%v is not a kind of code", k)
		}
		c.code = k
	}
}

// WithBorder draws a line around the label, on a box of its own. The text
// keeps DefaultBorderPaddingMM from it, unless WithPaddingMM is given.
func WithBorder(b BorderType) Option {
	return func(c *optionConfig) {
		if !b.Valid() {
			c.optionError("%v is not a border type", b)
		}
		c.border = b
	}
}

// WithBorderWidth sets the width of the border. It is DefaultBorderWidth
// by default.
func WithBorderWidth(w float64) Option {
	return func(c *optionConfig) {
		if w <= 0 {
			c.optionError("border width %v is not above 0", w)
		}
		c.borderWidth = w
	}
}

// WithIcon puts a PNG, JPEG or GIF picture before the text, as high as the
// tape prints, keeping its shape.
func WithIcon(picture []byte) Option {
	picture = bytes.Clone(picture)
	return func(c *optionConfig) { c.icon = picture }
}

// WithBackground stretches a PNG, JPEG or GIF picture under the whole
// label.
func WithBackground(picture []byte) Option {
	picture = bytes.Clone(picture)
	return func(c *optionConfig) { c.background = picture }
}

// WithImageMode sets how the app turns the icon and the background to black
// and white. It is ImageModeThreshold by default.
func WithImageMode(m ImageMode) Option {
	return func(c *optionConfig) {
		if !m.Valid() {
			c.optionError("%v is not an image mode", m)
		}
		c.imageMode = m
	}
}

// WithImageThreshold sets the grey level, from 0 to 255, below which a pixel
// of the icon and the background is black, with ImageModeThreshold. It is
// DefaultImageThreshold by default.
func WithImageThreshold(level int) Option {
	return func(c *optionConfig) {
		if level < 0 || level > 255 {
			c.optionError("image threshold %d is not from 0 to 255", level)
		}
		c.imageThreshold = level
	}
}

// WithPaddingMM sets the blank between the label's edges and what is on it,
// in mm: room for a border, or for the frame of a background.
func WithPaddingMM(mm float64) Option {
	return func(c *optionConfig) {
		if mm < 0 {
			c.optionError("padding %v mm is negative", mm)
		}
		c.paddingMM, c.paddingSet = mm, true
	}
}

// WithMarginMM sets the blank tape before and after the label, in mm, from
// 1 to 500 as the app takes it. It is DefaultMarginMM by default.
func WithMarginMM(mm float64) Option {
	return func(c *optionConfig) {
		if mm < 1 || mm > 500 {
			c.optionError("margin %v mm; the app takes 1 to 500 mm", mm)
		}
		c.marginMM = mm
	}
}

// WithLengthMM makes the label this long, margins included, in mm, from 10
// to 3000 as the app takes it. By default the label is as long as what is
// on it. Text that does not fit gets smaller, unless WithFontSize is given,
// and what is on the label is centred.
func WithLengthMM(mm float64) Option {
	return func(c *optionConfig) {
		if mm < 10 || mm > 3000 {
			c.optionError("length %v mm; the app takes 10 to 3000 mm", mm)
		}
		c.lengthMM = mm
	}
}
