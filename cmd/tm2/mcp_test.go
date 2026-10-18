package main

import (
	"context"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mpyw/tepra.tm2"
)

func TestMCP_tools(t *testing.T) {
	session := mcpConnect(t)
	res, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
	}
	if got := strings.Join(names, ","); got != "create_label_pages,create_pour_labels,create_text_label,inspect_label" {
		t.Errorf("tools %s", got)
	}
}

func TestMCP_createPourLabels(t *testing.T) {
	session := mcpConnect(t)
	path := filepath.Join(t.TempDir(), "日用品.tm2")
	out := mcpCall(t, session, "create_pour_labels", map[string]any{
		"path":   path,
		"labels": []string{"割り箸\nスプーン", "カイロ"},
		"column": "品名",
		"bold":   true,
	})
	// 12 mm tape when left out (#40).
	if out.Labels != 2 || out.FontSize != 11.5 || out.Path != path || !strings.Contains(out.Message, "on 12 mm tape") {
		t.Errorf("output %+v", out)
	}
	f, err := tm2.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.TableHeader[0] != "品名" || f.TableData[1][0] != "カイロ" {
		t.Errorf("column %q, row %q", f.TableHeader[0], f.TableData[1][0])
	}

	// The file is there now, so the same call fails until it may overwrite.
	res := mcpCallResult(t, session, "create_pour_labels", map[string]any{"path": path, "labels": []string{"a"}})
	if !res.IsError || !strings.Contains(mcpResultText(res), "overwrite: true") {
		t.Errorf("second call: %s", mcpResultText(res))
	}
	out = mcpCall(t, session, "create_pour_labels", map[string]any{"path": path, "labels": []string{"a"}, "overwrite": true})
	if out.Labels != 1 {
		t.Errorf("output %+v", out)
	}
}

func TestMCP_createLabelPages(t *testing.T) {
	session := mcpConnect(t)
	path := filepath.Join(t.TempDir(), "日用品.tm2")
	out := mcpCall(t, session, "create_label_pages", map[string]any{
		"path":   path,
		"labels": []string{"カイロ", "割り箸\nスプーン\nフォーク"},
	})
	if out.Labels != 2 || out.FontSize != 7.5 || out.LengthMM < 26.3 || out.LengthMM > 26.4 {
		t.Errorf("output %+v", out)
	}
	f, err := tm2.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Documents) != 2 || !f.Documents[1].MultiplePage {
		t.Errorf("%d documents", len(f.Documents))
	}
	res := mcpCallResult(t, session, "create_label_pages", map[string]any{"path": path, "labels": []string{}, "overwrite": true})
	if !res.IsError {
		t.Error("no labels: no error")
	}
}

func TestMCP_createTextLabel(t *testing.T) {
	session := mcpConnect(t)
	path := filepath.Join(t.TempDir(), "room.tm2")
	out := mcpCall(t, session, "create_text_label", map[string]any{"path": path, "text": "会議室A", "font_size": 14, "fill": "shaded", "border": "top-bottom", "border_width": 2.8, "align": "right", "valign": "bottom"})
	if out.Labels != 1 || out.FontSize != 14 {
		t.Errorf("output %+v", out)
	}
	f, err := tm2.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ds := f.Documents[0].Drawers
	if box, ok := ds[0].(*tm2.TextDrawer); !ok || box.BIType != tm2.BorderTopBottom || box.BILineWidthValue != 2.8 {
		t.Errorf("drawer 0 %+v, want lines 2.8 wide above and below", ds[0])
	}
	if d, ok := ds[len(ds)-1].(*tm2.TextDrawer); !ok || d.FillMode != tm2.FillShaded || d.Alignment != tm2.AlignRight || d.VerticalAlignment != tm2.VerticalAlignBottom {
		t.Errorf("last drawer %+v, want shaded text at the bottom right", ds[len(ds)-1])
	}
}

