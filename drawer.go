package tepra

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
)

// The drawer types this package knows.
const (
	DrawerTypeText  = "text"
	DrawerTypePour  = "pour"
	DrawerTypeImage = "image"
)

// Drawer is an object on a label: a TextDrawer, PourDrawer, ImageDrawer, or a
// RawDrawer for a type this package does not know.
type Drawer interface {
	// DrawerType is the drawer's "type" in the file.
	DrawerType() string
	// Base returns the fields every drawer has.
	Base() *BaseDrawer
}

// BaseDrawer holds the fields every drawer has.
type BaseDrawer struct {
	// Identifier is a UUID, unique in the file.
	Identifier string `json:"-"`

	// Rotate turns the drawer in Frame.
	Rotate Rotation `json:"Rotate"`
	// TapeForegroundColor is the color the app shows the text in, Black as
	// it writes it. White showed white text on the white tape.
	TapeForegroundColor Color `json:"tapeForegroundColor"`
	// Frame is where the drawer is in the document's frame. A text box fits
	// its text to it unless SizePriority is set. A picture fits in it,
	// keeping its shape, centred.
	Frame Rect `json:"frame"`
	// BIType is the line drawn on the edges of Frame.
	BIType BorderType `json:"BIType"`
	// BILineWidthValue is the width of the BIType line, DefaultBorderWidth
	// as the app writes it. It is spelled "BILineWidhValue" in the file.
	BILineWidthValue float64 `json:"BILineWidhValue"`

	// Below: written as the app writes them. Values tried showed no change
	// in the app, or none were tried.
	// TapeBackgroundColor: Black did not fill the box behind the text.
	TapeBackgroundColor Color `json:"tapeBackgroundColor"`
	// BILineThickness, BILineStyle and BILineWidth: thickness 1 to 3, style
	// 0 to 4 and width 0.5 to 2, with box and rounded borders.
	BILineThickness int     `json:"BILineThickness"`
	BILineStyle     int     `json:"BILineStyle"`
	BILineWidth     float64 `json:"BILineWidth"`
	// TapeVertical, TapeTapeFrame and FrameTop: set, and a frame of
	// [0, 0, 40, 20].
	TapeVertical  bool `json:"tapeVertical"`
	TapeTapeFrame Rect `json:"tapeTapeFrame"`
	FrameTop      bool `json:"frameTop"`
}

// newBaseDrawer returns the fields the app writes for a new object in frame.
func newBaseDrawer(frame Rect) BaseDrawer {
	return BaseDrawer{
		Identifier:          newUUID(),
		TapeForegroundColor: Black,
		TapeBackgroundColor: White,
		Frame:               frame,
		BILineStyle:         1,
		BILineWidthValue:    DefaultBorderWidth,
	}
}

// Base returns b.
func (b *BaseDrawer) Base() *BaseDrawer {
	return b
}

// TextDrawer is a text box.
type TextDrawer struct {
	BaseDrawer

	// Text is what the box shows. "\n" breaks a line.
	Text string `json:"TIText"`
	// FontName is FontGothic or FontGothicBold. Other fonts were not tried.
	FontName string `json:"TIFontName"`
	// FontSize is the size of the text, in points. The app uses it only with
	// SizePriority. Otherwise it fits the text to Frame.
	FontSize float64 `json:"TIFontSize"`
	// Alignment is where each line goes across Frame.
	Alignment Align `json:"TIAlignment"`
	// VerticalAlignment is where the lines go down Frame.
	VerticalAlignment VerticalAlign `json:"TIVerticalAlignment"`
	// FillMode is the box behind the text.
	FillMode FillMode `json:"TIFillMode"`
	// FontFace is 1 as the app writes it. 0 showed no change. The app
	// crashes on 2, so other values are not written.
	FontFace int `json:"TIFontFace"`
	// TextVertical writes the lines top to bottom, each to the left of the
	// one before.
	TextVertical bool `json:"textVertical"`
	// HasPlaceholder shows "未入力のテキストボックス" in grey instead of
	// Text. The app sets it on a box with no text.
	HasPlaceholder bool `json:"hasPlaceholder"`
	// HoldAspect, set as the app sets it, fits the text to Frame, unless
	// SizePriority is set. Unset, the text keeps FontSize, its lines close
	// up, and the app marks the box 固.
	HoldAspect bool `json:"holdAspect"`
	// SizePriority keeps FontSize instead of fitting the text to Frame.
	SizePriority bool `json:"sizePriority"`

	// Below: written as the app writes them. Values tried showed no change
	// in the app, or none were tried.
	// Color: grey, red and white. TapeForegroundColor is the color shown.
	Color Color `json:"TIColor"`
	// Bold and Italic: set. FontGothicBold is how the app writes bold.
	Bold   bool `json:"TIBold"`
	Italic bool `json:"TIItalic"`
	// LineBreak: 0, 1 and 3, on text of two lines broken by "\n".
	LineBreak int `json:"TILineBreak"`
	// TextMargin: up to [3, 3], fitted to Frame and with SizePriority.
	TextMargin [2]float64 `json:"textMargin"`
}

