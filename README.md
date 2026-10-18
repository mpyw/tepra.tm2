<div align="center">

<img src=".github/assets/icon.png" width="200" alt="A Go gopher holding a strip of label tape">

# tepra.tm2

Make label files for KING JIM's TEPRA LINK 2 app.

[![Test](https://github.com/mpyw/tepra.tm2/actions/workflows/test.yml/badge.svg)](https://github.com/mpyw/tepra.tm2/actions/workflows/test.yml)
[![Coverage](https://codecov.io/gh/mpyw/tepra.tm2/graph/badge.svg)](https://codecov.io/gh/mpyw/tepra.tm2)
[![Go Reference](https://pkg.go.dev/badge/github.com/mpyw/tepra.tm2.svg)](https://pkg.go.dev/github.com/mpyw/tepra.tm2)

</div>

TEPRA LINK 2 saves labels as `.tm2` files. tepra.tm2 writes them.
Type a list once, and get one label per line.
Then print from the app, as usual.

It works from an AI assistant over MCP, from the command line, and from Go.

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
| A release binary | Download the archive for your OS and CPU from [Releases](https://github.com/mpyw/tepra.tm2/releases). Put `tepra` on your `PATH`. `checksums.txt` has their SHA-256. |
| From source | `go install github.com/mpyw/tepra.tm2/cmd/tepra@latest` |

Releases cover macOS, Linux and Windows, each on amd64 and arm64.

### MCP Server

The MCP server is `tepra mcp`.
First install `tepra` as a [tool](#tool), as above.
Then add it to your assistant. For Claude Code:

```sh
claude mcp add tepra -- tepra mcp
```

### Library

```sh
go get github.com/mpyw/tepra.tm2
```

## Usage: As an MCP Server

Ask for labels in your own words.
The assistant writes a `.tm2` file with the [MCP tools](#mcp-tools).
Then [print the file](#print-the-file) from TEPRA LINK 2.

## Usage: As a Tool

One label of two lines:

```sh
tepra text -o sample.tm2 $'サンプル\nテキスト'
```

```text
wrote sample.tm2: 1 label, 18.2 mm long, on 12 mm tape
```

> [!TIP]
> `$'...'` turns `\n` into a line break in bash and zsh.
> A line break inside quotes works too. In PowerShell, write `` "サンプル`nテキスト" ``.

A label per row of a [CSV file](#the-csv-file), each in the size that fits it:

```sh
tepra pages -o pages.tm2 items.csv
```

```text
wrote pages.tm2: 3 pages, 12.0 to 22.5 mm long, on 12 mm tape
```

Then [print the file](#print-the-file) from TEPRA LINK 2.

## Usage: As a Library

```go
f, err := tepra.NewPages(
	tepra.Tape12mm,
	[]string{"1行目\n2行目", "1行目"},
	tepra.WithAlign(tepra.AlignCenter),
)
if err != nil {
	return err
}
return f.WriteFile("items.tm2")
```

## References

- All: [Kinds of label](#kinds-of-label), [Tapes](#tapes), [Layout](#layout), [Print the file](#print-the-file)
- MCP: [MCP tools](#mcp-tools)
- Tool: [Commands](#commands), [The CSV file](#the-csv-file), [Tool options](#tool-options), [tepra inspect](#tepra-inspect)
- Library: [Functions](#functions), [Library options](#library-options), [Option types](#option-types), [Errors](#errors), [Reading files](#reading-files)

### Kinds of label

| | text | pages | pour (流し込み) |
| --- | --- | --- | --- |
| Labels in a file | One | One per page | One per value |
| Font size and length | Fitted to the label | Fitted to each label | Same for every label, fitted to the longest |
| Edit in the app | As one label | Page by page | As pour data, in a table |
| QR codes and barcodes | No | No | Yes |
| MCP tool | `create_text_label` | `create_label_pages` | `create_pour_labels` |
| Command | `tepra text` | `tepra pages` | `tepra pour` |
| Go | [`NewText`][NewText] | [`NewPages`][NewPages] | [`NewPour`][NewPour] |

> [!TIP]
> For a list of things to label, use pages. Each label gets the size that fits it.
> Use pour for labels that must look alike, and for codes.

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
- Inverted text of more than one line is a box per line. In a pour file, each line is a column of its own: `Label`, `Label 2` and on.

> [!NOTE]
> On 4 mm tape, a border's padding of 1.5 mm leaves no room for text.
> Give a padding of 0.3 mm there.

### Print the file

1. Send the `.tm2` file to your phone, such as through Google Drive.
2. Open it with TEPRA LINK 2, through the share menu of the OS.
3. Check the labels in the app, and print.

### MCP tools

`tepra mcp` serves these on stdin and stdout:

| Tool | Kind | What it does |
| --- | --- | --- |
| `create_text_label` | text | Writes a file of one label |
| `create_label_pages` | pages | Writes a file of one page per string |
| `create_pour_labels` | pour | Writes a pour file: one label per string |
| `inspect_label` | | Describes a `.tm2` file |

Inputs:

- The tools that write a file take the [tool options](#tool-options), in snake case.
- A length in mm ends in `_mm`, and a picture in `_path`. Examples: `font_size`, `rotate`, `padding_mm`, `icon_path`.
- `code` and `column` are for `create_pour_labels` only.
- Paths must be absolute, for the file and for pictures.
- A file is not replaced unless `overwrite` is true.
- A line break in a label is `\n`.

### Commands

| Command | What it does |
| --- | --- |
| `tepra text [options] text` | Writes one label. A line break in the text breaks it into lines |
| `tepra pages [options] [file.csv]` | Writes a page per row of the file |
| `tepra pour [options] [file.csv]` | Writes a pour label per row of the file |
| `tepra inspect file.tm2` | Shows what a `.tm2` file holds |
| `tepra mcp` | Serves the [MCP tools](#mcp-tools) on stdin and stdout |

A pour file of the same `items.csv`:

```sh
tepra pour -o items.tm2 items.csv
```

```text
wrote items.tm2: 3 labels, 12.0 mm each, on 12 mm tape
```

A QR code per URL, one label each:

```sh
tepra pour --code qr -o qr.tm2 urls.csv
```

```text
wrote qr.tm2: 2 labels, 12.2 mm each, on 12 mm tape
```

A label of a fixed length, with a rounded border:

```sh
tepra text --length 60 --border rounded --align center -o fixed.tm2 サンプル
```

```text
wrote fixed.tm2: 1 label, 60.0 mm long, on 12 mm tape
```

Text turned 90 degrees, and text in white on black:

```sh
tepra text --rotate 90 -o turned.tm2 $'サンプル\nテキスト'
tepra text --fill inverted --bold -o inverted.tm2 サンプル
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

### tepra inspect

`tepra inspect` shows what a `.tm2` file holds.
It reads files the app wrote, too.

```text
$ tepra inspect items.tm2
file:    items.tm2 (version 2.0.0, saved 2026/10/18 10:15)
tape:    12 mm, pour, label 12.0 mm long, margin 1.0 mm
drawers:
  pour   x   1.6  y   1.1  w   6.7  h  10.2 mm  column 0, size 7.5, OTBGoStdR-Normal
rows:    3, column "Label"
    1  1行目 / 2行目 / 3行目
    2  1行目 / 2行目
    3  1行目
```

A file of pages shows each page. Pages count from 1, as in the app:

```text
$ tepra inspect pages.tm2
file:    pages.tm2 (version 2.0.0, saved 2026/10/18 10:15)
page 1:
tape:    12 mm, plain, label 12.0 mm long, margin 1.0 mm
drawers:
  text   x   1.6  y   1.1  w   6.7  h  10.2 mm  size 7.5, OTBGoStdR-Normal, "1行目\n2行目\n3行目"
page 2:
tape:    12 mm, plain, label 12.2 mm long, margin 1.0 mm
drawers:
  text   x   0.0  y   1.1  w  10.2  h  10.2 mm  size 11.5, OTBGoStdR-Normal, "1行目\n2行目"
page 3:
tape:    12 mm, plain, label 22.5 mm long, margin 1.0 mm
drawers:
  text   x   0.0  y   1.1  w  20.5  h  10.2 mm  size 23, OTBGoStdR-Normal, "1行目"
```

### Functions

| Function | Returns |
| --- | --- |
| [`NewText(tape, text, opts...)`][NewText] | A file of one label |
| [`NewPages(tape, texts, opts...)`][NewPages] | A file of a page per string, each fitted |
| [`NewPour(tape, column, values, opts...)`][NewPour] | A pour file of a label per string, under the column's title |
| [`ReadFile(path)`][ReadFile], [`ParseFile(r)`][ParseFile] | A file read back |

A [`File`][File] writes itself with [`WriteFile`][File.WriteFile] or [`WriteTo`][File.WriteTo].

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
- To build a list of options, make it a [`[]tepra.TextOption`][TextOption] or a [`[]tepra.PourOption`][PourOption].

```go
f, err := tepra.NewPour(
	tepra.Tape12mm,
	"URL",
	[]string{"https://example.com/"},
	tepra.WithCode(tepra.CodeQR),
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
