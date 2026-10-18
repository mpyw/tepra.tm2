package tm2

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// PatchOp is one operation of a JSON Patch (RFC 6902), as File.Patch
// applies it. Path and From are JSON Pointers (RFC 6901) into the file as
// WriteTo writes it, such as "/documents/0/drawers/1/TIText".
type PatchOp struct {
	// Op is add, remove, replace, move, copy or test.
	Op   string `json:"op"`
	Path string `json:"path"`
	// From is the value that move and copy take.
	From string `json:"from,omitzero"`
	// Value is the value that add, replace and test take.
	Value jsontext.Value `json:"value,omitzero"`
}

// Patch applies ops to f in order, as a JSON Patch (RFC 6902) on the file as
// WriteTo writes it. Either every operation applies, or f is left as it was.
//
// It keeps to what the app writes:
//
//   - A key is not added to an object, nor removed from one. The app writes
//     every key, so a key's value is replaced instead. Items of an array
//     are added and removed freely.
//   - A drawer added to a document's drawers starts as the app sets a new
//     one of its "type" (text, pour or image), in the document's frame. The
//     keys given replace its values. A key it does not have is an error.
//   - A document or a drawer whose identifier another before it has, as a
//     copy has, gets a new one.
//   - With more than one document, each has MultiplePage set, as NewPages
//     sets it: the app lists the pages only then. A CurrentPage past the
//     last document is the last.
//   - Each row of tableData, and tableHeader, gets empty cells up to
//     FileTableColumns.
//   - The picture of an image drawer may be a JPEG or a GIF. It is made a
//     PNG, as the app stores pictures.
//
// Patch does not lay the label out again. A drawer moved or a text made
// longer keeps the frames it was given.
func (f *File) Patch(ops ...PatchOp) error {
	b, err := json.Marshal(f)
	if err != nil {
		return err
	}
	t := &patchTarget{root: patchNodeOf(b)} // json.Marshal writes valid JSON
	for i, op := range ops {
		if err := t.apply(op); err != nil {
			return fmt.Errorf("operation %d (%s %s): %w", i+1, op.Op, op.Path, err)
		}
	}
	g, err := ParseFile(bytes.NewReader(t.root.bytes()))
	if err != nil {
		return fmt.Errorf("%w: the file it makes: %w", ErrBadPatch, err)
	}
	if err := g.patchFix(); err != nil {
		return err
	}
	// What the app would crash on is not written, so it is not made.
	if _, err := g.WriteTo(io.Discard); err != nil {
		return err
	}
	*f = *g
	return nil
}

// patchFix sets what Patch keeps to in f, the file the operations made.
func (f *File) patchFix() error {
	if len(f.Documents) > 1 {
		for i := range f.Documents {
			f.Documents[i].MultiplePage = true
		}
	}
	f.CurrentPage = min(f.CurrentPage, len(f.Documents)-1)
	seen := map[string]bool{}
	fresh := func(id *string) {
		if *id != "" && seen[*id] {
			*id = newUUID()
		}
		seen[*id] = true
	}
	for i := range f.Documents {
		doc := &f.Documents[i]
		fresh(&doc.Identifier)
		for j, d := range doc.Drawers {
			if _, raw := d.(*RawDrawer); raw {
				// Its identifier is written as it was read.
				seen[d.Base().Identifier] = true
				continue
			}
			fresh(&d.Base().Identifier)
			img, ok := d.(*ImageDrawer)
			if !ok || len(img.Image) == 0 || bytes.HasPrefix(img.Image, patchPNGSignature) {
				continue
			}
			png, _, _, err := decodeImage(img.Image)
			if err != nil {
				return fmt.Errorf("%w: page %d, drawer %d: %w", ErrBadImage, i+1, j+1, err)
			}
			img.Image = png
		}
	}
	for i, row := range f.TableData {
		padded, err := patchPadRow(row)
		if err != nil {
			return fmt.Errorf("tableData row %d: %w", i+1, err)
		}
		f.TableData[i] = padded
	}
	if len(f.TableHeader) > 0 {
		padded, err := patchPadRow(f.TableHeader)
		if err != nil {
			return fmt.Errorf("tableHeader: %w", err)
		}
		f.TableHeader = padded
	}
	return nil
}

