package tm2_test

import (
	"bytes"
	jsonv1 "encoding/json"
	"encoding/json/v2"
	"errors"
	"testing"

	"github.com/mpyw/tepra.tm2"
)

func TestDrawers_keepsUnknownTypes(t *testing.T) {
	in := `[{"type":"qr","identifier":"id-1","":"","frame":[1,2,3,4],"QRText":"x"}]`
	var ds tm2.Drawers
	if err := json.Unmarshal([]byte(in), &ds); err != nil {
		t.Fatal(err)
	}
	r, ok := ds[0].(*tm2.RawDrawer)
	if !ok {
		t.Fatalf("got %T, want *tm2.RawDrawer", ds[0])
	}
	if r.DrawerType() != "qr" || r.Base().Identifier != "id-1" || r.Base().Frame != (tm2.Rect{1, 2, 3, 4}) {
		t.Errorf("type %q, identifier %q, frame %v", r.DrawerType(), r.Base().Identifier, r.Base().Frame)
	}
	out, err := json.Marshal(ds)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != in {
		t.Errorf("got %s\nwant %s", out, in)
	}
}

func TestDrawers_imageIsBase64(t *testing.T) {
	png := []byte("\x89PNG\r\n")
	ds := tm2.Drawers{&tm2.ImageDrawer{BaseDrawer: tm2.BaseDrawer{Identifier: "id"}, Image: png, ImageThreshold: 128}}
	out, err := json.Marshal(ds)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(out, []byte(`[{"type":"image","identifier":"id","":"","Rotate":0,`)) {
		t.Errorf("head of %s", out)
	}
	if !bytes.Contains(out, []byte(`"image":"iVBORw0K"`)) {
		t.Errorf("image of %s", out)
	}
	var back tm2.Drawers
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatal(err)
	}
	img := fixtureAs[*tm2.ImageDrawer](t, back[0])
	if !bytes.Equal(img.Image, png) || img.ImageThreshold != 128 || img.Identifier != "id" {
		t.Errorf("read back %+v", img)
	}
}

func TestDrawers_rejectsBadDrawers(t *testing.T) {
	// A drawer that is not an object is kept as a RawDrawer (#45). What
	// fails is drawers that are not a list, and a known type of bad fields.
	for _, in := range []string{`{}`, `[{"type":"text","TIText":1}]`} {
		var ds tm2.Drawers
		if err := json.Unmarshal([]byte(in), &ds); err == nil {
			t.Errorf("%s: no error", in)
		}
	}
}

func TestNewTextDrawer_identifiersDiffer(t *testing.T) {
	a, b := tm2.NewTextDrawer("", tm2.Rect{}), tm2.NewTextDrawer("", tm2.Rect{})
	if a.Identifier == b.Identifier || len(a.Identifier) != 36 || a.Identifier[14] != '4' {
		t.Errorf("identifiers %q and %q", a.Identifier, b.Identifier)
	}
}

func TestNewImageDrawer_keysOfTheApp(t *testing.T) {
	d := tm2.NewImageDrawer([]byte("png"), tm2.Rect{0, 4.4, 56, 43.2})
	out, err := json.Marshal(tm2.Drawers{d})
	if err != nil {
		t.Fatal(err)
	}
	if want := `[{"type":"image","identifier":"` + d.Identifier + `","":"",`; !bytes.HasPrefix(out, []byte(want)) {
		t.Errorf("head of %s", out)
	}
	// The keys of the image drawer in a file the app wrote.
	want := []string{
		"type", "identifier", "", "Rotate", "tapeVertical", "tapeTapeFrame", "tapeForegroundColor",
		"tapeBackgroundColor", "frame", "BIType", "BILineThickness", "BILineStyle", "BILineWidth",
		"BILineWidhValue", "frameTop", "image", "imageKind", "imageMode", "imageThreshold", "imageBlock",
		"imageOffset",
	}
	got := fixtureAs[[]any](t, decodeFixture(t, out))
	if keys := fixtureAs[*fixtureObject](t, got[0]).keys; !fixtureSameKeys(keys, want) {
		t.Errorf("keys %q", keys)
	}
	if !bytes.Contains(out, []byte(`"image":"cG5n"`)) || !bytes.Contains(out, []byte(`"imageMode":2,"imageThreshold":128`)) {
		t.Errorf("values of %s", out)
	}
}

func TestDrawers_rejectsFillModesTheAppCrashesOn(t *testing.T) {
	text := tm2.NewTextDrawer("x", tm2.Rect{})
	text.FillMode = 3
	pour := tm2.NewPourDrawer(0, "x", tm2.Rect{})
	pour.FillMode = -1
	for _, d := range []tm2.Drawer{text, pour} {
		if _, err := json.Marshal(tm2.Drawers{d}); !errors.Is(err, tm2.ErrCrashesApp) {
			t.Errorf("%s: %v, want ErrCrashesApp", d.DrawerType(), err)
		}
	}
}

func TestDrawers_rejectsAlignmentsTheAppCrashesOn(t *testing.T) {
	d := tm2.NewPourDrawer(0, "x", tm2.Rect{})
	d.Alignment = 3
	if _, err := json.Marshal(tm2.Drawers{d}); !errors.Is(err, tm2.ErrCrashesApp) {
		t.Errorf("alignment 3: %v, want ErrCrashesApp", err)
	}
	d.Alignment = tm2.AlignRight
	if _, err := json.Marshal(tm2.Drawers{d}); err != nil {
		t.Errorf("AlignRight: %v", err)
	}
}

func TestDrawers_encodingJSONv1(t *testing.T) {
	f, err := tm2.ReadFile(fixturePourPath)
	if err != nil {
		t.Fatal(err)
	}
	ds := f.Documents[0].Drawers
	want, err := json.Marshal(ds)
	if err != nil {
		t.Fatal(err)
	}
	got, err := jsonv1.Marshal(ds)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("encoding/json wrote\n%s\nencoding/json/v2 wrote\n%s", got, want)
	}
	var back tm2.Drawers
	if err := jsonv1.Unmarshal(got, &back); err != nil {
		t.Fatal(err)
	}
	if len(back) != len(ds) || fixtureAs[*tm2.PourDrawer](t, back[1]).Identifier != "1a87ff22-bb84-4ac5-9e28-d451bc3011ae" {
		t.Errorf("read back %+v", back)
	}
}

func TestDrawers_rejectsRotationsAndFacesTheAppCrashesOn(t *testing.T) {
	rotated := tm2.NewTextDrawer("x", tm2.Rect{})
	rotated.Rotate = 90
	face := tm2.NewTextDrawer("x", tm2.Rect{})
	face.FontFace = 2
	img := tm2.NewImageDrawer([]byte("png"), tm2.Rect{})
	img.Rotate = 4
	for _, d := range []tm2.Drawer{rotated, face, img} {
		if _, err := json.Marshal(tm2.Drawers{d}); !errors.Is(err, tm2.ErrCrashesApp) {
			t.Errorf("%+v: %v, want ErrCrashesApp", d, err)
		}
	}
}
