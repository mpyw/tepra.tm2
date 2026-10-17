package tepra_test

import (
	"bytes"
	jsonv1 "encoding/json"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mpyw/tepra.tm2"
)

func TestReadFile_writesBackWhatTheAppWrote(t *testing.T) {
	paths, err := filepath.Glob("testdata/*.tm2")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		want := readFixture(t, path)
		f, err := tepra.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		if _, err := f.WriteTo(&buf); err != nil {
			t.Fatal(err)
		}
		if bytes.ContainsRune(buf.Bytes(), '\n') {
			t.Errorf("%s: written file has more than one line", path)
		}
		if err := fileCompareJSON(decodeFixture(t, want), decodeFixture(t, buf.Bytes()), "$"); err != "" {
			t.Errorf("%s: %s", path, err)
		}
	}
}

func TestReadFile_readsTheFixture(t *testing.T) {
	f, err := tepra.ReadFile(fixturePourPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Documents) != 1 {
		t.Fatalf("got %d documents, want 1", len(f.Documents))
	}
	doc := f.Documents[0]
	if !doc.PourMode || doc.TapeWidthMM != 18 {
		t.Errorf("PourMode %v, TapeWidthMM %d; want true, 18", doc.PourMode, doc.TapeWidthMM)
	}
	pour, ok := doc.Drawers[1].(*tepra.PourDrawer)
	if !ok {
		t.Fatalf("drawer 1 is %T, want *tepra.PourDrawer", doc.Drawers[1])
	}
	if pour.Identifier != "1a87ff22-bb84-4ac5-9e28-d451bc3011ae" || pour.FontSize != 17.5 {
		t.Errorf("pour drawer: identifier %q, font size %v", pour.Identifier, pour.FontSize)
	}
	if got, want := len(f.TableData), 10; got != want {
		t.Errorf("got %d rows, want %d", got, want)
	}
	if f.SavedLength == nil || *f.SavedLength-doc.LengthMM() > 1e-4 || doc.LengthMM()-*f.SavedLength > 1e-4 {
		t.Errorf("savedLength %v, but the document is %v mm", f.SavedLength, doc.LengthMM())
	}
}

func TestFile_WriteFile(t *testing.T) {
	f, err := tepra.NewText(tepra.Tape18mm, "a<b>&c")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "out.tm2")
	if err := f.WriteFile(path); err != nil {
		t.Fatal(err)
	}
	b := readFixture(t, path)
	// The app writes <, > and & as they are.
	if !bytes.Contains(b, []byte(`"TIText":"a<b>&c"`)) {
		t.Errorf("text escaped in %s", b)
	}
	got, err := tepra.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if fixtureAs[*tepra.TextDrawer](t, got.Documents[0].Drawers[0]).Text != "a<b>&c" {
		t.Error("text not read back")
	}
}

func TestParseFile_rejectsBadJSON(t *testing.T) {
	for _, in := range []string{`{`, `not json`, `{"documents":[{"drawers":[1]}]}`, `{"currentPage":"x"}`, `null`, `{}`, `{"documents":[]}`} {
		if _, err := tepra.ParseFile(bytes.NewReader([]byte(in))); !errors.Is(err, tepra.ErrBadFile) {
			t.Errorf("ParseFile(%s): %v, want ErrBadFile", in, err)
		}
	}
	// Errors of the file system are as they are (#6), reading too (#20).
	if _, err := tepra.ReadFile(filepath.Join(t.TempDir(), "none.tm2")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("missing file: %v, want fs.ErrNotExist", err)
	}
	var pathErr *fs.PathError
	if _, err := tepra.ReadFile(t.TempDir()); !errors.As(err, &pathErr) || errors.Is(err, tepra.ErrBadFile) {
		t.Errorf("a folder: %v, want the fs.PathError alone", err)
	}
}

// A file the package cannot encode is ErrBadFile, from WriteTo and
// WriteFile (#6).
func TestFile_encodeError(t *testing.T) {
	f, err := tepra.NewText(tepra.Tape18mm, "x")
	if err != nil {
		t.Fatal(err)
	}
	f.Documents[0].Margin = math.NaN()
	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); !errors.Is(err, tepra.ErrBadFile) {
		t.Errorf("WriteTo: %v, want ErrBadFile", err)
	}
	if err := f.WriteFile(filepath.Join(t.TempDir(), "x.tm2")); !errors.Is(err, tepra.ErrBadFile) {
		t.Errorf("WriteFile: %v, want ErrBadFile", err)
	}
}

