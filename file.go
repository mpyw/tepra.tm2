package tepra

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FileVersion is the "version" TEPRA LINK 2 writes.
const FileVersion = "2.0.0"

// FileTableColumns is how many columns the app writes in the table, used or
// not.
const FileTableColumns = 50

// File is a .tm2 file.
type File struct {
	// Documents are the pages of the file, a label each. The app lists them
	// when the first has MultiplePage set, and prints them together.
	Documents []Document `json:"documents"`
	// CurrentPage is the document the app opens, from 0. The app crashes on
	// a file whose CurrentPage has no document.
	CurrentPage int `json:"currentPage"`
	// FileName is the name the app shows the label by, with ".tm2". It is
	// not the name of the file the app opened.
	FileName string `json:"fileName"`
	// TableData is the data a PourDrawer fills from: one row, and one label,
	// per value, of FileTableColumns cells each.
	TableData [][]string `json:"tableData"`
	// TableHeader is the title of each column of TableData, shown in the
	// app's pour data editor.
	TableHeader []string `json:"tableHeader"`

	// Below: written as the app writes them. Values tried showed no change
	// in the app, or none were tried.
	// SavedLength and Version: 999 or left out, and "1.0.0". Date,
	// TextWords, Category, Keywords and the catalog titles do not show where
	// the app edits a label.
	SavedLength    *float64 `json:"savedLength,omitzero"`
	Version        string   `json:"version"`
	Date           string   `json:"date"`
	TextWords      string   `json:"textWords"`
	Category       string   `json:"category"`
	Keywords       string   `json:"keywords"`
	CatalogTitle   string   `json:"CatalogTitle"`
	CatalogTitleEN string   `json:"CatalogTitleEN"`
	KeywordsEN     string   `json:"keywordsEN"`

	// Unknown holds the keys this package does not know, as they were read,
	// and writes them back after the others. A newer app may write keys a
	// file read and written here must not lose.
	Unknown jsontext.Value `json:",embed"`
}

// newFile returns a file of docs, set as the app saves one. The app opens
// the first.
//
//declscope:shared // pour.go, text.go and pages.go wrap their documents in it
func newFile(c *optionConfig, docs ...Document) *File {
	length := docs[0].LengthMM()
	var words []string
	for _, doc := range docs {
		for _, d := range doc.Drawers {
			switch d := d.(type) {
			case *TextDrawer:
				words = append(words, d.Text)
			case *PourDrawer:
				words = append(words, d.Text)
			}
		}
	}
	return &File{
		Documents:   docs,
		Date:        c.at.Format("2006/01/02 15:04"),
		TextWords:   strings.Join(words, ","),
		FileName:    c.name + ".tm2",
		Version:     FileVersion,
		TableData:   [][]string{},
		TableHeader: make([]string, FileTableColumns),
		SavedLength: &length,
	}
}

// ParseFile reads a .tm2 file from r.
func ParseFile(r io.Reader) (*File, error) {
	var f File
	fr := &fileReader{r: r}
	if err := json.UnmarshalRead(fr, &f); err != nil {
		// An error of the reader comes back as it is, as err.go says, even
		// when jsontext wraps it in a SyntacticError, inside a value.
		if fr.err != nil {
			return nil, fr.err
		}
		return nil, fmt.Errorf("%w: %w", ErrBadFile, err)
	}
	// null and {} are JSON, but no label: the app writes a document always,
	// and a tape width in each.
	if len(f.Documents) == 0 {
		return nil, fmt.Errorf("%w: no documents", ErrBadFile)
	}
	for i, doc := range f.Documents {
		if doc.TapeWidthMM <= 0 {
			return nil, fmt.Errorf("%w: document %d has no tape width", ErrBadFile, i+1)
		}
	}
	return &f, nil
}

// fileReader reads r, and keeps its first error other than io.EOF.
type fileReader struct {
	r   io.Reader
	err error
}

// Read reads r.
func (fr *fileReader) Read(p []byte) (int, error) {
	n, err := fr.r.Read(p)
	if err != nil && !errors.Is(err, io.EOF) && fr.err == nil {
		fr.err = err
	}
	return n, err
}

// ReadFile reads the .tm2 file at path.
func ReadFile(path string) (*File, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = fh.Close() }() // read only: Close reports nothing lost
	return ParseFile(fh)
}

// MarshalJSONTo writes f, unless the app would crash on it.
func (f *File) MarshalJSONTo(enc *jsontext.Encoder) error {
	if f.CurrentPage < 0 || f.CurrentPage >= len(f.Documents) {
		return fmt.Errorf("tepra: current page %d of %d documents: %w", f.CurrentPage, len(f.Documents), ErrCrashesApp)
	}
	// fileJSON has File's fields without its methods, so encoding it does
	// not call MarshalJSONTo again.
	type fileJSON File
	return json.MarshalEncode(enc, (*fileJSON)(f))
}

// MarshalJSON is MarshalJSONTo, for encoding/json version 1. encoding/json
// escapes <, > and & in what it returns, which the app does not.
func (f *File) MarshalJSON() ([]byte, error) {
	// json.Marshal calls MarshalJSONTo, not MarshalJSON.
	return json.Marshal(f)
}

// WriteTo writes f to w as a .tm2 file: one line of JSON, with <, > and &
// as they are, as the app writes them.
func (f *File) WriteTo(w io.Writer) (int64, error) {
	b, err := f.fileEncode()
	if err != nil {
		return 0, err
	}
	n, err := w.Write(b)
	return int64(n), err
}

// WriteFile writes f to path, replacing a file there.
func (f *File) WriteFile(path string) error {
	b, err := f.fileEncode()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Clean(path), b, 0o644)
}

// fileEncode encodes f. An error of a sentinel, such as ErrCrashesApp,
// stays as it is. Any other is ErrBadFile.
func (f *File) fileEncode() ([]byte, error) {
	b, err := json.Marshal(f)
	if err != nil && !errors.Is(err, ErrCrashesApp) {
		return nil, fmt.Errorf("%w: %w", ErrBadFile, err)
	}
	return b, err
}
