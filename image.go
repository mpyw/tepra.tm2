package tepra

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/gif"  // decodeImage reads GIF
	_ "image/jpeg" // decodeImage reads JPEG
	"image/png"
	"math"
	"slices"
)

// ImageMode is how the app turns a picture to black and white. Each mode was
// seen in the app, opening a file this package wrote. 0 and 4 showed as
// ImageModeThreshold does.
type ImageMode int

// The modes the app showed differently.
const (
	// ImageModeDiffusion spreads the error of each pixel to the next ones:
	// grey shows as scattered dots.
	ImageModeDiffusion ImageMode = 1
	// ImageModeThreshold makes each pixel black below ImageThreshold. It is
	// the mode of a picture of a template of the app.
	ImageModeThreshold ImageMode = 2
	// ImageModeScreen shows grey as a regular screen of dots.
	ImageModeScreen ImageMode = 3
)

// DefaultImageThreshold is the threshold of a picture of a template of the
// app.
const DefaultImageThreshold = 128

// ParseImageMode returns the mode of a name: diffusion, threshold or
// screen.
func ParseImageMode(name string) (ImageMode, error) {
	i := slices.Index(imageModeNames, name)
	if i < 0 || name == "" {
		return 0, fmt.Errorf("%w: image mode %q; known: diffusion, threshold, screen", ErrUnknownName, name)
	}
	return ImageMode(i), nil
}

// Valid reports whether m is one of the modes known.
func (m ImageMode) Valid() bool {
	return m >= ImageModeDiffusion && m <= ImageModeScreen
}

// String returns the name of m, as ParseImageMode takes it.
func (m ImageMode) String() string {
	if !m.Valid() {
		return fmt.Sprintf("ImageMode(%d)", int(m))
	}
	return imageModeNames[m]
}

// imageModeNames are the names of String and ParseImageMode, by value.
var imageModeNames = []string{
	ImageModeDiffusion: "diffusion",
	ImageModeThreshold: "threshold",
	ImageModeScreen:    "screen",
}

// imageShapeHeight is the height in pixels of a picture made only for its
// shape.
const imageShapeHeight = 100

// imageDotsPerPoint is the printer's dots per point: 360 dots per inch. A
// stretched picture has no more pixels than the printer has dots for its
// frame, however large it was.
const imageDotsPerPoint = 360.0 / 72

// blackImage returns a black PNG of the shape of a w by h frame. The app
// keeps the shape of a picture, so one of another shape would not cover the
// frame.
//
//declscope:shared // layout.go lays it under inverted lines drawn apart
func blackImage(w, h float64) []byte {
	img := image.NewGray(image.Rect(0, 0, imageWidthFor(imageShapeHeight, w, h), imageShapeHeight)) // its pixels are 0, black
	var buf bytes.Buffer
	_ = png.Encode(&buf, img) // encoding to memory does not fail
	return buf.Bytes()
}

// stretchImage reads a PNG, JPEG or GIF picture, and returns it as a PNG
// stretched to the shape of a w by h frame. The app keeps the shape of a
// picture, centred in its frame, so a picture to fill a frame must have the
// frame's shape. It keeps the picture's height, or makes it
// imageShapeHeight when less, so the shape comes out close. It samples the
// nearest pixel, so line art stays sharp.
//
//declscope:shared // layout.go stretches a Background to the label
func stretchImage(data []byte, w, h float64) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	if b.Empty() {
		return nil, errors.New("picture has no pixels")
	}
	dh := min(max(b.Dy(), imageShapeHeight), int(math.Ceil(h*imageDotsPerPoint)))
	dst := image.NewNRGBA(image.Rect(0, 0, imageWidthFor(dh, w, h), dh))
	dw := dst.Bounds().Dx()
	// Each row and column of src is found once, not per pixel.
	cols := make([]int, dw)
	for x := range cols {
		cols[x] = b.Min.X + x*b.Dx()/dw
	}
	for y := range dh {
		sy := b.Min.Y + y*b.Dy()/dh
		for x, sx := range cols {
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// decodeImage reads a PNG, JPEG or GIF picture, and returns it as a PNG with
// its size in pixels. The app stores pictures as PNG.
//
//declscope:shared // layout.go puts the icon and the background on a label
func decodeImage(data []byte) (pngData []byte, width, height int, err error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, err
	}
	b := img.Bounds()
	if b.Empty() {
		return nil, 0, 0, fmt.Errorf("%s picture has no pixels", format)
	}
	if format == "png" {
		return data, b.Dx(), b.Dy(), nil
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, 0, 0, err
	}
	return buf.Bytes(), b.Dx(), b.Dy(), nil
}

// imageWidthFor is the width in pixels of a picture height pixels high, of
// the shape of a w by h frame.
func imageWidthFor(height int, w, h float64) int {
	return max(1, int(math.Round(float64(height)*w/h)))
}
