package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/urfave/cli/v3"

	"github.com/mpyw/tepra.tm2"
)

// mcpCommand is "tm2 mcp".
//
//declscope:shared // main.go makes it a subcommand
func mcpCommand() *cli.Command {
	return &cli.Command{
		Name:  "mcp",
		Usage: "serve the label commands as MCP tools on stdin and stdout",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return newMCPServer(cmd.Root().Version).Run(ctx, &mcp.StdioTransport{})
		},
	}
}

// mcpInstructions tell a client what the tools are for.
const mcpInstructions = "Make, read and change .tm2 label files for KING JIM's TEPRA LINK 2 app. " +
	"The tools write a file; they do not print. " +
	"To change a file, read it with inspect_label and full, then edit it with edit_label. " +
	"The user opens the file in TEPRA LINK 2 (on a phone, through the OS's share sheet) and prints from there. " +
	`Use "\n" in a label for a line break. ` +
	"For two or more labels, use create_label_pages; use create_pour_labels only for QR codes and barcodes."

// errMCPRelativePath is a path that is not absolute. The server's working
// directory is not the client's, so a relative path would point elsewhere.
var errMCPRelativePath = errors.New("path must be absolute")

// errMCPNotTM2 is a path to write that does not name a .tm2 file.
var errMCPNotTM2 = errors.New("path must name a .tm2 file, such as label.tm2")

// mcpLabelStyle are the inputs every tool that makes a label takes.
type mcpLabelStyle struct {
	Path           string   `json:"path" jsonschema:"absolute path of the .tm2 file to write, ending in .tm2"`
	Overwrite      bool     `json:"overwrite,omitempty" jsonschema:"replace the file at path when it exists"`
	TapeWidthMM    int      `json:"tape_width_mm,omitempty" jsonschema:"tape width in mm: 4, 6, 9, 12 (when left out), 18, 24, 36 or 50"`
	FontSize       *float64 `json:"font_size,omitempty" jsonschema:"font size in points; left out, the text fills the tape"`
	Bold           bool     `json:"bold,omitempty" jsonschema:"write the text in bold"`
	Align          string   `json:"align,omitempty" jsonschema:"where each line goes across: left (the default), center or right"`
	VerticalAlign  string   `json:"valign,omitempty" jsonschema:"where the text goes down: top, middle (the default) or bottom"`
	Rotate         string   `json:"rotate,omitempty" jsonschema:"turn the text clockwise, in degrees: 0 (the default), 90, 180 or 270"`
	Vertical       bool     `json:"vertical,omitempty" jsonschema:"write the text top to bottom"`
	Fill           string   `json:"fill,omitempty" jsonschema:"box behind the text: none (the default), inverted (white text on black) or shaded (black text on a light dotted box)"`
	Border         string   `json:"border,omitempty" jsonschema:"line around the label: none (the default), box, rounded, top-bottom, top or bottom"`
	BorderWidth    *float64 `json:"border_width,omitempty" jsonschema:"width of the border; 0.7 when left out, 0.35 is thin and 2.8 is thick"`
	IconPath       string   `json:"icon_path,omitempty" jsonschema:"absolute path of a PNG, JPEG or GIF picture to put before the text, as high as the tape prints"`
	BackgroundPath string   `json:"background_path,omitempty" jsonschema:"absolute path of a PNG, JPEG or GIF picture to stretch under the whole label, such as a frame"`
	ImageMode      string   `json:"image_mode,omitempty" jsonschema:"how the pictures turn black and white: threshold (the default), diffusion or screen"`
	ImageThreshold *int     `json:"image_threshold,omitempty" jsonschema:"grey level, 0 to 255, below which a pixel of a picture is black; 128 when left out"`
	PaddingMM      *float64 `json:"padding_mm,omitempty" jsonschema:"blank between the label's edges and its text and icon, in mm; room for a border or the frame of a background; 1.5 when left out with a border"`
	MarginMM       *float64 `json:"margin_mm,omitempty" jsonschema:"blank tape before and after the label, in mm, 1 to 500; 1 when left out"`
	LengthMM       *float64 `json:"length_mm,omitempty" jsonschema:"make the label this long, in mm, 10 to 3000; the text is centred and shrunk to fit"`
}