func TestMCP_pictures(t *testing.T) {
	session := mcpConnect(t)
	dir := t.TempDir()
	pic := filepath.Join(dir, "pic.png")
	if err := os.WriteFile(pic, fixturePicture(t), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "x.tm2")
	out := mcpCall(t, session, "create_pour_labels", map[string]any{
		"path":            path,
		"labels":          []string{"a", "b"},
		"icon_path":       pic,
		"background_path": pic,
		"padding_mm":      2,
		"font_size":       12,
	})
	// The font size comes from the text, not the pictures before it.
	if out.Labels != 2 || out.FontSize != 12 {
		t.Errorf("output %+v", out)
	}
	f, err := tm2.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(f.Documents[0].Drawers); n != 3 {
		t.Errorf("%d drawers, want the background, the icon and the text", n)
	}
	for name, args := range map[string]map[string]any{
		"relative icon":      {"path": filepath.Join(dir, "y.tm2"), "text": "a", "icon_path": "pic.png"},
		"missing background": {"path": filepath.Join(dir, "y.tm2"), "text": "a", "background_path": filepath.Join(dir, "none.png")},
	} {
		res := mcpCallResult(t, session, "create_text_label", args)
		if !res.IsError {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestMCP_newOptions(t *testing.T) {
	session := mcpConnect(t)
	path := filepath.Join(t.TempDir(), "x.tm2")
	out := mcpCall(t, session, "create_pour_labels", map[string]any{
		"path":      path,
		"labels":    []string{"https://example.com"},
		"code":      "qr",
		"rotate":    "180",
		"length_mm": 30,
		"margin_mm": 2,
	})
	if out.Labels != 1 {
		t.Errorf("output %+v", out)
	}
	f, err := tm2.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc := f.Documents[0]
	if d, ok := doc.Drawers[0].(*tm2.PourDrawer); !ok || d.Attribute != tm2.CodeQR || d.Rotate != tm2.Rotation180 {
		t.Errorf("drawer %+v, want a QR code turned 180°", doc.Drawers[0])
	}
	if doc.AutoLength || doc.FixedLength != tm2.MMToPoints(30) {
		t.Errorf("auto %v, fixed %v", doc.AutoLength, doc.FixedLength)
	}
	res := mcpCallResult(t, session, "create_text_label", map[string]any{"path": path, "text": "a", "overwrite": true, "image_threshold": 300})
	if !res.IsError {
		t.Error("threshold 300: no error")
	}
}

func TestMCP_rejects(t *testing.T) {
	session := mcpConnect(t)
	dir := t.TempDir()
	tests := map[string]struct {
		tool string
		args map[string]any
	}{
		"relative path":  {"create_text_label", map[string]any{"path": "x.tm2", "text": "a"}},
		"not .tm2":       {"create_text_label", map[string]any{"path": filepath.Join(dir, "x.txt"), "text": "a"}},
		"unknown tape":   {"create_text_label", map[string]any{"path": filepath.Join(dir, "x.tm2"), "text": "a", "tape_width_mm": 5}},
		"empty text":     {"create_text_label", map[string]any{"path": filepath.Join(dir, "x.tm2"), "text": ""}},
		"unknown align":  {"create_text_label", map[string]any{"path": filepath.Join(dir, "x.tm2"), "text": "a", "align": "justify"}},
		"unknown valign": {"create_text_label", map[string]any{"path": filepath.Join(dir, "x.tm2"), "text": "a", "valign": "center"}},
		"unknown border": {"create_text_label", map[string]any{"path": filepath.Join(dir, "x.tm2"), "text": "a", "border": "double"}},
		"unknown fill":   {"create_text_label", map[string]any{"path": filepath.Join(dir, "x.tm2"), "text": "a", "fill": "black"}},
		"no labels":      {"create_pour_labels", map[string]any{"path": filepath.Join(dir, "x.tm2"), "labels": []string{}}},
		"bad pour path":  {"create_pour_labels", map[string]any{"path": "x.tm2", "labels": []string{"a"}}},
		"relative read":  {"inspect_label", map[string]any{"path": "x.tm2"}},
		"missing file":   {"inspect_label", map[string]any{"path": filepath.Join(dir, "none.tm2")}},
		"missing labels": {"create_pour_labels", map[string]any{"path": filepath.Join(dir, "x.tm2")}},
	}
	for name, tt := range tests {
		res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: tt.tool, Arguments: tt.args})
		if err == nil && !res.IsError {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestMCP_reportsEveryError(t *testing.T) {
	// A bad name, which the tool reads, and a bad value, which the package
	// checks, come back together.
	session := mcpConnect(t)
	res := mcpCallResult(t, session, "create_pour_labels", map[string]any{
		"path":   filepath.Join(t.TempDir(), "x.tm2"),
		"labels": []string{"123"},
		"code":   "jan13",
		"rotate": "45",
	})
	text := mcpResultText(res)
	if !res.IsError || !strings.Contains(text, "rotation \"45\"") || !strings.Contains(text, "value 1:") {
		t.Errorf("result %s, want both errors", text)
	}
}

func TestMCP_inspectLabel(t *testing.T) {
	session := mcpConnect(t)
	path, err := filepath.Abs(filepath.Join("..", "..", "testdata", "pour-18mm.tm2"))
	if err != nil {
		t.Fatal(err)
	}
	res := mcpCallResult(t, session, "inspect_label", map[string]any{"path": path})
	if res.IsError || !strings.Contains(mcpResultText(res), "rows:    10, column \"品名\"") {
		t.Errorf("result %s", mcpResultText(res))
	}
}

// mcpConnect returns a client session of a server of newMCPServer.
func mcpConnect(t *testing.T) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	serverT, clientT := mcp.NewInMemoryTransports()
	ss, err := newMCPServer("test").Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil).Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cs.Close() // the test is over; the server side reports any failure
		_ = ss.Wait()  // returns once the client has closed
	})
	return cs
}

