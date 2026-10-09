package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/png"
)

// encodePNG writes a picture as PNG. image/png has no timestamps or other varying chunks, so
// the same picture gives the same bytes with the same Go.
func encodePNG(img *image.NRGBA, level png.CompressionLevel) ([]byte, error) {
	var b bytes.Buffer
	enc := png.Encoder{CompressionLevel: level}
	if err := enc.Encode(&b, img); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// encodeICO writes a Windows icon in the layout every Windows tool reads: each picture up to
// 128 px as a 32-bit BMP (a BITMAPINFOHEADER, BGRA rows bottom-up, then the 1-bit AND mask),
// and 256 px as PNG, which Windows reads since Vista.
func encodeICO(imgs []*image.NRGBA, encodePNG func(*image.NRGBA) ([]byte, error)) ([]byte, error) {
	var head, body bytes.Buffer
	le := binary.LittleEndian
	head.Write(le.AppendUint16(nil, 0)) // reserved
	head.Write(le.AppendUint16(nil, 1)) // type: icon
	head.Write(le.AppendUint16(nil, uint16(len(imgs))))
	offset := 6 + 16*len(imgs)
	for _, img := range imgs {
		n := img.Rect.Dx()
		if n != img.Rect.Dy() || n < 1 || n > 256 {
			return nil, fmt.Errorf("ico: cannot store a %dx%d picture", n, img.Rect.Dy())
		}
		var data []byte
		if n == 256 {
			var err error
			if data, err = encodePNG(img); err != nil {
				return nil, err
			}
		} else {
			data = icoBMP(img)
		}
		dim := byte(n % 256) // 0 means 256
		head.Write([]byte{dim, dim, 0, 0})
		head.Write(le.AppendUint16(nil, 1))  // colour planes
		head.Write(le.AppendUint16(nil, 32)) // bits per pixel
		head.Write(le.AppendUint32(nil, uint32(len(data))))
		head.Write(le.AppendUint32(nil, uint32(offset)))
		offset += len(data)
		body.Write(data)
	}
	return append(head.Bytes(), body.Bytes()...), nil
}

// icoBMP is one icon picture as a headerless BMP: the height counts the colour rows and the
// mask rows together. The mask marks pixels under half opacity as transparent, for the rare
// reader that ignores the alpha channel.
func icoBMP(img *image.NRGBA) []byte {
	n := img.Rect.Dx()
	maskStride := (n + 31) / 32 * 4
	le := binary.LittleEndian
	b := make([]byte, 0, 40+4*n*n+maskStride*n)
	b = le.AppendUint32(b, 40) // header size
	b = le.AppendUint32(b, uint32(n))
	b = le.AppendUint32(b, uint32(2*n))
	b = le.AppendUint16(b, 1)  // planes
	b = le.AppendUint16(b, 32) // bits per pixel
	b = le.AppendUint32(b, 0)  // BI_RGB
	b = le.AppendUint32(b, uint32(4*n*n+maskStride*n))
	b = append(b, make([]byte, 16)...) // resolution and palette: unused
	for y := n - 1; y >= 0; y-- {
		for x := range n {
			p := img.Pix[img.PixOffset(x, y):]
			b = append(b, p[2], p[1], p[0], p[3])
		}
	}
	for y := n - 1; y >= 0; y-- {
		row := make([]byte, maskStride)
		for x := range n {
			if img.Pix[img.PixOffset(x, y)+3] < 128 {
				row[x/8] |= 0x80 >> (x % 8)
			}
		}
		b = append(b, row...)
	}
	return b
}

// icoEntry is one picture read back from an icon file.
type icoEntry struct {
	size  int
	isPNG bool
	img   image.Image
}

// decodeICO reads an icon file written by encodeICO (or any 32-bit or PNG icon).
func decodeICO(data []byte) ([]icoEntry, error) {
	le := binary.LittleEndian
	if len(data) < 6 || le.Uint16(data) != 0 || le.Uint16(data[2:]) != 1 {
		return nil, errors.New("ico: not an icon file")
	}
	count := int(le.Uint16(data[4:]))
	if len(data) < 6+16*count {
		return nil, errors.New("ico: truncated directory")
	}
	var out []icoEntry
	for i := range count {
		e := data[6+16*i:]
		size := int(e[0])
		if size == 0 {
			size = 256
		}
		length, offset := int(le.Uint32(e[8:])), int(le.Uint32(e[12:]))
		if offset < 0 || length < 0 || offset+length > len(data) || offset+length < offset {
			return nil, fmt.Errorf("ico: entry %d lies outside the file", i)
		}
		pic := data[offset : offset+length]
		ent := icoEntry{size: size}
		if bytes.HasPrefix(pic, []byte("\x89PNG\r\n\x1a\n")) {
			img, err := png.Decode(bytes.NewReader(pic))
			if err != nil {
				return nil, fmt.Errorf("ico: entry %d: %w", i, err)
			}
			ent.isPNG, ent.img = true, img
		} else {
			img, err := decodeICOBMP(pic)
			if err != nil {
				return nil, fmt.Errorf("ico: entry %d: %w", i, err)
			}
			ent.img = img
		}
		out = append(out, ent)
	}
	return out, nil
}

func decodeICOBMP(b []byte) (*image.NRGBA, error) {
	le := binary.LittleEndian
	if len(b) < 40 || le.Uint32(b) != 40 {
		return nil, errors.New("not a BITMAPINFOHEADER bitmap")
	}
	w, h := int(int32(le.Uint32(b[4:]))), int(int32(le.Uint32(b[8:])))/2
	if le.Uint16(b[14:]) != 32 || le.Uint32(b[16:]) != 0 {
		return nil, errors.New("not an uncompressed 32-bit bitmap")
	}
	if w <= 0 || h <= 0 || w > 256 || h > 256 || len(b) < 40+4*w*h {
		return nil, errors.New("bad bitmap size")
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	px := b[40:]
	for y := range h {
		for x := range w {
			p := px[4*((h-1-y)*w+x):]
			copy(img.Pix[img.PixOffset(x, y):], []byte{p[2], p[1], p[0], p[3]})
		}
	}
	return img, nil
}
