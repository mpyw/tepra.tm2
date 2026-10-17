package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/mpyw/tepra.tm2"
)

// pourCommand is "tepra pour".
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

// pourAction runs "tepra pour".
func pourAction(_ context.Context, cmd *cli.Command) error {
	if cmd.NArg() > 1 {
		return errors.New("pour takes one file at most")
	}
	values, err := readPourFile(cmd)
	if err != nil {
		return err
	}
	// Each bad flag is reported at once, the code's too.
	tape, opts, err := labelOptions(cmd)
	pour, codeErr := labelPourOptions(opts, cmd.String("code"))
	if err := errors.Join(err, codeErr); err != nil {
		return err
	}
	f, err := tepra.NewPour(tape, cmd.String("column"), values, pour...)
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
// and so is a UTF-8 byte order mark, which spreadsheets write.
func readPourValues(r io.Reader) ([]string, error) {
	br := bufio.NewReader(r)
	if bom, err := br.Peek(len(pourBOM)); err == nil && string(bom) == pourBOM {
		_, _ = br.Discard(len(pourBOM)) // the bytes are buffered, so it cannot fail
	}
	cr := csv.NewReader(br)
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

// pourBOM is the UTF-8 byte order mark.
const pourBOM = "\ufeff"
