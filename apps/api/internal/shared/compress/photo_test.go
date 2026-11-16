package compress

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func photo(t *testing.T, w, h int) image.Image {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y += 7 {
		for x := 0; x < w; x += 7 {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 120, 255})
		}
	}
	return img
}

func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func size(t *testing.T, data []byte) (int, int, string) {
	t.Helper()
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	return cfg.Width, cfg.Height, format
}

func TestPhotoAlwaysDownscalesAndReencodes(t *testing.T) {
	out, mime, err := Photo(encodeJPEG(t, photo(t, 3000, 1500)), "image/jpeg")
	if err != nil || mime != "image/jpeg" {
		t.Fatal(err, mime)
	}
	if w, h, f := size(t, out); w != 2000 || h != 1000 || f != "jpeg" {
		t.Errorf("JPEG: %dx%d %s", w, h, f)
	}
	out, mime, err = Photo(encodePNG(t, photo(t, 1000, 2500)), "image/png")
	if err != nil || mime != "image/png" {
		t.Fatal(err, mime)
	}
	if w, h, f := size(t, out); w != 800 || h != 2000 || f != "png" {
		t.Errorf("PNG stays PNG: %dx%d %s", w, h, f)
	}
	small := encodeJPEG(t, photo(t, 40, 30))
	out, _, err = Photo(small, "image/jpeg")
	if err != nil {
		t.Fatal(err)
	}
	if w, h, _ := size(t, out); w != 40 || h != 30 {
		t.Errorf("a small photo keeps its size: %dx%d", w, h)
	}
}

// A 1x1 lossless WebP (Go can decode WebP but not encode it).
const tinyWebP = "UklGRhoAAABXRUJQVlA4TA0AAAAvAAAAEAcQERGIiP4HAA=="

func TestPhotoTurnsWebPIntoJPEG(t *testing.T) {
	data, _ := base64.StdEncoding.DecodeString(tinyWebP)
	out, mime, err := Photo(data, "image/webp")
	if err != nil || mime != "image/jpeg" {
		t.Fatal(err, mime)
	}
	if w, h, f := size(t, out); w != 1 || h != 1 || f != "jpeg" {
		t.Errorf("WebP: %dx%d %s", w, h, f)
	}
}

// pngWithHeader is a valid small PNG whose IHDR claims w x h: the header is
// all DecodeConfig reads, so this is what a decompression bomb looks like.
func pngWithHeader(t *testing.T, w, h uint32) []byte {
	t.Helper()
	data := encodePNG(t, photo(t, 8, 8))
	// signature (8) + length (4) + "IHDR" (4) → width at 16, height at 20, CRC at 29.
	binary.BigEndian.PutUint32(data[16:], w)
	binary.BigEndian.PutUint32(data[20:], h)
	binary.BigEndian.PutUint32(data[29:], crc32.ChecksumIEEE(data[12:29]))
	return data
}

func TestPhotoRefusesHugeCanvasesBeforeDecoding(t *testing.T) {
	for _, c := range [][2]uint32{{10000, 5000}, {30000, 10}, {7000, 7000}} {
		if _, _, err := Photo(pngWithHeader(t, c[0], c[1]), "image/png"); err != ErrPhotoTooLarge {
			t.Errorf("%dx%d: want ErrPhotoTooLarge, got %v", c[0], c[1], err)
		}
	}
	// Just under the limit is decoded (and fails later only because the pixels
	// are not really there).
	if _, _, err := Photo(pngWithHeader(t, 6000, 6000), "image/png"); err == ErrPhotoTooLarge {
		t.Error("36 MP is within the limit")
	}
}

func TestPhotoRefusesTheWrongType(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		mime string
	}{
		{"PNG declared JPEG", encodePNG(t, photo(t, 4, 4)), "image/jpeg"},
		{"JPEG declared PNG", encodeJPEG(t, photo(t, 4, 4)), "image/png"},
		{"garbage", []byte("not a photo"), "image/jpeg"},
		{"a PDF", []byte("%PDF-1.4\n"), "application/pdf"},
	}
	for _, c := range cases {
		if _, _, err := Photo(c.data, c.mime); err != ErrNotAPhoto {
			t.Errorf("%s: want ErrNotAPhoto, got %v", c.name, err)
		}
	}
}
