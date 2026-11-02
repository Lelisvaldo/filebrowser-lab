package web

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gtsteffaniak/filebrowser/backend/internal/adapters/fs/files"
	"github.com/gtsteffaniak/filebrowser/backend/internal/database/users"
	"github.com/gtsteffaniak/filebrowser/backend/pkg/settings"
)

// Cada usuário vê só o que apagou; o administrador vê tudo; pendrives e nuvens não têm lixeira.
func TestLabTrashVisibility(t *testing.T) {
	root := t.TempDir()
	usb := t.TempDir()
	old := settings.Config.Server.SourceMap
	settings.Config.Server.SourceMap = map[string]*settings.Source{
		root: {Name: "Media", Path: root},
		usb:  {Name: "Pendrives", Path: usb},
	}
	defer func() { settings.Config.Server.SourceMap = old }()

	for _, f := range []struct{ name, by string }{{"a.txt", "ana"}, {"b.txt", "bia"}} {
		p := filepath.Join(root, f.name)
		_ = os.WriteFile(p, []byte("x"), 0o644)
		if _, err := files.TrashMove(root, "Media", "/"+f.name, p, false, f.by); err != nil {
			t.Fatal(err)
		}
	}
	list := func(u *users.User) []files.TrashItem {
		w := httptest.NewRecorder()
		if _, err := labTrashListHandler(w, httptest.NewRequest("GET", "/lab/trash", nil), &Context{User: u}); err != nil {
			t.Fatal(err)
		}
		var out struct {
			Items []files.TrashItem `json:"items"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return out.Items
	}
	if got := list(&users.User{Username: "ana"}); len(got) != 1 || got[0].Name != "a.txt" {
		t.Fatalf("ana deveria ver só o dela: %+v", got)
	}
	if got := list(&users.User{Username: "bia"}); len(got) != 1 || got[0].Name != "b.txt" {
		t.Fatalf("bia deveria ver só o dela: %+v", got)
	}
	if got := list(&users.User{Username: "cris"}); len(got) != 0 {
		t.Fatalf("cris não apagou nada: %+v", got)
	}
	adm := &users.User{Username: "root"}
	adm.Permissions.Admin = true
	if got := list(adm); len(got) != 2 {
		t.Fatalf("o administrador deveria ver os dois: %+v", got)
	}
	// ana não consegue mexer no item da bia
	var bid string
	for _, it := range files.TrashList(root, "Media") {
		if it.By == "bia" {
			bid = it.ID
		}
	}
	if _, _, err := labTrashAuth(&Context{User: &users.User{Username: "ana"}}, labTrashRef{Source: "Media", ID: bid}); err == nil {
		t.Fatal("ana não deveria poder mexer no item da bia")
	}
	if _, _, err := labTrashAuth(&Context{User: &users.User{Username: "bia"}}, labTrashRef{Source: "Media", ID: bid}); err != nil {
		t.Fatalf("bia deveria poder mexer no dela: %v", err)
	}
	// pendrive não tem lixeira
	if _, _, err := labTrashAuth(&Context{User: adm}, labTrashRef{Source: "Pendrives", ID: bid}); err == nil {
		t.Fatal("pendrive não deveria ter lixeira")
	}
}
