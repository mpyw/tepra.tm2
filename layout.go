package tepra

import (
	"fmt"
	"math"
	"strings"
)

// layoutIconGapMM is the blank between an Icon and the text, in mm.
const layoutIconGapMM = 1.5

// layoutBarcodeLength is the length of the frame of a barcode, per point of
// its height. The app fits the barcode to the frame.
const layoutBarcodeLength = 2.5

// layoutMinLengthMM is the shortest label the app makes, margins included.
// It shows a shorter one this long, what is on it at the start.
const layoutMinLengthMM = 12

// layoutMaxLengthMM is the longest label the app makes, margins included.
const layoutMaxLengthMM = 3000

// layoutCodeFontSize is the font size written for a code. The app draws the
// code to its frame.
const layoutCodeFontSize = 11.5

// labelLayout is where layoutLabel puts what is on a label.
//
//declscope:shared // pour.go and text.go build their labels from it
type labelLayout struct {
	// texts are the frames of the text: one, or one per line when the lines
	// are drawn apart. A line drawn apart is the line of that index.
	texts []Rect
	// size is the font size of the text.
	size float64
	// pictures are the background, the border's box and the icon, to draw
	// before the text.
	pictures []Drawer
	// length is the length of the label's frame.
	length float64
}

// layoutLabel lays the label of texts and c out on tape. Along the tape
// come the icon, then the text. The background and the border lie around
// both, and the padding keeps them off the edges. texts are drawn in turn
// in the same frames, as PourDrawers are.
//
// Inverted text of more than one line gets a frame per line, stacked with no
// gap: the app shows only the last line of an inverted box. A black picture
// lies under them, as the app shows nothing for a box with no text, not even
// black.
//
// A label is at least layoutMinLengthMM long, what is on it centred.
//
// With a length, the label is that long. Text that does not fit at its size
// gets smaller, if its size is fitted, a code's frame gets shorter, and what
// is on the label is centred.
//
//declscope:shared // pour.go and text.go build their labels with it
func layoutLabel(tape Tape, texts []string, c *optionConfig) (labelLayout, error) {
	padMM := c.paddingMM
	pad := MMToPoints(padMM)
	top := tape.PrintableTop() + pad
	height := tape.PrintableHeight - 2*pad
	if pad < 0 || height <= 0 {
		return labelLayout{}, fmt.Errorf("%w: padding %v mm on %d mm tape", ErrNoRoom, padMM, tape.WidthMM)
	}

	var l labelLayout
	var icon *ImageDrawer
	iconWidth := 0.0
	if c.icon != nil {
		png, w, h, err := decodeImage(c.icon)
		if err != nil {
			return labelLayout{}, fmt.Errorf("%w: icon: %w", ErrBadImage, err)
		}
		iconWidth = height * float64(w) / float64(h)
		icon = NewImageDrawer(png, Rect{pad, top, iconWidth, height})
		c.styleImage(icon)
		iconWidth += MMToPoints(layoutIconGapMM)
	}

	sideways := c.vertical || c.rotation.sideways()
	var size, width float64
	switch c.code {
	case CodeText:
		size, width = layoutText(texts, height, c.fontSize, sideways)
		if size <= 0 {
			// Too many lines, or with the lines across the tape, a line too
			// long for its width. The padding may be what takes the room.
			cause := "too many lines"
			if sideways {
				cause = "a line too long to run across it"
			}
			if pad > 0 {
				cause += fmt.Sprintf(", with padding of %v mm", padMM)
			}
			return labelLayout{}, fmt.Errorf("%w: %s for %d mm tape", ErrNoRoom, cause, tape.WidthMM)
		}
	case CodeQR:
		size, width = layoutCodeFontSize, height
	default:
		size, width = layoutCodeFontSize, height*layoutBarcodeLength
	}

	length := pad + iconWidth + width + pad
	if c.lengthMM != 0 {
		inner := MMToPoints(c.lengthMM) - 2*MMToPoints(c.marginMM)
		if room := inner - (pad + iconWidth + pad); width > room {
			// A code fits its frame in the app. Text fits its size here,
			// unless the size was given.
			if (c.fontSize != 0 && c.code == CodeText) || room <= 0 {
				return labelLayout{}, fmt.Errorf("%w: a label of %v mm is too short for what is on it", ErrNoRoom, c.lengthMM)
			}
			if c.code == CodeText {
				// The app sets sizes in steps of 0.5.
				size = math.Floor(size*room/width*2) / 2
				if size <= 0 {
					return labelLayout{}, fmt.Errorf("%w: a label of %v mm is too short for its text", ErrNoRoom, c.lengthMM)
				}
			}
			width = room
		}
		length = inner
	} else if inner := MMToPoints(layoutMinLengthMM) - 2*MMToPoints(c.marginMM); length < inner {
		length = inner
	}
	// Centre what is on the label, when the label is longer.
	if shift := (length - (pad + iconWidth + width + pad)) / 2; shift > 0 {
		if icon != nil {
			icon.Frame[0] += shift
		}
		pad += shift
	}
	// A fixed length is checked by WithLengthMM. One of what is on the
	// label is checked here, before a background is made that long.
	if mm := PointsToMM(length) + 2*c.marginMM; c.lengthMM == 0 && mm > layoutMaxLengthMM {
		return labelLayout{}, fmt.Errorf("%w: a label of %.0f mm; the app takes up to %d mm", ErrOverLimit, mm, layoutMaxLengthMM)
	}
	x := pad + iconWidth
	l.size, l.length = size, length

	textBand := Rect{x, top, width, height}
	split := false
	if lines := layoutLines(texts); c.fill == FillInverted && lines > 1 {
		if sideways {
			return labelLayout{}, fmt.Errorf("%w: inverted text of more than one line cannot run across the tape", ErrInvalidOptions)
		}
		split = true
		h := height / float64(lines)
		for i := range lines {
			l.texts = append(l.texts, Rect{x, top + float64(i)*h, width, h})
		}
	} else {
		l.texts = []Rect{textBand}
	}

	band := Rect{0, tape.PrintableTop(), l.length, tape.PrintableHeight}
	if c.background != nil {
		png, err := stretchImage(c.background, band[2], band[3])
		if err != nil {
			return labelLayout{}, fmt.Errorf("%w: background: %w", ErrBadImage, err)
		}
		bg := NewImageDrawer(png, band)
		c.styleImage(bg)
		l.pictures = append(l.pictures, bg)
	}
	if c.border != BorderNone {
		// The box holds only the line, so the text in it is fitted to its own
		// frame, inside the padding. A border on the text's own frame would
		// touch the text, since the app fits the text to the frame.
		box := NewTextDrawer("", band)
		box.BIType = c.border
		box.BILineWidthValue = c.borderWidth
		l.pictures = append(l.pictures, box)
	}
	if icon != nil {
		l.pictures = append(l.pictures, icon)
	}
	if split {
		l.pictures = append(l.pictures, NewImageDrawer(blackImage(textBand[2], textBand[3]), textBand))
	}
	return l, nil
}

