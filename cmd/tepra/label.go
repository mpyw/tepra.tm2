package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/mpyw/tepra.tm2"
)

// labelFlags are the flags of every command that makes a label.
//
//declscope:shared // pour.go, pages.go and text.go take them
func labelFlags() []cli.Flag {
	// The flags of a number say their default in Usage, where it is not 0.
	// HideDefault keeps urfave/cli from adding "(default: 0)" after it.
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "output",
			Aliases:  []string{"o"},
			Usage:    "the .tm2 `FILE` to write",
			Required: true,
		},
		&cli.BoolFlag{
			Name:    "force",
			Aliases: []string{"f"},
			Usage:   "replace the output file when it exists",
		},
		&cli.IntFlag{
			Name:  "tape",
			Value: tepra.Tape12mm.WidthMM,
			Usage: "the tape width in `MM`: 4, 6, 9, 12, 18, 24, 36 or 50",
		},
		&cli.FloatFlag{Name: "font-size", HideDefault: true, Usage: "the font `SIZE`; by default the text fills the tape"},
		&cli.BoolFlag{Name: "bold", Usage: "write the text in bold"},
		&cli.StringFlag{Name: "align", Usage: "where each line goes across: left (default), center or right"},
		&cli.StringFlag{Name: "valign", Usage: "where the text goes down: top, middle (default) or bottom"},
		&cli.StringFlag{Name: "rotate", Usage: "turn the text clockwise: 0 (default), 90, 180 or 270 `DEGREES`"},
		&cli.BoolFlag{Name: "vertical", Usage: "write the text top to bottom"},
		&cli.StringFlag{Name: "fill", Usage: "the box behind the text: none (default), inverted (white on black) or shaded"},
		&cli.StringFlag{Name: "border", Usage: "the line around the label: none (default), box, rounded, top-bottom, top or bottom"},
		&cli.FloatFlag{Name: "border-width", HideDefault: true, Usage: "the `WIDTH` of the border: 0.7 by default, 0.35 thin, 2.8 thick"},
		&cli.StringFlag{Name: "icon", Usage: "a PNG, JPEG or GIF `FILE` to put before the text"},
		&cli.StringFlag{Name: "background", Usage: "a PNG, JPEG or GIF `FILE` to stretch under the whole label, such as a frame"},
		&cli.StringFlag{Name: "image-mode", Usage: "how pictures turn black and white: threshold (default), diffusion or screen"},
		&cli.IntFlag{Name: "image-threshold", HideDefault: true, Usage: "the grey `LEVEL`, 0 to 255, below which a pixel is black: 128 by default"},
		&cli.FloatFlag{Name: "padding", HideDefault: true, Usage: "the blank between the label's edges and its text, in `MM`: 1.5 by default with a border"},
		&cli.FloatFlag{Name: "margin", HideDefault: true, Usage: "the blank tape before and after the label, in `MM`, 1 to 500: 1 by default"},
		&cli.FloatFlag{Name: "length", HideDefault: true, Usage: "make the label this long, in `MM`, 10 to 3000; the text is centred"},
	}
}

// labelName is the name of the label in the app, of the file at path: its
// base name without ".tm2", in any case.
//
//declscope:shared // mcp.go names its labels with it
func labelName(path string) string {
	name := filepath.Base(path)
	if ext := filepath.Ext(name); strings.EqualFold(ext, ".tm2") {
		name = strings.TrimSuffix(name, ext)
	}
	return name
}

// labelIsTM2 reports whether path names a .tm2 file, in any case, with a
// name before ".tm2".
//
//declscope:shared // mcp.go and the commands check the file they write
func labelIsTM2(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".tm2") && labelName(path) != ""
}

// labelSettings are the settings of a label as the command line or MCP give
// them: "" or nil for what is not given, so the label is as the app sets it.
//
//declscope:shared // mcp.go reads the inputs of its tools into it
type labelSettings struct {
	name                                           string
	fill, align, valign, rotate, border, imageMode string
	bold, vertical                                 bool
	fontSize, borderWidth                          *float64
	padding, margin, length                        *float64
	imageThreshold                                 *int
	icon, background                               []byte
}

