package main

import (
	"context"
	"errors"
	"fmt"
	"io"
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
		Action:    inspectAction,
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
	_, err = io.WriteString(cmd.Writer, inspectFile(f))
	return err
}

// inspectFile describes what f holds: its tape, its drawers in mm, and the
// rows it pours.
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
			"drawers:")
		for _, d := range doc.Drawers {
			fr := d.Base().Frame
			lines = append(lines, fmt.Sprintf("  %-6s x %5.1f  y %5.1f  w %5.1f  h %5.1f mm  %s",
				d.DrawerType(), mm(fr[0]), mm(fr[1]), mm(fr[2]), mm(fr[3]), inspectDrawer(d)))
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

// inspectDrawer describes what is particular to d.
func inspectDrawer(d tm2.Drawer) string {
	switch d := d.(type) {
	case *tm2.PourDrawer:
		return fmt.Sprintf("column %d, size %g, %s", d.ColumnNumber, d.FontSize, d.FontName)
	case *tm2.TextDrawer:
		return fmt.Sprintf("size %g, %s, %q", d.FontSize, d.FontName, d.Text)
	case *tm2.ImageDrawer:
		return fmt.Sprintf("%d bytes, threshold %d", len(d.Image), d.ImageThreshold)
	default:
		return "(not known to tm2)"
	}
}
