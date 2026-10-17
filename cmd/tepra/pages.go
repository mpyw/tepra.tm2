package main

import (
	"context"
	"errors"

	"github.com/urfave/cli/v3"

	"github.com/mpyw/tepra.tm2"
)

// pagesCommand is "tepra pages".
//
//declscope:shared // main.go makes it a subcommand
func pagesCommand() *cli.Command {
	return &cli.Command{
		Name:      "pages",
		Usage:     "make one page per row of a CSV file of one column, or of stdin, each fitted to its row",
		ArgsUsage: "[file.csv]",
		Description: "The file reads as for pour. Each row is a label of its own, with its own\n" +
			"font size and length, as text makes it. TEPRA LINK 2 lists the pages,\n" +
			"and prints them together.",
		Flags:  labelFlags(),
		Action: pagesAction,
	}
}

// pagesAction runs "tepra pages".
func pagesAction(_ context.Context, cmd *cli.Command) error {
	if cmd.NArg() > 1 {
		return errors.New("pages takes one file at most")
	}
	values, err := readPourFile(cmd)
	if err != nil {
		return err
	}
	tape, opts, err := labelOptions(cmd)
	if err != nil {
		return err
	}
	f, err := tepra.NewPages(tape, values, labelTextOptions(opts)...)
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
