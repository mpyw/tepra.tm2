package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/urfave/cli/v3"

	"github.com/mpyw/tepra.tm2"
)

// pourCommand is "tm2 pour".
//
//declscope:shared // main.go makes it a subcommand
func pourCommand() *cli.Command {
	return &cli.Command{
		Name:      "pour",
		Usage:     "make one label per row of a CSV file of one column, or of stdin",
		ArgsUsage: "[file.csv]",
		Description: "Each row is a label; blank lines are skipped. A value in double quotes may hold\n" +
			"line breaks, which break the label into lines, and commas.\n" +
			"The labels share one frame, sized for the longest. Open the file in TEPRA LINK 2,\n" +
			"where the labels show in its pour (流し込み) form.",
		Flags: append(labelFlags(),
			&cli.StringFlag{
				Name:  "code",
				Usage: "show each value as a code: text (default), qr, code39, code128 or jan13 (12 digits)",
			},
			&cli.StringFlag{
				Name:  "column",
				Value: "Label",
				Usage: "the `TITLE` of the column in the app's pour data",
			},
		),
		Action: pourAction,
	}
}

// pourAction runs "tm2 pour".
func pourAction(_ context.Context, cmd *cli.Command) error {
	if cmd.NArg() > 1 {
		return errors.New("pour takes one file at most")
	}
	values, err := readPourFile(cmd)
	if err != nil {
		return err
	}
	tape, opts, err := labelOptions(cmd)
	pour, codeErr := labelPourOptions(opts, cmd.String("code"))
	f, err := labelBuild(tape, errors.Join(err, codeErr), func() (*tm2.File, error) {
		return tm2.NewPour(tape, cmd.String("column"), values, pour...)
	})
	if err != nil {
		return err
	}
	out := cmd.String("output")
	if err := writeLabel(out, f, cmd.Bool("force"), "--force"); err != nil {
		return err
	}
	reportLabel(cmd.Writer, out, f)
	return nil
}

// readPourFile reads the values of the file cmd names, or of its stdin.
//
//declscope:shared // pages.go reads its file with it
func readPourFile(cmd *cli.Command) ([]string, error) {
	path := cmd.Args().First()
	if path == "" || path == "-" {
		return readPourValues(cmd.Reader)
	}
	fh, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	defer func() { _ = fh.Close() }() // read only: Close reports nothing lost
	return readPourValues(fh)
}

// readPourValues reads a label from each row of r, a CSV file of one column,
// as the app's own pour import reads one. A value in double quotes may hold
// line breaks. Blank rows are skipped, spaces around a value are dropped,
// and so is a UTF-8 byte order mark, which spreadsheets write. Rows may end
// in "\r\n", "\n" or a lone "\r". The file must be UTF-8.
func readPourValues(r io.Reader) ([]string, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	b = bytes.TrimPrefix(b, []byte(pourBOM))
	if !utf8.Valid(b) {
		// Excel saves CSV in Shift_JIS by default in Japan.
		return nil, fmt.Errorf("%w: save the file as UTF-8 (\"CSV UTF-8\" in Excel)", tm2.ErrBadText)
	}
	// encoding/csv ends rows at "\n" only.
	b = bytes.ReplaceAll(bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n")), []byte("\r"), []byte("\n"))
	// Spaces around a quoted value are taken, as around any other. The CSV
	// stays strict otherwise: with LazyQuotes, a quote left open would take
	// every row after it into one value, and nobody would notice.
	cr := csv.NewReader(bytes.NewReader(pourTrimRowEnds(b)))
	cr.TrimLeadingSpace = true
	cr.FieldsPerRecord = 1
	var values []string
	for {
		row, err := cr.Read()
		if errors.Is(err, io.EOF) {
			return values, nil
		}
		if errors.Is(err, csv.ErrFieldCount) {
			return nil, fmt.Errorf("%w: the file takes one column; put a value with a comma in double quotes", err)
		}
		if err != nil {
			return nil, err
		}
		if v := strings.TrimSpace(row[0]); v != "" {
			values = append(values, v)
		}
	}
}

// pourTrimRowEnds drops the blanks at the end of each row of b, so
// encoding/csv takes blanks after a closing quote. A line that ends inside
// quotes ends inside a value, whose blanks stay. A quote doubled in a value
// turns the state twice, so counting quotes is enough.
func pourTrimRowEnds(b []byte) []byte {
	lines := bytes.Split(b, []byte("\n"))
	quoted := false
	for i, line := range lines {
		if bytes.Count(line, []byte(`"`))%2 == 1 {
			quoted = !quoted
		}
		if !quoted {
			lines[i] = bytes.TrimRightFunc(line, unicode.IsSpace)
		}
	}
	return bytes.Join(lines, []byte("\n"))
}

// pourBOM is the UTF-8 byte order mark.
const pourBOM = "\ufeff"
