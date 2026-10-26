<div align="center">

<img src=".github/assets/icon.png" width="200" alt="A Go gopher holding a strip of label tape">

# tepra.tm2

Make label files for KING JIM's TEPRA LINK 2 app.

[![Test](https://github.com/mpyw/tepra.tm2/actions/workflows/test.yml/badge.svg)](https://github.com/mpyw/tepra.tm2/actions/workflows/test.yml)
[![Coverage](https://codecov.io/gh/mpyw/tepra.tm2/graph/badge.svg)](https://codecov.io/gh/mpyw/tepra.tm2)
[![Go Reference](https://pkg.go.dev/badge/github.com/mpyw/tepra.tm2.svg)](https://pkg.go.dev/github.com/mpyw/tepra.tm2)
[![M8ven Score](https://m8ven.ai/badge/mcp/mpyw-tepra-tm2-1oawlc?v=f27303c8668b6535627a69b0e025b5c7)](https://m8ven.ai/mcp/mpyw-tepra-tm2-1oawlc?s=readme)

</div>

TEPRA LINK 2 saves labels as `.tm2` files. tepra.tm2 writes them.
Type a list once, and get one label per line.
Then print from the app, as usual.

It works from an AI assistant over MCP, from the command line, and from Go.

> [!IMPORTANT]
> `.tm2` is the format of TEPRA LINK 2 only.
> TEPRA Creator (テプラ クリエイター), the app for Windows, does not open these files.
> tepra.tm2 does not write files for TEPRA Creator.

> [!WARNING]
> This is an unofficial tool. KING JIM does not document the `.tm2` format.
>
> - tepra.tm2 writes every key that files from the app have.
> - Each setting was checked on the screen of TEPRA LINK 2 for Android, with an SR-MK1.
> - A new release of the app may change the format.

## Installation

### Tool

| Way | How |
| --- | --- |
| [mise](https://mise.jdx.dev/) (recommended) | `mise use -g github:mpyw/tepra.tm2` |
| A release binary | Download the archive for your OS and CPU from [Releases](https://github.com/mpyw/tepra.tm2/releases). Put `tm2` on your `PATH`. `checksums.txt` has their SHA-256. |
| From source | `go install github.com/mpyw/tepra.tm2/cmd/tm2@latest` |

Releases cover macOS, Linux and Windows, each on amd64 and arm64.

### MCP Server

The MCP server is `tm2 mcp`.

| Assistant | How |
| --- | --- |
| Claude Desktop, on macOS or Windows | Download [`tm2.mcpb`](https://github.com/mpyw/tepra.tm2/releases/latest/download/tm2.mcpb) and open it. Claude Desktop asks you to install it. It needs nothing else installed |
| Claude Code | First install `tm2` as a [tool](#tool), as above. Then run `claude mcp add tm2 -- tm2 mcp` |
| Another assistant | First install `tm2` as a [tool](#tool). Add a server whose command is the absolute path of `tm2`, with the argument `mcp` |

> [!IMPORTANT]
> `tm2.mcpb` adds the tools to the chat of Claude Desktop only.
> The Code tab of Claude Desktop runs Claude Code, and its sessions do not get them.

> [!TIP]
> Claude Desktop does not read the `PATH` of your shell.
> If you set up `tm2` by hand in `claude_desktop_config.json`, give the absolute path of `tm2` as the command.
>
> ```json
> {
>   "mcpServers": {
>     "tm2": { "command": "/absolute/path/to/tm2", "args": ["mcp"] }
>   }
> }
> ```

### Library

```sh
go get github.com/mpyw/tepra.tm2
```

The package is named `tm2`.

## Usage: As an MCP Server

Ask for labels in your own words.
The assistant writes a `.tm2` file with the [MCP tools](#mcp-tools).
It can also read a `.tm2` file and change it, one the app saved too.
Then [print the file](#print-the-file) from TEPRA LINK 2.

## Usage: As a Tool

One label of two lines:

```sh
tm2 text -o sample.tm2 $'サンプル\nテキスト'
```

```text
wrote sample.tm2: 1 label, 18.2 mm long, on 12 mm tape
```

> [!TIP]
> `$'...'` turns `\n` into a line break in bash and zsh.
> A line break inside quotes works too. In PowerShell, write `` "サンプル`nテキスト" ``.

A label per row of a [CSV file](#the-csv-file), each in the size that fits it:

```sh
tm2 pages -o pages.tm2 items.csv
```

```text
wrote pages.tm2: 3 pages, 12.0 to 22.5 mm long, on 12 mm tape
```

Then [print the file](#print-the-file) from TEPRA LINK 2.

## Usage: As a Library

```go
f, err := tm2.NewPages(
	tm2.Tape12mm,
	[]string{"1行目\n2行目", "1行目"},
	tm2.WithAlign(tm2.AlignCenter),
)
if err != nil {
	return err
}
return f.WriteFile("items.tm2")
```

## References

- All: [Kinds of label](#kinds-of-label), [Tapes](#tapes), [Layout](#layout), [Print the file](#print-the-file)
- MCP: [MCP tools](#mcp-tools)
- Tool: [Commands](#commands), [The CSV file](#the-csv-file), [Tool options](#tool-options), [tm2 inspect](#tm2-inspect), [tm2 edit](#tm2-edit)
- All: [Patches](#patches)
- Library: [Functions](#functions), [Library options](#library-options), [Option types](#option-types), [Errors](#errors), [Reading files](#reading-files)

### Kinds of label

| | text | pages | pour (流し込み) |
| --- | --- | --- | --- |
| Labels in a file | One | One per page | One per value |
| Font size and length | Fitted to the label | Fitted to each label | Same for every label, fitted to the longest |
| Edit in the app | As one label | Page by page | As pour data, in a table |
| QR codes and barcodes | No | No | Yes |
| MCP tool | `create_text_label` | `create_label_pages` | `create_pour_labels` |
| Command | `tm2 text` | `tm2 pages` | `tm2 pour` |
| Go | [`NewText`][NewText] | [`NewPages`][NewPages] | [`NewPour`][NewPour] |

> [!TIP]
> For a list of things to label, use pages. Each label gets the size that fits it.
> Give a length, and every label is as long.
> Use pour only for QR codes and barcodes, or to edit the labels as one table in the app.
> Pour lays text out less well: see [Layout](#layout).

### Tapes

| Width | Go |
| --- | --- |
| 4, 6, 9, 12, 18, 24, 36 and 50 mm | [`Tape4mm`][Tape4mm] to [`Tape50mm`][Tape50mm], or [`LookupTape(width)`][LookupTape] |

The sizes of each tape come from a file the app wrote on it.

### Layout

Size and length:

- Text fills the height of the tape, unless a font size is given.
- A label is as long as what is on it, and at least 12 mm with its margins. The app makes no shorter label.
- A label of a fixed length has its content centred. Text too long for it gets smaller, unless a font size is given.
- All the labels of a pour file have one font size, the one that fits the value of most lines.

What you see when you edit the label in the app:

- A border is an empty box of its own, around the text.
  A border set on a pour drawer itself (`BIType`) does not show.
- Inverted text of more than one line is a box per line. In a pour file, each line is a column of its own: `Label`, `Label 2` and on.

What pour cannot do:

- The text sits at the left of its frame. The app ignores the alignment of a pour drawer.
- Inverted text is black only behind the letters, not over the whole frame.
- The text keeps its font size, and its width is an estimate. Bold text and wide letters, such as `w` and `m`, can run past the frame.

Pages with a length have none of these limits.

> [!NOTE]
> On 4 mm tape, a border's padding of 1.5 mm leaves no room for text.
> Give a padding of 0.3 mm there.

### Print the file

1. Send the `.tm2` file to your phone, such as through Google Drive.
2. Open it with TEPRA LINK 2, through the share menu of the OS.
3. Check the labels in the app, and print.

### MCP tools

`tm2 mcp` serves these on stdin and stdout:

| Tool | Kind | What it does |
| --- | --- | --- |
| `create_text_label` | text | Writes a file of one label |
| `create_label_pages` | pages | Writes a file of one page per string |
| `create_pour_labels` | pour | Writes a pour file: one label per string |
| `inspect_label` | | Describes a `.tm2` file. With `full`, it also returns every key as JSON, and each picture |
| `edit_label` | | Changes a `.tm2` file by a [patch](#patches) |

Inputs:

- The tools that write a file take the [tool options](#tool-options), in snake case.
- A length in mm ends in `_mm`, and a picture in `_path`. Examples: `font_size`, `rotate`, `padding_mm`, `icon_path`.
- `code` and `column` are for `create_pour_labels` only.
- Paths must be absolute, for the file and for pictures.
- A file is not replaced unless `overwrite` is true.
  `edit_label` changes the file at `path` in place without it. It writes to `output` instead when given.
  The label then takes the name of `output` in the app, unless the patch sets `/fileName`.
- A line break in a label is `\n`.

### Commands

| Command | What it does |
| --- | --- |
| `tm2 text [options] text` | Writes one label. A line break in the text breaks it into lines |
| `tm2 pages [options] [file.csv]` | Writes a page per row of the file |
| `tm2 pour [options] [file.csv]` | Writes a pour label per row of the file |
| `tm2 inspect [--json] file.tm2` | Shows what a `.tm2` file holds |
| `tm2 edit [-o out.tm2] [-f] file.tm2 [patch.json]` | Changes a `.tm2` file by a [patch](#patches) |
| `tm2 mcp` | Serves the [MCP tools](#mcp-tools) on stdin and stdout |

A pour file of the same `items.csv`:

```sh
tm2 pour -o items.tm2 items.csv
```

```text
wrote items.tm2: 3 labels, 12.0 mm each, on 12 mm tape
```

A QR code per URL, one label each:

```sh
tm2 pour --code qr -o qr.tm2 urls.csv
```

```text
wrote qr.tm2: 2 labels, 12.2 mm each, on 12 mm tape
```

A label of a fixed length, with a rounded border:

```sh
tm2 text --length 60 --border rounded --align center -o fixed.tm2 サンプル
```

```text
wrote fixed.tm2: 1 label, 60.0 mm long, on 12 mm tape
```

Text turned 90 degrees, and text in white on black:

```sh
tm2 text --rotate 90 -o turned.tm2 $'サンプル\nテキスト'
tm2 text --fill inverted --bold -o inverted.tm2 サンプル
```

```text
wrote turned.tm2: 1 label, 12.0 mm long, on 12 mm tape
wrote inverted.tm2: 1 label, 34.5 mm long, on 12 mm tape
```

### The CSV file

`pages` and `pour` read a CSV file of one column.
Without a file, or with `-`, they read stdin.

- Each row is a label.
- A line break in a value breaks the label into lines.
- A value in double quotes may hold line breaks and commas.
- A double quote in a value is doubled, inside double quotes: `"He said ""hi"""`.
- Blank rows are skipped. So are spaces around a value.
- A row of two or more columns is an error.
- The file must be UTF-8. In Excel, save it as "CSV UTF-8".
- Rows may end in CRLF, LF or CR.

This `items.csv` makes three labels:

```csv
"1行目
2行目
3行目"
"1行目
2行目"
1行目
```

> [!TIP]
> A file with one value per line is a CSV file of one column already.
> Quote only the values that hold a line break, a comma or a double quote.
> Spreadsheets quote such values when they save a CSV file.

### Tool options

`text`, `pages` and `pour` all take these.
In `pages` and `pour` they apply to every label.

| Option | Default | Meaning |
| --- | --- | --- |
| `-o`, `--output` | (required) | The `.tm2` file to write. Its name, without `.tm2`, names the label in the app |
| `-f`, `--force` | off | Replace the file when it exists |
| `--tape` | `12` | Tape width in mm: `4`, `6`, `9`, `12`, `18`, `24`, `36` or `50` |
| `--font-size` | fit | Font size in points. By default the text fills the tape |
| `--bold` | off | Bold text |
| `--align` | `left` | Where each line goes across: `left`, `center` or `right` |
| `--valign` | `middle` | Where the text goes down: `top`, `middle` or `bottom` |
| `--rotate` | `0` | Turn the text clockwise: `0`, `90`, `180` or `270` degrees |
| `--vertical` | off | Write the text top to bottom |
| `--fill` | `none` | The box behind the text: `none`, `inverted` (white on black) or `shaded` (a light dotted box) |
| `--border` | `none` | The line around the label: `none`, `box`, `rounded`, `top-bottom`, `top` or `bottom` |
| `--border-width` | `0.7` | The width of the border. `0.35` is thin, `2.8` is thick |
| `--icon` | none | A PNG, JPEG or GIF file to put before the text, as high as the tape prints |
| `--background` | none | A PNG, JPEG or GIF file to stretch under the whole label, such as a frame |
| `--image-mode` | `threshold` | How pictures turn black and white: `threshold`, `diffusion` or `screen` |
| `--image-threshold` | `128` | The grey level, `0` to `255`, below which a pixel is black |
| `--padding` | `0` | Blank between the label's edges and its text, in mm. `1.5` with a border |
| `--margin` | `1` | Blank tape before and after the label, in mm, `1` to `500` |
| `--length` | fit | Make the label this long, margins included, in mm, `10` to `3000` |

Only `pour` takes these:

| Option | Default | Meaning |
| --- | --- | --- |
| `--code` | `text` | Show each value as a code: `text`, `qr`, `code39`, `code128` or `jan13` |
| `--column` | `Label` | Title of the pour column in the app |

> [!NOTE]
> `--code jan13` takes 12 digits. The app adds the check digit.

### tm2 inspect

`tm2 inspect` shows what a `.tm2` file holds.
It reads files the app wrote, too.

```text
$ tm2 inspect items.tm2
file:    items.tm2 (version 2.0.0, saved 2026/10/18 10:15)
tape:    12 mm, pour, label 12.0 mm long, margin 1.0 mm
drawers: /documents/0/drawers
   0 pour   x   1.6  y   1.1  w   6.7  h  10.2 mm  column 0, size 7.5, OTBGoStdR-Normal, keeps its size
rows:    3, column "Label"
    1  1行目 / 2行目 / 3行目
    2  1行目 / 2行目
    3  1行目
```

A file of pages shows each page. Pages count from 1, as in the app:

```text
$ tm2 inspect pages.tm2
file:    pages.tm2 (version 2.0.0, saved 2026/10/18 10:15)
page 1:
tape:    12 mm, plain, label 12.0 mm long, margin 1.0 mm
drawers: /documents/0/drawers
   0 text   x   1.6  y   1.1  w   6.7  h  10.2 mm  size 7.5, OTBGoStdR-Normal, "1行目\n2行目\n3行目"
page 2:
tape:    12 mm, plain, label 12.2 mm long, margin 1.0 mm
drawers: /documents/1/drawers
   0 text   x   0.0  y   1.1  w  10.2  h  10.2 mm  size 11.5, OTBGoStdR-Normal, "1行目\n2行目"
page 3:
tape:    12 mm, plain, label 22.5 mm long, margin 1.0 mm
drawers: /documents/2/drawers
   0 text   x   0.0  y   1.1  w  20.5  h  10.2 mm  size 23, OTBGoStdR-Normal, "1行目"
```

- Each drawer starts with its index. `drawers:` gives the path of the list, for a [patch](#patches).
- After the size and the font come the settings that differ from a new drawer, such as `align center` or `border box 0.7`.
- `--json` prints every key of the file, at the paths a patch takes. A picture shows as its size.

### tm2 edit

`tm2 edit` changes a `.tm2` file by a [patch](#patches).
It reads the patch from a file, or from stdin without one.
Without `-o`, it changes the file in place.
With `-o`, the label takes the name of the new file in the app, as a label made there would.
A patch that sets `/fileName` keeps its own name.

> [!WARNING]
> An edit in place writes over the file, as `--force` does.
> A failed write, such as on a full disk, can leave it broken.
> Keep a copy of a file the app saved, or write to another with `-o`.

This `patch.json` changes the third row, and centres the text:

```json
[
  {"op": "replace", "path": "/tableData/2/0", "value": "4行目"},
  {"op": "replace", "path": "/documents/0/drawers/0/TIAlignment", "value": 1}
]
```

```sh
tm2 edit items.tm2 patch.json
```

```text
wrote items.tm2: 3 labels, 12.0 mm each, on 12 mm tape
```

### Patches

A patch is a JSON Patch ([RFC 6902](https://www.rfc-editor.org/rfc/rfc6902)).
It applies to the file as `tm2 inspect --json` prints it.
`tm2 edit`, `edit_label` and [`File.Patch`][File.Patch] take it.

| Op | What it does |
| --- | --- |
| `replace` | Sets the value of a key, or of an item of an array |
| `add` | Puts an item in an array. `-` as the index puts it last. On a key, it is `replace` |
| `remove` | Takes an item out of an array |
| `move`, `copy` | Moves or copies a value from `from` to `path` |
| `test` | Fails the patch unless the value at `path` is `value` |

What a patch keeps:

| Rule | Why |
| --- | --- |
| A key is not added or removed | The app writes every key. A new key would be one no file of the app has |
| A whole file, document or drawer put in place needs every key it has, and no other | A key left out would be written as 0, not as the app sets it. So `type` changes only with the keys of the new type |
| A drawer added starts as the app makes one of its `type`: `text`, `pour` or `image` | It needs only `type` and the keys to change. Its frame is the label's frame |
| A drawer of a type tm2 does not know may move, but not be copied or changed | It is written as it was read, so its identifier cannot change |
| The empty key after a drawer's `identifier` stays | The app writes it |
| A copy of a document or drawer, or one with no identifier, gets a new identifier | The app gives each one its own |
| `hasPlaceholder` is cleared once `TIText` has text | The app shows the placeholder instead of the text while it is set |
| With two or more pages, each gets `MultiplePage` | The app lists the pages only then |
| A row of `tableData` gets empty cells up to 50 | The app writes 50 columns |
| A picture is checked, and a JPEG or GIF is made PNG | The app stores pictures as PNG |
| A value the app crashes on fails the patch | Such a file is never written |

Either every operation applies, or nothing changes.

> [!IMPORTANT]
> A patch does not lay the label out again. Text made longer keeps its frame, and the app fits it there.
> Set `frame` too when the size must change. A frame is x, y, width and height in points. 1 mm is 2.835 points.

`image_path` in an operation gives a picture, from a PNG, JPEG or GIF file.
It is the value of a path that ends in `/image`, given with no `value`.
Or it is the image of a drawer of type `image`, given as the `value`:

```json
[{"op": "add", "path": "/documents/0/drawers/-", "value": {"type": "image", "frame": [0, 0, 20, 20]}, "image_path": "/path/to/logo.png"}]
```

The numbers of the keys that show in the app:

| Key | Values |
| --- | --- |
| `TIAlignment` | `0` left, `1` center, `2` right |
| `TIVerticalAlignment` | `0` top, `1` middle, `2` bottom |
| `TIFillMode` | `0` none, `1` inverted, `2` shaded |
| `Rotate` | `0` none, `1` 180°, `2` 270°, `3` 90° |
| `BIType` | `0` none, `1` box, `2` rounded, `3` top-bottom, `4` top, `5` bottom |
| `Attribute` (pour) | `0` text, `1` QR, `2` CODE39, `3` CODE128, `4` JAN-13 |
| `imageMode` | `1` diffusion, `2` threshold, `3` screen |
| `TIFontName` | `OTBGoStdR-Normal` regular, `OTBGoStdB-Normal` bold |

### Functions

| Function | Returns |
| --- | --- |
| [`NewText(tape, text, opts...)`][NewText] | A file of one label |
| [`NewPages(tape, texts, opts...)`][NewPages] | A file of a page per string, each fitted |
| [`NewPour(tape, column, values, opts...)`][NewPour] | A pour file of a label per string, under the column's title |
| [`ReadFile(path)`][ReadFile], [`ParseFile(r)`][ParseFile] | A file read back |

A [`File`][File] writes itself with [`WriteFile`][File.WriteFile] or [`WriteTo`][File.WriteTo].
[`Patch`][File.Patch] changes it by a [patch](#patches).

```go
f, err := tm2.ReadFile("items.tm2")
if err != nil {
	return err
}
err = f.Patch(tm2.PatchOp{
	Op:    "replace",
	Path:  "/tableData/2/0",
	Value: jsontext.Value(`"4行目"`),
})
if err != nil {
	return err
}
return f.WriteFile("items.tm2")
```

### Library options

Each `With` function sets one thing. What no option sets is as the app sets it.

| Option | Default | Meaning |
| --- | --- | --- |
| [`WithName(name)`][WithName] | `"label"` | Names the label in the app, without `.tm2` |
| [`WithTime(t)`][WithTime] | now | The file's date |
| [`WithFontSize(pt)`][WithFontSize] | fit | Font size in points. By default the text fills the tape |
| [`WithBold()`][WithBold] | off | Bold text |
| [`WithAlign(a)`][WithAlign] | `AlignLeft` | Where each line goes across: `AlignLeft`, `AlignCenter` or `AlignRight` |
| [`WithVerticalAlign(a)`][WithVerticalAlign] | `VerticalAlignMiddle` | Where the text goes down: `VerticalAlignTop`, `VerticalAlignMiddle` or `VerticalAlignBottom` |
| [`WithRotation(r)`][WithRotation] | `RotationNone` | Turn the text clockwise: `RotationNone`, `Rotation90`, `Rotation180` or `Rotation270` |
| [`WithVertical()`][WithVertical] | off | Write the text top to bottom |
| [`WithFill(m)`][WithFill] | `FillNone` | The box behind the text: `FillNone`, `FillInverted` or `FillShaded` |
| [`WithBorder(b)`][WithBorder] | `BorderNone` | The line around the label: `BorderNone`, `BorderBox`, `BorderRounded`, `BorderTopBottom`, `BorderTop` or `BorderBottom` |
| [`WithBorderWidth(w)`][WithBorderWidth] | `0.7` | The width of the border |
| [`WithIcon(picture)`][WithIcon] | none | PNG, JPEG or GIF bytes to put before the text |
| [`WithBackground(picture)`][WithBackground] | none | PNG, JPEG or GIF bytes to stretch under the whole label |
| [`WithImageMode(m)`][WithImageMode] | `ImageModeThreshold` | How pictures turn black and white: `ImageModeThreshold`, `ImageModeDiffusion` or `ImageModeScreen` |
| [`WithImageThreshold(level)`][WithImageThreshold] | `128` | The grey level, 0 to 255, below which a pixel is black |
| [`WithPaddingMM(mm)`][WithPaddingMM] | `0` | Blank between the label's edges and its text. `1.5` with a border |
| [`WithMarginMM(mm)`][WithMarginMM] | `1` | Blank tape before and after the label, 1 to 500 |
| [`WithLengthMM(mm)`][WithLengthMM] | fit | Make the label this long, margins included, 10 to 3000 |
| [`WithCode(k)`][WithCode] | `CodeText` | Pour only. Show each value as a code: `CodeText`, `CodeQR`, `CodeCODE39`, `CodeCODE128` or `CodeJAN13` |

### Option types

The type of an option says which functions take it:

| Type | Returned by | Taken by |
| --- | --- | --- |
| [`Option`][Option] | Every `With` function but [`WithCode`][WithCode] | [`NewText`][NewText], [`NewPages`][NewPages] and [`NewPour`][NewPour] |
| [`PourOnlyOption`][PourOnlyOption] | [`WithCode`][WithCode] | [`NewPour`][NewPour] |

- So [`WithCode`][WithCode] given to [`NewText`][NewText] does not compile.
- To build a list of options, make it a [`[]tm2.TextOption`][TextOption] or a [`[]tm2.PourOption`][PourOption].

```go
f, err := tm2.NewPour(
	tm2.Tape12mm,
	"URL",
	[]string{"https://example.com/"},
	tm2.WithCode(tm2.CodeQR),
)
```

### Errors

Every error wraps one of these. Test it with `errors.Is`.
Errors of the file system come back as they are, such as `fs.ErrNotExist`.

| Error | When |
| --- | --- |
| [`ErrInvalidOptions`][ErrInvalidOptions] | An option out of range, such as a margin under 1 mm |
| [`ErrCrashesApp`][ErrCrashesApp] | A value the app crashes on, such as alignment 3. It is never written |
| [`ErrTapeNotSupported`][ErrTapeNotSupported] | A tape width whose sizes are not known |
| [`ErrUnknownName`][ErrUnknownName] | A name [`ParseRotation`][ParseRotation], [`ParseCodeKind`][ParseCodeKind] and the like do not know |
| [`ErrNoText`][ErrNoText] | No text, no values or pages, or one of blanks only |
| [`ErrBadText`][ErrBadText] | Text that is not UTF-8 |
| [`ErrBadFile`][ErrBadFile] | A file that is not a `.tm2` file the package can read, or one it cannot encode |
| [`ErrOverLimit`][ErrOverLimit] | More rows, characters or columns than the app takes, or a label over 3000 mm |
| [`ErrNoRoom`][ErrNoRoom] | No room on the tape. Padding takes it, or there are too many lines, a line too long to run across, a font size too large, or a fixed length too short |
| [`ErrBadImage`][ErrBadImage] | A picture that is not PNG, JPEG or GIF, or over 8192 by 8192 pixels |
| [`ErrBadCode`][ErrBadCode] | A value the kind of code cannot hold, such as 13 digits for JAN-13 |
| [`ErrBadPatch`][ErrBadPatch] | An operation [`Patch`][File.Patch] cannot apply, such as one that adds a key, or a patch that makes no `.tm2` file |

### Reading files

- [`ReadFile`][ReadFile] and [`ParseFile`][ParseFile] read a `.tm2` file, from the app or from this package.
- Drawer types the package does not know are kept as [`RawDrawer`][RawDrawer].
- Keys it does not know, of the file, a document or a drawer, are kept in their `Unknown` field.
- Writing the file back keeps both as they were.

## License

tepra.tm2 is under the [MIT License](LICENSE).

The mascot is a Go gopher holding a strip of label tape.
The Go gopher was designed by Renée French, under [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/).

[NewText]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#NewText
[NewPages]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#NewPages
[NewPour]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#NewPour
[ReadFile]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#ReadFile
[ParseFile]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#ParseFile
[File]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#File
[File.WriteFile]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#File.WriteFile
[File.WriteTo]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#File.WriteTo
[File.Patch]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#File.Patch
[Tape4mm]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#Tape4mm
[Tape50mm]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#Tape50mm
[LookupTape]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#LookupTape
[Option]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#Option
[PourOnlyOption]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#PourOnlyOption
[TextOption]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#TextOption
[PourOption]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#PourOption
[WithName]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#WithName
[WithTime]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#WithTime
[WithFontSize]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#WithFontSize
[WithBold]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#WithBold
[WithAlign]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#WithAlign
[WithVerticalAlign]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#WithVerticalAlign
[WithRotation]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#WithRotation
[WithVertical]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#WithVertical
[WithFill]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#WithFill
[WithBorder]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#WithBorder
[WithBorderWidth]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#WithBorderWidth
[WithIcon]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#WithIcon
[WithBackground]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#WithBackground
[WithImageMode]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#WithImageMode
[WithImageThreshold]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#WithImageThreshold
[WithPaddingMM]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#WithPaddingMM
[WithMarginMM]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#WithMarginMM
[WithLengthMM]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#WithLengthMM
[WithCode]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#WithCode
[RawDrawer]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#RawDrawer
[ParseRotation]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#ParseRotation
[ParseCodeKind]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#ParseCodeKind
[ErrInvalidOptions]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#ErrInvalidOptions
[ErrCrashesApp]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#ErrCrashesApp
[ErrTapeNotSupported]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#ErrTapeNotSupported
[ErrUnknownName]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#ErrUnknownName
[ErrNoText]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#ErrNoText
[ErrOverLimit]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#ErrOverLimit
[ErrBadFile]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#ErrBadFile
[ErrBadText]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#ErrBadText
[ErrNoRoom]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#ErrNoRoom
[ErrBadImage]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#ErrBadImage
[ErrBadCode]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#ErrBadCode
[ErrBadPatch]: https://pkg.go.dev/github.com/mpyw/tepra.tm2#ErrBadPatch
