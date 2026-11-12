package fileutils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLabProgressCopy(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "origem")
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "a.bin"), make([]byte, 3<<20), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "sub", "b.bin"), make([]byte, 1500), 0o644); err != nil {
		t.Fatal(err)
	}
	p := StartJob("job-teste", 7, 1)
	if p == nil {
		t.Fatal("StartJob devolveu nil")
	}
	if GetJob("job-teste", 8) != nil {
		t.Fatal("outro usuário não pode ver o job")
	}
	p.BeginItem(src)
	if s := p.Snapshot(); s.Total != 3<<20+1500 || s.Item != 1 || s.Name != "origem" {
		t.Fatalf("snapshot inicial errado: %+v", s)
	}
	if err := CopyFile(src, filepath.Join(dir, "destino")); err != nil {
		t.Fatal(err)
	}
	if s := p.Snapshot(); s.Done != s.Total {
		t.Fatalf("depois da cópia done=%d total=%d", s.Done, s.Total)
	}
	p.EndItem(src)
	p.Finish("job-teste")
	if !p.Snapshot().Finished {
		t.Fatal("deveria estar terminado")
	}
	// sem job registrado a cópia segue pelo caminho normal
	if err := CopyFile(src, filepath.Join(dir, "destino2")); err != nil {
		t.Fatal(err)
	}
	if progressFor(src) != nil {
		t.Fatal("origem deveria estar desregistrada")
	}
	var nilp *Progress
	nilp.BeginItem(src) // nil-safe
	nilp.EndItem(src)
}
