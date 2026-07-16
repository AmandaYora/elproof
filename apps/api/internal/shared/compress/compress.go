// Package compress re-compresses evidence uploads server-side — the
// authoritative pass regardless of what the frontend already did (ADR-0010).
// Scope is deliberately narrow: JPEG (lossy, quality 82) and PNG (lossless,
// max compression), both downscaled if oversized. Every other file type is
// returned byte-identical — see ADR-0010 for why documents are never
// recompressed here.
package compress

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"

	"golang.org/x/image/draw"
)

const (
	// MaxDimension caps the longest side of an evidence photo — WO evidence
	// photos never need to exceed this for screen/print use.
	MaxDimension = 2000
	jpegQuality  = 82
)

// Image re-compresses JPEG/PNG bytes per the rules above. For any other
// mimeType it returns the input unchanged — this is intentional, not a
// fallback for an unhandled case (see ADR-0010).
func Image(data []byte, mimeType string) ([]byte, error) {
	switch mimeType {
	case "image/jpeg", "image/jpg":
		return recompressJPEG(data)
	case "image/png":
		return recompressPNG(data)
	default:
		return data, nil
	}
}

func recompressJPEG(data []byte) ([]byte, error) {
	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		// Not a decodable JPEG (or already-corrupt) — pass through rather
		// than fail the whole upload; storing the original is safer than
		// rejecting evidence that just happens to be an unusual encoding.
		return data, nil //nolint:nilerr
	}
	img = downscale(img)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func recompressPNG(data []byte) ([]byte, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return data, nil //nolint:nilerr
	}
	img = downscale(img)

	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	var out bytes.Buffer
	if err := encoder.Encode(&out, img); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// downscale resizes img so its longest side is at most MaxDimension,
// preserving aspect ratio exactly — never crops, never stretches. Images
// already within bounds are returned unchanged.
func downscale(img image.Image) image.Image {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= MaxDimension && height <= MaxDimension {
		return img
	}

	var newWidth, newHeight int
	if width >= height {
		newWidth = MaxDimension
		newHeight = int(float64(height) * float64(MaxDimension) / float64(width))
	} else {
		newHeight = MaxDimension
		newWidth = int(float64(width) * float64(MaxDimension) / float64(height))
	}

	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
	return dst
}