// layoutLineOf returns what frame i of n frames of lines drawn apart shows
// of text. A text of fewer lines sits in the middle frames, or above the
// middle when it cannot sit exactly there. The frames around it are "".
//
//declscope:shared // pour.go and text.go fill the frames of lines drawn apart
func layoutLineOf(text string, i, n int) string {
	lines := strings.Split(text, "\n")
	j := i - (n-len(lines))/2
	if j < 0 || j >= len(lines) {
		return ""
	}
	return lines[j]
}

// layoutLines is the most lines of any text in texts.
func layoutLines(texts []string) int {
	lines := 1
	for _, t := range texts {
		lines = max(lines, strings.Count(t, "\n")+1)
	}
	return lines
}

// layoutLineHeight is the height of a line of text, per point of font size.
// With it, two lines of 17.5 fill the 43.2 points of an 18 mm tape, as in a
// label made in the app.
const layoutLineHeight = 43.2 / 2 / 17.5

// layoutText sizes one frame for every text in texts: fontSize, or the size
// at which the text of the most lines fills height when it is 0, and the
// width of the widest line at that size. Sideways, the lines run across the
// tape, side by side along it: the size is that at which the longest line
// fills height, and the width that of the lines side by side.
func layoutText(texts []string, height, fontSize float64, sideways bool) (size, width float64) {
	lines, ems := 1, 0.0
	for _, t := range texts {
		ls := strings.Split(t, "\n")
		lines = max(lines, len(ls))
		for _, l := range ls {
			ems = max(ems, layoutEms(l))
		}
	}
	ems = max(ems, 1)
	size = fontSize
	if sideways {
		if size == 0 {
			// The app sets sizes in steps of 0.5.
			size = math.Floor(height/ems*2) / 2
		}
		return size, math.Ceil(float64(lines) * layoutLineHeight * size)
	}
	if size == 0 {
		size = math.Floor(height/(float64(lines)*layoutLineHeight)*2) / 2
	}
	return size, math.Ceil(ems * size)
}

// layoutEms is the width of line in ems: 1 for a wide character, such as
// kana and kanji, and 0.5 for a narrow one, such as ASCII and half-width
// katakana. It is an estimate; the app measures the font.
func layoutEms(line string) float64 {
	var ems float64
	for _, r := range line {
		switch {
		case r < 0x1100, r >= 0xff61 && r <= 0xffdc:
			ems += 0.5
		default:
			ems++
		}
	}
	return ems
}
