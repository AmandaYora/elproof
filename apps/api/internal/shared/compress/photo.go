package compress

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"image/png"

	_ "golang.org/x/image/webp" // register the WebP decoder for image.Decode
)

// MaxPhotoPixels bounds how large a picture Photo will decode. A small file can
// declare a huge canvas (a decompression bomb): 40 megapixels is well above any
// phone camera and keeps one decode under ~160 MB of RGBA.
const MaxPhotoPixels = 40_000_000

// maxPhotoSide bounds each side on its own, before the area is computed.
const maxPhotoSide = 20_000

// ErrPhotoTooLarge: the picture's dimensions are past MaxPhotoPixels.
var ErrPhotoTooLarge = errors.New("photo dimensions exceed the limit")

// ErrNotAPhoto: the bytes cannot be decoded as a JPEG, PNG or WebP picture.
var ErrNotAPhoto = errors.New("not a decodable photo")

// Photo re-encodes a JPEG, PNG or WebP picture for display (a project's cover
// photo, ADR-0047). Unlike Image, it ALWAYS decodes and re-encodes, so the
// stored file is guaranteed to be at most MaxDimension on its longest side and
// to carry no metadata (EXIF location included): PNG stays PNG (lossless, best
// compression), JPEG and WebP become JPEG at the evidence quality (Go has no
// WebP encoder). It returns the bytes and their MIME type. The dimensions are
// read from the header first, and a picture past MaxPhotoPixels is refused
// without being decoded.
func Photo(data []byte, mimeType string) ([]byte, string, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, "", ErrNotAPhoto
	}
	if !formatIs(format, mimeType) {
		return nil, "", ErrNotAPhoto
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, "", ErrNotAPhoto
	}
	if cfg.Width > maxPhotoSide || cfg.Height > maxPhotoSide || int64(cfg.Width)*int64(cfg.Height) > MaxPhotoPixels {
		return nil, "", ErrPhotoTooLarge
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", ErrNotAPhoto
	}
	img = downscale(img)

	var out bytes.Buffer
	if mimeType == "image/png" {
		encoder := png.Encoder{CompressionLevel: png.BestCompression}
		if err := encoder.Encode(&out, img); err != nil {
			return nil, "", err
		}
		return out.Bytes(), "image/png", nil
	}
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, "", err
	}
	return out.Bytes(), "image/jpeg", nil
}

// formatIs: the decoder that recognised the bytes is the declared type's.
func formatIs(format, mimeType string) bool {
	switch mimeType {
	case "image/jpeg":
		return format == "jpeg"
	case "image/png":
		return format == "png"
	case "image/webp":
		return format == "webp"
	}
	return false
}
