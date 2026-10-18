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
//   - An object put in place of the file, a document or a drawer, by any
//     operation, has every key tm2 writes for it, and no other key but
//     those the file had there. So a drawer's "type" changes only with its
//     keys. A drawer of a type tm2 does not know is one the file had. It
//     may move, but not be copied: it is written as it was read, so its
//     identifier cannot change.
//   - A drawer added to a document's drawers starts as the app sets a new
//     one of its "type" (text, pour or image), in the document's frame. The
//     keys given replace its values. A key it does not have is an error.
//   - The empty key after a drawer's identifier stays as the app writes it.
//   - A document or a drawer with no identifier, or one that another
//     before it has, as a copy has, gets a new one.
//   - A text box with text shows it, not its placeholder. A box with no
//     text keeps its HasPlaceholder: tm2's own border box is empty, with
//     no placeholder.
//   - With more than one document, each has MultiplePage set, as NewPages
//     sets it: the app lists the pages only then. A CurrentPage past the
//     last document is the last.
//   - Each row of tableData, and tableHeader, gets empty cells up to
//     FileTableColumns.
//   - A picture the patch gives is checked as NewText checks an icon. It
//     may be a JPEG or a GIF, and is made a PNG, as the app stores
//     pictures.
//
// Patch does not lay the label out again. A drawer moved or a text made
// longer keeps the frames it was given.
func (f *File) Patch(ops ...PatchOp) error {
	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		return err
	}
	root := patchNodeOf(buf.Bytes()) // WriteTo writes valid JSON
	shape := patchShapeOf(root)
	t := &patchTarget{root: root}
	for i, op := range ops {
		if err := t.apply(op); err != nil {
			return fmt.Errorf("operation %d (%s %s): %w", i+1, op.Op, op.Path, err)
		}
	}
	if err := shape.check(t.root); err != nil {
		return err
	}
	g, err := ParseFile(bytes.NewReader(t.root.bytes()))
	if err != nil {
		return fmt.Errorf("%w: the file it makes: %w", ErrBadPatch, err)
	}
	if err := g.patchFix(f); err != nil {
		return err
	}
	// What the app would crash on is not written, so it is not made.
	if _, err := g.WriteTo(io.Discard); err != nil {
		return err
	}
	*f = *g
	return nil
}

