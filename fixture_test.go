package tm2_test

import (
	"bytes"
	"encoding/json/jsontext"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"slices"
	"testing"

	"github.com/mpyw/tepra.tm2"
)

// fixturePourPath is a pour label of ten rows on 18 mm tape, saved by TEPRA
// LINK 2 for Android. Besides the pour drawer it holds an empty text drawer
// the app left outside the tape.
//
//declscope:shared // the tests of file.go, pour.go and text.go compare with it
const fixturePourPath = "testdata/pour-18mm.tm2"

// fixtureObject is a JSON object with its keys in order.
//
//declscope:shared // the tests compare the key order of what they write
type fixtureObject struct {
	keys   []string
	values map[string]any
}

// readFixture returns the bytes of the file at path.
//
//declscope:shared // the tests read fixturePourPath with it
func readFixture(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// fixturePicture returns a w by h picture, black on white, as a PNG, or as
// a JPEG when jpg is set.
//
//declscope:shared // the tests of icons and backgrounds pass it as Options
func fixturePicture(t *testing.T, w, h int, jpg bool) []byte {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, w, h))
	for i := range img.Pix {
		img.Pix[i] = byte(255 * (i % 2))
	}
	var buf bytes.Buffer
	var err error
	if jpg {
		err = jpeg.Encode(&buf, img, nil)
	} else {
		err = png.Encode(&buf, img)
	}
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// fixtureFrameNear reports whether frames a and b differ by less than a
// thousandth of a point.
//
//declscope:shared // the tests compare frames placed by arithmetic
func fixtureFrameNear(a, b tm2.Rect) bool {
	for i := range a {
		if d := a[i] - b[i]; d > 1e-3 || d < -1e-3 {
			return false
		}
	}
	return true
}

// fixtureShapeNear reports whether the PNG png has the shape of frame, to a
// hundredth: the app keeps a picture's shape, so it fills only a frame of
// its own shape.
//
//declscope:shared // the tests of pictures that must cover their frame check it
func fixtureShapeNear(t *testing.T, pngData []byte, frame tm2.Rect) bool {
	t.Helper()
	cfg, err := png.DecodeConfig(bytes.NewReader(pngData))
	if err != nil {
		t.Fatal(err)
	}
	d := float64(cfg.Width)/float64(cfg.Height) - frame[2]/frame[3]
	return d < 0.01 && d > -0.01
}

// fixtureSameKeys reports whether a and b hold the same keys, in any order.
// The order of fields of the types decides the order of keys written, not
// the order of files the app wrote.
//
//declscope:shared // the tests compare the keys they write with the fixture's
func fixtureSameKeys(a, b []string) bool {
	return slices.Equal(slices.Sorted(slices.Values(a)), slices.Sorted(slices.Values(b)))
}

// fixtureAs returns v as a T, or fails the test.
//
//declscope:shared // the tests take drawers and decoded JSON apart with it
func fixtureAs[T any](t *testing.T, v any) T {
	t.Helper()
	got, ok := v.(T)
	if !ok {
		var want T
		t.Fatalf("got %T, want %T", v, want)
	}
	return got
}

// decodeFixture decodes JSON, keeping the order of keys: an object becomes a
// *fixtureObject, and the rest decode as encoding/json does.
//
//declscope:shared // the tests decode both the fixture and what they write
func decodeFixture(t *testing.T, data []byte) any {
	t.Helper()
	dec := jsontext.NewDecoder(bytes.NewReader(data))
	v, err := decodeFixtureValue(dec)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dec.ReadToken(); !errors.Is(err, io.EOF) {
		t.Fatalf("data after the JSON value: %v", err)
	}
	return v
}

// fixtureDrawer returns the first drawer of type typ in the first document
// of file, as decodeFixture decoded it.
//
//declscope:shared // the tests compare a drawer of what they write with the fixture's
func fixtureDrawer(t *testing.T, file any, typ string) *fixtureObject {
	t.Helper()
	doc := fixtureDocument(t, file)
	for _, d := range fixtureAs[[]any](t, doc.values["drawers"]) {
		obj := fixtureAs[*fixtureObject](t, d)
		if obj.values["type"] == typ {
			return obj
		}
	}
	t.Fatalf("no drawer of type %q", typ)
	return nil
}

// fixtureDocument returns the first document of file, as decodeFixture
// decoded it.
//
//declscope:shared // the tests compare the document of what they write with the fixture's
func fixtureDocument(t *testing.T, file any) *fixtureObject {
	t.Helper()
	docs := fixtureAs[[]any](t, fixtureAs[*fixtureObject](t, file).values["documents"])
	if len(docs) == 0 {
		t.Fatal("no documents")
	}
	return fixtureAs[*fixtureObject](t, docs[0])
}

// decodeFixtureValue decodes the next value of dec. A number becomes a
// float64, so 0.0 and 0 compare equal.
func decodeFixtureValue(dec *jsontext.Decoder) (any, error) {
	tok, err := dec.ReadToken()
	if err != nil {
		return nil, err
	}
	switch tok.Kind() {
	case jsontext.KindBeginObject:
		obj := &fixtureObject{values: map[string]any{}}
		for dec.PeekKind() != jsontext.KindEndObject {
			tok, err := dec.ReadToken()
			if err != nil {
				return nil, err
			}
			// A token is valid only until the next read.
			name := tok.String()
			v, err := decodeFixtureValue(dec)
			if err != nil {
				return nil, err
			}
			obj.keys = append(obj.keys, name)
			obj.values[name] = v
		}
		_, err := dec.ReadToken()
		return obj, err
	case jsontext.KindBeginArray:
		arr := []any{}
		for dec.PeekKind() != jsontext.KindEndArray {
			v, err := decodeFixtureValue(dec)
			if err != nil {
				return nil, err
			}
			arr = append(arr, v)
		}
		_, err := dec.ReadToken()
		return arr, err
	case jsontext.KindNumber:
		return tok.Float()
	case jsontext.KindString:
		return tok.String(), nil
	case jsontext.KindTrue, jsontext.KindFalse:
		return tok.Bool(), nil
	default:
		return nil, nil
	}
}
