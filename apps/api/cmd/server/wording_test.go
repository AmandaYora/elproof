package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A refusal never names who would be allowed (knowledge/DESIGN.md §7, 18 Nov 2026): a WO member or a client who reaches
// an internal endpoint gets "Anda tidak memiliki akses untuk aksi ini", not a hint that ElProof's own team uses the same
// app. Messages only the platform team sees (inside the Platform Console) may still say "admin platform".
var revealsInternalTeam = regexp.MustCompile(`"[^"]*(?i:hanya (admin|super admin|tim) (platform|elproof)|(hanya )?tersedia untuk admin platform|ke endpoint ini)[^"]*"`)

func TestRefusalsDoNotNameTheInternalTeam(t *testing.T) {
	root := filepath.Join("..", "..", "internal")
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(src), "\n") {
			if m := revealsInternalTeam.FindString(line); m != "" {
				t.Errorf("%s:%d: %s (use \"Anda tidak memiliki akses untuk aksi ini\")", path, i+1, m)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
