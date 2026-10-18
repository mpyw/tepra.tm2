package tm2_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mpyw/tepra.tm2"
)

func TestOption_Border(t *testing.T) {
	f, err := tm2.NewText(tm2.Tape18mm, "会議室", tm2.WithBorder(tm2.BorderRounded), tm2.WithBorderWidth(1.4), tm2.WithFontSize(10))
	if err != nil {
		t.Fatal(err)
	}
	doc := f.Documents[0]
	if len(doc.Drawers) != 2 {
		t.Fatalf("got %d drawers, want the border's box and the text", len(doc.Drawers))
	}
	box := fixtureAs[*tm2.TextDrawer](t, doc.Drawers[0])
	text := fixtureAs[*tm2.TextDrawer](t, doc.Drawers[1])
	if box.BIType != tm2.BorderRounded || box.BILineWidthValue != 1.4 || box.Text != "" {
		t.Errorf("box: border %v, width %v, text %q", box.BIType, box.BILineWidthValue, box.Text)
	}
	if text.BIType != tm2.BorderNone {
		t.Errorf("text has border %v", text.BIType)
	}
	// The text, 3 ems of 10 points, sits DefaultBorderPaddingMM inside the
	// box, which covers the label.
	pad := tm2.MMToPoints(tm2.DefaultBorderPaddingMM)
	if want := (tm2.Rect{pad, 4.4 + pad, 30, 43.2 - 2*pad}); !fixtureFrameNear(text.Frame, want) {
		t.Errorf("text frame %v, want %v", text.Frame, want)
	}
	if want := (tm2.Rect{0, 4.4, 30 + 2*pad, 43.2}); !fixtureFrameNear(box.Frame, want) {
		t.Errorf("box frame %v, want %v", box.Frame, want)
	}

	f, err = tm2.NewPour(tm2.Tape18mm, "c", []string{"x"}, tm2.WithBorder(tm2.BorderTop), tm2.WithPaddingMM(3))
	if err != nil {
		t.Fatal(err)
	}
	box = fixtureAs[*tm2.TextDrawer](t, f.Documents[0].Drawers[0])
	pour := fixtureAs[*tm2.PourDrawer](t, f.Documents[0].Drawers[1])
	if box.BIType != tm2.BorderTop || box.BILineWidthValue != tm2.DefaultBorderWidth {
		t.Errorf("box: border %v, width %v", box.BIType, box.BILineWidthValue)
	}
	if got, want := pour.Frame[0], tm2.MMToPoints(3); got != want {
		t.Errorf("pour drawer at x %v, want the padding given, %v", got, want)
	}
	for name, opt := range map[string]tm2.Option{"border 6": tm2.WithBorder(6), "negative width": tm2.WithBorderWidth(-1)} {
		if _, err := tm2.NewText(tm2.Tape18mm, "x", opt); !errors.Is(err, tm2.ErrInvalidOptions) {
			t.Errorf("%s: %v, want ErrInvalidOptions", name, err)
		}
	}
}

