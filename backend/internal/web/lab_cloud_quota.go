package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Espaço usado e livre de cada nuvem, perguntado ao provedor com o token que o rclone mantém no rclone.conf.
// O token renovado aqui fica só na memória: quem grava no rclone.conf é o rclone.

type labCloudQuotaOut struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Used  int64  `json:"used"`
	Total int64  `json:"total"` // 0 = ilimitado ou desconhecido
	Err   string `json:"err,omitempty"`
}

var (
	labCloudQuotaMu    sync.Mutex
	labCloudQuotaCache = map[string]struct {
		at  time.Time
		out labCloudQuotaOut
	}{}
	labCloudAccess = map[string]struct {
		tok string
		exp time.Time
	}{}
)

// token guardado pelo rclone para a seção [Folder]
func labCloudConfToken(p *labCloudProv) (access, refresh string, exp time.Time) {
	b, err := os.ReadFile(filepath.Join(labCloudDir(), "rclone.conf"))
	if err != nil {
		return
	}
	in := false
	for _, ln := range strings.Split(string(b), "\n") {
		ln = strings.TrimSpace(ln)
		if strings.HasPrefix(ln, "[") {
			in = ln == "["+p.Folder+"]"
			continue
		}
		if in && strings.HasPrefix(ln, "token") {
			var t struct {
				A string `json:"access_token"`
				R string `json:"refresh_token"`
				E string `json:"expiry"`
			}
			if json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(ln, "token"), " ="))), &t) == nil {
				e, _ := time.Parse(time.RFC3339, t.E)
				return t.A, t.R, e
			}
		}
	}
	return
}

// token de acesso válido: o do rclone.conf, ou um novo pedido com o refresh_token
func labCloudFreshAccess(p *labCloudProv, app labCloudApp, force bool) (string, error) {
	if c, ok := labCloudAccess[p.ID]; ok && !force && time.Now().Before(c.exp) {
		return c.tok, nil
	}
	acc, ref, exp := labCloudConfToken(p)
	if !force && acc != "" && time.Now().Add(60*time.Second).Before(exp) {
		return acc, nil
	}
	if ref == "" {
		return "", fmt.Errorf("sem token; conecte de novo")
	}
	form := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {ref}, "client_id": {app.ClientID}, "client_secret": {app.ClientSecret}}
	resp, err := labCloudHTTP.PostForm(p.override("TOKEN", p.Token), form)
	if err != nil {
		return "", fmt.Errorf("sem conexão com o %s", p.Name)
	}
	defer resp.Body.Close()
	var t labCloudTok
	if json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&t) != nil || t.AccessToken == "" {
		return "", fmt.Errorf("o %s recusou renovar o login; conecte de novo", p.Name)
	}
	labCloudAccess[p.ID] = struct {
		tok string
		exp time.Time
	}{t.AccessToken, time.Now().Add(time.Duration(max(t.ExpiresIn, 120)-60) * time.Second)}
	return t.AccessToken, nil
}

func labCloudQuotaURL(p *labCloudProv) (method, u string) {
	switch p.ID {
	case "onedrive":
		return http.MethodGet, p.override("QUOTA", "https://graph.microsoft.com/v1.0/me/drive")
	case "drive":
		return http.MethodGet, p.override("QUOTA", "https://www.googleapis.com/drive/v3/about?fields=storageQuota")
	case "dropbox":
		return http.MethodPost, p.override("QUOTA", "https://api.dropboxapi.com/2/users/get_space_usage")
	}
	return "", ""
}

func labCloudNum(v any) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case string:
		var n int64
		fmt.Sscan(x, &n)
		return n
	}
	return 0
}

func labCloudQuotaOne(p *labCloudProv, app labCloudApp) labCloudQuotaOut {
	out := labCloudQuotaOut{ID: p.ID, Name: p.Name}
	method, u := labCloudQuotaURL(p)
	if u == "" {
		out.Err = "indisponível para este serviço"
		return out
	}
	for try := 0; try < 2; try++ {
		tok, err := labCloudFreshAccess(p, app, try == 1)
		if err != nil {
			out.Err = err.Error()
			return out
		}
		req, _ := http.NewRequest(method, u, nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		resp, err := labCloudHTTP.Do(req)
		if err != nil {
			out.Err = "sem conexão com o " + p.Name
			return out
		}
		var m map[string]any
		derr := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&m)
		resp.Body.Close()
		if resp.StatusCode == http.StatusUnauthorized && try == 0 {
			continue // token vencido antes da hora: renova e tenta de novo
		}
		if resp.StatusCode != http.StatusOK || derr != nil {
			out.Err = fmt.Sprintf("o %s respondeu %d", p.Name, resp.StatusCode)
			return out
		}
		switch p.ID {
		case "onedrive":
			if q, ok := m["quota"].(map[string]any); ok {
				out.Used, out.Total = labCloudNum(q["used"]), labCloudNum(q["total"])
			}
		case "drive":
			if q, ok := m["storageQuota"].(map[string]any); ok {
				out.Used, out.Total = labCloudNum(q["usage"]), labCloudNum(q["limit"])
			}
		case "dropbox":
			out.Used = labCloudNum(m["used"])
			if a, ok := m["allocation"].(map[string]any); ok {
				out.Total = labCloudNum(a["allocated"])
			}
		}
		return out
	}
	out.Err = "login vencido; conecte de novo"
	return out
}

// GET /api/lab/cloud/quota   -> {items:[...]}  (cache de 5 min; ?force=1 ignora)
func labCloudQuotaHandler(w http.ResponseWriter, r *http.Request, d *Context) (int, error) {
	if err := labCloudOn(); err != nil {
		return http.StatusNotFound, err
	}
	force := r.URL.Query().Get("force") != ""
	labCloudMu.Lock()
	st := labCloudLoad()
	labCloudMu.Unlock()
	labCloudQuotaMu.Lock()
	defer labCloudQuotaMu.Unlock()
	items := []labCloudQuotaOut{}
	for i := range labCloudProvs {
		p := &labCloudProvs[i]
		if _, on := st.Conns[p.ID]; !on || p.Kind != "oauth" {
			continue
		}
		if c, ok := labCloudQuotaCache[p.ID]; ok && !force && time.Since(c.at) < 5*time.Minute && c.out.Err == "" {
			items = append(items, c.out)
			continue
		}
		o := labCloudQuotaOne(p, st.Apps[p.ID])
		labCloudQuotaCache[p.ID] = struct {
			at  time.Time
			out labCloudQuotaOut
		}{time.Now(), o}
		items = append(items, o)
	}
	return labJSON(w, map[string]any{"items": items})
}
