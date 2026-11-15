// Package filetype checks that the bytes of an uploaded file really are the
// type the client declared. A file name or a Content-Type is only a claim; a
// renamed binary declared as image/jpeg must be refused before it is stored and
// shown to anyone as proof (QA EP-02).
//
// The check reads the file's signature (magic bytes) and, for the raster image
// types Go can parse, the image header too, so a file that merely starts with
// the right bytes but cannot be opened as a picture is refused as well.
package filetype

import (
	"bytes"
	"image"
	_ "image/jpeg" // register decoders for DecodeConfig
	_ "image/png"
	"io"
	"mime"
	"strings"

	_ "golang.org/x/image/webp"
)

// HeadSize is how many leading bytes are enough to recognise every signature
// this package knows. Callers that stream from storage read at least this much.
const HeadSize = 1024

// headerReadLimit bounds how much of a stream DecodeConfig may read: a JPEG's
// size marker can sit after a large EXIF block, but never past this.
const headerReadLimit = 512 * 1024

// Known reports whether this package can verify the given MIME type. For an
// unknown type Matches has nothing to compare against.
func Known(mimeType string) bool {
	switch normalize(mimeType) {
	case "image/jpeg", "image/jpg", "image/png", "image/webp", "image/heic", "image/heif", "application/pdf":
		return true
	}
	return false
}

// Matches reports whether data is a file of the declared type. Unknown types
// match (there is no signature to check); callers that only accept a fixed
// list check the list themselves.
func Matches(data []byte, mimeType string) bool {
	return MatchesReader(bytes.NewReader(data), mimeType)
}

// MatchesReader is Matches for a stream (an object in storage): it reads the
// signature and, for JPEG/PNG/WebP, at most headerReadLimit bytes of header.
func MatchesReader(r io.Reader, mimeType string) bool {
	m := normalize(mimeType)
	if !Known(m) {
		return true
	}
	head := make([]byte, HeadSize)
	n, err := io.ReadFull(r, head)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return false
	}
	head = head[:n]
	if !signatureMatches(head, m) {
		return false
	}
	switch m {
	case "image/jpeg", "image/jpg", "image/png", "image/webp":
		// The signature is right; the header must also parse as that image.
		rest := io.MultiReader(bytes.NewReader(head), io.LimitReader(r, headerReadLimit))
		cfg, format, err := image.DecodeConfig(rest)
		if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
			return false
		}
		return formatMatches(format, m)
	}
	return true
}

func signatureMatches(head []byte, m string) bool {
	switch m {
	case "image/jpeg", "image/jpg":
		return bytes.HasPrefix(head, []byte{0xFF, 0xD8, 0xFF})
	case "image/png":
		return bytes.HasPrefix(head, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'})
	case "image/webp":
		return len(head) >= 12 && bytes.Equal(head[0:4], []byte("RIFF")) && bytes.Equal(head[8:12], []byte("WEBP"))
	case "image/heic", "image/heif":
		return isHEIF(head)
	case "application/pdf":
		// The PDF header may follow a few junk bytes; readers accept it within the first 1024.
		return bytes.Contains(head, []byte("%PDF-"))
	}
	return false
}

// isHEIF: an ISO-BMFF "ftyp" box whose major or compatible brands name HEIF/HEIC.
func isHEIF(head []byte) bool {
	if len(head) < 12 || !bytes.Equal(head[4:8], []byte("ftyp")) {
		return false
	}
	boxSize := int(head[0])<<24 | int(head[1])<<16 | int(head[2])<<8 | int(head[3])
	if boxSize < 16 || boxSize > len(head) {
		boxSize = len(head)
	}
	brands := map[string]bool{"heic": true, "heix": true, "hevc": true, "hevx": true, "heim": true, "heis": true, "mif1": true, "msf1": true}
	if brands[string(head[8:12])] {
		return true
	}
	for i := 16; i+4 <= boxSize; i += 4 {
		if brands[string(head[i:i+4])] {
			return true
		}
	}
	return false
}

func formatMatches(format, m string) bool {
	switch m {
	case "image/jpeg", "image/jpg":
		return format == "jpeg"
	case "image/png":
		return format == "png"
	case "image/webp":
		return format == "webp"
	}
	return false
}

func normalize(m string) string {
	base, _, err := mime.ParseMediaType(strings.TrimSpace(m))
	if err != nil {
		return strings.ToLower(strings.TrimSpace(m))
	}
	return strings.ToLower(base)
}
