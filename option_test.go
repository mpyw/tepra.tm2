package tepra_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mpyw/tepra.tm2"
)

func TestOption_Border(t *testing.T) {
	f, err := tepra.NewText(tepra.Tape18mm, "会議室", tepra.WithBorder(tepra.BorderRounded), tepra.WithBorderWidth(1.4), tepra.WithFontSize(10))
	if err != nil {
		t.Fatal(err)
	}
	doc := f.Documents[0]
	if len(doc.Drawers) != 2 {
		t.Fatalf("got %d drawers, want the border's box and the text", len(doc.Drawers))
	}
	box := fixtureAs[*tepra.TextDrawer](t, doc.Drawers[0])
	text := fixtureAs[*tepra.TextDrawer](t, doc.Drawers[1])
	if box.BIType != tepra.BorderRounded || box.BILineWidthValue != 1.4 || box.Text != "" {
		t.Errorf("box: border %v, width %v, text %q", box.BIType, box.BILineWidthValue, box.Text)
	}
	if text.BIType != tepra.BorderNone {
		t.Errorf("text has border %v", text.BIType)
	}
	// The text, 3 ems of 10 points, sits DefaultBorderPaddingMM inside the
	// box, which covers the label.
	pad := tepra.MMToPoints(tepra.DefaultBorderPaddingMM)
	if want := (tepra.Rect{pad, 4.4 + pad, 30, 43.2 - 2*pad}); !fixtureFrameNear(text.Frame, want) {
		t.Errorf("text frame %v, want %v", text.Frame, want)
	}
	if want := (tepra.Rect{0, 4.4, 30 + 2*pad, 43.2}); !fixtureFrameNear(box.Frame, want) {
		t.Errorf("box frame %v, want %v", box.Frame, want)
	}

	f, err = tepra.NewPour(tepra.Tape18mm, "c", []string{"x"}, tepra.WithBorder(tepra.BorderTop), tepra.WithPaddingMM(3))
	if err != nil {
		t.Fatal(err)
	}
	box = fixtureAs[*tepra.TextDrawer](t, f.Documents[0].Drawers[0])
	pour := fixtureAs[*tepra.PourDrawer](t, f.Documents[0].Drawers[1])
	if box.BIType != tepra.BorderTop || box.BILineWidthValue != tepra.DefaultBorderWidth {
		t.Errorf("box: border %v, width %v", box.BIType, box.BILineWidthValue)
	}
	if got, want := pour.Frame[0], tepra.MMToPoints(3); got != want {
		t.Errorf("pour drawer at x %v, want the padding given, %v", got, want)
	}
	for name, opt := range map[string]tepra.Option{"border 6": tepra.WithBorder(6), "negative width": tepra.WithBorderWidth(-1)} {
		if _, err := tepra.NewText(tepra.Tape18mm, "x", opt); !errors.Is(err, tepra.ErrInvalidOptions) {
			t.Errorf("%s: %v, want ErrInvalidOptions", name, err)
		}
	}
}