// mcpPourInput is the input of create_pour_labels.
type mcpPourInput struct {
	mcpLabelStyle

	Labels []string `json:"labels" jsonschema:"one string per label, in print order"`
	Column string   `json:"column,omitempty" jsonschema:"title of the column in the app's pour data; Label when left out"`
	Code   string   `json:"code,omitempty" jsonschema:"show each label as a code: text (the default), qr, code39, code128 or jan13 (12 digits; the app adds the check digit)"`
}

// mcpPagesInput is the input of create_label_pages.
type mcpPagesInput struct {
	mcpLabelStyle

	Labels []string `json:"labels" jsonschema:"one string per label, in print order"`
}

// mcpTextInput is the input of create_text_label.
type mcpTextInput struct {
	mcpLabelStyle

	Text string `json:"text" jsonschema:"the text of the label"`
}

// mcpInspectInput is the input of inspect_label.
type mcpInspectInput struct {
	Path string `json:"path" jsonschema:"absolute path of the .tm2 file to read"`
	Full bool   `json:"full,omitempty" jsonschema:"also return every key of the file as JSON, at the paths edit_label takes, and each picture"`
}

// mcpEditInput is the input of edit_label.
type mcpEditInput struct {
	Path       string   `json:"path" jsonschema:"absolute path of the .tm2 file to change"`
	Operations []editOp `json:"operations" jsonschema:"a JSON Patch (RFC 6902) on the file as inspect_label with full shows it, applied in order; all apply, or nothing is written"`
	Output     string   `json:"output,omitempty" jsonschema:"absolute path of the .tm2 file to write, ending in .tm2; the file at path when left out. The label takes its name in the app, unless an operation sets /fileName"`
	Overwrite  bool     `json:"overwrite,omitempty" jsonschema:"replace the file at output when it exists; the file at path is changed in place without it"`
}

// mcpLabelOutput is what the tools that make a label return.
type mcpLabelOutput struct {
	Path     string  `json:"path" jsonschema:"the file written"`
	Labels   int     `json:"labels" jsonschema:"how many labels the file prints"`
	LengthMM float64 `json:"length_mm" jsonschema:"length of tape each label takes, in mm; of the longest label, for pages"`
	FontSize float64 `json:"font_size" jsonschema:"font size of the text; of the smallest text, for pages"`
	Message  string  `json:"message" jsonschema:"what was written, and how to print it"`
}

// newMCPServer returns the MCP server of "tm2 mcp".
func newMCPServer(version string) *mcp.Server {
	s := mcp.NewServer(
		&mcp.Implementation{Name: "tm2", Title: "TEPRA LINK 2 labels", Version: version},
		&mcp.ServerOptions{Instructions: mcpInstructions},
	)
	mcp.AddTool(s, &mcp.Tool{
		Name: "create_pour_labels",
		Description: "Write a .tm2 file that prints one label per string, through the app's pour (流し込み) form. " +
			"Use it for QR codes and barcodes; for other lists, use create_label_pages. " +
			"The labels share one frame, sized for the longest. The app shows the text at the left of that frame, whatever align says, " +
			"and inverted fill only behind the letters.",
	}, mcpPour)
	mcp.AddTool(s, &mcp.Tool{
		Name: "create_label_pages",
		Description: "Write a .tm2 file that prints one label per string, each on a page of its own. " +
			"Each label gets the font size and length that fit it, as create_text_label makes it. " +
			"Use it for any list of two or more text labels. With length_mm, every label is as long. The app prints the pages together.",
	}, mcpPages)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_text_label",
		Description: "Write a .tm2 file that prints one label of text.",
	}, mcpText)
	mcp.AddTool(s, &mcp.Tool{
		Name: "inspect_label",
		Description: "Describe what a .tm2 file holds: its tape, its objects (drawers) and the rows it pours. " +
			"With full, it also returns every key of the file as JSON, and each picture.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, mcpInspect)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "edit_label",
		Description: mcpEditDescription,
	}, mcpEdit)
	return s
}

