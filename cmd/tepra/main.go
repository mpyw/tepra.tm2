// Command tepra makes .tm2 label files for TEPRA LINK 2.
//
//	tepra pour -o out.tm2 [file]     one label per line of file, or of stdin
//	tepra pages -o out.tm2 [file]    a page per line, each fitted to its line
//	tepra text -o out.tm2 text       one label; a line break in text breaks it
//	tepra inspect file.tm2           what a .tm2 file holds
//	tepra mcp                        an MCP server on stdin and stdout
//
// Open the file in TEPRA LINK 2 to print it.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/mpyw/tepra.tm2"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err := newMainCommand().Run(ctx, os.Args)
	stop()
	os.Exit(mainExitCode(os.Stderr, err))
}

// newMainCommand is the tepra command and its subcommands.
func newMainCommand() *cli.Command {
	cmd := &cli.Command{
		Name:    "tepra",
		Usage:   "make .tm2 label files for TEPRA LINK 2",
		Version: mainVersion(),
		// main prints errors and exits; urfave/cli must not do it as well.
		ExitErrHandler: func(context.Context, *cli.Command, error) {},
		// With no subcommand, the help. With one tepra does not have, an
		// error: urfave/cli would look for a help topic, and exit 3.
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.NArg() > 0 {
				return fmt.Errorf("unknown command %q; see tepra --help", cmd.Args().First())
			}
			return cli.ShowAppHelp(cmd)
		},
		Commands: []*cli.Command{
			pourCommand(),
			pagesCommand(),
			textCommand(),
			inspectCommand(),
			mcpCommand(),
		},
	}
	// urfave/cli prints a usage error and the help itself, before returning
	// it. main prints errors, so each command returns it only. Subcommands
	// do not inherit the handler. The help commands urfave/cli adds as it
	// runs would not have it, so there are none: -h and --help remain.
	quiet := func(_ context.Context, _ *cli.Command, err error, _ bool) error { return err }
	for _, c := range append([]*cli.Command{cmd}, cmd.Commands...) {
		c.OnUsageError = quiet
		c.HideHelpCommand = true
	}
	return cmd
}

// mainSentinels are the errors of package tepra. Each says "tepra: ",
// which main prints at the start of each line already.
var mainSentinels = []error{
	tepra.ErrTapeNotSupported, tepra.ErrUnknownName, tepra.ErrInvalidOptions,
	tepra.ErrCrashesApp, tepra.ErrBadFile, tepra.ErrBadText, tepra.ErrNoText,
	tepra.ErrOverLimit, tepra.ErrNoRoom, tepra.ErrBadImage, tepra.ErrBadCode,
}

// mainMessage is err's message without the "tepra: " of the package's
// errors. A "tepra: " in a value the user gave stays.
func mainMessage(err error) string {
	msg := err.Error()
	for _, s := range mainSentinels {
		msg = strings.ReplaceAll(msg, s.Error(), strings.TrimPrefix(s.Error(), "tepra: "))
	}
	return msg
}

// mainExitCode reports err on w, and returns the exit status for it.
func mainExitCode(w io.Writer, err error) int {
	var msg string
	code := 0
	exit, isExit := errors.AsType[cli.ExitCoder](err)
	switch {
	case isExit:
		msg, code = strings.TrimSpace(err.Error()), exit.ExitCode()
	case err != nil:
		msg, code = mainMessage(err), 1
	}
	if msg == "" {
		return code
	}
	// Errors joined together come a line each.
	for line := range strings.Lines(msg) {
		// A failed write to stderr has nowhere to be reported.
		_, _ = fmt.Fprintf(w, "tepra: %s\n", strings.TrimSuffix(line, "\n"))
	}
	return code
}

// version is the release, set by goreleaser with -ldflags "-X
// main.version=...".
//
//declscope:ignore qualify // goreleaser sets it by this name
var version string

// mainVersion is the release, or the module version go install built, with
// the Go release and platform.
func mainVersion() string {
	v := version
	if v == "" {
		v = "(devel)"
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
			v = info.Main.Version
		}
	} else if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return v + " (" + runtime.Version() + " " + runtime.GOOS + "/" + runtime.GOARCH + ")"
}