// patchFix sets what Patch keeps to in f, the file the operations made
// of old.
func (f *File) patchFix(old *File) error {
	oldImages := map[string]bool{}
	for _, doc := range old.Documents {
		for _, d := range doc.Drawers {
			if img, ok := d.(*ImageDrawer); ok {
				oldImages[string(img.Image)] = true
			}
		}
	}
	if len(f.Documents) > 1 {
		for i := range f.Documents {
			f.Documents[i].MultiplePage = true
		}
	}
	f.CurrentPage = min(f.CurrentPage, len(f.Documents)-1)
	// A drawer of a type tm2 does not know keeps its identifier, so no
	// drawer before it may take it.
	seen := map[string]bool{}
	for _, doc := range f.Documents {
		for _, d := range doc.Drawers {
			if _, raw := d.(*RawDrawer); raw {
				seen[d.Base().Identifier] = true
			}
		}
	}
	fresh := func(id *string) {
		if *id == "" || seen[*id] {
			*id = newUUID()
		}
		seen[*id] = true
	}
	for i := range f.Documents {
		doc := &f.Documents[i]
		fresh(&doc.Identifier)
		for j, d := range doc.Drawers {
			if _, raw := d.(*RawDrawer); raw {
				continue // it is written as it was read
			}
			fresh(&d.Base().Identifier)
			switch d := d.(type) {
			case *TextDrawer:
				d.HasPlaceholder = d.HasPlaceholder && d.Text == ""
			case *PourDrawer:
				d.HasPlaceholder = d.HasPlaceholder && d.Text == ""
			}
			img, ok := d.(*ImageDrawer)
			if !ok || len(img.Image) == 0 || oldImages[string(img.Image)] {
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
	padded, err := patchPadRow(f.TableHeader)
	if err != nil {
		return fmt.Errorf("tableHeader: %w", err)
	}
	f.TableHeader = padded
	return nil
}

// patchPadRow returns row with empty cells up to FileTableColumns.
func patchPadRow(row []string) ([]string, error) {
	if len(row) > FileTableColumns {
		return nil, fmt.Errorf("%d columns, up to %d: %w", len(row), FileTableColumns, ErrOverLimit)
	}
	return append(row, make([]string, FileTableColumns-len(row))...), nil
}

// patchShape is the keys each object of a file may have.
type patchShape struct {
	file, document patchKeys
	// drawers are the keys of the drawers of each type tm2 knows.
	drawers map[string]patchKeys
	// raws are the drawers of the file of no type tm2 knows.
	raws []*patchNode
}

// patchKeys are the keys of an object: those tm2 writes, all of which it
// needs, and the others the file had on such an object.
type patchKeys struct {
	needed  []string
	allowed map[string]bool
}

// patchShapeOf returns the shape of root, a file as WriteTo writes it.
func patchShapeOf(root *patchNode) *patchShape {
	// What tm2 writes for each, from objects it makes. A document of no
	// tape width is still written.
	b, _ := json.Marshal(&File{Documents: []Document{{}}}) // a file of one document encodes
	file := patchNodeOf(b)
	s := &patchShape{
		file:     patchKeysOf(file),
		document: patchKeysOf(file.member("documents").items[0]),
		drawers:  map[string]patchKeys{},
	}
	// SavedLength is written only when it is set.
	s.file.allowed["savedLength"] = true
	for _, d := range []Drawer{NewTextDrawer("", Rect{}), NewPourDrawer(0, "", Rect{}), NewImageDrawer(nil, Rect{})} {
		b, _ := json.Marshal(Drawers{d}) // a new drawer of a known type encodes
		s.drawers[d.DrawerType()] = patchKeysOf(patchNodeOf(b).items[0])
	}
	s.file.allow(root)
	for _, doc := range root.member("documents").items {
		s.document.allow(doc)
		for _, d := range doc.member("drawers").items {
			if keys, ok := s.drawers[patchDrawerType(d)]; ok {
				keys.allow(d)
			} else {
				s.raws = append(s.raws, d.clone())
			}
		}
	}
	return s
}

// patchKeysOf returns the keys of n, all needed.
func patchKeysOf(n *patchNode) patchKeys {
	k := patchKeys{needed: n.keys, allowed: map[string]bool{}}
	for _, key := range n.keys {
		k.allowed[key] = true
	}
	return k
}

// allow allows the keys of n.
func (k patchKeys) allow(n *patchNode) {
	for _, key := range n.keys {
		k.allowed[key] = true
	}
}

// check reports a key of n, at path, that k does not allow, or one it needs
// that n lacks.
func (k patchKeys) check(n *patchNode, path, what string) error {
	if n.kind != jsontext.KindBeginObject {
		return fmt.Errorf("%w: %s is not an object, as %s is", ErrBadPatch, path, what)
	}
	for _, key := range n.keys {
		if !k.allowed[key] {
			return fmt.Errorf("%w: %s: %s has no key %q, and tm2 writes only the keys the app writes", ErrBadPatch, path, what, key)
		}
	}
	for _, key := range k.needed {
		if n.index(key) < 0 {
			return fmt.Errorf("%w: %s: %s needs the key %q; replace the values of its keys instead, or copy one", ErrBadPatch, path, what, key)
		}
	}
	return nil
}

// check reports an object of root, the patched file, that s does not
// allow. Values of the wrong type are left to ParseFile.
func (s *patchShape) check(root *patchNode) error {
	if err := s.file.check(root, "the file", "a file"); err != nil {
		return err
	}
	docs := root.member("documents")
	if docs == nil || docs.kind != jsontext.KindBeginArray {
		return nil
	}
	raws := slices.Clone(s.raws)
	for i, doc := range docs.items {
		path := fmt.Sprintf("/documents/%d", i)
		if err := s.document.check(doc, path, "a document"); err != nil {
			return err
		}
		drawers := doc.member("drawers")
		if drawers.kind != jsontext.KindBeginArray {
			continue
		}
		for j, d := range drawers.items {
			path := fmt.Sprintf("%s/drawers/%d", path, j)
			typ := patchDrawerType(d)
			if keys, ok := s.drawers[typ]; ok {
				if err := keys.check(d, path, "a "+typ+" drawer"); err != nil {
					return err
				}
				continue
			}
			// Byte for byte: it is written as it was read.
			k := slices.IndexFunc(raws, func(r *patchNode) bool { return bytes.Equal(r.bytes(), d.bytes()) })
			if k < 0 {
				return fmt.Errorf("%w: %s is a drawer of type %q, which tm2 does not know; it can only be one the file has, once", ErrBadPatch, path, typ)
			}
			raws = slices.Delete(raws, k, k+1)
		}
	}
	return nil
}

// patchDrawerType is the "type" of the drawer d, or "" for none.
func patchDrawerType(d *patchNode) string {
	var typ string
	if t := d.member("type"); t != nil {
		_ = json.Unmarshal(t.value, &typ) // a type that is no string is none
	}
	return typ
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

// index returns the index of key in object n, or -1.
func (n *patchNode) index(key string) int {
	return slices.Index(n.keys, key)
}

// member returns the value of key in object n, or nil.
func (n *patchNode) member(key string) *patchNode {
	if i := n.index(key); i >= 0 {
		return n.members[i]
	}
	return nil
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
	if op.Op != "test" && patchIsDrawerHead(path) {
		return fmt.Errorf("%w: the empty key of a drawer stays as the app writes it", ErrBadPatch)
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
			got := v.bytes()
			if len(got) > patchShownBytes {
				got = append(got[:patchShownBytes:patchShownBytes], "..."...)
			}
			return fmt.Errorf("%w: the value is %s", ErrBadPatch, got)
		}
		return nil
	default:
		return fmt.Errorf("%w: op %q; known: add, remove, replace, move, copy, test", ErrBadPatch, op.Op)
	}
}

// patchShownBytes is the most of a value an error shows. A picture is
// megabytes.
const patchShownBytes = 200

// get returns the value at path.
func (t *patchTarget) get(path []string) (*patchNode, error) {
	n := t.root
	for i, tok := range path {
		switch n.kind {
		case jsontext.KindBeginObject:
			j := n.index(tok)
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
		i := parent.index(last)
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
	if len(path) != 4 || path[0] != "documents" || path[2] != "drawers" {
		return v, nil
	}
	if v.kind != jsontext.KindBeginObject {
		return nil, fmt.Errorf("%w: a drawer added is an object of its type and the keys to change", ErrBadPatch)
	}
	doc, err := t.get(path[:2])
	if err != nil {
		return nil, err
	}
	if doc.kind != jsontext.KindBeginObject {
		return v, nil // add reports that it holds no drawers
	}
	var frame Rect
	if fr := doc.member("frame"); fr != nil {
		_ = json.Unmarshal(fr.bytes(), &frame) // a frame that is no Rect puts the drawer at 0
	}
	var d Drawer
	switch typ := patchDrawerType(v); typ {
	case DrawerTypeText:
		d = NewTextDrawer("", frame)
	case DrawerTypePour:
		d = NewPourDrawer(0, "", frame)
	case DrawerTypeImage:
		d = NewImageDrawer(nil, frame)
	default:
		return nil, fmt.Errorf("%w: drawer type %q; a drawer added is of type text, pour or image", ErrBadPatch, typ)
	}
	b, _ := json.Marshal(Drawers{d}) // a new drawer of a known type encodes
	n := patchNodeOf(b).items[0]
	for i, key := range v.keys {
		j := n.index(key)
		if j < 0 {
			return nil, fmt.Errorf("%w: a %s drawer has no key %q", ErrBadPatch, d.DrawerType(), key)
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
		// A ~ is ~0 or ~1, nothing else.
		if strings.Count(tok, "~") != strings.Count(tok, "~0")+strings.Count(tok, "~1") {
			return nil, fmt.Errorf("%w: %q: a ~ in a JSON Pointer is ~0 or ~1", ErrBadPatch, p)
		}
		path[i] = strings.NewReplacer("~1", "/", "~0", "~").Replace(tok)
	}
	return path, nil
}

// patchIsDrawerHead reports whether path is the empty key after a drawer's
// identifier. The app writes it, and tm2 writes it as the app does.
func patchIsDrawerHead(path []string) bool {
	return len(path) == 5 && path[0] == "documents" && path[2] == "drawers" && path[4] == ""
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