// patchPNGSignature is how a PNG file starts.
var patchPNGSignature = []byte("\x89PNG\r\n\x1a\n")

// patchPadRow returns row with empty cells up to FileTableColumns.
func patchPadRow(row []string) ([]string, error) {
	if len(row) > FileTableColumns {
		return nil, fmt.Errorf("%d columns, up to %d: %w", len(row), FileTableColumns, ErrOverLimit)
	}
	return append(row, make([]string, FileTableColumns-len(row))...), nil
}

// patchNode is a JSON value of the file being patched. An object keeps the
// order of its keys, and a number keeps its digits.
type patchNode struct {
	kind jsontext.Kind
	// value is the JSON of a value that is neither an object nor an array.
	value jsontext.Value
	// keys and members are an object's, in order.
	keys    []string
	members []*patchNode
	// items are an array's.
	items []*patchNode
}

// patchParse reads one JSON value.
func patchParse(b []byte) (*patchNode, error) {
	v := jsontext.Value(b)
	if !v.IsValid() {
		return nil, fmt.Errorf("not one JSON value: %s", b)
	}
	return patchNodeOf(v), nil
}

// patchNodeOf returns the node of v, valid JSON. So no read fails.
func patchNodeOf(v jsontext.Value) *patchNode {
	kind := v.Kind()
	if kind != jsontext.KindBeginObject && kind != jsontext.KindBeginArray {
		return &patchNode{kind: kind, value: v.Clone()}
	}
	n := &patchNode{kind: kind}
	dec := jsontext.NewDecoder(bytes.NewReader(v))
	_, _ = dec.ReadToken() // v is valid, so it reads
	for dec.PeekKind() != jsontext.KindEndObject && dec.PeekKind() != jsontext.KindEndArray {
		if kind == jsontext.KindBeginObject {
			name, _ := dec.ReadToken() // v is valid, so it reads
			n.keys = append(n.keys, name.String())
			item, _ := dec.ReadValue() // v is valid, so it reads
			n.members = append(n.members, patchNodeOf(item))
		} else {
			item, _ := dec.ReadValue() // v is valid, so it reads
			n.items = append(n.items, patchNodeOf(item))
		}
	}
	return n
}

// bytes is the JSON of n.
func (n *patchNode) bytes() []byte {
	switch n.kind {
	case jsontext.KindBeginObject:
		b := []byte{'{'}
		for i, key := range n.keys {
			if i > 0 {
				b = append(b, ',')
			}
			b, _ = jsontext.AppendQuote(b, key) // a key read from valid JSON is valid UTF-8
			b = append(b, ':')
			b = append(b, n.members[i].bytes()...)
		}
		return append(b, '}')
	case jsontext.KindBeginArray:
		b := []byte{'['}
		for i, item := range n.items {
			if i > 0 {
				b = append(b, ',')
			}
			b = append(b, item.bytes()...)
		}
		return append(b, ']')
	default:
		return n.value
	}
}

// clone returns a copy of n that shares nothing with it.
func (n *patchNode) clone() *patchNode {
	c := &patchNode{kind: n.kind, value: n.value, keys: slices.Clone(n.keys)}
	for _, m := range n.members {
		c.members = append(c.members, m.clone())
	}
	for _, item := range n.items {
		c.items = append(c.items, item.clone())
	}
	return c
}

// member returns the index of key in object n, or -1.
func (n *patchNode) member(key string) int {
	return slices.Index(n.keys, key)
}

// equal reports whether n and m are the same JSON value, as test compares
// them: numbers by value, objects whatever the order of their keys.
func (n *patchNode) equal(m *patchNode) bool {
	var a, b any
	return json.Unmarshal(n.bytes(), &a) == nil && json.Unmarshal(m.bytes(), &b) == nil && reflect.DeepEqual(a, b)
}

// patchTarget is the file being patched.
type patchTarget struct {
	root *patchNode
}

