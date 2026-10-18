package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"image"
	_ "image/png" // inspectJSON reads the size of a picture
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/mpyw/tepra.tm2"
)

// inspectCommand is "tm2 inspect".
//
//declscope:shared // main.go makes it a subcommand
func inspectCommand() *cli.Command {
	return &cli.Command{
		Name:      "inspect",
		Usage:     "print what a .tm2 file holds",
		ArgsUsage: "file.tm2",
		Description: "--json prints every key of the file, at the paths \"tm2 edit\" takes.\n" +
			"A picture is shown by its size.",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "json", Usage: "print every key of the file as JSON"},
		},
		Action: inspectAction,
	}
}

// inspectAction runs "tm2 inspect".
func inspectAction(_ context.Context, cmd *cli.Command) error {
	if cmd.NArg() != 1 {
		return errors.New("inspect takes one file")
	}
	f, err := tm2.ReadFile(cmd.Args().First())
	if err != nil {
		return err
	}
	text := inspectFile(f)
	if cmd.Bool("json") {
		if text, _, err = inspectJSON(f); err != nil {
			return err
		}
	}
	_, err = io.WriteString(cmd.Writer, text)
	return err
}

// inspectFile describes what f holds: its tape, its drawers in mm, each by
// its index in the file's drawers, and the rows it pours.
//
//declscope:shared // mcp.go answers inspect_label with it
func inspectFile(f *tm2.File) string {
	mm := tm2.PointsToMM
	lines := []string{fmt.Sprintf("file:    %s (version %s, saved %s)", f.FileName, f.Version, f.Date)}
	for i, doc := range f.Documents {
		if len(f.Documents) > 1 {
			lines = append(lines, fmt.Sprintf("page %d:", i+1))
		}
		mode := "plain"
		if doc.PourMode {
			mode = "pour"
		}
		lines = append(lines,
			fmt.Sprintf("tape:    %d mm, %s, label %.1f mm long, margin %.1f mm", doc.TapeWidthMM, mode, doc.LengthMM(), mm(doc.Margin)),
			fmt.Sprintf("drawers: /documents/%d/drawers", i))
		for j, d := range doc.Drawers {
			fr := d.Base().Frame
			lines = append(lines, fmt.Sprintf("  %2d %-6s x %5.1f  y %5.1f  w %5.1f  h %5.1f mm  %s",
				j, d.DrawerType(), mm(fr[0]), mm(fr[1]), mm(fr[2]), mm(fr[3]), inspectDrawer(d)))
		}
	}
	if len(f.TableData) > 0 {
		// A file need not name its columns.
		var column string
		if len(f.TableHeader) > 0 {
			column = f.TableHeader[0]
		}
		lines = append(lines, fmt.Sprintf("rows:    %d, column %q", len(f.TableData), column))
		for i, row := range f.TableData {
			// A value may be split into a column per line.
			var cells []string
			for _, c := range row {
				if c != "" {
					cells = append(cells, strings.ReplaceAll(c, "\n", " / "))
				}
			}
			lines = append(lines, fmt.Sprintf("  %3d  %s", i+1, strings.Join(cells, " / ")))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

// inspectDrawer describes what is particular to d, and each setting that
// shows in the app and is not as a new drawer has it.
func inspectDrawer(d tm2.Drawer) string {
	var parts []string
	switch d := d.(type) {
	case *tm2.PourDrawer:
		parts = append(parts, fmt.Sprintf("column %d, size %g, %s", d.ColumnNumber, d.FontSize, d.FontName))
		if d.Attribute != tm2.CodeText {
			parts = append(parts, "code "+d.Attribute.String())
		}
		parts = append(parts, inspectText(&d.TextDrawer)...)
	case *tm2.TextDrawer:
		parts = append(parts, fmt.Sprintf("size %g, %s, %q", d.FontSize, d.FontName, d.Text))
		parts = append(parts, inspectText(d)...)
	case *tm2.ImageDrawer:
		parts = append(parts, fmt.Sprintf("%d bytes, threshold %d", len(d.Image), d.ImageThreshold))
		if d.ImageMode != tm2.ImageModeThreshold {
			parts = append(parts, "mode "+d.ImageMode.String())
		}
	default:
		return "(not known to tm2)"
	}
	b := d.Base()
	if b.Rotate != tm2.RotationNone {
		parts = append(parts, "rotate "+b.Rotate.String())
	}
	if b.BIType != tm2.BorderNone {
		parts = append(parts, fmt.Sprintf("border %s %g", b.BIType, b.BILineWidthValue))
	}
	if b.TapeForegroundColor != tm2.Black {
		parts = append(parts, fmt.Sprintf("color %v", b.TapeForegroundColor))
	}
	return strings.Join(parts, ", ")
}

// inspectText lists the settings of a text box that show in the app and are
// not as a new one has them.
func inspectText(d *tm2.TextDrawer) []string {
	var parts []string
	if d.Alignment != tm2.AlignLeft {
		parts = append(parts, "align "+d.Alignment.String())
	}
	if d.VerticalAlignment != tm2.VerticalAlignMiddle {
		parts = append(parts, "valign "+d.VerticalAlignment.String())
	}
	if d.FillMode != tm2.FillNone {
		parts = append(parts, "fill "+d.FillMode.String())
	}
	if d.TextVertical {
		parts = append(parts, "vertical")
	}
	if d.SizePriority {
		parts = append(parts, "keeps its size")
	} else if !d.HoldAspect {
		parts = append(parts, "fixed (固)")
	}
	if d.HasPlaceholder {
		parts = append(parts, "placeholder")
	}
	return parts
}

// inspectImagePath matches the path of a drawer's picture in the file.
var inspectImagePath = regexp.MustCompile(`^/documents/\d+/drawers/\d+/image$`)

// inspectJSON is every key of f, as WriteTo writes it, indented. It
// returns each picture besides, as a PNG, and shows it in the JSON by its
// size and its place in the pictures returned, from 1.
//
//declscope:shared // mcp.go answers inspect_label with it, when full
func inspectJSON(f *tm2.File) (string, [][]byte, error) {
	b, err := json.Marshal(f)
	if err != nil {
		return "", nil, err
	}
	var pictures [][]byte
	return inspectIndent(b, "", "", &pictures) + "\n", pictures, nil
}

// inspectIndent returns v, valid JSON at path in the file, a member or an
// item a line, each line after the first beginning with indent. An array
// of values that are neither objects nor arrays, such as a frame or a row
// of tableData, takes one line. A PNG picture is added to pictures, and
// shown by its size.
func inspectIndent(v jsontext.Value, indent, path string, pictures *[][]byte) string {
	kind := v.Kind()
	if kind == '"' && inspectImagePath.MatchString(path) {
		var data string
		_ = json.Unmarshal(v, &data) // a string of valid JSON reads
		png, err := base64.StdEncoding.DecodeString(data)
		if cfg, format, cfgErr := image.DecodeConfig(bytes.NewReader(png)); err == nil && cfgErr == nil && format == "png" {
			*pictures = append(*pictures, png)
			q, _ := json.Marshal(fmt.Sprintf("(picture %d: PNG, %d by %d pixels, %d bytes)", len(*pictures), cfg.Width, cfg.Height, len(png))) // a string encodes
			return string(q)
		}
	}
	if kind != '{' && kind != '[' {
		return string(v)
	}
	dec := jsontext.NewDecoder(bytes.NewReader(v))
	_, _ = dec.ReadToken() // v is valid, so it reads
	var items []string
	flat := kind == '['
	for i := 0; dec.PeekKind() != jsontext.KindEndObject && dec.PeekKind() != jsontext.KindEndArray; i++ {
		var name, at string
		if kind == '{' {
			tok, _ := dec.ReadToken()          // v is valid, so it reads
			q, _ := json.Marshal(tok.String()) // a string encodes
			name, at = string(q)+": ", tok.String()
		} else {
			at = strconv.Itoa(i)
		}
		item, _ := dec.ReadValue() // v is valid, so it reads
		if k := item.Kind(); k == '{' || k == '[' {
			flat = false
		}
		items = append(items, name+inspectIndent(item, indent+"  ", path+"/"+at, pictures))
	}
	open, end := "{", "}"
	if kind == '[' {
		open, end = "[", "]"
	}
	switch {
	case len(items) == 0:
		return open + end
	case flat:
		return open + strings.Join(items, ", ") + end
	default:
		inner := indent + "  "
		return open + "\n" + inner + strings.Join(items, ",\n"+inner) + "\n" + indent + end
	}
}