// labelOptions are the options of s. A name it does not know is an error.
//
//declscope:shared // mcp.go turns the inputs of its tools into options with it
func (s labelSettings) labelOptions() ([]tepra.Option, error) {
	opts := []tepra.Option{tepra.WithName(s.name)}
	add := func(opt tepra.Option) { opts = append(opts, opt) }
	if s.bold {
		add(tepra.WithBold())
	}
	if s.vertical {
		add(tepra.WithVertical())
	}
	if s.fontSize != nil {
		add(tepra.WithFontSize(*s.fontSize))
	}
	if s.borderWidth != nil {
		add(tepra.WithBorderWidth(*s.borderWidth))
	}
	if s.padding != nil {
		add(tepra.WithPaddingMM(*s.padding))
	}
	if s.margin != nil {
		add(tepra.WithMarginMM(*s.margin))
	}
	if s.length != nil {
		add(tepra.WithLengthMM(*s.length))
	}
	if s.imageThreshold != nil {
		add(tepra.WithImageThreshold(*s.imageThreshold))
	}
	if s.icon != nil {
		add(tepra.WithIcon(s.icon))
	}
	if s.background != nil {
		add(tepra.WithBackground(s.background))
	}
	var errs []error
	parse := func(name string, with func(string) (tepra.Option, error)) {
		if name == "" {
			return
		}
		opt, err := with(name)
		if err != nil {
			errs = append(errs, err)
			return
		}
		add(opt)
	}
	parse(s.fill, labelParse(tepra.ParseFillMode, tepra.WithFill))
	parse(s.align, labelParse(tepra.ParseAlign, tepra.WithAlign))
	parse(s.valign, labelParse(tepra.ParseVerticalAlign, tepra.WithVerticalAlign))
	parse(s.rotate, labelParse(tepra.ParseRotation, tepra.WithRotation))
	parse(s.border, labelParse(tepra.ParseBorderType, tepra.WithBorder))
	parse(s.imageMode, labelParse(tepra.ParseImageMode, tepra.WithImageMode))
	return opts, errors.Join(errs...)
}

// labelParse makes an option of a name, parsed by parse and set by with.
func labelParse[T, O any](parse func(string) (T, error), with func(T) O) func(string) (O, error) {
	return func(name string) (O, error) {
		v, err := parse(name)
		if err != nil {
			var none O
			return none, err
		}
		return with(v), nil
	}
}

// labelTextOptions are opts, as NewText and NewPages take them.
//
//declscope:shared // pages.go, text.go and mcp.go pass their options on with it
func labelTextOptions(opts []tepra.Option) []tepra.TextOption {
	text := make([]tepra.TextOption, len(opts))
	for i, opt := range opts {
		text[i] = opt
	}
	return text
}

// labelPourOptions are opts as NewPour takes them, with the kind of code
// named code. An empty code is text.
//
//declscope:shared // pour.go and mcp.go pass their options on with it
func labelPourOptions(opts []tepra.Option, code string) ([]tepra.PourOption, error) {
	pour := make([]tepra.PourOption, len(opts), len(opts)+1)
	for i, opt := range opts {
		pour[i] = opt
	}
	if code == "" {
		return pour, nil
	}
	opt, err := labelParse(tepra.ParseCodeKind, tepra.WithCode)(code)
	if err != nil {
		return nil, err
	}
	return append(pour, opt), nil
}

// labelOptions reads the flags of labelFlags. The file's name in the app is
// the output's, without ".tm2".
//
//declscope:shared // pour.go, pages.go and text.go read their flags with it
func labelOptions(cmd *cli.Command) (tepra.Tape, []tepra.Option, error) {
	if out := cmd.String("output"); !labelIsTM2(out) {
		return tepra.Tape{}, nil, fmt.Errorf("--output must name a .tm2 file, such as label.tm2: %q", out)
	}
	tape, err := tepra.LookupTape(cmd.Int("tape"))
	if err != nil {
		return tepra.Tape{}, nil, err
	}
	s := labelSettings{
		name:      labelName(cmd.String("output")),
		fill:      cmd.String("fill"),
		align:     cmd.String("align"),
		valign:    cmd.String("valign"),
		rotate:    cmd.String("rotate"),
		border:    cmd.String("border"),
		imageMode: cmd.String("image-mode"),
		bold:      cmd.Bool("bold"),
		vertical:  cmd.Bool("vertical"),
	}
	float := func(name string) *float64 {
		if !cmd.IsSet(name) {
			return nil
		}
		v := cmd.Float(name)
		return &v
	}
	s.fontSize, s.borderWidth = float("font-size"), float("border-width")
	s.padding, s.margin, s.length = float("padding"), float("margin"), float("length")
	if cmd.IsSet("image-threshold") {
		v := int(cmd.Int("image-threshold"))
		s.imageThreshold = &v
	}
	if s.icon, err = labelReadPicture(cmd.String("icon")); err != nil {
		return tepra.Tape{}, nil, err
	}
	if s.background, err = labelReadPicture(cmd.String("background")); err != nil {
		return tepra.Tape{}, nil, err
	}
	opts, err := s.labelOptions()
	return tape, opts, err
}