// NewTextDrawer returns a text box in frame, set as the app sets a new one.
func NewTextDrawer(text string, frame Rect) *TextDrawer {
	return &TextDrawer{
		BaseDrawer:        newBaseDrawer(frame),
		Text:              text,
		FontName:          FontGothic,
		FontSize:          11.5,
		VerticalAlignment: VerticalAlignMiddle,
		LineBreak:         2,
		Color:             Black,
		FontFace:          1,
		TextMargin:        [2]float64{0.1, 0},
		HoldAspect:        true,
	}
}

// DrawerType returns DrawerTypeText.
func (*TextDrawer) DrawerType() string {
	return DrawerTypeText
}

// PourDrawer is a text box filled from a column of the file's table, one
// label per row.
type PourDrawer struct {
	TextDrawer

	// ColumnNumber is the column of the file's TableData the box shows, from
	// 0.
	ColumnNumber int `json:"ColumnNumber"`
	// Attribute is what the column shows as: text, or a code made of it.
	Attribute CodeKind `json:"Attribute"`
}

// NewPourDrawer returns a text box in frame for column, showing text in the
// editor.
func NewPourDrawer(column int, text string, frame Rect) *PourDrawer {
	return &PourDrawer{
		TextDrawer:   *NewTextDrawer(text, frame),
		ColumnNumber: column,
	}
}

// DrawerType returns DrawerTypePour.
func (*PourDrawer) DrawerType() string {
	return DrawerTypePour
}

// ImageDrawer is a picture.
type ImageDrawer struct {
	BaseDrawer

	// Image is a PNG, base64 in the file. The app fits it in Frame, keeping
	// its shape, centred.
	Image []byte `json:"image"`
	// ImageKind is 3 as the app writes it. 1 showed the picture in grey,
	// not turned to black and white. 0 and 2 showed no change.
	ImageKind int `json:"imageKind"`
	// ImageMode is how the picture turns to black and white.
	ImageMode ImageMode `json:"imageMode"`
	// ImageThreshold is the grey level, from 0 to 255, below which a pixel
	// is black, with ImageModeThreshold.
	ImageThreshold int `json:"imageThreshold"`

	// Below: written as the app writes them. Values tried showed no change
	// in the app, or none were tried.
	// ImageBlock and ImageOffset: 50, and 0.
	ImageBlock  int `json:"imageBlock"`
	ImageOffset int `json:"imageOffset"`
}

// NewImageDrawer returns a picture in frame, set as in a file the app wrote.
// The app fits png in frame, keeping its shape, centred. It turns it to
// black and white at ImageThreshold.
func NewImageDrawer(png []byte, frame Rect) *ImageDrawer {
	return &ImageDrawer{
		BaseDrawer:     newBaseDrawer(frame),
		Image:          png,
		ImageKind:      3,
		ImageMode:      ImageModeThreshold,
		ImageThreshold: DefaultImageThreshold,
		ImageBlock:     210,
		ImageOffset:    18,
	}
}

// DrawerType returns DrawerTypeImage.
func (*ImageDrawer) DrawerType() string {
	return DrawerTypeImage
}

// RawDrawer is a drawer of a type this package does not know. It is written
// back as it was read.
type RawDrawer struct {
	// BaseDrawer holds the fields every drawer has, as read. Changing them
	// does not change what is written.
	BaseDrawer

	// Type is the drawer's "type".
	Type string
	// JSON is the drawer as read.
	JSON jsontext.Value
}

// DrawerType returns r.Type.
func (r *RawDrawer) DrawerType() string {
	return r.Type
}

// Drawers are the objects of a Document, drawn in order.
type Drawers []Drawer

// MarshalJSONTo writes each drawer as the app does: "type", "identifier" and
// an empty key come first, then the drawer's fields.
func (ds Drawers) MarshalJSONTo(enc *jsontext.Encoder) error {
	if err := enc.WriteToken(jsontext.BeginArray); err != nil {
		return err
	}
	for i, d := range ds {
		if err := marshalDrawer(enc, d); err != nil {
			return fmt.Errorf("tepra: drawer %d: %w", i, err)
		}
	}
	return enc.WriteToken(jsontext.EndArray)
}

