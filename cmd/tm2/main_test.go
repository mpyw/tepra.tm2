package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/mpyw/tepra.tm2"
)

func TestPour(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "items.csv")
	if err := os.WriteFile(in, []byte("\"割り箸\nスプーン\"\n\n  カイロ  \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "日用品.tm2")
	stdout, err := runMain(t, "", "pour", "--column", "品名", "--bold", "--border", "rounded", "--border-width", "1.4", "--align", "center", "--valign", "top", "-o", out, in)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stdout, "wrote "+out+": 2 labels, ") {
		t.Errorf("stdout %q", stdout)
	}
	f, err := tm2.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if f.FileName != "日用品.tm2" || f.TableHeader[0] != "品名" {
		t.Errorf("fileName %q, column %q", f.FileName, f.TableHeader[0])
	}
	if got := []string{f.TableData[0][0], f.TableData[1][0]}; got[0] != "割り箸\nスプーン" || got[1] != "カイロ" {
		t.Errorf("rows %q", got)
	}
	ds := f.Documents[0].Drawers
	if len(ds) != 2 {
		t.Fatalf("%d drawers, want the border's box and the pour drawer", len(ds))
	}
	if box, ok := ds[0].(*tm2.TextDrawer); !ok || box.BIType != tm2.BorderRounded || box.BILineWidthValue != 1.4 {
		t.Errorf("drawer 0 %+v, want a rounded border 1.4 wide", ds[0])
	}
	if d, ok := ds[1].(*tm2.PourDrawer); !ok || d.FontName != tm2.FontGothicBold || d.Alignment != tm2.AlignCenter || d.VerticalAlignment != tm2.VerticalAlignTop {
		t.Errorf("drawer 1 %+v, want a pour drawer in bold, centred at the top", ds[1])
	}
}

func TestPour_readsStdin(t *testing.T) {
	out := filepath.Join(t.TempDir(), "x.tm2")
	if _, err := runMain(t, "a\nb\n", "pour", "-o", out); err != nil {
		t.Fatal(err)
	}
	f, err := tm2.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.TableData) != 2 || f.TableHeader[0] != "Label" {
		t.Errorf("%d rows, column %q", len(f.TableData), f.TableHeader[0])
	}
}

