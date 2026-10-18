package tm2_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mpyw/tepra.tm2"
)

func TestFilePatch_noOperationsWritesTheSame(t *testing.T) {
	paths, err := filepath.Glob("testdata/*.tm2")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		f, err := tm2.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var want, got bytes.Buffer
		if _, err := f.WriteTo(&want); err != nil {
			t.Fatal(err)
		}
		if err := f.Patch(); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if _, err := f.WriteTo(&got); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(want.Bytes(), got.Bytes()) {
			t.Errorf("%s: written differently after an empty patch", path)
		}
	}
}

func TestFilePatch_replaceMoveTest(t *testing.T) {
	f := patchFixture(t)
	err := f.Patch(
		patchOp("test", "/documents/0/drawers/0/TIText", `"a"`),
		patchOp("replace", "/documents/0/drawers/0/TIText", `"b"`),
		patchOp("replace", "/documents/0/drawers/0/TIAlignment", `2`),
		patchOp("replace", "/fileName", `"x.tm2"`),
	)
	if err != nil {
		t.Fatal(err)
	}
	d := fixtureAs[*tm2.TextDrawer](t, f.Documents[0].Drawers[0])
	if d.Text != "b" || d.Alignment != tm2.AlignRight || f.FileName != "x.tm2" {
		t.Errorf("text %q, align %v, name %q", d.Text, d.Alignment, f.FileName)
	}

	// Two drawers; the last moves first, under the other.
	if err := f.Patch(patchOp("add", "/documents/0/drawers/-", `{"type":"text","TIText":"c"}`)); err != nil {
		t.Fatal(err)
	}
	if err := f.Patch(tm2.PatchOp{Op: "move", From: "/documents/0/drawers/1", Path: "/documents/0/drawers/0"}); err != nil {
		t.Fatal(err)
	}
	if d := fixtureAs[*tm2.TextDrawer](t, f.Documents[0].Drawers[0]); d.Text != "c" {
		t.Errorf("first drawer %q, want c", d.Text)
	}
}

func TestFilePatch_addsADrawerAsTheAppMakesOne(t *testing.T) {
	f := patchFixture(t)
	if err := f.Patch(patchOp("add", "/documents/0/drawers/-", `{"type":"text","TIText":"new","BIType":1}`)); err != nil {
		t.Fatal(err)
	}
	doc := f.Documents[0]
	d := fixtureAs[*tm2.TextDrawer](t, doc.Drawers[1])
	want := tm2.NewTextDrawer("new", doc.Frame)
	if d.Text != "new" || d.BIType != tm2.BorderBox || d.Frame != doc.Frame || d.FontFace != want.FontFace || d.TapeForegroundColor != tm2.Black || !d.HoldAspect {
		t.Errorf("drawer %+v, want a new text box in the label's frame", d)
	}
	if d.Identifier == "" || d.Identifier == doc.Drawers[0].Base().Identifier {
		t.Errorf("identifier %q", d.Identifier)
	}
}

func TestFilePatch_copiesGetNewIdentifiers(t *testing.T) {
	f := patchFixture(t)
	if err := f.Patch(tm2.PatchOp{Op: "copy", From: "/documents/0", Path: "/documents/-"}); err != nil {
		t.Fatal(err)
	}
	if len(f.Documents) != 2 {
		t.Fatalf("%d documents", len(f.Documents))
	}
	a, b := f.Documents[0], f.Documents[1]
	if a.Identifier == b.Identifier || a.Drawers[0].Base().Identifier == b.Drawers[0].Base().Identifier {
		t.Error("the copy has the identifiers of the original")
	}
	// The app lists the pages only with MultiplePage set.
	if !a.MultiplePage || !b.MultiplePage {
		t.Errorf("MultiplePage %v, %v", a.MultiplePage, b.MultiplePage)
	}

	// The page the app opens stays on a document.
	f.CurrentPage = 1
	if err := f.Patch(tm2.PatchOp{Op: "remove", Path: "/documents/1"}); err != nil {
		t.Fatal(err)
	}
	if f.CurrentPage != 0 {
		t.Errorf("current page %d", f.CurrentPage)
	}
}

func TestFilePatch_makesAPicturePNG(t *testing.T) {
	f := patchFixture(t)
	jpg := base64.StdEncoding.EncodeToString(fixturePicture(t, 8, 4, true))
	if err := f.Patch(patchOp("add", "/documents/0/drawers/0", `{"type":"image","image":"`+jpg+`"}`)); err != nil {
		t.Fatal(err)
	}
	img := fixtureAs[*tm2.ImageDrawer](t, f.Documents[0].Drawers[0])
	if !bytes.HasPrefix(img.Image, []byte("\x89PNG")) || img.ImageThreshold != tm2.DefaultImageThreshold {
		t.Errorf("image %q..., threshold %d", img.Image[:4], img.ImageThreshold)
	}
	bad := base64.StdEncoding.EncodeToString([]byte("not a picture"))
	if err := f.Patch(patchOp("replace", "/documents/0/drawers/0/image", `"`+bad+`"`)); !errors.Is(err, tm2.ErrBadImage) {
		t.Errorf("not a picture: %v", err)
	}
}

