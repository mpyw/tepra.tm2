package main

import (
	"context"
	"encoding/base64"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/mpyw/tepra.tm2"
)

// editCommand is "tm2 edit".
//
//declscope:shared // main.go makes it a subcommand
func editCommand() *cli.Command {
	return &cli.Command{
		Name:      "edit",
		Usage:     "change a .tm2 file by a JSON Patch, of a file or of stdin",
		ArgsUsage: "file.tm2 [patch.json]",
		Description: "The patch is a JSON array of operations (RFC 6902) on the file as\n" +
			"\"tm2 inspect --json\" prints it, such as\n" +
			"  [{\"op\": \"replace\", \"path\": \"/documents/0/drawers/0/TIText\", \"value\": \"A\"}]\n" +
			"\"image_path\" in an operation gives a picture: the value of a path ending in\n" +
			"/image, or the image of an image drawer added.\n" +
			"Keys are not added or removed, since the app writes every key. A drawer added\n" +
			"starts as the app makes one of its type. Without -o the file is changed in place.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "the .tm2 `FILE` to write; the file read when left out",
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "replace the output file when it exists",
			},
		},
		Action: editAction,
	}
}

// editAction runs "tm2 edit".
func editAction(_ context.Context, cmd *cli.Command) error {
	if cmd.NArg() == 0 || cmd.NArg() > 2 {
		return errors.New("edit takes the .tm2 file, and the patch unless it is on stdin")
	}
	path := cmd.Args().First()
	var patch []byte
	var err error
	if cmd.NArg() == 2 {
		patch, err = os.ReadFile(filepath.Clean(cmd.Args().Get(1)))
	} else {
		patch, err = io.ReadAll(cmd.Reader)
	}
	if err != nil {
		return err
	}
	var ops []editOp
	if err := json.Unmarshal(patch, &ops); err != nil {
		return fmt.Errorf("the patch is not a JSON array of operations: %w", err)
	}
	out := cmd.String("output")
	if out == "" {
		out = path
	}
	if !labelIsTM2(out) {
		return fmt.Errorf("%s does not name a .tm2 file", out)
	}
	f, err := editFile(path, ops, labelReadPicture)
	if err != nil {
		return err
	}
	if err := writeLabel(out, f, cmd.Bool("force") || editInPlace(path, out), "--force"); err != nil {
		return err
	}
	reportLabel(cmd.Writer, out, f)
	return nil
}

// editOp is an operation of a JSON Patch, as "tm2 edit" and edit_label take
// it.
//
//declscope:shared // mcp.go takes it in edit_label
type editOp struct {
	Op        string `json:"op" jsonschema:"add, remove, replace, move, copy or test"`
	Path      string `json:"path" jsonschema:"JSON Pointer to the value, such as /documents/0/drawers/1/TIText; /documents/0/drawers/- adds a drawer last, on top"`
	From      string `json:"from,omitempty" jsonschema:"JSON Pointer to the value that move and copy take"`
	Value     any    `json:"value,omitempty" jsonschema:"the value that add, replace and test take"`
	ImagePath string `json:"image_path,omitempty" jsonschema:"path of a PNG, JPEG or GIF picture: the value of a path ending in /image, or the image of an image drawer added"`
}

// editFile reads the .tm2 file at path, and applies ops to it. readPicture
// reads the picture of an ImagePath.
//
//declscope:shared // mcp.go edits a file with it
func editFile(path string, ops []editOp, readPicture func(string) ([]byte, error)) (*tm2.File, error) {
	f, err := tm2.ReadFile(path)
	if err != nil {
		return nil, err
	}
	patch := make([]tm2.PatchOp, len(ops))
	for i, op := range ops {
		if patch[i], err = op.editPatchOp(readPicture); err != nil {
			return nil, fmt.Errorf("operation %d (%s %s): %w", i+1, op.Op, op.Path, err)
		}
	}
	if err := f.Patch(patch...); err != nil {
		return nil, err
	}
	return f, nil
}

// editInPlace reports whether out is the file read, at path, which an edit
// replaces without being told to.
//
//declscope:shared // mcp.go writes an edit with it
func editInPlace(path, out string) bool {
	return filepath.Clean(path) == filepath.Clean(out)
}

// editPatchOp converts op for package tm2.
func (op editOp) editPatchOp(readPicture func(string) ([]byte, error)) (tm2.PatchOp, error) {
	value := op.Value
	if op.ImagePath != "" {
		pic, err := readPicture(op.ImagePath)
		if err != nil {
			return tm2.PatchOp{}, err
		}
		enc := base64.StdEncoding.EncodeToString(pic)
		switch v := value.(type) {
		case map[string]any:
			v["image"] = enc
		case nil:
			if !strings.HasSuffix(op.Path, "/image") {
				return tm2.PatchOp{}, errors.New("image_path is for a path ending in /image, or a value that is an image drawer")
			}
			value = enc
		default:
			return tm2.PatchOp{}, errors.New("image_path is for a path ending in /image, or a value that is an image drawer")
		}
	}
	p := tm2.PatchOp{Op: op.Op, Path: op.Path, From: op.From}
	if value != nil {
		p.Value, _ = json.Marshal(value) // a value read from JSON, or a string added, encodes
	}
	return p, nil
}
