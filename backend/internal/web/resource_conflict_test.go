package web

import (
	"os"
	"path/filepath"
	"testing"
)

// Nome repetido ganha " (2)", " (3)"… (o original é o 1).
func TestAddVersionSuffix(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "BIOS.bin")
	_ = os.WriteFile(f, []byte("a"), 0o644)
	if got := addVersionSuffix(f); got != filepath.Join(dir, "BIOS (2).bin") {
		t.Fatalf("esperava BIOS (2).bin, veio %s", got)
	}
	_ = os.WriteFile(filepath.Join(dir, "BIOS (2).bin"), []byte("b"), 0o644)
	if got := addVersionSuffix(f); got != filepath.Join(dir, "BIOS (3).bin") {
		t.Fatalf("esperava BIOS (3).bin, veio %s", got)
	}
	if got := addVersionSuffix(filepath.Join(dir, "livre.txt")); got != filepath.Join(dir, "livre.txt") {
		t.Fatalf("nome livre não deveria mudar: %s", got)
	}
}
