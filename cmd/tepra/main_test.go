package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
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
	f, err := tepra.ReadFile(out)
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
	if box, ok := ds[0].(*tepra.TextDrawer); !ok || box.BIType != tepra.BorderRounded || box.BILineWidthValue != 1.4 {
		t.Errorf("drawer 0 %+v, want a rounded border 1.4 wide", ds[0])
	}
	if d, ok := ds[1].(*tepra.PourDrawer); !ok || d.FontName != tepra.FontGothicBold || d.Alignment != tepra.AlignCenter || d.VerticalAlignment != tepra.VerticalAlignTop {
		t.Errorf("drawer 1 %+v, want a pour drawer in bold, centred at the top", ds[1])
	}
}

func TestPour_readsStdin(t *testing.T) {
	out := filepath.Join(t.TempDir(), "x.tm2")
	if _, err := runMain(t, "a\nb\n", "pour", "-o", out); err != nil {
		t.Fatal(err)
	}
	f, err := tepra.ReadFile(out)
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
	if _, err := tepra.ReadFile(out); err != nil {
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
	if want := "wrote " + out + ": 2 pages, 18.2 to 39.0 mm long, on 18 mm tape\n"; stdout != want {
		t.Errorf("stdout %q, want %q", stdout, want)
	}
	f, err := tepra.ReadFile(out)
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
	f, err := tepra.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	// Inverted text gets a box per line.
	var texts []string
	for _, d := range f.Documents[0].Drawers {
		if d, ok := d.(*tepra.TextDrawer); ok && d.FillMode == tepra.FillInverted {
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
		"  pour   x   0.0  y   1.6  w  54.0  h  15.2 mm  column 0, size 17.5, OTBGoStdR-Normal\n",
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
	if !strings.HasPrefix(stdout, "tepra version (devel) (go") {
		t.Errorf("stdout %q", stdout)
	}
}

// runMain runs tepra with args and stdin, and returns what it wrote to
// stdout.
func runMain(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	cmd := newMainCommand()
	var stdout bytes.Buffer
	cmd.Reader = strings.NewReader(stdin)
	cmd.Writer = &stdout
	cmd.ErrWriter = &stdout
	err := cmd.Run(context.Background(), append([]string{"tepra"}, args...))
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
	f, err := tepra.ReadFile(out)
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
		{errors.New("tepra: no text"), 1, "tepra: no text\n"},
		{errors.New("pour takes one file at most"), 1, "tepra: pour takes one file at most\n"},
		{cli.Exit("quiet", 3), 3, "tepra: quiet\n"},
		{cli.Exit("", 4), 4, ""},
		{errors.Join(errors.New("tepra: bad option"), errors.New("value 1: tepra: bad value")), 1, "tepra: bad option\ntepra: value 1: bad value\n"},
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
	f, err := tepra.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	doc := f.Documents[0]
	if d, ok := doc.Drawers[0].(*tepra.PourDrawer); !ok || d.Attribute != tepra.CodeJAN13 {
		t.Errorf("drawer %+v, want JAN-13", doc.Drawers[0])
	}
	if doc.AutoLength || doc.FixedLength != tepra.MMToPoints(40) || doc.Margin != tepra.MMToPoints(2) {
		t.Errorf("auto %v, fixed %v, margin %v", doc.AutoLength, doc.FixedLength, doc.Margin)
	}

	pic := filepath.Join(dir, "pic.png")
	if err := os.WriteFile(pic, fixturePicture(t), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := runMain(t, "", "text", "--rotate", "90", "--icon", pic, "--image-mode", "screen", "--image-threshold", "64", "-f", "-o", out, "x"); err != nil {
		t.Fatal(err)
	}
	if f, err = tepra.ReadFile(out); err != nil {
		t.Fatal(err)
	}
	ds := f.Documents[0].Drawers
	if img, ok := ds[0].(*tepra.ImageDrawer); !ok || img.ImageMode != tepra.ImageModeScreen || img.ImageThreshold != 64 {
		t.Errorf("drawer 0 %+v, want a screened picture", ds[0])
	}
	if d, ok := ds[1].(*tepra.TextDrawer); !ok || d.Rotate != tepra.Rotation90 {
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
	if _, err := runMain(t, "a\n", "pour", "--margin", "0", "-f", "-o", out); !errors.Is(err, tepra.ErrInvalidOptions) {
		t.Errorf("margin 0: %v", err)
	}
}

func TestPour_readsCSV(t *testing.T) {
	out := filepath.Join(t.TempDir(), "x.tm2")
	in := "\ufeff\"a, b\"\r\n\"c\r\nd\"\r\ne\r\n"
	if _, err := runMain(t, in, "pour", "-o", out); err != nil {
		t.Fatal(err)
	}
	f, err := tepra.ReadFile(out)
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
	f, err := tepra.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	f.Documents[0].Drawers = append(f.Documents[0].Drawers, &tepra.RawDrawer{Type: "shape", JSON: []byte(`{"type":"shape","frame":[0,0,1,1]}`)})
	if err := f.WriteFile(out); err != nil {
		t.Fatal(err)
	}
	stdout, err := runMain(t, "", "inspect", out)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"  image  ", "bytes, threshold 128", "  shape  ", "(not known to tepra)"} {
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
