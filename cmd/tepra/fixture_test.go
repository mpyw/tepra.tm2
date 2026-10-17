package main

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

// fixturePicture returns a small PNG.
//
//declscope:shared // the tests of --icon, --background and their MCP inputs write it out
func fixturePicture(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewGray(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
