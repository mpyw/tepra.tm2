# tepra.tm2

Writes `.tm2` files, the labels of TEPRA LINK 2. See README.md for the CLI, MCP and Go API.

## Rules

- Files the app wrote are the only source of truth for the format. Each key and its default come from one. Key order does not: a type's fields come in two groups, what was seen in the app first, then a blank line and the rest. Do not add a key, a drawer type or a tape width without such a file in `testdata/`.
- `TestReadFile_writesBackWhatTheAppWrote` must pass for every file in `testdata/`. A new sample gets the same round-trip test.
- Keep the app's quirks: the `"": ""` key after each drawer's identifier, the misspelled `BILineWidhValue`, 50 columns in `tableData` and `tableHeader`. Nobody knows which of them the app needs to read a file.
- `testdata/` holds only files made for this repository. A user's own labels can be personal. Files from the app's templates carry KING JIM's artwork.
- Check every error, or drop it with `_ =` and a comment saying why. errcheck runs with none of its default exclusions, in tests too.
- Every error of package tm2 wraps a sentinel of `err.go`, so callers test it with `errors.Is`. A new kind of failure gets a new sentinel there. The one exception: errors of the file system and of a reader or writer come back as they are, since `errors.Is` already tests them (`fs.ErrNotExist`).
- JSON goes through `encoding/json/v2`. A type with its own encoding implements `MarshalJSONTo` and `UnmarshalJSONFrom`, and `MarshalJSON` and `UnmarshalJSON` that call them, for `encoding/json`.
- A field the app writes stays, even when changing it prints nothing. Its comment says what was tried. The options and the CLI offer only what showed differently.

## Skills

Read the matching one in `.agents/skills/` before you start.

| Skill | When |
| --- | --- |
| `declscope-authoring` | Adding, naming or moving any declaration |
| `declscope-adoption` | Changing declscope's configuration |

## Before you finish

```sh
go vet ./... && go test ./...
mise exec -- golangci-lint run ./...
mise exec -- declscope ./...
```

declscope runs with `.declscope.yaml` (`qualify: ondemand`, `exported: true`). Each file is its own namespace, in `cmd/tm2` too. Only `tm2.go`, the types every file uses, is core.

A change to what the files hold is done only when a file it wrote opens in TEPRA LINK 2. Tests cannot show that. Say so when it is not checked yet.

## Rejected designs

| Idea | Why not |
| --- | --- |
| Print directly over Bluetooth or Wi-Fi | That needs the printer protocol, which comes from decompiling the app. Printing stays in the app. |
| Read the app's code to learn the format | Same. The format is learned from files the app saves and shares. |
| Only write CSV for the app's pour import | It works, but the layout must then be made by hand in the app each time. A `.tm2` file carries both. |
| A struct per drawer with its own `type` field | The app writes `type`, `identifier` and `""` first. `Drawers.MarshalJSONTo` writes those, then the struct's members. |
| The border on the text's own frame (`BIType` of the text box) | The app fits the text to its frame, so the line touches the text. The border goes on an empty box around it, with padding. |
| One inverted box for text of many lines | The app prints only its last line. Each line gets a box, stacked with no gap. |
| Set `sizePriority` on a text box of one label | The text then keeps the estimated size. Fitting to a padded frame keeps it in the frame when the estimate is off. Pour drawers are the exception, below. |
| Let the app stretch a picture to its frame | The app keeps a picture's shape, centred. A picture to fill a frame is made to the frame's shape first (`stretchImage`, `blackImage`). |
| Let the app fit each poured value to the shared frame | A value of fewer lines grows to the frame's height and runs past its sides. Pour drawers set `sizePriority`, so every label keeps the size of the value of most lines. |
| An option package per kind of label (`textopt`, `pouropt`, `pageopt`) | The kinds share every option but `WithCode`. Packages would repeat the shared ones, or need a fourth package and the config in `internal`. Option types do the same in one package: `Option` is a `TextOption` and a `PourOption`, and `WithCode` returns a `PourOnlyOption`, so a code given to `NewText` does not compile. |
| Each label of a list in a pour file at its own size | A pour drawer has one `TIFontSize` for every row. `NewPages` puts each label in a document of its own instead, with `MultiplePage` set, so the app lists them. |
| `tm2 text` taking each argument as a line | `tm2 text a b` then reads as two lines or as one with a space, and nobody can tell which. `pages` and `pour` break lines at line breaks, so `text` takes one argument and does the same. |
