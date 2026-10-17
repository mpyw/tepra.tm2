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
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err := newMainCommand().Run(ctx, os.Args)
	stop()
	os.Exit(mainExitCode(os.Stderr, err))
}

// newMainCommand is the tepra command and its subcommands.
func newMainCommand() *cli.Command {
	return &cli.Command{
		Name:    "tepra",
		Usage:   "make .tm2 label files for TEPRA LINK 2",
		Version: mainVersion(),
		// main prints errors and exits; urfave/cli must not do it as well.
		ExitErrHandler: func(context.Context, *cli.Command, error) {},
		Commands: []*cli.Command{
			pourCommand(),
			pagesCommand(),
			textCommand(),
			inspectCommand(),
			mcpCommand(),
		},
	}
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
		// Errors of package tepra say "tepra: " already, even when wrapped.
		msg, code = strings.ReplaceAll(err.Error(), "tepra: ", ""), 1
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
