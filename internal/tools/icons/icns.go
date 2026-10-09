package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/png"
)

// icnsTypes are the pictures in WOPR.app's icon, smallest first. Most are PNG. The 16 and 32 px
// ones at 1x are 24-bit RGB with an 8-bit mask instead, the types every macOS draws: macOS
// draws PNG in those sizes' newer types (icp4, icp5) in a standalone .icns but not as an app's
// icon, and no form of icp6 draws in both, according to rendering tests on every macOS from
// 10.0 to 26 (https://github.com/relikd/icns-archive). The @2x types (ic11 to ic14) are what a
// Retina display uses.
var icnsTypes = []struct {
	typ  string
	size int
	kind byte // 'p' PNG, 'r' 24-bit RGB, 'm' 8-bit mask
}{
	{"is32", 16, 'r'},
	{"s8mk", 16, 'm'},
	{"il32", 32, 'r'},
	{"l8mk", 32, 'm'},
	{"ic11", 32, 'p'},   // 16 pt @2x
	{"ic12", 64, 'p'},   // 32 pt @2x
	{"ic07", 128, 'p'},  // 128 pt
	{"ic13", 256, 'p'},  // 128 pt @2x
	{"ic08", 256, 'p'},  // 256 pt
	{"ic14", 512, 'p'},  // 256 pt @2x
	{"ic09", 512, 'p'},  // 512 pt
	{"ic10", 1024, 'p'}, // 512 pt @2x
}

// maskOf names the mask that goes with each RGB type.
var maskOf = map[string]string{"is32": "s8mk", "il32": "l8mk"}

// encodeICNS writes a macOS icon file: the "icns" magic and the file's length, then each
// picture as its type, its length (counting this 8-byte header) and its data, all big-endian.
func encodeICNS(pic func(size int) *image.NRGBA, encodePNG func(*image.NRGBA) ([]byte, error)) ([]byte, error) {
	be := binary.BigEndian
	var body []byte
	for _, t := range icnsTypes {
		img := pic(t.size)
		var data []byte
		switch t.kind {
		case 'p':
			var err error
			if data, err = encodePNG(img); err != nil {
				return nil, err
			}
		case 'r':
			for ch := range 3 {
				data = append(data, packBits(plane(img, ch))...)
			}
			// macOS on Apple silicon drops the last value of compressed RGB data in an app's
			// icon; a spare zero byte after it is ignored everywhere (icns-archive, "Known
			// issues").
			data = append(data, 0)
		case 'm':
			data = plane(img, 3)
		}
		body = append(body, t.typ...)
		body = be.AppendUint32(body, uint32(8+len(data)))
		body = append(body, data...)
	}
	out := append([]byte("icns"), be.AppendUint32(nil, uint32(8+len(body)))...)
	return append(out, body...), nil
}

// plane returns one channel of a picture, row by row.
func plane(img *image.NRGBA, ch int) []byte {
	n := img.Rect.Dx() * img.Rect.Dy()
	out := make([]byte, n)
	for i := range n {
		out[i] = img.Pix[4*i+ch]
	}
	return out
}

// packBits compresses bytes as icns RGB data does: a header byte below 128 is followed by that
// many plus one literal bytes, and one of 128 or more by a byte repeated (header - 125) times.
func packBits(src []byte) []byte {
	var out []byte
	for i := 0; i < len(src); {
		run := 1
		for i+run < len(src) && src[i+run] == src[i] && run < 130 {
			run++
		}
		if run >= 3 {
			out = append(out, byte(run+125), src[i])
			i += run
			continue
		}
		start := i
		for i < len(src) && i-start < 128 {
			if i+2 < len(src) && src[i] == src[i+1] && src[i] == src[i+2] {
				break
			}
			i++
		}
		out = append(out, byte(i-start-1))
		out = append(out, src[start:i]...)
	}
	return out
}

// unpackBits expands n bytes of packBits data, returning how many bytes it read.
func unpackBits(src []byte, n int) ([]byte, int, error) {
	out := make([]byte, 0, n)
	i := 0
	for len(out) < n {
		if i >= len(src) {
			return nil, i, errors.New("compressed data ends early")
		}
		h := int(src[i])
		i++
		if h < 128 {
			if i+h+1 > len(src) {
				return nil, i, errors.New("compressed data ends early")
			}
			out = append(out, src[i:i+h+1]...)
			i += h + 1
		} else {
			if i >= len(src) {
				return nil, i, errors.New("compressed data ends early")
			}
			out = append(out, bytes.Repeat(src[i:i+1], h-125)...)
			i++
		}
	}
	if len(out) != n {
		return nil, i, errors.New("a run crosses the end of the picture")
	}
	return out, i, nil
}

// icnsElem is one picture read back from an icon file. An RGB picture carries the alpha of its
// mask, and a mask reads as an alpha picture.
type icnsElem struct {
	typ string
	img image.Image
}

// decodeICNS reads an icon file holding the kinds of picture encodeICNS writes.
func decodeICNS(data []byte) ([]icnsElem, error) {
	be := binary.BigEndian
	if len(data) < 8 || string(data[:4]) != "icns" || int(be.Uint32(data[4:])) != len(data) {
		return nil, errors.New("icns: not an icon file, or its length is wrong")
	}
	raw := map[string][]byte{}
	var order []string
	for i := 8; i < len(data); {
		if i+8 > len(data) {
			return nil, errors.New("icns: truncated element header")
		}
		typ, n := string(data[i:i+4]), int(be.Uint32(data[i+4:]))
		if n < 8 || i+n > len(data) {
			return nil, fmt.Errorf("icns: element %q has a bad length", typ)
		}
		raw[typ] = data[i+8 : i+n]
		order = append(order, typ)
		i += n
	}
	sizes := map[string]int{}
	for _, t := range icnsTypes {
		sizes[t.typ] = t.size
	}
	var out []icnsElem
	for _, typ := range order {
		d := raw[typ]
		size, known := sizes[typ]
		var img image.Image
		switch {
		case bytes.HasPrefix(d, []byte("\x89PNG\r\n\x1a\n")):
			var err error
			if img, err = png.Decode(bytes.NewReader(d)); err != nil {
				return nil, fmt.Errorf("icns: %s: %w", typ, err)
			}
		case known && maskOf[typ] != "":
			rgb, _, err := unpackBits(d, 3*size*size)
			if err != nil {
				return nil, fmt.Errorf("icns: %s: %w", typ, err)
			}
			mask, hasMask := raw[maskOf[typ]]
			if hasMask && len(mask) != size*size {
				return nil, fmt.Errorf("icns: %s has the wrong length", maskOf[typ])
			}
			m := image.NewNRGBA(image.Rect(0, 0, size, size))
			for p := range size * size {
				a := byte(255)
				if hasMask {
					a = mask[p]
				}
				copy(m.Pix[4*p:], []byte{rgb[p], rgb[size*size+p], rgb[2*size*size+p], a})
			}
			img = m
		case known && len(d) == size*size:
			img = &image.Alpha{Pix: d, Stride: size, Rect: image.Rect(0, 0, size, size)}
		default:
			return nil, fmt.Errorf("icns: cannot read element %q", typ)
		}
		out = append(out, icnsElem{typ, img})
	}
	return out, nil
}
