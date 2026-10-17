package tepra

import (
	"errors"
	"fmt"
	"time"
)

// Option sets something every kind of label has. NewText, NewPages and
// NewPour all take it: it is a TextOption and a PourOption. Each With
// function returns one, but WithCode. What no option sets is as the app
// sets it.
type Option func(*optionConfig)

// TextOption sets something of the labels NewText and NewPages make. An
// Option is one.
type TextOption interface {
	applyText(c *optionConfig)
}

// PourOption sets something of the labels NewPour makes. An Option is one,
// and so is a PourOnlyOption.
type PourOption interface {
	applyPour(c *optionConfig)
}

// PourOnlyOption sets what only the labels of NewPour have. It is not a
// TextOption, so NewText and NewPages do not compile with it.
type PourOnlyOption func(*optionConfig)

func (o Option) applyText(c *optionConfig)         { o(c) }
func (o Option) applyPour(c *optionConfig)         { o(c) }
func (o PourOnlyOption) applyPour(c *optionConfig) { o(c) }

// optionConfig is what the options of a label set, defaults filled in.
//
//declscope:shared // option_with.go sets it; layout.go, pour.go, text.go and file.go read it
type optionConfig struct {
	name           string
	at             time.Time
	fontSize       float64 // 0 fits the text to the tape
	bold           bool
	fill           FillMode
	align          Align
	verticalAlign  VerticalAlign
	rotation       Rotation
	vertical       bool
	code           CodeKind
	border         BorderType
	borderWidth    float64
	icon           []byte
	background     []byte
	imageMode      ImageMode
	imageThreshold int
	paddingMM      float64
	marginMM       float64
	lengthMM       float64 // 0 makes the label as long as what is on it
	// paddingSet is whether WithPaddingMM was given, so a border's default
	// padding does not replace it.
	paddingSet bool
	// errs are what the options found wrong in their own arguments.
	//
	//declscope:private
	errs []error
}

// newTextOptionConfig reads the options of NewText or NewPages.
//
//declscope:shared // pages.go and text.go read their options with it
func newTextOptionConfig(opts []TextOption) (*optionConfig, error) {
	return newOptionConfig(opts, TextOption.applyText)
}

// newPourOptionConfig reads the options of NewPour.
//
//declscope:shared // pour.go reads its options with it
func newPourOptionConfig(opts []PourOption) (*optionConfig, error) {
	return newOptionConfig(opts, PourOption.applyPour)
}

// newOptionConfig applies opts over the defaults, each with apply, and
// reports what the app would not take.
func newOptionConfig[O any](opts []O, apply func(O, *optionConfig)) (*optionConfig, error) {
	c := &optionConfig{
		name:           "label",
		verticalAlign:  VerticalAlignMiddle,
		borderWidth:    DefaultBorderWidth,
		imageMode:      ImageModeThreshold,
		imageThreshold: DefaultImageThreshold,
		marginMM:       DefaultMarginMM,
	}
	for _, opt := range opts {
		apply(opt, c)
	}
	if c.vertical && c.rotation.sideways() {
		c.optionError("vertical text cannot turn a quarter turn as well")
	}
	if !c.paddingSet && c.border != BorderNone {
		c.paddingMM = DefaultBorderPaddingMM
	}
	if c.at.IsZero() {
		c.at = time.Now()
	}
	if err := errors.Join(c.errs...); err != nil {
		return nil, err
	}
	return c, nil
}

// optionError records what is wrong with an option. format may wrap
// errors with %w, such as ErrCrashesApp.
//
//declscope:shared // option_with.go reports bad arguments with it
func (c *optionConfig) optionError(format string, args ...any) {
	c.errs = append(c.errs, fmt.Errorf("%w: %w", ErrInvalidOptions, fmt.Errorf(format, args...)))
}

// styleText sets the font, the fill, the alignment and the turn of d.
//
//declscope:shared // pour.go and text.go style each text box they make
func (c *optionConfig) styleText(d *TextDrawer) {
	if c.bold {
		d.FontName = FontGothicBold
	}
	d.FillMode = c.fill
	d.Alignment = c.align
	d.VerticalAlignment = c.verticalAlign
	d.Rotate = c.rotation
	d.TextVertical = c.vertical
}

// styleImage sets how d turns to black and white.
//
//declscope:shared // layout.go styles the icon and the background with it
func (c *optionConfig) styleImage(d *ImageDrawer) {
	d.ImageMode = c.imageMode
	d.ImageThreshold = c.imageThreshold
}

// styleDocument sets the length of doc.
//
//declscope:shared // pour.go and text.go fix the length of their label with it
func (c *optionConfig) styleDocument(doc *Document) {
	if c.lengthMM != 0 {
		doc.AutoLength = false
		doc.FixedLength = MMToPoints(c.lengthMM)
	}
}