// fileCompareJSON returns where want and got differ, or "". Keys may come in
// another order. decodeFixture made numbers float64, so 0.0 equals 0.
func fileCompareJSON(want, got any, path string) string {
	switch w := want.(type) {
	case *fixtureObject:
		g, ok := got.(*fixtureObject)
		if !ok {
			return path + ": not an object"
		}
		if !fixtureSameKeys(w.keys, g.keys) {
			return path + ": keys differ:\nwant " + fileQuote(w.keys) + "\ngot  " + fileQuote(g.keys)
		}
		for _, k := range w.keys {
			if d := fileCompareJSON(w.values[k], g.values[k], path+"."+k); d != "" {
				return d
			}
		}
	case []any:
		g, ok := got.([]any)
		if !ok || len(g) != len(w) {
			return path + ": arrays differ in length"
		}
		for i := range w {
			if d := fileCompareJSON(w[i], g[i], path+"[]"); d != "" {
				return d
			}
		}
	case float64:
		if g, ok := got.(float64); !ok || w != g {
			return fmt.Sprintf("%s: numbers differ: %v and %v", path, want, got)
		}
	default:
		if want != got {
			return path + ": values differ"
		}
	}
	return ""
}

// fileQuote writes keys as JSON, for a message.
func fileQuote(keys []string) string {
	b, _ := json.Marshal(keys) // strings always marshal
	return string(b)
}

func TestFile_rejectsPagesTheAppCrashesOn(t *testing.T) {
	f, err := tepra.NewText(tepra.Tape18mm, "x")
	if err != nil {
		t.Fatal(err)
	}
	for _, page := range []int{1, -1} {
		f.CurrentPage = page
		var buf bytes.Buffer
		if _, err := f.WriteTo(&buf); !errors.Is(err, tepra.ErrCrashesApp) {
			t.Errorf("page %d: %v, want ErrCrashesApp", page, err)
		}
		if _, err := jsonv1.Marshal(f); err == nil {
			t.Errorf("page %d, encoding/json: no error", page)
		}
	}
}

// Keys the package does not know survive a read and a write, on the file,
// on a document and on each kind of drawer (#7).
func TestReadFile_keepsUnknownKeys(t *testing.T) {
	f, err := tepra.NewText(tepra.Tape18mm, "x", tepra.WithIcon(fixturePicture(t, 2, 2, false)))
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	// Add a key to the file, the document and both drawers.
	in := buf.String()
	in = strings.Replace(in, `{"documents":`, `{"futureFile":1,"documents":`, 1)
	in = strings.Replace(in, `"drawers":[`, `"futureDocument":[true],"drawers":[`, 1)
	in = strings.ReplaceAll(in, `"":"",`, `"":"","futureDrawer":"kept",`)
	g, err := tepra.ParseFile(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	if _, err := g.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for key, want := range map[string]int{`"futureFile":1`: 1, `"futureDocument":[true]`: 1, `"futureDrawer":"kept"`: 2} {
		if got := strings.Count(out, key); got != want {
			t.Errorf("%s %d times, want %d, in\n%s", key, got, want, out)
		}
	}
	// The head of each drawer is written once, by Drawers.
	if got := strings.Count(out, `"type":`); got != 2 {
		t.Errorf(`"type" %d times, want 2`, got)
	}
	if got := strings.Count(out, `"":""`); got != 2 {
		t.Errorf(`"" %d times, want 2`, got)
	}
}

// A document that is null or {} is no label (#31).
func TestParseFile_emptyDocuments(t *testing.T) {
	for _, in := range []string{`{"documents":[null]}`, `{"documents":[{}]}`, `{"documents":[{"SettingsTapeWidth":18},{}]}`} {
		if _, err := tepra.ParseFile(strings.NewReader(in)); !errors.Is(err, tepra.ErrBadFile) {
			t.Errorf("ParseFile(%s): %v, want ErrBadFile", in, err)
		}
	}
}

// A drawer of an unknown type is kept, whatever its keys hold (#31).
func TestParseFile_unknownDrawerWithOddKeys(t *testing.T) {
	in := `{"documents":[{"SettingsTapeWidth":18,"drawers":[{"type":"shape","identifier":"id","frame":"x"}]}]}`
	f, err := tepra.ParseFile(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	r := fixtureAs[*tepra.RawDrawer](t, f.Documents[0].Drawers[0])
	if r.Type != "shape" || r.Identifier != "id" || !strings.Contains(string(r.JSON), `"frame":"x"`) {
		t.Errorf("%+v", r)
	}
}