// mcpCall calls tool, which must succeed, and returns its structured output.
func mcpCall(t *testing.T, session *mcp.ClientSession, tool string, args map[string]any) mcpLabelOutput {
	t.Helper()
	res := mcpCallResult(t, session, tool, args)
	if res.IsError {
		t.Fatalf("%s: %s", tool, mcpResultText(res))
	}
	b, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var out mcpLabelOutput
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// mcpCallResult calls tool and returns its result, an error result included.
func mcpCallResult(t *testing.T, session *mcp.ClientSession, tool string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// mcpResultText joins the text of res's content.
func mcpResultText(res *mcp.CallToolResult) string {
	var parts []string
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			parts = append(parts, tc.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// inspect_label on a file with rows but no column titles answers, and the
// server lives on (#4).
func TestMCP_inspectRowsWithoutTitles(t *testing.T) {
	session := mcpConnect(t)
	path := filepath.Join(t.TempDir(), "x.tm2")
	if err := os.WriteFile(path, []byte(`{"documents":[{"SettingsTapeWidth":18,"drawers":[]}],"currentPage":0,"tableData":[["a"]]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		res := mcpCallResult(t, session, "inspect_label", map[string]any{"path": path})
		if res.IsError || !strings.Contains(mcpResultText(res), "rows:    1") {
			t.Errorf("result %s", mcpResultText(res))
		}
	}
}

// A path that is absolute but names no .tm2 file says so (#21).
func TestMCP_pathNotTM2(t *testing.T) {
	session := mcpConnect(t)
	dir := t.TempDir()
	for _, path := range []string{filepath.Join(dir, "x.txt"), filepath.Join(dir, ".tm2"), filepath.Join(dir, ".TM2")} {
		res := mcpCallResult(t, session, "create_text_label", map[string]any{"path": path, "text": "a"})
		if text := mcpResultText(res); !res.IsError || !strings.Contains(text, "must name a .tm2 file") || strings.Contains(text, "absolute") {
			t.Errorf("%s: %s", path, text)
		}
	}
}

// MCP takes .TM2 in upper case, as the CLI does (#37).
func TestMCP_upperCaseTM2(t *testing.T) {
	session := mcpConnect(t)
	path := filepath.Join(t.TempDir(), "M.TM2")
	out := mcpCall(t, session, "create_text_label", map[string]any{"path": path, "text": "a"})
	f, err := tm2.ReadFile(out.Path)
	if err != nil {
		t.Fatal(err)
	}
	if f.FileName != "M.tm2" {
		t.Errorf("fileName %q, want M.tm2", f.FileName)
	}
}