func TestOption_invertedLinesApart(t *testing.T) {
	f, err := tepra.NewText(tepra.Tape18mm, "会議室A\n3F", tepra.WithFill(tepra.FillInverted), tepra.WithFontSize(14))
	if err != nil {
		t.Fatal(err)
	}
	ds := f.Documents[0].Drawers
	if len(ds) != 3 {
		t.Fatalf("got %d drawers, want a black picture and a box per line", len(ds))
	}
	black := fixtureAs[*tepra.ImageDrawer](t, ds[0])
	first := fixtureAs[*tepra.TextDrawer](t, ds[1])
	second := fixtureAs[*tepra.TextDrawer](t, ds[2])
	// The picture covers both boxes: the app prints a box with no text white.
	if want := (tepra.Rect{first.Frame[0], first.Frame[1], first.Frame[2], first.Frame[3] * 2}); black.Frame != want {
		t.Errorf("black picture at %v, want %v", black.Frame, want)
	}
	if !fixtureShapeNear(t, black.Image, black.Frame) {
		t.Error("black picture is not of its frame's shape")
	}
	if first.Text != "会議室A" || second.Text != "3F" || first.FillMode != tepra.FillInverted || second.FillMode != tepra.FillInverted {
		t.Errorf("lines %q (%v) and %q (%v)", first.Text, first.FillMode, second.Text, second.FillMode)
	}
	// Stacked with no gap, so the black boxes join.
	if first.Frame[1]+first.Frame[3] != second.Frame[1] || first.Frame[3] != 43.2/2 {
		t.Errorf("frames %v and %v", first.Frame, second.Frame)
	}

	f, err = tepra.NewPour(tepra.Tape18mm, "品名", []string{"割り箸\nスプーン\nフォーク", "カイロ"}, tepra.WithFill(tepra.FillInverted))
	if err != nil {
		t.Fatal(err)
	}
	ds = f.Documents[0].Drawers
	if len(ds) != 4 {
		t.Fatalf("got %d drawers, want a black picture and one per line of the longest value", len(ds))
	}
	fixtureAs[*tepra.ImageDrawer](t, ds[0])
	for c, d := range ds[1:] {
		if p := fixtureAs[*tepra.PourDrawer](t, d); p.ColumnNumber != c {
			t.Errorf("drawer %d pours column %d", c, p.ColumnNumber)
		}
	}
	if got := f.TableHeader[:4]; got[0] != "品名" || got[1] != "品名 2" || got[2] != "品名 3" || got[3] != "" {
		t.Errorf("header %q", got)
	}
	if got := f.TableData[0][:3]; got[0] != "割り箸" || got[1] != "スプーン" || got[2] != "フォーク" {
		t.Errorf("row 0 %q", got)
	}
	// A value of fewer lines sits in the middle.
	if got := f.TableData[1][:3]; got[0] != "" || got[1] != "カイロ" || got[2] != "" {
		t.Errorf("row 1 %q", got)
	}

	// One line, or another fill, keeps one frame.
	for text, fill := range map[string]tepra.FillMode{"a": tepra.FillInverted, "a\nb": tepra.FillShaded} {
		f, err := tepra.NewText(tepra.Tape18mm, text, tepra.WithFill(fill))
		if err != nil {
			t.Fatal(err)
		}
		if n := len(f.Documents[0].Drawers); n != 1 {
			t.Errorf("%v, %q: %d drawers", fill, text, n)
		}
	}
}

func TestOption_alignment(t *testing.T) {
	bottom := tepra.VerticalAlignBottom
	f, err := tepra.NewText(tepra.Tape18mm, "x", tepra.WithAlign(tepra.AlignCenter), tepra.WithVerticalAlign(bottom))
	if err != nil {
		t.Fatal(err)
	}
	d := fixtureAs[*tepra.TextDrawer](t, f.Documents[0].Drawers[0])
	if d.Alignment != tepra.AlignCenter || d.VerticalAlignment != tepra.VerticalAlignBottom {
		t.Errorf("alignment %v, %v", d.Alignment, d.VerticalAlignment)
	}
	f, err = tepra.NewText(tepra.Tape18mm, "x")
	if err != nil {
		t.Fatal(err)
	}
	d = fixtureAs[*tepra.TextDrawer](t, f.Documents[0].Drawers[0])
	if d.Alignment != tepra.AlignLeft || d.VerticalAlignment != tepra.VerticalAlignMiddle {
		t.Errorf("default alignment %v, %v; want the app's", d.Alignment, d.VerticalAlignment)
	}
	three := tepra.VerticalAlign(3)
	// Alignment 3 crashes the app. Vertical alignment 3 does not, but it is
	// not offered.
	if _, err := tepra.NewText(tepra.Tape18mm, "x", tepra.WithAlign(3)); !errors.Is(err, tepra.ErrInvalidOptions) || !errors.Is(err, tepra.ErrCrashesApp) {
		t.Errorf("Align 3: %v, want ErrInvalidOptions and ErrCrashesApp", err)
	}
	_, err = tepra.NewText(tepra.Tape18mm, "x", tepra.WithVerticalAlign(three))
	if !errors.Is(err, tepra.ErrInvalidOptions) || errors.Is(err, tepra.ErrCrashesApp) {
		t.Errorf("VerticalAlign 3: %v, want ErrInvalidOptions alone", err)
	}
}

