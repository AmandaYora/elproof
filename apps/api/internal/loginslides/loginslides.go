// Package loginslides is a one-off, re-runnable operator command that
// compresses the login page's 3 fixed marketing photos and uploads them to
// object storage under a stable key per slide. Not request-path code — same
// "administrative tooling" category as internal/adminseed, invoked from
// cmd/server's own dispatch rather than a request handler.
package loginslides

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"elproof/internal/shared/compress"
	"elproof/internal/shared/storage"
)

const keyPrefix = "elproof/marketing/login-slides/"

// flattenJPEGQuality is used only for the one PNG source below, converting it
// to JPEG before the shared compress pipeline runs. This is intentionally not
// done inside internal/shared/compress: that package's PNG path stays
// lossless for callers (tenant logos) that may need real transparency: this
// package's one source photo has none, and a quality re-encode is meaningfully
// lighter for a full-bleed background photo than PNG's lossless mode.
const flattenJPEGQuality = 92

type slide struct {
	sourceFile string
	key        string
}

var slides = []slide{
	{"ChatGPT Image Jul 30, 2026 at 03_30_38 PM.png", keyPrefix + "slide-01.jpg"},
	{"SRNTY-440.jpg", keyPrefix + "slide-02.jpg"},
	{"SRNTY-465.jpg", keyPrefix + "slide-03.jpg"},
}

// Run reads each fixed source file from dir, compresses it, and uploads it to
// object storage under its fixed key. Safe to re-run: Save overwrites the
// same key, so swapping a photo later just means replacing the file under the
// same source name and running this again.
func Run(ctx context.Context, storageClient *storage.Client, dir string) error {
	for _, s := range slides {
		path := filepath.Join(dir, s.sourceFile)
		original, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %q: %w", path, err)
		}

		data := original
		if strings.HasSuffix(strings.ToLower(s.sourceFile), ".png") {
			data, err = flattenToJPEG(original)
			if err != nil {
				return fmt.Errorf("flatten %q to jpeg: %w", path, err)
			}
		}

		compressed, err := compress.Image(data, "image/jpeg")
		if err != nil {
			return fmt.Errorf("compress %q: %w", path, err)
		}

		if _, err := storageClient.Save(ctx, s.key, compressed, "image/jpeg"); err != nil {
			return fmt.Errorf("upload %q as %q: %w", path, s.key, err)
		}

		log.Printf("upload-login-slides: %s -> %s (%d bytes -> %d bytes)", s.sourceFile, s.key, len(original), len(compressed))
	}
	return nil
}

func flattenToJPEG(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: flattenJPEGQuality}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