// mcpEditDescription tells a client how to edit a file, and what the
// numbers of the file mean.
const mcpEditDescription = "Change a .tm2 file by a JSON Patch (RFC 6902) on the file as inspect_label with full shows it. " +
	"Any key can be replaced, and items of arrays (documents, drawers, rows of tableData) added, removed, moved or copied. " +
	"Keys are not added to an object or removed from one, since the app writes every key. " +
	"A whole document or drawer put in place needs every key it has; replace the values of its keys instead. " +
	`A drawer added to /documents/N/drawers needs only its "type" (text, pour or image) and the keys to change; ` +
	"it starts as the app makes one, in the whole label's frame. A copy gets a new identifier. " +
	"A text box shows its text once it has some (hasPlaceholder is cleared). " +
	"Nothing is laid out again: set frame (x, y, width, height in points, 1 mm = 2.835 points) yourself. " +
	"A text box fits its text to its frame unless sizePriority is set. " +
	"A border is an empty text drawer of its own, around the text: add one with BIType set. " +
	"BIType on a pour drawer shows no border, and on a text drawer with text it draws a line that touches the text. " +
	"Values: TIAlignment 0 left, 1 center, 2 right; TIVerticalAlignment 0 top, 1 middle, 2 bottom; " +
	"TIFillMode 0 none, 1 inverted, 2 shaded; Rotate 0 none, 1 180°, 2 270°, 3 90°; " +
	"BIType 0 none, 1 box, 2 rounded, 3 top-bottom, 4 top, 5 bottom; " +
	"Attribute (pour) 0 text, 1 QR, 2 CODE39, 3 CODE128, 4 JAN-13; imageMode 1 diffusion, 2 threshold, 3 screen; " +
	"TIFontName OTBGoStdR-Normal (regular) or OTBGoStdB-Normal (bold). " +
	"A pour drawer shows column ColumnNumber of each row of tableData, a label per row."

// mcpPour handles create_pour_labels.
func mcpPour(_ context.Context, _ *mcp.CallToolRequest, in mcpPourInput) (*mcp.CallToolResult, mcpLabelOutput, error) {
	tape, opts, err := in.mcpOptions()
	pour, codeErr := labelPourOptions(opts, in.Code)
	column := in.Column
	if column == "" {
		column = "Label"
	}
	f, err := labelBuild(tape, errors.Join(err, codeErr), func() (*tm2.File, error) {
		return tm2.NewPour(tape, column, in.Labels, pour...)
	})
	if err != nil {
		return nil, mcpLabelOutput{}, err
	}
	out, err := in.mcpWrite(f)
	return nil, out, err
}

// mcpPages handles create_label_pages.
func mcpPages(_ context.Context, _ *mcp.CallToolRequest, in mcpPagesInput) (*mcp.CallToolResult, mcpLabelOutput, error) {
	tape, opts, err := in.mcpOptions()
	f, err := labelBuild(tape, err, func() (*tm2.File, error) {
		return tm2.NewPages(tape, in.Labels, labelTextOptions(opts)...)
	})
	if err != nil {
		return nil, mcpLabelOutput{}, err
	}
	out, err := in.mcpWrite(f)
	return nil, out, err
}

// mcpText handles create_text_label.
func mcpText(_ context.Context, _ *mcp.CallToolRequest, in mcpTextInput) (*mcp.CallToolResult, mcpLabelOutput, error) {
	tape, opts, err := in.mcpOptions()
	f, err := labelBuild(tape, err, func() (*tm2.File, error) {
		return tm2.NewText(tape, in.Text, labelTextOptions(opts)...)
	})
	if err != nil {
		return nil, mcpLabelOutput{}, err
	}
	out, err := in.mcpWrite(f)
	return nil, out, err
}

// mcpInspect handles inspect_label.
func mcpInspect(_ context.Context, _ *mcp.CallToolRequest, in mcpInspectInput) (*mcp.CallToolResult, any, error) {
	if !filepath.IsAbs(in.Path) {
		return nil, nil, fmt.Errorf("%w: %s", errMCPRelativePath, in.Path)
	}
	f, err := tm2.ReadFile(in.Path)
	if err != nil {
		return nil, nil, err
	}
	content := []mcp.Content{&mcp.TextContent{Text: inspectFile(f)}}
	if !in.Full {
		return &mcp.CallToolResult{Content: content}, nil, nil
	}
	text, pictures, err := inspectJSON(f)
	if err != nil {
		return nil, nil, err
	}
	content = append(content, &mcp.TextContent{Text: text})
	for _, png := range pictures {
		content = append(content, &mcp.ImageContent{Data: png, MIMEType: "image/png"})
	}
	return &mcp.CallToolResult{Content: content}, nil, nil
}