// labelBuild makes a file with build. err is what reading the options
// found: build still runs, so the errors of the values come with it, all
// at once. It does not run without a tape, which a bad tape width or an
// unreadable picture leaves.
//
//declscope:shared // the commands and the MCP tools make their files with it
func labelBuild(tape tepra.Tape, err error, build func() (*tepra.File, error)) (*tepra.File, error) {
	if tape.WidthMM == 0 {
		return nil, err
	}
	f, buildErr := build()
	if err := errors.Join(err, buildErr); err != nil {
		return nil, err
	}
	return f, nil
}

// labelReadPicture reads the picture at path. An empty path is no picture.
//
//declscope:shared // mcp.go reads the pictures of its tools with it
func labelReadPicture(path string) ([]byte, error) {
	if path == "" {
		return nil, nil
	}
	return os.ReadFile(filepath.Clean(path))
}

// writeLabel writes f to path. It does not replace a file there unless
// overwrite is set; hint says how to set it. The file is encoded first, so
// a file that cannot be encoded writes nothing, and leaves a file there as
// it was.
//
// A replaced file is written in place, as other tools write: a symlink
// writes its target, and the file keeps its mode and its hard links. A
// temporary file renamed over it would lose all three, and needs a folder
// it can write to.
//
//declscope:shared // the commands and the MCP tools write their files with it
func writeLabel(path string, f *tepra.File, overwrite bool, hint string) error {
	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		return err
	}
	flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	if !overwrite {
		flags |= os.O_EXCL
	}
	fh, err := os.OpenFile(filepath.Clean(path), flags, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("%w: %s; replace it with %s", fs.ErrExist, path, hint)
	}
	if err != nil {
		return err
	}
	if _, err := fh.Write(buf.Bytes()); err != nil {
		_ = fh.Close() // the write error is the one to report
		return err
	}
	return fh.Close()
}

// reportLabel writes what describeLabel says to w. The file is written by
// then, so a lost line changes nothing, and its error is dropped.
//
//declscope:shared // pour.go, pages.go and text.go report what they wrote with it
func reportLabel(w io.Writer, path string, f *tepra.File) {
	_, _ = fmt.Fprintln(w, describeLabel(path, f))
}

// describeLabel is one line about f, for after it is written.
//
//declscope:shared // mcp.go reports what its tools wrote with it
func describeLabel(path string, f *tepra.File) string {
	doc := f.Documents[0]
	labels := fmt.Sprintf("1 label, %.1f mm long", doc.LengthMM())
	if n := len(f.TableData); doc.PourMode && n != 1 {
		labels = fmt.Sprintf("%d labels, %.1f mm each", n, doc.LengthMM())
	}
	if n := len(f.Documents); n > 1 {
		short, long := labelLengths(f)
		labels = fmt.Sprintf("%d pages, %.1f to %.1f mm long", n, short, long)
		if fmt.Sprintf("%.1f", short) == fmt.Sprintf("%.1f", long) {
			labels = fmt.Sprintf("%d pages, %.1f mm each", n, long)
		}
	}
	return fmt.Sprintf("wrote %s: %s, on %d mm tape", path, labels, doc.TapeWidthMM)
}

// labelLengths are the shortest and the longest length of the pages of
// f, in mm.
//
//declscope:shared // mcp.go reports the longest with it
func labelLengths(f *tepra.File) (short, long float64) {
	short = f.Documents[0].LengthMM()
	for _, doc := range f.Documents {
		short, long = min(short, doc.LengthMM()), max(long, doc.LengthMM())
	}
	return short, long
}