// apply applies op.
func (t *patchTarget) apply(op PatchOp) error {
	path, err := patchPointer(op.Path)
	if err != nil {
		return err
	}
	var value *patchNode
	switch op.Op {
	case "add", "replace", "test":
		if len(op.Value) == 0 {
			return fmt.Errorf("%w: %s needs a value", ErrBadPatch, op.Op)
		}
		if value, err = patchParse(op.Value); err != nil {
			return fmt.Errorf("%w: value: %w", ErrBadPatch, err)
		}
	case "move", "copy":
		if _, err := patchPointer(op.From); err != nil {
			return err
		}
	}
	switch op.Op {
	case "add":
		if value, err = t.drawerAdded(path, value); err != nil {
			return err
		}
		return t.add(path, value)
	case "remove":
		_, err := t.remove(path)
		return err
	case "replace":
		return t.replace(path, value)
	case "move":
		if op.Path == op.From {
			return nil
		}
		if strings.HasPrefix(op.Path, op.From+"/") {
			return fmt.Errorf("%w: a value cannot move into itself", ErrBadPatch)
		}
		from, _ := patchPointer(op.From) // checked above
		v, err := t.remove(from)
		if err != nil {
			return err
		}
		return t.add(path, v)
	case "copy":
		from, _ := patchPointer(op.From) // checked above
		v, err := t.get(from)
		if err != nil {
			return err
		}
		return t.add(path, v.clone())
	case "test":
		v, err := t.get(path)
		if err != nil {
			return err
		}
		if !v.equal(value) {
			return fmt.Errorf("%w: the value is %s", ErrBadPatch, v.bytes())
		}
		return nil
	default:
		return fmt.Errorf("%w: op %q; known: add, remove, replace, move, copy, test", ErrBadPatch, op.Op)
	}
}

// get returns the value at path.
func (t *patchTarget) get(path []string) (*patchNode, error) {
	n := t.root
	for i, tok := range path {
		switch n.kind {
		case jsontext.KindBeginObject:
			j := n.member(tok)
			if j < 0 {
				return nil, fmt.Errorf("%w: %s has no key %q", ErrBadPatch, patchPointerString(path[:i]), tok)
			}
			n = n.members[j]
		case jsontext.KindBeginArray:
			j, err := patchIndex(tok, len(n.items))
			if err != nil {
				return nil, fmt.Errorf("%w: %s: %w", ErrBadPatch, patchPointerString(path[:i]), err)
			}
			n = n.items[j]
		default:
			return nil, fmt.Errorf("%w: %s is neither an object nor an array", ErrBadPatch, patchPointerString(path[:i]))
		}
	}
	return n, nil
}

// add sets the value of a key that is there, or puts v in an array before
// the item at path, or at its end for "-".
func (t *patchTarget) add(path []string, v *patchNode) error {
	if len(path) == 0 {
		t.root = v
		return nil
	}
	parent, last, err := t.parent(path)
	if err != nil {
		return err
	}
	if parent.kind == jsontext.KindBeginObject {
		return t.replace(path, v)
	}
	i := len(parent.items)
	if last != "-" {
		if i, err = patchIndex(last, len(parent.items)+1); err != nil {
			return fmt.Errorf("%w: %s: %w", ErrBadPatch, patchPointerString(path[:len(path)-1]), err)
		}
	}
	parent.items = slices.Insert(parent.items, i, v)
	return nil
}

// remove takes the item at path out of its array, and returns it.
func (t *patchTarget) remove(path []string) (*patchNode, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("%w: the file cannot be removed", ErrBadPatch)
	}
	parent, last, err := t.parent(path)
	if err != nil {
		return nil, err
	}
	if parent.kind == jsontext.KindBeginObject {
		return nil, fmt.Errorf("%w: %s is a key, which the app writes always; replace its value instead", ErrBadPatch, patchPointerString(path))
	}
	i, err := patchIndex(last, len(parent.items))
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrBadPatch, patchPointerString(path[:len(path)-1]), err)
	}
	v := parent.items[i]
	parent.items = slices.Delete(parent.items, i, i+1)
	return v, nil
}