// UnmarshalJSONFrom reads drawers by their "type". An unknown type becomes a
// RawDrawer.
func (ds *Drawers) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	var raws []jsontext.Value
	if err := json.UnmarshalDecode(dec, &raws); err != nil {
		return err
	}
	out := make(Drawers, len(raws))
	for i, raw := range raws {
		d, err := unmarshalDrawer(raw)
		if err != nil {
			return fmt.Errorf("tepra: drawer %d: %w", i, err)
		}
		out[i] = d
	}
	*ds = out
	return nil
}

// MarshalJSON is MarshalJSONTo, for encoding/json version 1. encoding/json
// escapes <, > and & in what it returns, which the app does not.
func (ds Drawers) MarshalJSON() ([]byte, error) {
	// json.Marshal calls MarshalJSONTo, not MarshalJSON.
	return json.Marshal(ds)
}

// UnmarshalJSON is UnmarshalJSONFrom, for encoding/json version 1.
func (ds *Drawers) UnmarshalJSON(data []byte) error {
	// json.Unmarshal calls UnmarshalJSONFrom, not UnmarshalJSON.
	return json.Unmarshal(data, ds)
}

// marshalDrawer writes d to enc.
func marshalDrawer(enc *jsontext.Encoder, d Drawer) error {
	if r, ok := d.(*RawDrawer); ok {
		return enc.WriteValue(r.JSON)
	}
	if err := checkDrawer(d); err != nil {
		return err
	}
	fields, err := json.Marshal(d)
	if err != nil {
		return err
	}
	// The app writes "": "" after the identifier. Whether it needs the key
	// to read a file is not known, so it is kept.
	for _, t := range []jsontext.Token{
		jsontext.BeginObject,
		jsontext.String("type"), jsontext.String(d.DrawerType()),
		jsontext.String("identifier"), jsontext.String(d.Base().Identifier),
		jsontext.String(""), jsontext.String(""),
	} {
		if err := enc.WriteToken(t); err != nil {
			return err
		}
	}
	// Then each name and value of fields, in the order of the struct.
	dec := jsontext.NewDecoder(bytes.NewReader(fields))
	if _, err := dec.ReadToken(); err != nil {
		return err
	}
	for dec.PeekKind() != jsontext.KindEndObject {
		v, err := dec.ReadValue()
		if err != nil {
			return err
		}
		if err := enc.WriteValue(v); err != nil {
			return err
		}
	}
	return enc.WriteToken(jsontext.EndObject)
}

// unmarshalDrawer reads a drawer of the type raw names.
func unmarshalDrawer(raw jsontext.Value) (Drawer, error) {
	var head struct {
		Type       string `json:"type"`
		Identifier string `json:"identifier"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, err
	}
	// fields is what the drawer is read into. A RawDrawer keeps the JSON, so
	// only the fields every drawer has are read.
	var d Drawer
	var fields any
	switch head.Type {
	case DrawerTypeText:
		d = &TextDrawer{}
		fields = d
	case DrawerTypePour:
		d = &PourDrawer{}
		fields = d
	case DrawerTypeImage:
		d = &ImageDrawer{}
		fields = d
	default:
		r := &RawDrawer{Type: head.Type, JSON: raw}
		d, fields = r, &r.BaseDrawer
	}
	if err := json.Unmarshal(raw, fields); err != nil {
		return nil, fmt.Errorf("%s: %w", head.Type, err)
	}
	d.Base().Identifier = head.Identifier
	return d, nil
}

// checkDrawer reports a field of d that crashes the app.
func checkDrawer(d Drawer) error {
	if r := d.Base().Rotate; !r.Valid() {
		return fmt.Errorf("%v is not a rotation: %w", r, ErrCrashesApp)
	}
	var t *TextDrawer
	switch d := d.(type) {
	case *TextDrawer:
		t = d
	case *PourDrawer:
		t = &d.TextDrawer
	default:
		return nil
	}
	if !t.FillMode.Valid() {
		return fmt.Errorf("%v is not a fill mode: %w", t.FillMode, ErrCrashesApp)
	}
	if !t.Alignment.Valid() {
		return fmt.Errorf("%v is not an alignment: %w", t.Alignment, ErrCrashesApp)
	}
	if t.FontFace != 0 && t.FontFace != 1 {
		return fmt.Errorf("font face %d: %w", t.FontFace, ErrCrashesApp)
	}
	return nil
}
