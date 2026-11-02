package web

// Computadores do Google Drive (filebrowser-lab): a seção "Computadores" do Drive não aparece na raiz pela API,
// mas cada computador é uma pasta sua sem pasta-pai. Achamos essas pastas e o rclone as junta com "Meu Drive"
// numa pasta só (backend "combine"): GoogleDrive/Meu Drive e GoogleDrive/Computadores/<nome>, como o próprio Google Drive mostra.

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

type labCloudPasta struct {
	Name     string `json:"name"`
	FolderID string `json:"folderId"`
}

var labCloudNomeBad = regexp.MustCompile(`[^\p{L}\p{N} ._()+@-]+`)

// nome seguro para um caminho do combine (sem aspas, =, :, / nem controles)
func labCloudPastaName(name, id string) string {
	n := strings.TrimSpace(labCloudNomeBad.ReplaceAllString(name, "_"))
	if n == "" {
		n = "Computador"
	}
	return n
}

func labCloudShortID(id string) string {
	id = regexp.MustCompile(`[^A-Za-z0-9_-]`).ReplaceAllString(id, "")
	if len(id) > 8 {
		id = id[:8]
	}
	return id
}

// trecho do rclone.conf: um alias por computador, um combine só dos computadores e a pasta visível (combine).
// Dois níveis porque o combine do rclone não aceita "Computadores/PSD" como nome de pasta.
func labCloudCombine(p *labCloudProv, comps []labCloudPasta) string {
	var sb strings.Builder
	ups := []string{`"Meu Drive=` + p.section() + `:"`}
	if len(comps) > 0 {
		var pcs []string
		for _, c := range comps {
			al := p.section() + "_" + labCloudShortID(c.FolderID)
			sb.WriteString("[" + al + "]\ntype = alias\nremote = " + p.section() + ",root_folder_id=" + c.FolderID + ":\n\n")
			pcs = append(pcs, `"`+c.Name+"="+al+`:"`)
		}
		sb.WriteString("[" + p.section() + "_computadores]\ntype = combine\nupstreams = " + strings.Join(pcs, " ") + "\n\n")
		ups = append(ups, `"Computadores=`+p.section()+`_computadores:"`)
	}
	sb.WriteString("[" + p.Folder + "]\ntype = combine\nupstreams = " + strings.Join(ups, " ") + "\n\n")
	return sb.String()
}

// pastas suas (no Drive) que não estão dentro de nenhuma outra: são os computadores
func labCloudFindComputers(p *labCloudProv, app labCloudApp) ([]labCloudPasta, error) {
	base := p.override("FILES", "https://www.googleapis.com/drive/v3/files")
	q := "mimeType='application/vnd.google-apps.folder' and 'me' in owners and trashed=false"
	var out []labCloudPasta
	seen := map[string]bool{}
	pageToken := ""
	for page := 0; page < 30; page++ {
		u := base + "?" + url.Values{"q": {q}, "fields": {"nextPageToken,files(id,name,parents)"}, "pageSize": {"1000"}, "pageToken": {pageToken}}.Encode()
		var res struct {
			NextPageToken string `json:"nextPageToken"`
			Files         []struct {
				ID      string   `json:"id"`
				Name    string   `json:"name"`
				Parents []string `json:"parents"`
			} `json:"files"`
		}
		var lastErr error
		for try := 0; try < 2; try++ {
			tok, err := labCloudFreshAccess(p, app, try == 1)
			if err != nil {
				return nil, err
			}
			req, _ := http.NewRequest(http.MethodGet, u, nil)
			req.Header.Set("Authorization", "Bearer "+tok)
			resp, err := labCloudHTTP.Do(req)
			if err != nil {
				lastErr = fmt.Errorf("sem conexão com o %s", p.Name)
				continue
			}
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
			resp.Body.Close()
			if resp.StatusCode == http.StatusUnauthorized {
				lastErr = fmt.Errorf("o %s recusou o login; conecte de novo", p.Name)
				continue
			}
			if resp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("o %s respondeu %d", p.Name, resp.StatusCode)
			}
			if err := json.Unmarshal(body, &res); err != nil {
				return nil, fmt.Errorf("resposta inválida do %s", p.Name)
			}
			lastErr = nil
			break
		}
		if lastErr != nil {
			return nil, lastErr
		}
		for _, f := range res.Files {
			if len(f.Parents) == 0 && f.ID != "" && !seen[f.ID] {
				seen[f.ID] = true
				out = append(out, labCloudPasta{Name: f.Name, FolderID: f.ID})
			}
		}
		if res.NextPageToken == "" {
			break
		}
		pageToken = res.NextPageToken
	}
	// nomes seguros e sem repetição (se dois computadores tiverem o mesmo nome, ganham o começo do ID)
	count := map[string]int{}
	for i := range out {
		out[i].Name = labCloudPastaName(out[i].Name, out[i].FolderID)
		count[out[i].Name]++
	}
	for i := range out {
		if count[out[i].Name] > 1 {
			out[i].Name += " (" + labCloudShortID(out[i].FolderID) + ")"
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// POST /api/lab/cloud/computadores?id=drive: procura os computadores e atualiza a pasta GoogleDrive (só remonta se mudou)
func labCloudComputersHandler(w http.ResponseWriter, r *http.Request, d *Context) (int, error) {
	if err := labCloudOn(); err != nil {
		return http.StatusNotFound, err
	}
	id := r.URL.Query().Get("id")
	p := labCloudProvByID(id)
	if p == nil || p.RcloneType != "drive" {
		return http.StatusBadRequest, fmt.Errorf("só o Google Drive tem computadores")
	}
	labCloudMu.Lock()
	defer labCloudMu.Unlock()
	st := labCloudLoad()
	if _, on := st.Conns[id]; !on {
		return http.StatusPreconditionFailed, fmt.Errorf("conecte o %s antes", p.Name)
	}
	comps, err := labCloudFindComputers(p, st.Apps[id])
	if err != nil {
		return http.StatusBadGateway, err
	}
	changed := fmt.Sprint(comps) != fmt.Sprint(st.Pastas[id])
	if changed {
		if st.Pastas == nil {
			st.Pastas = map[string][]labCloudPasta{}
		}
		if len(comps) == 0 {
			delete(st.Pastas, id)
		} else {
			st.Pastas[id] = comps
		}
		if err := labCloudSave(st); err != nil {
			return http.StatusInternalServerError, err
		}
	}
	return labJSON(w, map[string]any{"status": "ok", "changed": changed, "computers": comps})
}
