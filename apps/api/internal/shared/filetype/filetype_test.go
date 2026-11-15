package filetype

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"testing"
)

func encoded(t *testing.T, enc func(*bytes.Buffer, image.Image) error) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := enc(&buf, image.NewRGBA(image.Rect(0, 0, 4, 3))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestMatches(t *testing.T) {
	jpg := encoded(t, func(b *bytes.Buffer, i image.Image) error { return jpeg.Encode(b, i, nil) })
	pngBytes := encoded(t, func(b *bytes.Buffer, i image.Image) error { return png.Encode(b, i) })
	heic := append([]byte{0, 0, 0, 24, 'f', 't', 'y', 'p', 'h', 'e', 'i', 'c', 0, 0, 0, 0, 'm', 'i', 'f', '1', 'h', 'e', 'i', 'c'}, make([]byte, 40)...)
	webpHeader := []byte("RIFF\x00\x00\x00\x00WEBPVP8 ")

	cases := []struct {
		name string
		data []byte
		mime string
		want bool
	}{
		{"real jpeg", jpg, "image/jpeg", true},
		{"jpeg with parameter", jpg, "image/jpeg; charset=binary", true},
		{"real png", pngBytes, "image/png", true},
		{"png declared as jpeg", pngBytes, "image/jpeg", false},
		{"text renamed to jpg", []byte("hello, I am not a photo"), "image/jpeg", false},
		{"jpeg signature then garbage", append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{1}, 100)...), "image/jpeg", false},
		{"pdf", []byte("%PDF-1.7\n..."), "application/pdf", true},
		{"pdf after junk bytes", append([]byte("\r\n  "), []byte("%PDF-1.4")...), "application/pdf", true},
		{"html as pdf", []byte("<html></html>"), "application/pdf", false},
		{"heic", heic, "image/heic", true},
		{"jpeg as heic", jpg, "image/heic", false},
		{"truncated webp", webpHeader, "image/webp", false},
		{"empty", nil, "image/png", false},
		{"unknown type passes", []byte("anything"), "application/zip", true},
	}
	for _, c := range cases {
		if got := Matches(c.data, c.mime); got != c.want {
			t.Errorf("%s: Matches = %v, want %v", c.name, got, c.want)
		}
	}
}
