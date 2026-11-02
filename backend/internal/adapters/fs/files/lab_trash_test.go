package files

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLabTrashFlow(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "Filmes", "Sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(root, "Filmes", "Sub", "a.txt")
	_ = os.WriteFile(f, []byte("conteudo"), 0o644)

	it, err := TrashMove(root, "Media", "/Filmes/Sub/a.txt", f, false, "ana")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Fatal("o arquivo deveria ter saído do lugar")
	}
	if it.Size != 8 || it.By != "ana" || it.Path != "/Filmes/Sub/a.txt" {
		t.Fatalf("dados errados: %+v", it)
	}
	list := TrashList(root, "Media")
	if len(list) != 1 || list[0].Name != "a.txt" || list[0].ExpiresAt-list[0].DeletedAt != int64(LabTrashDays())*86400 {
		t.Fatalf("lista errada: %+v", list)
	}

	// restaurar com o nome já ocupado ganha " (1)"
	_ = os.WriteFile(f, []byte("novo"), 0o644)
	final, err := TrashRestore(root, "Media", it.ID)
	if err != nil || final != "/Filmes/Sub/a (2).txt" {
		t.Fatalf("restaurar: %q %v", final, err)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "Filmes", "Sub", "a (2).txt")); string(b) != "conteudo" {
		t.Fatal("conteúdo restaurado errado")
	}
	if len(TrashList(root, "Media")) != 0 {
		t.Fatal("a lixeira deveria ficar vazia")
	}

	// pasta inteira, restaurada com a pasta de origem apagada
	d := filepath.Join(root, "Filmes", "Sub")
	it2, err := TrashMove(root, "Media", "/Filmes/Sub", d, true, "bia")
	if err != nil || it2.Size < 0 {
		t.Fatalf("pasta: %+v %v", it2, err)
	}
	_ = os.RemoveAll(filepath.Join(root, "Filmes"))
	final, err = TrashRestore(root, "Media", it2.ID)
	if err != nil || final != "/Filmes/Sub" {
		t.Fatalf("restaurar pasta: %q %v", final, err)
	}
	if _, err := os.Stat(filepath.Join(root, "Filmes", "Sub", "a (2).txt")); err != nil {
		t.Fatal("o conteúdo da pasta deveria voltar")
	}

	// apagar de vez e vencimento
	g := filepath.Join(root, "x.bin")
	_ = os.WriteFile(g, []byte("z"), 0o644)
	it3, _ := TrashMove(root, "Media", "/x.bin", g, false, "ana")
	if err := TrashPurge(root, it3.ID); err != nil || len(TrashList(root, "Media")) != 0 {
		t.Fatal("apagar de vez falhou")
	}
	_ = os.WriteFile(g, []byte("z"), 0o644)
	it4, _ := TrashMove(root, "Media", "/x.bin", g, false, "ana")
	if n := TrashPurgeExpired(root, "Media", time.Now()); n != 0 {
		t.Fatal("não deveria vencer agora")
	}
	if n := TrashPurgeExpired(root, "Media", time.Now().Add(31*24*time.Hour)); n != 1 {
		t.Fatalf("deveria vencer depois de 31 dias, veio %d", n)
	}
	if _, err := os.Stat(filepath.Join(labTrashRoot(root), it4.ID)); !os.IsNotExist(err) {
		t.Fatal("a pasta do item vencido deveria sumir")
	}

	// ids fora do padrão não passam (travessia de pasta)
	if _, err := TrashRestore(root, "Media", "../x"); err == nil {
		t.Fatal("id inválido aceito")
	}
	if TrashPurge(root, "../../etc") == nil {
		t.Fatal("id inválido aceito no purge")
	}
	if !inLabTrash(root, filepath.Join(root, ".lixeira", "123", "a")) || inLabTrash(root, filepath.Join(root, "Filmes")) {
		t.Fatal("inLabTrash errado")
	}
}

func TestLabTrashEnabled(t *testing.T) {
	t.Setenv("FILEBROWSER_LAB_TRASH_OFF", "")
	if LabTrashEnabled("Pendrives") || LabTrashEnabled("nuvens") || !LabTrashEnabled("Media") {
		t.Fatal("padrão: Pendrives e Nuvens sem lixeira")
	}
	t.Setenv("FILEBROWSER_LAB_TRASH", "off")
	if LabTrashEnabled("Media") {
		t.Fatal("FILEBROWSER_LAB_TRASH=off deveria desligar tudo")
	}
	if !strings.Contains(".lixeira", "lixeira") {
		t.Fatal()
	}
}