func TestPour_keepsAnExistingFile(t *testing.T) {
	out := filepath.Join(t.TempDir(), "x.tm2")
	if err := os.WriteFile(out, []byte("mine"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := runMain(t, "a\n", "pour", "-o", out)
	if !errors.Is(err, fs.ErrExist) || !strings.Contains(err.Error(), "--force") {
		t.Errorf("error %v, want fs.ErrExist, mentioning --force", err)
	}
	if b, _ := os.ReadFile(out); string(b) != "mine" {
		t.Error("file replaced")
	}
	if _, err := runMain(t, "a\n", "pour", "-f", "-o", out); err != nil {
		t.Fatal(err)
	}
	if _, err := tm2.ReadFile(out); err != nil {
		t.Errorf("file not replaced: %v", err)
	}
}

func TestPour_rejects(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "x.tm2")
	tests := map[string]struct {
		stdin string
		args  []string
	}{
		"no labels":      {"\n\n", []string{"pour", "-o", out}},
		"two files":      {"", []string{"pour", "-o", out, "a", "b"}},
		"missing file":   {"", []string{"pour", "-o", out, filepath.Join(dir, "none.txt")}},
		"unknown tape":   {"a\n", []string{"pour", "--tape", "5", "-o", out}},
		"unknown fill":   {"a\n", []string{"pour", "--fill", "black", "-o", out}},
		"unknown border": {"a\n", []string{"pour", "--border", "double", "-o", out}},
		"unknown align":  {"a\n", []string{"pour", "--align", "justify", "-o", out}},
		"unknown valign": {"a\n", []string{"pour", "--valign", "center", "-o", out}},
		"missing output": {"a\n", []string{"pour"}},
	}
	for name, tt := range tests {
		if _, err := runMain(t, tt.stdin, tt.args...); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestPages(t *testing.T) {
	out := filepath.Join(t.TempDir(), "x.tm2")
	stdout, err := runMain(t, "カイロ\n\"割り箸\nスプーン\nフォーク\"\n", "pages", "--align", "center", "-o", out)
	if err != nil {
		t.Fatal(err)
	}
	if want := "wrote " + out + ": 2 pages, 12.6 to 26.3 mm long, on 12 mm tape\n"; stdout != want {
		t.Errorf("stdout %q, want %q", stdout, want)
	}
	f, err := tm2.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Documents) != 2 {
		t.Fatalf("%d documents", len(f.Documents))
	}
	// inspect counts pages from 1, as the app's list of labels does.
	stdout, err = runMain(t, "", "inspect", out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "\npage 1:\n") || !strings.Contains(stdout, "\npage 2:\n") {
		t.Errorf("inspect %q, want pages 1 and 2", stdout)
	}
	// Pages of one length are described as pour labels are.
	stdout, err = runMain(t, "a\nb\n", "pages", "--tape", "4", "-f", "-o", out)
	if err != nil {
		t.Fatal(err)
	}
	if want := "wrote " + out + ": 2 pages, 12.0 mm each, on 4 mm tape\n"; stdout != want {
		t.Errorf("stdout %q, want %q", stdout, want)
	}
	for _, args := range [][]string{{"pages", "-o", out, "-f"}, {"pages", "--code", "qr", "-o", out, "-f"}} {
		if _, err := runMain(t, "", args...); err == nil {
			t.Errorf("%q: no error", args)
		}
	}
}

func TestText(t *testing.T) {
	out := filepath.Join(t.TempDir(), "room.tm2")
	stdout, err := runMain(t, "", "text", "--margin", "5", "--fill", "inverted", "-o", out, "会議室A\n3F")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stdout, "wrote "+out+": 1 label, ") {
		t.Errorf("stdout %q", stdout)
	}
	f, err := tm2.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	// Inverted text gets a box per line.
	var texts []string
	for _, d := range f.Documents[0].Drawers {
		if d, ok := d.(*tm2.TextDrawer); ok && d.FillMode == tm2.FillInverted {
			texts = append(texts, d.Text)
		}
	}
	if got := strings.Join(texts, ","); got != "会議室A,3F" {
		t.Errorf("inverted lines %q", got)
	}
	for _, args := range [][]string{{"text", "-o", out}, {"text", "-o", out, "x"}, {"text", "-f", "-o", out, "a", "b"}} {
		if _, err := runMain(t, "", args...); err == nil {
			t.Errorf("%q: no error", args)
		}
	}
}

func TestInspect(t *testing.T) {
	stdout, err := runMain(t, "", "inspect", filepath.Join("..", "..", "testdata", "pour-18mm.tm2"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"file:    test.tm2 (version 2.0.0, saved 2026/10/17 16:39)\n",
		"tape:    18 mm, pour, label 56.0 mm long, margin 1.0 mm\n",
		"drawers: /documents/0/drawers\n",
		"   1 pour   x   0.0  y   1.6  w  54.0  h  15.2 mm  column 0, size 17.5, OTBGoStdR-Normal\n",
		"rows:    10, column \"品名\"\n",
		"    2  割り箸 / スプーン / フォーク\n",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("no %q in\n%s", want, stdout)
		}
	}
	if _, err := runMain(t, "", "inspect"); err == nil {
		t.Error("no file: no error")
	}
	if _, err := runMain(t, "", "inspect", "none.tm2"); err == nil {
		t.Error("missing file: no error")
	}
}

func TestVersion(t *testing.T) {
	stdout, err := runMain(t, "", "--version")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stdout, "tm2 version (devel) (go") {
		t.Errorf("stdout %q", stdout)
	}
}