func TestFilePatch_padsRows(t *testing.T) {
	f := patchFixture(t)
	if err := f.Patch(patchOp("add", "/tableData/-", `["a"]`)); err != nil {
		t.Fatal(err)
	}
	if len(f.TableData) != 1 || len(f.TableData[0]) != tm2.FileTableColumns || f.TableData[0][0] != "a" {
		t.Errorf("rows %q", f.TableData)
	}
	long := `["` + strings.Repeat(`a","`, tm2.FileTableColumns) + `a"]`
	if err := f.Patch(patchOp("add", "/tableData/-", long)); !errors.Is(err, tm2.ErrOverLimit) {
		t.Errorf("51 columns: %v", err)
	}
}

func TestFilePatch_keepsNumbersOfUnknownKeys(t *testing.T) {
	f := patchFixture(t)
	f.Unknown = jsontext.Value(`{"big":12345678901234567890}`)
	if err := f.Patch(patchOp("replace", "/fileName", `"y.tm2"`)); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(f.Unknown, []byte("12345678901234567890")) {
		t.Errorf("unknown keys %s", f.Unknown)
	}
}

func TestFilePatch_rejects(t *testing.T) {
	tests := map[string]struct {
		op   tm2.PatchOp
		want error
	}{
		"new key":          {patchOp("add", "/documents/0/foo", `1`), tm2.ErrBadPatch},
		"replace no key":   {patchOp("replace", "/documents/0/foo", `1`), tm2.ErrBadPatch},
		"remove key":       {tm2.PatchOp{Op: "remove", Path: "/documents/0/name"}, tm2.ErrBadPatch},
		"move a key":       {tm2.PatchOp{Op: "move", From: "/documents/0/name", Path: "/fileName"}, tm2.ErrBadPatch},
		"move into itself": {tm2.PatchOp{Op: "move", From: "/documents/0", Path: "/documents/0/drawers/0"}, tm2.ErrBadPatch},
		"remove the file":  {tm2.PatchOp{Op: "remove", Path: ""}, tm2.ErrBadPatch},
		"failed test":      {patchOp("test", "/fileName", `"z.tm2"`), tm2.ErrBadPatch},
		"no value":         {tm2.PatchOp{Op: "replace", Path: "/fileName"}, tm2.ErrBadPatch},
		"bad value":        {patchOp("replace", "/fileName", `{`), tm2.ErrBadPatch},
		"unknown op":       {patchOp("merge", "/fileName", `"a"`), tm2.ErrBadPatch},
		"no slash":         {patchOp("replace", "fileName", `"a"`), tm2.ErrBadPatch},
		"bad index":        {patchOp("replace", "/documents/01", `{}`), tm2.ErrBadPatch},
		"past the end":     {tm2.PatchOp{Op: "remove", Path: "/documents/1"}, tm2.ErrBadPatch},
		"into a value":     {patchOp("replace", "/fileName/x", `"a"`), tm2.ErrBadPatch},
		"drawer, no type":  {patchOp("add", "/documents/0/drawers/-", `{"TIText":"a"}`), tm2.ErrBadPatch},
		"drawer, shape":    {patchOp("add", "/documents/0/drawers/-", `{"type":"shape"}`), tm2.ErrBadPatch},
		"drawer, bad key":  {patchOp("add", "/documents/0/drawers/-", `{"type":"text","color":1}`), tm2.ErrBadPatch},
		"no documents":     {tm2.PatchOp{Op: "remove", Path: "/documents/0"}, tm2.ErrBadPatch},
		"crashes the app":  {patchOp("replace", "/documents/0/drawers/0/TIFillMode", `7`), tm2.ErrCrashesApp},
		"wrong value type": {patchOp("replace", "/documents/0/SettingsTapeWidth", `"wide"`), tm2.ErrBadPatch},
		"two values":       {patchOp("replace", "/fileName", `"a" "b"`), tm2.ErrBadPatch},
		"duplicate keys":   {patchOp("add", "/documents/0/drawers/-", `{"type":"text","type":"pour"}`), tm2.ErrBadPatch},
		"from no slash":    {tm2.PatchOp{Op: "move", From: "fileName", Path: "/fileName"}, tm2.ErrBadPatch},
		"copy no value":    {tm2.PatchOp{Op: "copy", From: "/foo", Path: "/fileName"}, tm2.ErrBadPatch},
		"move no value":    {tm2.PatchOp{Op: "move", From: "/documents/3", Path: "/documents/0"}, tm2.ErrBadPatch},
		"no key on path":   {patchOp("replace", "/foo/x", `1`), tm2.ErrBadPatch},
		"no item on path":  {patchOp("replace", "/documents/5/name", `"a"`), tm2.ErrBadPatch},
		"through a value":  {patchOp("replace", "/fileName/x/y", `1`), tm2.ErrBadPatch},
		"test no value":    {patchOp("test", "/foo", `1`), tm2.ErrBadPatch},
		"add past the end": {patchOp("add", "/documents/2", `{}`), tm2.ErrBadPatch},
		"add under no key": {patchOp("add", "/foo/0", `1`), tm2.ErrBadPatch},
		"remove under":     {tm2.PatchOp{Op: "remove", Path: "/foo/0"}, tm2.ErrBadPatch},
		"replace past":     {patchOp("replace", "/documents/1", `{}`), tm2.ErrBadPatch},
		"drawer, no page":  {patchOp("add", "/documents/9/drawers/-", `{"type":"text"}`), tm2.ErrBadPatch},
		"51 titles":        {patchOp("add", "/tableHeader/-", `"x"`), tm2.ErrOverLimit},
	}
	for name, tt := range tests {
		f := patchFixture(t)
		var before bytes.Buffer
		if _, err := f.WriteTo(&before); err != nil {
			t.Fatal(err)
		}
		if err := f.Patch(tt.op); !errors.Is(err, tt.want) {
			t.Errorf("%s: %v, want %v", name, err, tt.want)
		}
		var after bytes.Buffer
		if _, err := f.WriteTo(&after); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(before.Bytes(), after.Bytes()) {
			t.Errorf("%s: the file changed", name)
		}
	}
}