func TestOption_invertedLinesApart(t *testing.T) {
	f, err := tm2.NewText(tm2.Tape18mm, "会議室A\n3F", tm2.WithFill(tm2.FillInverted), tm2.WithFontSize(14))
	if err != nil {
		t.Fatal(err)
	}
	ds := f.Documents[0].Drawers
	if len(ds) != 3 {
		t.Fatalf("got %d drawers, want a black picture and a box per line", len(ds))
	}
	black := fixtureAs[*tm2.ImageDrawer](t, ds[0])
	first := fixtureAs[*tm2.TextDrawer](t, ds[1])
	second := fixtureAs[*tm2.TextDrawer](t, ds[2])
	// The picture covers both boxes: the app prints a box with no text white.
	if want := (tm2.Rect{first.Frame[0], first.Frame[1], first.Frame[2], first.Frame[3] * 2}); black.Frame != want {
		t.Errorf("black picture at %v, want %v", black.Frame, want)
	}
	if !fixtureShapeNear(t, black.Image, black.Frame) {
		t.Error("black picture is not of its frame's shape")
	}
	if first.Text != "会議室A" || second.Text != "3F" || first.FillMode != tm2.FillInverted || second.FillMode != tm2.FillInverted {
		t.Errorf("lines %q (%v) and %q (%v)", first.Text, first.FillMode, second.Text, second.FillMode)
	}
	// Stacked with no gap, so the black boxes join.
	if first.Frame[1]+first.Frame[3] != second.Frame[1] || first.Frame[3] != 43.2/2 {
		t.Errorf("frames %v and %v", first.Frame, second.Frame)
	}

	f, err = tm2.NewPour(tm2.Tape18mm, "品名", []string{"割り箸\nスプーン\nフォーク", "カイロ"}, tm2.WithFill(tm2.FillInverted))
	if err != nil {
		t.Fatal(err)
	}
	ds = f.Documents[0].Drawers
	if len(ds) != 4 {
		t.Fatalf("got %d drawers, want a black picture and one per line of the longest value", len(ds))
	}
	fixtureAs[*tm2.ImageDrawer](t, ds[0])
	for c, d := range ds[1:] {
		if p := fixtureAs[*tm2.PourDrawer](t, d); p.ColumnNumber != c {
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
	for text, fill := range map[string]tm2.FillMode{"a": tm2.FillInverted, "a\nb": tm2.FillShaded} {
		f, err := tm2.NewText(tm2.Tape18mm, text, tm2.WithFill(fill))
		if err != nil {
			t.Fatal(err)
		}
		if n := len(f.Documents[0].Drawers); n != 1 {
			t.Errorf("%v, %q: %d drawers", fill, text, n)
		}
	}
}

func TestOption_alignment(t *testing.T) {
	bottom := tm2.VerticalAlignBottom
	f, err := tm2.NewText(tm2.Tape18mm, "x", tm2.WithAlign(tm2.AlignCenter), tm2.WithVerticalAlign(bottom))
	if err != nil {
		t.Fatal(err)
	}
	d := fixtureAs[*tm2.TextDrawer](t, f.Documents[0].Drawers[0])
	if d.Alignment != tm2.AlignCenter || d.VerticalAlignment != tm2.VerticalAlignBottom {
		t.Errorf("alignment %v, %v", d.Alignment, d.VerticalAlignment)
	}
	f, err = tm2.NewText(tm2.Tape18mm, "x")
	if err != nil {
		t.Fatal(err)
	}
	d = fixtureAs[*tm2.TextDrawer](t, f.Documents[0].Drawers[0])
	if d.Alignment != tm2.AlignLeft || d.VerticalAlignment != tm2.VerticalAlignMiddle {
		t.Errorf("default alignment %v, %v; want the app's", d.Alignment, d.VerticalAlignment)
	}
	three := tm2.VerticalAlign(3)
	// Alignment 3 crashes the app. Vertical alignment 3 does not, but it is
	// not offered.
	if _, err := tm2.NewText(tm2.Tape18mm, "x", tm2.WithAlign(3)); !errors.Is(err, tm2.ErrInvalidOptions) || !errors.Is(err, tm2.ErrCrashesApp) {
		t.Errorf("Align 3: %v, want ErrInvalidOptions and ErrCrashesApp", err)
	}
	_, err = tm2.NewText(tm2.Tape18mm, "x", tm2.WithVerticalAlign(three))
	if !errors.Is(err, tm2.ErrInvalidOptions) || errors.Is(err, tm2.ErrCrashesApp) {
		t.Errorf("VerticalAlign 3: %v, want ErrInvalidOptions alone", err)
	}
}

func TestOption_Fill(t *testing.T) {
	f, err := tm2.NewText(tm2.Tape18mm, "x", tm2.WithFill(tm2.FillInverted))
	if err != nil {
		t.Fatal(err)
	}
	if d := fixtureAs[*tm2.TextDrawer](t, f.Documents[0].Drawers[0]); d.FillMode != tm2.FillInverted {
		t.Errorf("text fill %v", d.FillMode)
	}
	f, err = tm2.NewPour(tm2.Tape18mm, "c", []string{"x"}, tm2.WithFill(tm2.FillShaded))
	if err != nil {
		t.Fatal(err)
	}
	if d := fixtureAs[*tm2.PourDrawer](t, f.Documents[0].Drawers[0]); d.FillMode != tm2.FillShaded {
		t.Errorf("pour fill %v", d.FillMode)
	}
}

func TestOption_rejectsFillModesTheAppCrashesOn(t *testing.T) {
	opts := tm2.WithFill(3)
	if _, err := tm2.NewText(tm2.Tape18mm, "x", opts); !errors.Is(err, tm2.ErrInvalidOptions) || !errors.Is(err, tm2.ErrCrashesApp) {
		t.Errorf("NewText: %v, want ErrInvalidOptions and ErrCrashesApp", err)
	}
	if _, err := tm2.NewPour(tm2.Tape18mm, "c", []string{"x"}, opts); !errors.Is(err, tm2.ErrCrashesApp) {
		t.Errorf("NewPour: %v, want ErrCrashesApp", err)
	}
}

func TestOption_sideways(t *testing.T) {
	sideways := []struct {
		name     string
		opt      tm2.Option
		rotation tm2.Rotation
		vertical bool
	}{
		{"90", tm2.WithRotation(tm2.Rotation90), tm2.Rotation90, false},
		{"270", tm2.WithRotation(tm2.Rotation270), tm2.Rotation270, false},
		{"vertical", tm2.WithVertical(), tm2.RotationNone, true},
	}
	for _, tt := range sideways {
		f, err := tm2.NewText(tm2.Tape18mm, "会議室\n3F", tt.opt)
		if err != nil {
			t.Fatal(err)
		}
		d := fixtureAs[*tm2.TextDrawer](t, f.Documents[0].Drawers[0])
		// The longest line, 3 ems, runs across the 43.2 points of the tape:
		// 14.4 points, in steps of 0.5. The two lines lie side by side.
		if d.FontSize != 14 || d.Frame[2] != 35 || d.Rotate != tt.rotation || d.TextVertical != tt.vertical {
			t.Errorf("%s: size %v, frame %v, rotate %v, vertical %v", tt.name, d.FontSize, d.Frame, d.Rotate, d.TextVertical)
		}
	}
	f, err := tm2.NewText(tm2.Tape18mm, "会議室\n3F", tm2.WithRotation(tm2.Rotation180))
	if err != nil {
		t.Fatal(err)
	}
	if d := fixtureAs[*tm2.TextDrawer](t, f.Documents[0].Drawers[0]); d.FontSize != 17.5 || d.Rotate != tm2.Rotation180 {
		t.Errorf("180: size %v, rotate %v", d.FontSize, d.Rotate)
	}
	tests := map[string][]tm2.TextOption{
		"rotation 90":           {tm2.WithRotation(90)},
		"vertical and rotated":  {tm2.WithVertical(), tm2.WithRotation(tm2.Rotation90)},
		"inverted lines across": {tm2.WithVertical(), tm2.WithFill(tm2.FillInverted)},
	}
	for name, opts := range tests {
		if _, err := tm2.NewText(tm2.Tape18mm, "a\nb", opts...); !errors.Is(err, tm2.ErrInvalidOptions) {
			t.Errorf("%s: %v, want ErrInvalidOptions", name, err)
		}
	}
}

func TestOption_LengthMM(t *testing.T) {
	f, err := tm2.NewText(tm2.Tape18mm, "カイロ", tm2.WithLengthMM(50))
	if err != nil {
		t.Fatal(err)
	}
	doc := f.Documents[0]
	d := fixtureAs[*tm2.TextDrawer](t, doc.Drawers[0])
	if doc.AutoLength || doc.FixedLength != tm2.MMToPoints(50) || doc.LengthMM()-50 > 1e-9 || 50-doc.LengthMM() > 1e-9 {
		t.Errorf("auto %v, fixed %v, length %v mm", doc.AutoLength, doc.FixedLength, doc.LengthMM())
	}
	// The text is centred in the 48 mm inside the margins.
	if left, right := d.Frame[0], doc.Frame[2]-d.Frame[0]-d.Frame[2]; left-right > 1e-9 || right-left > 1e-9 {
		t.Errorf("text at %v in %v", d.Frame, doc.Frame[2])
	}

	// Text too long for the length gets smaller, unless its size is given.
	f, err = tm2.NewText(tm2.Tape18mm, "とても長いラベルの文字", tm2.WithLengthMM(30))
	if err != nil {
		t.Fatal(err)
	}
	d = fixtureAs[*tm2.TextDrawer](t, f.Documents[0].Drawers[0])
	// The text stays on the label: it starts at or after its left edge, and
	// ends at or before its right one.
	if d.FontSize >= 35 || d.Frame[0] < 0 || d.Frame[0]+d.Frame[2] > f.Documents[0].Frame[2]+1e-9 {
		t.Errorf("size %v, frame %v in %v", d.FontSize, d.Frame, f.Documents[0].Frame)
	}
	if _, err := tm2.NewText(tm2.Tape18mm, "とても長いラベルの文字", tm2.WithLengthMM(30), tm2.WithFontSize(30)); !errors.Is(err, tm2.ErrNoRoom) {
		t.Errorf("given size: %v, want ErrNoRoom", err)
	}

	pour, err := tm2.NewPour(tm2.Tape18mm, "c", []string{"a", "b"}, tm2.WithLengthMM(40))
	if err != nil {
		t.Fatal(err)
	}
	if doc := pour.Documents[0]; doc.AutoLength || doc.FixedLength != tm2.MMToPoints(40) {
		t.Errorf("pour: auto %v, fixed %v", doc.AutoLength, doc.FixedLength)
	}
}

func TestOption_ranges(t *testing.T) {
	tests := map[string]tm2.Option{
		"margin below 1 mm":   tm2.WithMarginMM(0.5),
		"margin over 500 mm":  tm2.WithMarginMM(501),
		"length below 10 mm":  tm2.WithLengthMM(9),
		"length over 3000 mm": tm2.WithLengthMM(3001),
		"threshold over 255":  tm2.WithImageThreshold(256),
		"negative threshold":  tm2.WithImageThreshold(-1),
		"unknown image mode":  tm2.WithImageMode(9),
		"font size 0":         tm2.WithFontSize(0),
		"negative padding":    tm2.WithPaddingMM(-1),
		"vertical align 3":    tm2.WithVerticalAlign(3),
	}
	for name, opt := range tests {
		if _, err := tm2.NewText(tm2.Tape18mm, "x", opt); !errors.Is(err, tm2.ErrInvalidOptions) {
			t.Errorf("%s: %v, want ErrInvalidOptions", name, err)
		}
	}
	for name, opt := range map[string]tm2.Option{"margin 1": tm2.WithMarginMM(1), "margin 500": tm2.WithMarginMM(500), "length 3000": tm2.WithLengthMM(3000)} {
		if _, err := tm2.NewText(tm2.Tape18mm, "x", opt); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	// Every bad option is reported, not only the first.
	_, err := tm2.NewText(tm2.Tape18mm, "x", tm2.WithMarginMM(0), tm2.WithLengthMM(1))
	if err == nil || !strings.Contains(err.Error(), "margin") || !strings.Contains(err.Error(), "length") {
		t.Errorf("two bad options: %v", err)
	}
}
