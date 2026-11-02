package web

// Lixeira do Google Drive (filebrowser-lab, HL-129): quem apaga em Nuvens/GoogleDrive já manda o item para a
// lixeira do Drive (o rclone usa a lixeira por padrão). Esta tela lista essa lixeira, restaura e apaga de vez.
// Só administrador: a conta do Drive é única e o Drive não diz qual usuário do filebrowser apagou.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
)

type labCloudTrashItem struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsDir     bool   `json:"isDir"`
	Size      int64  `json:"size"`
	DeletedAt string `json:"deletedAt,omitempty"`
}

var labCloudDriveID = regexp.MustCompile(`^[A-Za-z0-9_-]{10,100}$`)

func labCloudDriveConnected() bool {
	if labCloudDir() == "" {
		return false
	}
	labCloudMu.Lock()
	defer labCloudMu.Unlock()
	_, on := labCloudLoad().Conns["drive"]
	return on
}

// chamada à API do Drive com o token do usuário conectado (renova uma vez se o token foi recusado)
func labCloudDriveCall(method, rawURL string, body any) ([]byte, int, error) {
	p := labCloudProvByID("drive")
	labCloudMu.Lock()
	app := labCloudLoad().Apps["drive"]
	labCloudMu.Unlock()
	var payload []byte
	if body != nil {
		payload, _ = json.Marshal(body)
	}
	var lastErr error
	for try := 0; try < 2; try++ {
		tok, err := labCloudFreshAccess(p, app, try == 1)
		if err != nil {
			return nil, 0, err
		}
		req, _ := http.NewRequest(method, rawURL, bytes.NewReader(payload))
		req.Header.Set("Authorization", "Bearer "+tok)
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := labCloudHTTP.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("sem conexão com o Google Drive")
			continue
		}
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()
		if resp.StatusCode == http.StatusUnauthorized {
			lastErr = fmt.Errorf("o Google Drive recusou o login; conecte de novo")
			continue
		}
		return b, resp.StatusCode, nil
	}
	return nil, 0, lastErr
}

func labCloudDriveBase() string {
	return labCloudProvByID("drive").override("FILES", "https://www.googleapis.com/drive/v3/files")
}

func labCloudTrashOn() error {
	if err := labCloudOn(); err != nil {
		return err
	}
	if !labCloudDriveConnected() {
		return fmt.Errorf("o Google Drive não está conectado")
	}
	return nil
}

// GET /api/lab/cloud/trash
func labCloudTrashListHandler(w http.ResponseWriter, r *http.Request, d *Context) (int, error) {
	if err := labCloudTrashOn(); err != nil {
		return http.StatusPreconditionFailed, err
	}
	out := []labCloudTrashItem{}
	pageToken := ""
	for page := 0; page < 5; page++ {
		u := labCloudDriveBase() + "?" + url.Values{"q": {"trashed=true and 'me' in owners"}, "orderBy": {"modifiedTime desc"},
			"fields": {"nextPageToken,files(id,name,mimeType,size,trashedTime)"}, "pageSize": {"200"}, "pageToken": {pageToken}}.Encode()
		b, st, err := labCloudDriveCall(http.MethodGet, u, nil)
		if err != nil {
			return http.StatusBadGateway, err
		}
		if st != http.StatusOK {
			return http.StatusBadGateway, fmt.Errorf("o Google Drive respondeu %d", st)
		}
		var res struct {
			NextPageToken string `json:"nextPageToken"`
			Files         []struct {
				ID, Name, MimeType, Size, TrashedTime string
			} `json:"files"`
		}
		if json.Unmarshal(b, &res) != nil {
			return http.StatusBadGateway, fmt.Errorf("resposta inválida do Google Drive")
		}
		for _, f := range res.Files {
			sz, _ := strconv.ParseInt(f.Size, 10, 64)
			out = append(out, labCloudTrashItem{ID: f.ID, Name: f.Name, IsDir: f.MimeType == "application/vnd.google-apps.folder", Size: sz, DeletedAt: f.TrashedTime})
		}
		if res.NextPageToken == "" {
			break
		}
		pageToken = res.NextPageToken
	}
	return labJSON(w, map[string]any{"items": out})
}

type labCloudTrashIDs struct {
	IDs []string `json:"ids"`
	All bool     `json:"all"`
}

func labCloudTrashIDsOf(r *http.Request) (labCloudTrashIDs, error) {
	var in labCloudTrashIDs
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
		return in, fmt.Errorf("pedido inválido")
	}
	for _, id := range in.IDs {
		if !labCloudDriveID.MatchString(id) {
			return in, fmt.Errorf("item inválido")
		}
	}
	return in, nil
}

// POST /api/lab/cloud/trash/restore   {ids:[...]}
func labCloudTrashRestoreHandler(w http.ResponseWriter, r *http.Request, d *Context) (int, error) {
	if err := labCloudTrashOn(); err != nil {
		return http.StatusPreconditionFailed, err
	}
	in, err := labCloudTrashIDsOf(r)
	if err != nil || len(in.IDs) == 0 {
		return http.StatusBadRequest, fmt.Errorf("pedido inválido")
	}
	restored, failed := 0, []labTrashFail{}
	for _, id := range in.IDs {
		_, st, err := labCloudDriveCall(http.MethodPatch, labCloudDriveBase()+"/"+id, map[string]bool{"trashed": false})
		if err != nil || st != http.StatusOK {
			msg := fmt.Sprintf("o Google Drive respondeu %d", st)
			if err != nil {
				msg = err.Error()
			}
			failed = append(failed, labTrashFail{ID: id, Message: msg})
			continue
		}
		restored++
	}
	return labJSON(w, map[string]any{"restored": restored, "failed": failed})
}

// POST /api/lab/cloud/trash/purge   {ids:[...]} ou {all:true} (esvazia a lixeira inteira do Drive)
func labCloudTrashPurgeHandler(w http.ResponseWriter, r *http.Request, d *Context) (int, error) {
	if err := labCloudTrashOn(); err != nil {
		return http.StatusPreconditionFailed, err
	}
	in, err := labCloudTrashIDsOf(r)
	if err != nil {
		return http.StatusBadRequest, err
	}
	if in.All {
		_, st, err := labCloudDriveCall(http.MethodDelete, labCloudDriveBase()+"/trash", nil)
		if err != nil {
			return http.StatusBadGateway, err
		}
		if st != http.StatusNoContent && st != http.StatusOK {
			return http.StatusBadGateway, fmt.Errorf("o Google Drive respondeu %d", st)
		}
		return labJSON(w, map[string]any{"purged": -1, "failed": []labTrashFail{}})
	}
	if len(in.IDs) == 0 {
		return http.StatusBadRequest, fmt.Errorf("pedido inválido")
	}
	purged, failed := 0, []labTrashFail{}
	for _, id := range in.IDs {
		_, st, err := labCloudDriveCall(http.MethodDelete, labCloudDriveBase()+"/"+id, nil)
		if err != nil || (st != http.StatusNoContent && st != http.StatusOK) {
			msg := fmt.Sprintf("o Google Drive respondeu %d", st)
			if err != nil {
				msg = err.Error()
			}
			failed = append(failed, labTrashFail{ID: id, Message: msg})
			continue
		}
		purged++
	}
	return labJSON(w, map[string]any{"purged": purged, "failed": failed})
}