func TestOption_Fill(t *testing.T) {
	f, err := tepra.NewText(tepra.Tape18mm, "x", tepra.WithFill(tepra.FillInverted))
	if err != nil {
		t.Fatal(err)
	}
	if d := fixtureAs[*tepra.TextDrawer](t, f.Documents[0].Drawers[0]); d.FillMode != tepra.FillInverted {
		t.Errorf("text fill %v", d.FillMode)
	}
	f, err = tepra.NewPour(tepra.Tape18mm, "c", []string{"x"}, tepra.WithFill(tepra.FillShaded))
	if err != nil {
		t.Fatal(err)
	}
	if d := fixtureAs[*tepra.PourDrawer](t, f.Documents[0].Drawers[0]); d.FillMode != tepra.FillShaded {
		t.Errorf("pour fill %v", d.FillMode)
	}
}

func TestOption_rejectsFillModesTheAppCrashesOn(t *testing.T) {
	opts := tepra.WithFill(3)
	if _, err := tepra.NewText(tepra.Tape18mm, "x", opts); !errors.Is(err, tepra.ErrInvalidOptions) || !errors.Is(err, tepra.ErrCrashesApp) {
		t.Errorf("NewText: %v, want ErrInvalidOptions and ErrCrashesApp", err)
	}
	if _, err := tepra.NewPour(tepra.Tape18mm, "c", []string{"x"}, opts); !errors.Is(err, tepra.ErrCrashesApp) {
		t.Errorf("NewPour: %v, want ErrCrashesApp", err)
	}
}

func TestOption_sideways(t *testing.T) {
	sideways := []struct {
		name     string
		opt      tepra.Option
		rotation tepra.Rotation
		vertical bool
	}{
		{"90", tepra.WithRotation(tepra.Rotation90), tepra.Rotation90, false},
		{"270", tepra.WithRotation(tepra.Rotation270), tepra.Rotation270, false},
		{"vertical", tepra.WithVertical(), tepra.RotationNone, true},
	}
	for _, tt := range sideways {
		f, err := tepra.NewText(tepra.Tape18mm, "会議室\n3F", tt.opt)
		if err != nil {
			t.Fatal(err)
		}
		d := fixtureAs[*tepra.TextDrawer](t, f.Documents[0].Drawers[0])
		// The longest line, 3 ems, runs across the 43.2 points of the tape:
		// 14.4 points, in steps of 0.5. The two lines lie side by side.
		if d.FontSize != 14 || d.Frame[2] != 35 || d.Rotate != tt.rotation || d.TextVertical != tt.vertical {
			t.Errorf("%s: size %v, frame %v, rotate %v, vertical %v", tt.name, d.FontSize, d.Frame, d.Rotate, d.TextVertical)
		}
	}
	f, err := tepra.NewText(tepra.Tape18mm, "会議室\n3F", tepra.WithRotation(tepra.Rotation180))
	if err != nil {
		t.Fatal(err)
	}
	if d := fixtureAs[*tepra.TextDrawer](t, f.Documents[0].Drawers[0]); d.FontSize != 17.5 || d.Rotate != tepra.Rotation180 {
		t.Errorf("180: size %v, rotate %v", d.FontSize, d.Rotate)
	}
	tests := map[string][]tepra.TextOption{
		"rotation 90":           {tepra.WithRotation(90)},
		"vertical and rotated":  {tepra.WithVertical(), tepra.WithRotation(tepra.Rotation90)},
		"inverted lines across": {tepra.WithVertical(), tepra.WithFill(tepra.FillInverted)},
	}
	for name, opts := range tests {
		if _, err := tepra.NewText(tepra.Tape18mm, "a\nb", opts...); !errors.Is(err, tepra.ErrInvalidOptions) {
			t.Errorf("%s: %v, want ErrInvalidOptions", name, err)
		}
	}
}