func TestInspect_json(t *testing.T) {
	dir := t.TempDir()
	pic := filepath.Join(dir, "pic.png")
	if err := os.WriteFile(pic, fixturePicture(t), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "x.tm2")
	if _, err := runMain(t, "", "text", "--icon", pic, "--border", "box", "--align", "center", "-o", out, "a"); err != nil {
		t.Fatal(err)
	}
	stdout, err := runMain(t, "", "inspect", out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "border box 0.7") || !strings.Contains(stdout, "align center") {
		t.Errorf("inspect\n%s", stdout)
	}
	stdout, err = runMain(t, "", "inspect", "--json", out)
	if err != nil {
		t.Fatal(err)
	}
	// A frame takes one line, and a picture shows by its size.
	for _, want := range []string{`"SettingsTapeWidth": 12,`, `"TIText": "a",`, `"tapeForegroundColor": [0, 0, 0, 1],`, `"image": "(picture 1: PNG, 4 by 4 pixels, `} {
		if !strings.Contains(stdout, want) {
			t.Errorf("no %q in\n%s", want, stdout)
		}
	}
}

func TestInspect_settings(t *testing.T) {
	f, err := tm2.NewPour(tm2.Tape12mm, "URL", []string{"https://example.com/"},
		tm2.WithCode(tm2.CodeQR), tm2.WithRotation(tm2.Rotation180), tm2.WithFill(tm2.FillShaded),
		tm2.WithVerticalAlign(tm2.VerticalAlignTop), tm2.WithVertical(), tm2.WithIcon(fixturePicture(t)),
		tm2.WithImageMode(tm2.ImageModeDiffusion))
	if err != nil {
		t.Fatal(err)
	}
	doc := &f.Documents[0]
	box := tm2.NewTextDrawer("固", doc.Frame)
	box.HoldAspect = false
	box.TapeForegroundColor = tm2.White
	doc.Drawers = append(doc.Drawers, box)
	path := filepath.Join(t.TempDir(), "x.tm2")
	if err := f.WriteFile(path); err != nil {
		t.Fatal(err)
	}
	stdout, err := runMain(t, "", "inspect", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"mode diffusion", "code qr", "valign top", "fill shaded", "vertical", "keeps its size", "rotate 180",
		`"固", fixed (固), color [1 1 1 1]`,
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("no %q in\n%s", want, stdout)
		}
	}
}

