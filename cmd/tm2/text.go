package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mpyw/tepra.tm2"
)

// textCommand is "tm2 text".
//
//declscope:shared // main.go makes it a subcommand
func textCommand() *cli.Command {
	return &cli.Command{
		Name:      "text",
		Usage:     "make one label; a line break in the text breaks it into lines",
		ArgsUsage: "text",
		Description: "Give the text as one argument, line breaks and all:\n" +
			"  tm2 text -o out.tm2 $'line 1\\nline 2'    (bash, zsh)\n" +
			"  tm2 text -o out.tm2 \"line 1`nline 2\"    (PowerShell)",
		Flags:  labelFlags(),
		Action: textAction,
	}
}

// textAction runs "tm2 text".
func textAction(_ context.Context, cmd *cli.Command) error {
	if cmd.NArg() == 0 {
		return errors.New("text needs the text of the label")
	}
	if cmd.NArg() > 1 {
		// Arguments as lines would read "a b" as two lines, unlike the
		// line breaks of pages and pour.
		return fmt.Errorf("text takes one argument, not %d; put line breaks in it, as $'line 1\\nline 2'", cmd.NArg())
	}
	tape, opts, err := labelOptions(cmd)
	f, err := labelBuild(tape, err, func() (*tm2.File, error) {
		return tm2.NewText(tape, cmd.Args().First(), labelTextOptions(opts)...)
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