func TestOption_LengthMM(t *testing.T) {
	f, err := tepra.NewText(tepra.Tape18mm, "カイロ", tepra.WithLengthMM(50))
	if err != nil {
		t.Fatal(err)
	}
	doc := f.Documents[0]
	d := fixtureAs[*tepra.TextDrawer](t, doc.Drawers[0])
	if doc.AutoLength || doc.FixedLength != tepra.MMToPoints(50) || doc.LengthMM()-50 > 1e-9 || 50-doc.LengthMM() > 1e-9 {
		t.Errorf("auto %v, fixed %v, length %v mm", doc.AutoLength, doc.FixedLength, doc.LengthMM())
	}
	// The text is centred in the 48 mm inside the margins.
	if left, right := d.Frame[0], doc.Frame[2]-d.Frame[0]-d.Frame[2]; left-right > 1e-9 || right-left > 1e-9 {
		t.Errorf("text at %v in %v", d.Frame, doc.Frame[2])
	}

	// Text too long for the length gets smaller, unless its size is given.
	f, err = tepra.NewText(tepra.Tape18mm, "とても長いラベルの文字", tepra.WithLengthMM(30))
	if err != nil {
		t.Fatal(err)
	}
	d = fixtureAs[*tepra.TextDrawer](t, f.Documents[0].Drawers[0])
	// The text stays on the label: it starts at or after its left edge, and
	// ends at or before its right one.
	if d.FontSize >= 35 || d.Frame[0] < 0 || d.Frame[0]+d.Frame[2] > f.Documents[0].Frame[2]+1e-9 {
		t.Errorf("size %v, frame %v in %v", d.FontSize, d.Frame, f.Documents[0].Frame)
	}
	if _, err := tepra.NewText(tepra.Tape18mm, "とても長いラベルの文字", tepra.WithLengthMM(30), tepra.WithFontSize(30)); !errors.Is(err, tepra.ErrNoRoom) {
		t.Errorf("given size: %v, want ErrNoRoom", err)
	}

	pour, err := tepra.NewPour(tepra.Tape18mm, "c", []string{"a", "b"}, tepra.WithLengthMM(40))
	if err != nil {
		t.Fatal(err)
	}
	if doc := pour.Documents[0]; doc.AutoLength || doc.FixedLength != tepra.MMToPoints(40) {
		t.Errorf("pour: auto %v, fixed %v", doc.AutoLength, doc.FixedLength)
	}
}

func TestOption_ranges(t *testing.T) {
	tests := map[string]tepra.Option{
		"margin below 1 mm":   tepra.WithMarginMM(0.5),
		"margin over 500 mm":  tepra.WithMarginMM(501),
		"length below 10 mm":  tepra.WithLengthMM(9),
		"length over 3000 mm": tepra.WithLengthMM(3001),
		"threshold over 255":  tepra.WithImageThreshold(256),
		"negative threshold":  tepra.WithImageThreshold(-1),
		"unknown image mode":  tepra.WithImageMode(9),
		"font size 0":         tepra.WithFontSize(0),
		"negative padding":    tepra.WithPaddingMM(-1),
		"vertical align 3":    tepra.WithVerticalAlign(3),
	}
	for name, opt := range tests {
		if _, err := tepra.NewText(tepra.Tape18mm, "x", opt); !errors.Is(err, tepra.ErrInvalidOptions) {
			t.Errorf("%s: %v, want ErrInvalidOptions", name, err)
		}
	}
	for name, opt := range map[string]tepra.Option{"margin 1": tepra.WithMarginMM(1), "margin 500": tepra.WithMarginMM(500), "length 3000": tepra.WithLengthMM(3000)} {
		if _, err := tepra.NewText(tepra.Tape18mm, "x", opt); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	// Every bad option is reported, not only the first.
	_, err := tepra.NewText(tepra.Tape18mm, "x", tepra.WithMarginMM(0), tepra.WithLengthMM(1))
	if err == nil || !strings.Contains(err.Error(), "margin") || !strings.Contains(err.Error(), "length") {
		t.Errorf("two bad options: %v", err)
	}
}
