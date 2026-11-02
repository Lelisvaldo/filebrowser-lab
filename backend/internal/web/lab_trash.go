package web

// Lixeira (filebrowser-lab, HL-129): cada usuário vê o que apagou; o administrador vê tudo.

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"

	"github.com/gtsteffaniak/filebrowser/backend/internal/adapters/fs/files"
)

type labTrashRef struct {
	Source string `json:"source"`
	ID     string `json:"id"`
}

type labTrashFail struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

func labTrashVisible(d *Context, it files.TrashItem) bool {
	return d.User.Permissions.Admin || it.By == d.User.Username
}

// GET /api/lab/trash
func labTrashListHandler(w http.ResponseWriter, r *http.Request, d *Context) (int, error) {
	out := []files.TrashItem{}
	for _, s := range files.TrashSources() {
		for _, it := range files.TrashList(s.Root, s.Name) {
			if labTrashVisible(d, it) {
				out = append(out, it)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DeletedAt > out[j].DeletedAt })
	return labJSON(w, map[string]any{"items": out, "days": files.LabTrashDays(), "admin": d.User.Permissions.Admin, "drive": d.User.Permissions.Admin && labCloudDriveConnected()})
}

// item pedido, se o usuário pode mexer nele (dono ou administrador)
func labTrashAuth(d *Context, ref labTrashRef) (root string, it *files.TrashItem, err error) {
	root, ok := files.TrashSourceRoot(ref.Source)
	if !ok {
		return "", nil, fmt.Errorf("esta pasta não tem lixeira")
	}
	it, err = files.TrashGet(root, ref.Source, ref.ID)
	if err != nil {
		return "", nil, err
	}
	if !labTrashVisible(d, *it) {
		return "", nil, fmt.Errorf("este item não é seu")
	}
	return root, it, nil
}

func labTrashBody(r *http.Request, v any) error {
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(v)
}

// POST /api/lab/trash/restore   {items:[{source,id}]}
func labTrashRestoreHandler(w http.ResponseWriter, r *http.Request, d *Context) (int, error) {
	var in struct {
		Items []labTrashRef `json:"items"`
	}
	if err := labTrashBody(r, &in); err != nil || len(in.Items) == 0 {
		return http.StatusBadRequest, fmt.Errorf("pedido inválido")
	}
	restored, failed := 0, []labTrashFail{}
	for _, ref := range in.Items {
		root, it, err := labTrashAuth(d, ref)
		if err == nil {
			if p, e := effectiveFilePerms(d, ref.Source); e != nil || !p.Create {
				err = fmt.Errorf("sem permissão para criar em %s", ref.Source)
			}
		}
		if err == nil {
			var final string
			if final, err = files.TrashRestore(root, ref.Source, it.ID); err == nil {
				files.TrashRefresh(ref.Source, final, it.IsDir)
				restored++
				continue
			}
		}
		failed = append(failed, labTrashFail{ID: ref.ID, Message: err.Error()})
	}
	return labJSON(w, map[string]any{"restored": restored, "failed": failed})
}

// POST /api/lab/trash/purge   {items:[{source,id}]} ou {all:true}
func labTrashPurgeHandler(w http.ResponseWriter, r *http.Request, d *Context) (int, error) {
	var in struct {
		Items []labTrashRef `json:"items"`
		All   bool          `json:"all"`
	}
	if err := labTrashBody(r, &in); err != nil {
		return http.StatusBadRequest, fmt.Errorf("pedido inválido")
	}
	if in.All {
		for _, s := range files.TrashSources() {
			for _, it := range files.TrashList(s.Root, s.Name) {
				if labTrashVisible(d, it) {
					in.Items = append(in.Items, labTrashRef{Source: s.Name, ID: it.ID})
				}
			}
		}
	}
	purged, failed := 0, []labTrashFail{}
	for _, ref := range in.Items {
		root, it, err := labTrashAuth(d, ref)
		if err == nil {
			err = files.TrashPurge(root, it.ID)
		}
		if err != nil {
			failed = append(failed, labTrashFail{ID: ref.ID, Message: err.Error()})
			continue
		}
		purged++
	}
	return labJSON(w, map[string]any{"purged": purged, "failed": failed})
}