// mcpEdit handles edit_label.
func mcpEdit(_ context.Context, _ *mcp.CallToolRequest, in mcpEditInput) (*mcp.CallToolResult, any, error) {
	out := in.Output
	if out == "" {
		out = in.Path
	}
	for _, path := range []string{in.Path, out} {
		if !filepath.IsAbs(path) {
			return nil, nil, fmt.Errorf("%w: %s", errMCPRelativePath, path)
		}
	}
	if !labelIsTM2(out) {
		return nil, nil, fmt.Errorf("%w: %s", errMCPNotTM2, out)
	}
	f, err := editFile(in.Path, out, in.Operations, mcpReadPicture)
	if err != nil {
		return nil, nil, err
	}
	if err := writeLabel(out, f, in.Overwrite || editInPlace(in.Path, out), "overwrite: true"); err != nil {
		return nil, nil, err
	}
	text := describeLabel(out, f) + ". Open it in TEPRA LINK 2 to print.\n" + inspectFile(f)
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, nil, nil
}

// mcpOptions checks s and converts it for package tm2.
func (s mcpLabelStyle) mcpOptions() (tm2.Tape, []tm2.Option, error) {
	if !filepath.IsAbs(s.Path) {
		return tm2.Tape{}, nil, fmt.Errorf("%w: %s", errMCPRelativePath, s.Path)
	}
	if !labelIsTM2(s.Path) {
		return tm2.Tape{}, nil, fmt.Errorf("%w: %s", errMCPNotTM2, s.Path)
	}
	width := s.TapeWidthMM
	if width == 0 {
		width = tm2.Tape12mm.WidthMM
	}
	tape, err := tm2.LookupTape(width)
	if err != nil {
		return tm2.Tape{}, nil, err
	}
	settings := labelSettings{
		name:           labelName(s.Path),
		fill:           s.Fill,
		align:          s.Align,
		valign:         s.VerticalAlign,
		rotate:         s.Rotate,
		border:         s.Border,
		imageMode:      s.ImageMode,
		bold:           s.Bold,
		vertical:       s.Vertical,
		fontSize:       s.FontSize,
		borderWidth:    s.BorderWidth,
		padding:        s.PaddingMM,
		margin:         s.MarginMM,
		length:         s.LengthMM,
		imageThreshold: s.ImageThreshold,
	}
	if settings.icon, err = mcpReadPicture(s.IconPath); err != nil {
		return tm2.Tape{}, nil, err
	}
	if settings.background, err = mcpReadPicture(s.BackgroundPath); err != nil {
		return tm2.Tape{}, nil, err
	}
	opts, err := settings.labelOptions()
	return tape, opts, err
}

// mcpReadPicture reads the picture at path, which must be absolute. An
// empty path is no picture.
func mcpReadPicture(path string) ([]byte, error) {
	if path == "" {
		return nil, nil
	}
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("%w: %s", errMCPRelativePath, path)
	}
	return labelReadPicture(path)
}

// mcpWrite writes f to s.Path and reports what it wrote. The error, if any,
// is in the second result.
func (s mcpLabelStyle) mcpWrite(f *tm2.File) (mcpLabelOutput, error) {
	if err := writeLabel(s.Path, f, s.Overwrite, "overwrite: true"); err != nil {
		return mcpLabelOutput{}, err
	}
	_, long := labelLengths(f)
	out := mcpLabelOutput{
		Path:     s.Path,
		Labels:   len(f.Documents),
		LengthMM: long,
		Message:  describeLabel(s.Path, f) + ". Open it in TEPRA LINK 2 to print.",
	}
	for i, doc := range f.Documents {
		// NewPour, NewPages and NewText put the text last, after the
		// pictures.
		var size float64
		switch d := doc.Drawers[len(doc.Drawers)-1].(type) {
		case *tm2.PourDrawer:
			out.Labels = len(f.TableData)
			size = d.FontSize
		case *tm2.TextDrawer:
			size = d.FontSize
		}
		if i == 0 || size < out.FontSize {
			out.FontSize = size
		}
	}
	return out, nil
}