func TestFilePatch_wholeFileAndItems(t *testing.T) {
	f := patchFixture(t)
	file := `{"documents":[{"SettingsTapeWidth":12,"drawers":[]}],"currentPage":0,"fileName":"w.tm2"}`
	if err := f.Patch(patchOp("replace", "", file)); err != nil {
		t.Fatal(err)
	}
	if f.FileName != "w.tm2" || len(f.Documents[0].Drawers) != 0 {
		t.Errorf("name %q, %d drawers", f.FileName, len(f.Documents[0].Drawers))
	}
	if err := f.Patch(patchOp("add", "", strings.Replace(file, "w.tm2", "v.tm2", 1))); err != nil {
		t.Fatal(err)
	}
	if f.FileName != "v.tm2" {
		t.Errorf("name %q", f.FileName)
	}

	// A pour drawer, then replaced in the array by a text box.
	if err := f.Patch(patchOp("add", "/documents/0/drawers/0", `{"type":"pour","ColumnNumber":1}`)); err != nil {
		t.Fatal(err)
	}
	if d := fixtureAs[*tm2.PourDrawer](t, f.Documents[0].Drawers[0]); d.ColumnNumber != 1 || d.FontFace != 1 {
		t.Errorf("pour drawer %+v", d)
	}
	// A move to where the value is changes nothing.
	if err := f.Patch(tm2.PatchOp{Op: "move", From: "/documents/0/drawers/0", Path: "/documents/0/drawers/0"}); err != nil {
		t.Fatal(err)
	}
	text, err := patchDrawerJSON(patchFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Patch(patchOp("replace", "/documents/0/drawers/0", string(text))); err != nil {
		t.Fatal(err)
	}
	if d := fixtureAs[*tm2.TextDrawer](t, f.Documents[0].Drawers[0]); d.Text != "a" {
		t.Errorf("drawer %+v", d)
	}

	// A drawer added where a document is no object: the add fails.
	err = f.Patch(patchOp("replace", "/documents/0", `1`), patchOp("add", "/documents/0/drawers/-", `{"type":"text"}`))
	if !errors.Is(err, tm2.ErrBadPatch) {
		t.Errorf("drawer in a number: %v", err)
	}
}

func TestFilePatch_keepsRawDrawers(t *testing.T) {
	f := patchFixture(t)
	raw := &tm2.RawDrawer{Type: "shape", JSON: jsontext.Value(`{"type":"shape","identifier":"x"}`)}
	raw.Identifier = "x"
	f.Documents[0].Drawers = append(f.Documents[0].Drawers, raw)
	if err := f.Patch(patchOp("replace", "/fileName", `"r.tm2"`)); err != nil {
		t.Fatal(err)
	}
	if d := fixtureAs[*tm2.RawDrawer](t, f.Documents[0].Drawers[1]); d.Type != "shape" || d.Identifier != "x" {
		t.Errorf("raw drawer %+v", d)
	}
}

func TestFilePatch_fileTheAppCrashesOn(t *testing.T) {
	f := patchFixture(t)
	f.CurrentPage = 3
	if err := f.Patch(); !errors.Is(err, tm2.ErrCrashesApp) {
		t.Errorf("current page 3 of 1: %v", err)
	}
}

// patchDrawerJSON is the first drawer of f, as the file holds it.
func patchDrawerJSON(f *tm2.File) (jsontext.Value, error) {
	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		return nil, err
	}
	var file struct {
		Documents []struct {
			Drawers []jsontext.Value `json:"drawers"`
		} `json:"documents"`
	}
	if err := json.Unmarshal(buf.Bytes(), &file); err != nil {
		return nil, err
	}
	return file.Documents[0].Drawers[0], nil
}

// patchFixture returns a label of one text box, "a".
func patchFixture(t *testing.T) *tm2.File {
	t.Helper()
	f, err := tm2.NewText(tm2.Tape12mm, "a")
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// patchOp is an operation of path with value, as JSON.
func patchOp(op, path, value string) tm2.PatchOp {
	return tm2.PatchOp{Op: op, Path: path, Value: jsontext.Value(value)}
}