// replace sets the value at path, which must be there.
func (t *patchTarget) replace(path []string, v *patchNode) error {
	if len(path) == 0 {
		t.root = v
		return nil
	}
	parent, last, err := t.parent(path)
	if err != nil {
		return err
	}
	if parent.kind == jsontext.KindBeginObject {
		i := parent.member(last)
		if i < 0 {
			return fmt.Errorf("%w: %s has no key %q, and tm2 writes only the keys the app writes", ErrBadPatch, patchPointerString(path[:len(path)-1]), last)
		}
		parent.members[i] = v
		return nil
	}
	i, err := patchIndex(last, len(parent.items))
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrBadPatch, patchPointerString(path[:len(path)-1]), err)
	}
	parent.items[i] = v
	return nil
}

// parent returns the object or array that holds the value at path, and the
// value's key or index in it.
func (t *patchTarget) parent(path []string) (*patchNode, string, error) {
	parent, err := t.get(path[:len(path)-1])
	if err != nil {
		return nil, "", err
	}
	if parent.kind != jsontext.KindBeginObject && parent.kind != jsontext.KindBeginArray {
		return nil, "", fmt.Errorf("%w: %s is neither an object nor an array", ErrBadPatch, patchPointerString(path[:len(path)-1]))
	}
	return parent, path[len(path)-1], nil
}

// drawerAdded returns v as the drawer it adds when path is in a document's
// drawers: a new drawer of its type, with v's keys over it. Any other v is
// returned as it is.
func (t *patchTarget) drawerAdded(path []string, v *patchNode) (*patchNode, error) {
	if len(path) != 4 || path[0] != "documents" || path[2] != "drawers" || v.kind != jsontext.KindBeginObject {
		return v, nil
	}
	doc, err := t.get(path[:2])
	if err != nil {
		return nil, err
	}
	if doc.kind != jsontext.KindBeginObject {
		return v, nil // add reports that it holds no drawers
	}
	var frame Rect
	if i := doc.member("frame"); i >= 0 {
		_ = json.Unmarshal(doc.members[i].bytes(), &frame) // a frame that is no Rect puts the drawer at 0
	}
	var typ string
	if i := v.member("type"); i < 0 || json.Unmarshal(v.members[i].value, &typ) != nil {
		return nil, fmt.Errorf("%w: a drawer added needs its type: text, pour or image", ErrBadPatch)
	}
	var d Drawer
	switch typ {
	case DrawerTypeText:
		d = NewTextDrawer("", frame)
	case DrawerTypePour:
		d = NewPourDrawer(0, "", frame)
	case DrawerTypeImage:
		d = NewImageDrawer(nil, frame)
	default:
		return nil, fmt.Errorf("%w: drawer type %q; tm2 adds text, pour or image", ErrBadPatch, typ)
	}
	b, _ := json.Marshal(Drawers{d}) // a new drawer of a known type encodes
	n := patchNodeOf(b).items[0]
	for i, key := range v.keys {
		j := n.member(key)
		if j < 0 {
			return nil, fmt.Errorf("%w: a %s drawer has no key %q", ErrBadPatch, typ, key)
		}
		n.members[j] = v.members[i]
	}
	return n, nil
}

// patchPointer splits a JSON Pointer into its keys and indexes.
func patchPointer(p string) ([]string, error) {
	if p == "" {
		return nil, nil
	}
	if p[0] != '/' {
		return nil, fmt.Errorf("%w: %q is not a JSON Pointer, which starts with /", ErrBadPatch, p)
	}
	path := strings.Split(p[1:], "/")
	for i, tok := range path {
		path[i] = strings.NewReplacer("~1", "/", "~0", "~").Replace(tok)
	}
	return path, nil
}

// patchPointerString is the JSON Pointer of path, or "the file" for none.
func patchPointerString(path []string) string {
	if len(path) == 0 {
		return "the file"
	}
	escaped := make([]string, len(path))
	for i, tok := range path {
		escaped[i] = strings.NewReplacer("~", "~0", "/", "~1").Replace(tok)
	}
	return "/" + strings.Join(escaped, "/")
}

// patchIndex reads tok as an index of an array of n items.
func patchIndex(tok string, n int) (int, error) {
	i, err := strconv.Atoi(tok)
	if err != nil || i < 0 || tok != strconv.Itoa(i) {
		return 0, fmt.Errorf("%q is not an index", tok)
	}
	if i >= n {
		return 0, fmt.Errorf("index %d of %d items", i, n)
	}
	return i, nil
}