// A file the app would crash on is described, but not as JSON, which
// would write it.
func TestInspect_jsonOfAFileTheAppCrashesOn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.tm2")
	if err := os.WriteFile(path, []byte(`{"documents":[{"SettingsTapeWidth":18,"drawers":[]}],"currentPage":3}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := runMain(t, "", "inspect", path); err != nil {
		t.Error(err)
	}
	if _, err := runMain(t, "", "inspect", "--json", path); !errors.Is(err, tm2.ErrCrashesApp) {
		t.Errorf("--json: %v", err)
	}
}

func TestEdit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.tm2")
	if _, err := runMain(t, "", "text", "-o", path, "a"); err != nil {
		t.Fatal(err)
	}
	// In place, from stdin.
	stdout, err := runMain(t, `[{"op": "replace", "path": "/documents/0/drawers/0/TIText", "value": "b"}]`, "edit", path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stdout, "wrote "+path+": 1 label") {
		t.Errorf("stdout %q", stdout)
	}
	f, err := tm2.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if d, ok := f.Documents[0].Drawers[0].(*tm2.TextDrawer); !ok || d.Text != "b" {
		t.Errorf("drawer %+v, want text b", f.Documents[0].Drawers[0])
	}

	// To another file, from a patch file, with a picture.
	pic := filepath.Join(dir, "pic.png")
	if err := os.WriteFile(pic, fixturePicture(t), 0o600); err != nil {
		t.Fatal(err)
	}
	patch := filepath.Join(dir, "patch.json")
	if err := os.WriteFile(patch, []byte(`[{"op": "add", "path": "/documents/0/drawers/0", "value": {"type": "image"}, "image_path": "`+pic+`"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "y.tm2")
	if _, err := runMain(t, "", "edit", "-o", out, path, patch); err != nil {
		t.Fatal(err)
	}
	f, err = tm2.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if img, ok := f.Documents[0].Drawers[0].(*tm2.ImageDrawer); !ok || !bytes.Equal(img.Image, fixturePicture(t)) {
		t.Errorf("drawer %T, want the picture", f.Documents[0].Drawers[0])
	}
	// The picture replaced.
	if _, err := runMain(t, `[{"op": "replace", "path": "/documents/0/drawers/0/image", "image_path": "`+pic+`"}]`, "edit", out); err != nil {
		t.Fatal(err)
	}
	// out is there now.
	if _, err := runMain(t, "", "edit", "-o", out, path, patch); err == nil || !strings.Contains(err.Error(), "--force") {
		t.Errorf("existing output: %v", err)
	}
	if _, err := runMain(t, "", "edit", "-f", "-o", out, path, patch); err != nil {
		t.Error(err)
	}
}

func TestEdit_rejects(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.tm2")
	if _, err := runMain(t, "", "text", "-o", path, "a"); err != nil {
		t.Fatal(err)
	}
	tests := map[string]struct {
		stdin string
		args  []string
	}{
		"no file":        {"[]", []string{"edit"}},
		"three files":    {"", []string{"edit", path, "a", "b"}},
		"missing patch":  {"", []string{"edit", path, filepath.Join(dir, "none.json")}},
		"not an array":   {`{"op": "remove"}`, []string{"edit", path}},
		"not .tm2":       {"[]", []string{"edit", "-o", filepath.Join(dir, "x.txt"), path}},
		"missing file":   {"[]", []string{"edit", filepath.Join(dir, "none.tm2")}},
		"new key":        {`[{"op": "add", "path": "/foo", "value": 1}]`, []string{"edit", path}},
		"missing image":  {`[{"op": "replace", "path": "/documents/0/drawers/0/image", "image_path": "none.png"}]`, []string{"edit", path}},
		"image for text": {`[{"op": "replace", "path": "/documents/0/drawers/0/TIText", "image_path": "` + path + `"}]`, []string{"edit", path}},
		"image in value": {`[{"op": "replace", "path": "/fileName", "value": "a", "image_path": "` + path + `"}]`, []string{"edit", path}},
		// A file read into a drawer that is not a picture would carry it.
		"image into text": {`[{"op": "add", "path": "/documents/0/drawers/-", "value": {"type": "text"}, "image_path": "` + path + `"}]`, []string{"edit", path}},
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for name, tt := range tests {
		if _, err := runMain(t, tt.stdin, tt.args...); err == nil {
			t.Errorf("%s: no error", name)
		}
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(before, after) {
			t.Fatalf("%s: the file changed", name)
		}
	}
}

// Another path to the file read, such as a symlink, is an edit in place.
func TestEdit_inPlaceThroughASymlink(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.tm2")
	if _, err := runMain(t, "", "text", "-o", path, "a"); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.tm2")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := runMain(t, `[{"op": "replace", "path": "/fileName", "value": "b.tm2"}]`, "edit", "-o", link, path); err != nil {
		t.Fatal(err)
	}
	f, err := tm2.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.FileName != "b.tm2" {
		t.Errorf("name %q", f.FileName)
	}
}

// runMain runs tm2 with args and stdin, and returns what it wrote to
// stdout.
func runMain(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	cmd := newMainCommand()
	var stdout bytes.Buffer
	cmd.Reader = strings.NewReader(stdin)
	cmd.Writer = &stdout
	cmd.ErrWriter = &stdout
	err := cmd.Run(context.Background(), append([]string{"tm2"}, args...))
	return stdout.String(), err
}

func TestText_pictures(t *testing.T) {
	dir := t.TempDir()
	pic := filepath.Join(dir, "pic.png")
	if err := os.WriteFile(pic, fixturePicture(t), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "x.tm2")
	if _, err := runMain(t, "", "text", "--icon", pic, "--background", pic, "--padding", "2", "-o", out, "カイロ"); err != nil {
		t.Fatal(err)
	}
	f, err := tm2.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var types []string
	for _, d := range f.Documents[0].Drawers {
		types = append(types, d.DrawerType())
	}
	if got := strings.Join(types, ","); got != "image,image,text" {
		t.Errorf("drawers %s, want the background, the icon and the text", got)
	}
	for _, args := range [][]string{
		{"text", "--icon", filepath.Join(dir, "none.png"), "-o", out, "-f", "x"},
		{"text", "--background", filepath.Join(dir, "none.png"), "-o", out, "-f", "x"},
	} {
		if _, err := runMain(t, "", args...); err == nil {
			t.Errorf("%q: no error", args)
		}
	}
}

func TestMainExitCode(t *testing.T) {
	tests := []struct {
		err  error
		code int
		out  string
	}{
		{nil, 0, ""},
		{tm2.ErrNoText, 1, "tm2: no text\n"},
		{errors.New("pour takes one file at most"), 1, "tm2: pour takes one file at most\n"},
		{cli.Exit("quiet", 3), 3, "tm2: quiet\n"},
		{cli.Exit("", 4), 4, ""},
		{errors.Join(fmt.Errorf("%w: margin", tm2.ErrInvalidOptions), fmt.Errorf("value 1: %w", tm2.ErrNoText)), 1, "tm2: invalid options: margin\ntm2: value 1: no text\n"},
		// A "tm2: " of the user's own stays (#9).
		{fmt.Errorf("%w: %q", tm2.ErrBadCode, "tm2: 12"), 1, "tm2: bad value for a code: \"tm2: 12\"\n"},
	}
	for _, tt := range tests {
		var w bytes.Buffer
		if code := mainExitCode(&w, tt.err); code != tt.code || w.String() != tt.out {
			t.Errorf("mainExitCode(%v) = %d, %q; want %d, %q", tt.err, code, w.String(), tt.code, tt.out)
		}
	}
}

func TestPour_newOptions(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "x.tm2")
	if _, err := runMain(t, "490123456789\n451234567890\n", "pour", "--code", "jan13", "--length", "40", "--margin", "2", "-o", out); err != nil {
		t.Fatal(err)
	}
	f, err := tm2.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	doc := f.Documents[0]
	if d, ok := doc.Drawers[0].(*tm2.PourDrawer); !ok || d.Attribute != tm2.CodeJAN13 {
		t.Errorf("drawer %+v, want JAN-13", doc.Drawers[0])
	}
	if doc.AutoLength || doc.FixedLength != tm2.MMToPoints(40) || doc.Margin != tm2.MMToPoints(2) {
		t.Errorf("auto %v, fixed %v, margin %v", doc.AutoLength, doc.FixedLength, doc.Margin)
	}

	pic := filepath.Join(dir, "pic.png")
	if err := os.WriteFile(pic, fixturePicture(t), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := runMain(t, "", "text", "--rotate", "90", "--icon", pic, "--image-mode", "screen", "--image-threshold", "64", "-f", "-o", out, "x"); err != nil {
		t.Fatal(err)
	}
	if f, err = tm2.ReadFile(out); err != nil {
		t.Fatal(err)
	}
	ds := f.Documents[0].Drawers
	if img, ok := ds[0].(*tm2.ImageDrawer); !ok || img.ImageMode != tm2.ImageModeScreen || img.ImageThreshold != 64 {
		t.Errorf("drawer 0 %+v, want a screened picture", ds[0])
	}
	if d, ok := ds[1].(*tm2.TextDrawer); !ok || d.Rotate != tm2.Rotation90 {
		t.Errorf("drawer 1 %+v, want text turned 90°", ds[1])
	}
	if _, err := runMain(t, "", "text", "--vertical", "-f", "-o", out, "縦"); err != nil {
		t.Fatal(err)
	}

	// Each bad flag is reported at once.
	_, err = runMain(t, "a\n", "pour", "--rotate", "45", "--code", "ean8", "-f", "-o", out)
	if err == nil || !strings.Contains(err.Error(), "45") || !strings.Contains(err.Error(), "ean8") {
		t.Errorf("two bad names: %v", err)
	}
	if _, err := runMain(t, "a\n", "pour", "--margin", "0", "-f", "-o", out); !errors.Is(err, tm2.ErrInvalidOptions) {
		t.Errorf("margin 0: %v", err)
	}
}

func TestPour_readsCSV(t *testing.T) {
	out := filepath.Join(t.TempDir(), "x.tm2")
	in := "\ufeff\"a, b\"\r\n\"c\r\nd\"\r\ne\r\n"
	if _, err := runMain(t, in, "pour", "-o", out); err != nil {
		t.Fatal(err)
	}
	f, err := tm2.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, row := range f.TableData {
		got = append(got, row[0])
	}
	// The byte order mark goes; a quoted comma and line break stay.
	if strings.Join(got, "|") != "a, b|c\nd|e" {
		t.Errorf("values %q", got)
	}

	_, err = runMain(t, "a,b\n", "pour", "-f", "-o", out)
	if !errors.Is(err, csv.ErrFieldCount) || !strings.Contains(err.Error(), "double quotes") {
		t.Errorf("two columns: %v", err)
	}
	if _, err := runMain(t, "\"open\n", "pour", "-f", "-o", out); err == nil {
		t.Error("an unclosed quote: no error")
	}
	// Spaces around a quoted value go, as around any other (#21). A quote
	// in a value is doubled, inside quotes.
	if _, err := runMain(t, "  \"a,b\"  \n\"He said \"\"hi\"\"\"\n", "pour", "-f", "-o", out); err != nil {
		t.Fatal(err)
	}
	if f, err := tm2.ReadFile(out); err != nil || f.TableData[0][0] != "a,b" || f.TableData[1][0] != `He said "hi"` {
		t.Errorf("%v, rows %q", err, f.TableData)
	}
}

func TestPages_rejects(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "x.tm2")
	tests := map[string][]string{
		"two files":      {"pages", "-o", out, "a", "b"},
		"missing file":   {"pages", "-o", out, filepath.Join(dir, "none.csv")},
		"missing folder": {"pages", "-o", filepath.Join(dir, "none", "x.tm2")},
	}
	for name, args := range tests {
		if _, err := runMain(t, "a\n", args...); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestInspect_picturesAndUnknownDrawers(t *testing.T) {
	dir := t.TempDir()
	pic := filepath.Join(dir, "pic.png")
	if err := os.WriteFile(pic, fixturePicture(t), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "x.tm2")
	if _, err := runMain(t, "", "text", "--icon", pic, "-o", out, "x"); err != nil {
		t.Fatal(err)
	}
	f, err := tm2.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	f.Documents[0].Drawers = append(f.Documents[0].Drawers, &tm2.RawDrawer{Type: "shape", JSON: []byte(`{"type":"shape","frame":[0,0,1,1]}`)})
	if err := f.WriteFile(out); err != nil {
		t.Fatal(err)
	}
	stdout, err := runMain(t, "", "inspect", out)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{" image  ", "bytes, threshold 128", " shape  ", "(not known to tm2)"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("no %q in\n%s", want, stdout)
		}
	}
}

func TestMainVersion_release(t *testing.T) {
	// GoReleaser sets version without the "v" of the tag.
	old := version
	t.Cleanup(func() { version = old })
	version = "1.2.3"
	if got := mainVersion(); !strings.HasPrefix(got, "v1.2.3 (go") {
		t.Errorf("mainVersion() = %q", got)
	}
}

// A file that cannot be encoded writes nothing, and leaves an old file as
// it was (#3).
func TestWriteLabel_failedWriteKeepsTheOldFile(t *testing.T) {
	out := filepath.Join(t.TempDir(), "x.tm2")
	if _, err := runMain(t, "", "text", "-o", out, "good"); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	bad, err := tm2.NewText(tm2.Tape18mm, "x")
	if err != nil {
		t.Fatal(err)
	}
	bad.CurrentPage = 1 // the app crashes on it, so it is not encoded
	if err := writeLabel(out, bad, true, "--force"); !errors.Is(err, tm2.ErrCrashesApp) {
		t.Fatalf("%v, want ErrCrashesApp", err)
	}
	if after, _ := os.ReadFile(out); !bytes.Equal(after, before) {
		t.Error("the old file changed")
	}
	fresh := filepath.Join(filepath.Dir(out), "fresh.tm2")
	if err := writeLabel(fresh, bad, false, "--force"); err == nil {
		t.Fatal("no error")
	}
	if _, err := os.Stat(fresh); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a file was left: %v", err)
	}
}

func TestWriteLabel_replaces(t *testing.T) {
	out := filepath.Join(t.TempDir(), "x.tm2")
	for _, text := range []string{"one", "two"} {
		if _, err := runMain(t, "", "text", "-f", "-o", out, text); err != nil {
			t.Fatal(err)
		}
	}
	f, err := tm2.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if got := f.TextWords; got != "two" {
		t.Errorf("text %q, want two", got)
	}
}

// A replaced label is written in place: through a symlink, keeping its
// mode, in a folder it cannot create files in (#17).
func TestWriteLabel_replacesInPlace(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real.tm2")
	if _, err := runMain(t, "", "text", "-o", real, "one"); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(real, 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.tm2")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) }) // so TempDir can remove it
	if _, err := runMain(t, "", "text", "-f", "-o", link, "two"); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&fs.ModeSymlink == 0 {
		t.Errorf("link: %v, %v; want it still a symlink", info, err)
	}
	f, err := tm2.ReadFile(real)
	if err != nil {
		t.Fatal(err)
	}
	if f.TextWords != "two" {
		t.Errorf("target %q, want two", f.TextWords)
	}
	if info, err := os.Stat(real); err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("target mode %v, %v; want 0600", info.Mode().Perm(), err)
	}
}

// A file with rows but no column titles is described, not a panic (#4).
func TestInspect_rowsWithoutTitles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.tm2")
	if err := os.WriteFile(path, []byte(`{"documents":[{"SettingsTapeWidth":18,"drawers":[]}],"currentPage":0,"tableData":[["a"]]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, err := runMain(t, "", "inspect", path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "rows:    1, column \"\"\n") || !strings.Contains(stdout, "    1  a\n") {
		t.Errorf("stdout\n%s", stdout)
	}
}

// A CSV of rows ending in a lone "\r" is read row by row, and one that is
// not UTF-8 says so (#5).
func TestPour_csvLineEndsAndEncoding(t *testing.T) {
	out := filepath.Join(t.TempDir(), "x.tm2")
	if _, err := runMain(t, "a\rb\rc\r", "pour", "-o", out); err != nil {
		t.Fatal(err)
	}
	f, err := tm2.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.TableData) != 3 {
		t.Errorf("%d rows, want 3", len(f.TableData))
	}
	// "カイロ" in Shift_JIS.
	_, err = runMain(t, "\x83J\x83C\x83\x8d\n", "pour", "-f", "-o", out)
	if !errors.Is(err, tm2.ErrBadText) || !strings.Contains(err.Error(), "UTF-8") {
		t.Errorf("%v, want ErrBadText", err)
	}
}

// Help says each default once, as its usage says it (#21).
func TestHelp_defaults(t *testing.T) {
	stdout, err := runMain(t, "", "text", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout, "(default: 0)") {
		t.Errorf("help says (default: 0):\n%s", stdout)
	}
}

// Blanks at the end of a line inside a quoted value stay (#29).
func TestPour_csvBlanksInsideQuotes(t *testing.T) {
	out := filepath.Join(t.TempDir(), "x.tm2")
	if _, err := runMain(t, "\"a  \nb\"  \n\"c\t\n\td\"\n\"e \"\"x\"\"  \nf\"\n", "pour", "-o", out); err != nil {
		t.Fatal(err)
	}
	f, err := tm2.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, row := range f.TableData {
		got = append(got, row[0])
	}
	if want := []string{"a  \nb", "c\t\n\td", "e \"x\"  \nf"}; strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("values %q, want %q", got, want)
	}
}

// A usage error is returned for main to print once, with no help (#30).
func TestUsageError_printsNothing(t *testing.T) {
	for _, args := range [][]string{{"text", "a"}, {"text", "--nope", "-o", "x.tm2", "a"}, {"--nope"}, {"help", "--nope"}, {"text", "help", "--nope"}} {
		stdout, err := runMain(t, "", args...)
		if err == nil || stdout != "" {
			t.Errorf("%q: %v, printed %q", args, err, stdout)
		}
	}
}

// The label's name drops ".tm2" in any case (#30).
func TestLabelName(t *testing.T) {
	for path, want := range map[string]string{"a/X.TM2": "X", "x.tm2": "x", "x.Tm2": "x", "x.txt": "x.txt", "x": "x"} {
		if got := labelName(path); got != want {
			t.Errorf("labelName(%q) = %q, want %q", path, got, want)
		}
	}
}

// The CLI writes a .tm2 file with a name only, as MCP does (#37, #55).
func TestText_outputNotTM2(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{".tm2", "out.txt", "out"} {
		out := filepath.Join(dir, name)
		if _, err := runMain(t, "", "text", "-o", out, "a"); err == nil || !strings.Contains(err.Error(), "must name a .tm2 file") {
			t.Errorf("%s: %v", name, err)
		}
		if _, err := os.Stat(out); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s: a file was written: %v", name, err)
		}
	}
	if _, err := runMain(t, "", "text", "-o", filepath.Join(dir, "UP.TM2"), "a"); err != nil {
		t.Errorf("UP.TM2: %v", err)
	}
}

// A full-width space after a closing quote is skipped, as everywhere else
// around a value (#37).
func TestPour_csvFullWidthSpace(t *testing.T) {
	out := filepath.Join(t.TempDir(), "x.tm2")
	if _, err := runMain(t, "\"a\"\u3000\n\u3000\"b\"\n", "pour", "-o", out); err != nil {
		t.Fatal(err)
	}
	f, err := tm2.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.TableData) != 2 || f.TableData[0][0] != "a" || f.TableData[1][0] != "b" {
		t.Errorf("rows %q", f.TableData)
	}
}

// An unknown command is a usage error, exit 1; no command shows the help
// (#46).
func TestMain_unknownCommand(t *testing.T) {
	for _, name := range []string{"nope", "help"} {
		_, err := runMain(t, "", name)
		if err == nil || !strings.Contains(err.Error(), "unknown command \""+name+"\"") {
			t.Errorf("%s: %v", name, err)
		}
		var w bytes.Buffer
		if code := mainExitCode(&w, err); code != 1 {
			t.Errorf("%s: exit %d, want 1", name, code)
		}
	}
	stdout, err := runMain(t, "")
	if err != nil || !strings.Contains(stdout, "USAGE:") {
		t.Errorf("no command: %v, %q", err, stdout)
	}
}
