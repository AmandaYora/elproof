package compress

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func lowQualityJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 640, 480))
	for y := 0; y < 480; y++ {
		for x := 0; x < 640; x++ {
			img.Set(x, y, color.RGBA{uint8(x * 7), uint8(y * 13), uint8((x ^ y) * 3), 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 25}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// EP-01: re-encoding must never return more bytes than it was given.
func TestImageNeverGrows(t *testing.T) {
	src := lowQualityJPEG(t)
	// Precondition: a plain re-encode at our quality really is bigger here.
	decoded, _ := jpeg.Decode(bytes.NewReader(src))
	var plain bytes.Buffer
	_ = jpeg.Encode(&plain, decoded, &jpeg.Options{Quality: jpegQuality})
	if plain.Len() <= len(src) {
		t.Skip("fixture no longer grows when re-encoded")
	}
	out, err := Image(src, "image/jpeg")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, src) {
		t.Errorf("a re-encode that would grow the file must return the original (%d → %d)", len(src), len(out))
	}
}

func TestNonImagesPassThrough(t *testing.T) {
	in := []byte("%PDF-1.4")
	out, _ := Image(in, "application/pdf")
	if !bytes.Equal(in, out) {
		t.Error("a PDF is returned unchanged")
	}
}
