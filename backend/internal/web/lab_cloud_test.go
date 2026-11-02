package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLabCloudFluxoOAuth(t *testing.T) {
	prov := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/token") {
			_ = r.ParseForm()
			if r.Form.Get("code") != "cod123" || r.Form.Get("client_secret") != "seg" {
				_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
				return
			}
			_, _ = w.Write([]byte(`{"access_token":"AT","token_type":"Bearer","refresh_token":"RT","expires_in":3600}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"drv1","driveType":"personal","owner":{"user":{"displayName":"Fulano"}}}`))
	}))
	defer prov.Close()
	dir := t.TempDir()
	t.Setenv("FILEBROWSER_LAB_CLOUD_DIR", dir)
	t.Setenv("FILEBROWSER_LAB_CLOUD_ONEDRIVE_TOKEN", prov.URL+"/token")
	t.Setenv("FILEBROWSER_LAB_CLOUD_ONEDRIVE_INFO", prov.URL+"/info")

	// sem app cadastrado, o login é recusado
	w := httptest.NewRecorder()
	if st, err := labCloudStartHandler(w, httptest.NewRequest("POST", "/lab/cloud/start?id=onedrive", nil), nil); err == nil || st != http.StatusPreconditionFailed {
		t.Fatalf("esperava 412, veio %d %v", st, err)
	}
	// cadastra o app
	w = httptest.NewRecorder()
	body := strings.NewReader(`{"clientId":"cid","clientSecret":"seg"}`)
	if _, err := labCloudAppHandler(w, httptest.NewRequest("PUT", "/lab/cloud/app?id=onedrive", body), nil); err != nil {
		t.Fatal(err)
	}
	// início do login
	w = httptest.NewRecorder()
	if _, err := labCloudStartHandler(w, httptest.NewRequest("POST", "/lab/cloud/start?id=onedrive", nil), nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(w.Body.String(), "client_id=cid") || !strings.Contains(w.Body.String(), "state=") {
		t.Fatalf("url de login sem client_id/state: %s", w.Body.String())
	}
	var state string
	for st := range labCloudPend {
		state = st
	}
	// state inválido
	w = httptest.NewRecorder()
	_, _ = labCloudCallbackHandler(w, httptest.NewRequest("GET", "/cb?code=cod123&state=xx", nil), nil)
	if !strings.Contains(w.Header().Get("Location"), "nuvem=erro") {
		t.Fatalf("state inválido deveria dar erro: %s", w.Header().Get("Location"))
	}
	// retorno certo
	w = httptest.NewRecorder()
	_, _ = labCloudCallbackHandler(w, httptest.NewRequest("GET", "/cb?code=cod123&state="+url.QueryEscape(state), nil), nil)
	if !strings.Contains(w.Header().Get("Location"), "nuvem=ok") {
		t.Fatalf("login deveria dar certo: %s", w.Header().Get("Location"))
	}
	conf, _ := os.ReadFile(filepath.Join(dir, "rclone.conf"))
	for _, want := range []string{"[OneDrive]", "type = onedrive", "drive_id = drv1", "drive_type = personal", `"refresh_token":"RT"`} {
		if !strings.Contains(string(conf), want) {
			t.Fatalf("rclone.conf sem %q:\n%s", want, conf)
		}
	}
	if fi, _ := os.Stat(filepath.Join(dir, "rclone.conf")); fi.Mode().Perm() != 0o600 {
		t.Fatalf("permissão do rclone.conf = %v", fi.Mode().Perm())
	}
	if _, err := os.Stat(filepath.Join(dir, "escrita-OneDrive")); err == nil {
		t.Fatal("nuvem nova deveria ser só leitura")
	}
	// state é de uso único
	w = httptest.NewRecorder()
	_, _ = labCloudCallbackHandler(w, httptest.NewRequest("GET", "/cb?code=cod123&state="+state, nil), nil)
	if !strings.Contains(w.Header().Get("Location"), "nuvem=erro") {
		t.Fatal("state reutilizado deveria falhar")
	}
	// libera escrita e desconecta
	w = httptest.NewRecorder()
	_, _ = labCloudModeHandler(w, httptest.NewRequest("POST", "/m?id=onedrive&readOnly=false", nil), nil)
	if _, err := os.Stat(filepath.Join(dir, "escrita-OneDrive")); err != nil {
		t.Fatal("escrita deveria estar liberada")
	}
	w = httptest.NewRecorder()
	_, _ = labCloudDisconnectHandler(w, httptest.NewRequest("DELETE", "/lab/cloud?id=onedrive", nil), nil)
	conf, _ = os.ReadFile(filepath.Join(dir, "rclone.conf"))
	if strings.Contains(string(conf), "OneDrive") {
		t.Fatalf("desconectar deveria limpar o rclone.conf:\n%s", conf)
	}
}

func TestLabCloudMega(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FILEBROWSER_LAB_CLOUD_DIR", dir)
	w := httptest.NewRecorder()
	body := strings.NewReader(`{"user":"a@b.c","pass":"s3nha"}`)
	if _, err := labCloudLoginHandler(w, httptest.NewRequest("POST", "/lab/cloud/senha?id=mega", body), nil); err != nil {
		t.Fatal(err)
	}
	conf, _ := os.ReadFile(filepath.Join(dir, "rclone.conf"))
	if !strings.Contains(string(conf), "[Mega]") || strings.Contains(string(conf), "s3nha") {
		t.Fatalf("conf do mega errada (senha deve estar ofuscada):\n%s", conf)
	}
	t.Logf("%s", conf)
}

func labCloudQuotaSetup(t *testing.T, expiry string) {
	t.Helper()
	t.Setenv("FILEBROWSER_LAB_CLOUD_DIR", t.TempDir())
	labCloudQuotaCache = map[string]struct {
		at  time.Time
		out labCloudQuotaOut
	}{}
	labCloudAccess = map[string]struct {
		tok string
		exp time.Time
	}{}
	st := labCloudState{Apps: map[string]labCloudApp{"onedrive": {ClientID: "cid", ClientSecret: "seg"}},
		Conns: map[string]labCloudInfo{"onedrive": {Account: "x"}}}
	if err := labCloudSave(st); err != nil {
		t.Fatal(err)
	}
	conf := "[OneDrive]\ntype = onedrive\ntoken = {\"access_token\":\"VELHO\",\"token_type\":\"Bearer\",\"refresh_token\":\"RT\",\"expiry\":\"" + expiry + "\"}\n"
	if err := os.WriteFile(filepath.Join(labCloudDir(), "rclone.conf"), []byte(conf), 0o600); err != nil {
		t.Fatal(err)
	}
}

func labCloudQuotaGet(t *testing.T) labCloudQuotaOut {
	t.Helper()
	w := httptest.NewRecorder()
	if _, err := labCloudQuotaHandler(w, httptest.NewRequest("GET", "/lab/cloud/quota?force=1", nil), nil); err != nil {
		t.Fatal(err)
	}
	var r struct{ Items []labCloudQuotaOut }
	if err := json.Unmarshal(w.Body.Bytes(), &r); err != nil || len(r.Items) != 1 {
		t.Fatalf("resposta inesperada: %s", w.Body.String())
	}
	return r.Items[0]
}

func TestLabCloudQuota(t *testing.T) {
	refreshes := 0
	deny := false
	prov := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/token") {
			refreshes++
			if deny {
				_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
				return
			}
			_, _ = w.Write([]byte(`{"access_token":"NOVO","expires_in":3600}`))
			return
		}
		if r.Header.Get("Authorization") != "Bearer NOVO" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"quota":{"total":1099511627776,"used":101580132352}}`))
	}))
	t.Setenv("FILEBROWSER_LAB_CLOUD_ONEDRIVE_TOKEN", prov.URL+"/token")
	t.Setenv("FILEBROWSER_LAB_CLOUD_ONEDRIVE_QUOTA", prov.URL+"/quota")

	// token vencido no rclone.conf: renova antes de perguntar
	labCloudQuotaSetup(t, "2020-01-01T00:00:00Z")
	if o := labCloudQuotaGet(t); o.Err != "" || o.Used != 101580132352 || o.Total != 1099511627776 || refreshes != 1 {
		t.Fatalf("token vencido: %+v, renovações=%d", o, refreshes)
	}
	// token que parece válido mas o provedor recusa (401): renova e tenta de novo
	refreshes = 0
	labCloudQuotaSetup(t, "2099-01-01T00:00:00Z")
	if o := labCloudQuotaGet(t); o.Err != "" || o.Used == 0 || refreshes != 1 {
		t.Fatalf("401: %+v, renovações=%d", o, refreshes)
	}
	// renovação recusada: pede para conectar de novo
	deny = true
	labCloudQuotaSetup(t, "2020-01-01T00:00:00Z")
	if o := labCloudQuotaGet(t); !strings.Contains(o.Err, "conecte de novo") {
		t.Fatalf("renovação recusada: %+v", o)
	}
	// rede caída: erro claro, sem travar nem derrubar a lista
	prov.Close()
	labCloudQuotaSetup(t, "2020-01-01T00:00:00Z")
	if o := labCloudQuotaGet(t); !strings.Contains(o.Err, "sem conexão") {
		t.Fatalf("rede caída: %+v", o)
	}
}

// Google Drive: o remoto real fica oculto (_GoogleDrive) e a pasta visível junta "Meu Drive" com "Computadores/<nome>".
func TestLabCloudDriveComputers(t *testing.T) {
	var gotQ string
	prov := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer AT" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		gotQ = r.URL.Query().Get("q")
		_, _ = w.Write([]byte(`{"files":[
			{"id":"1imhT3abcdefghijk","name":"PSD"},
			{"id":"2otherFolderIdxyz","name":"Fotos","parents":["1imhT3abcdefghijk"]},
			{"id":"3dupAAAAAAAAAAAA","name":"Casa \"=:/x"},
			{"id":"4dupBBBBBBBBBBBB","name":"Casa \"=:/x"}]}`))
	}))
	defer prov.Close()
	dir := t.TempDir()
	t.Setenv("FILEBROWSER_LAB_CLOUD_DIR", dir)
	t.Setenv("FILEBROWSER_LAB_CLOUD_DRIVE_FILES", prov.URL)
	exp := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	tok := `{"access_token":"AT","token_type":"Bearer","refresh_token":"RT","expiry":"` + exp + `"}`
	if err := os.WriteFile(filepath.Join(dir, "rclone.conf"), []byte("[_GoogleDrive]\ntoken = "+tok+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	p := labCloudProvByID("drive")
	comps, err := labCloudFindComputers(p, labCloudApp{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotQ, "'me' in owners") {
		t.Fatalf("consulta sem filtro de dono: %q", gotQ)
	}
	if len(comps) != 3 || comps[2].Name != "PSD" || !strings.HasPrefix(comps[0].Name, "Casa _x (") {
		t.Fatalf("computadores inesperados: %+v", comps)
	}

	// conf: seção oculta com o login, um alias por computador e a pasta visível (combine)
	_ = labCloudSetSection("drive", map[string]string{"type": "drive", "token": tok, "scope": "drive"})
	st := labCloudState{Apps: map[string]labCloudApp{}, Conns: map[string]labCloudInfo{"drive": {ReadOnly: false}}, Pastas: map[string][]labCloudPasta{"drive": comps[2:]}}
	if err := labCloudWriteConf(st); err != nil {
		t.Fatal(err)
	}
	conf, _ := os.ReadFile(filepath.Join(dir, "rclone.conf"))
	for _, want := range []string{"[_GoogleDrive]\n", "type = drive", "[_GoogleDrive_1imhT3ab]\ntype = alias\nremote = _GoogleDrive,root_folder_id=1imhT3abcdefghijk:",
		"[_GoogleDrive_computadores]\ntype = combine\nupstreams = \"PSD=_GoogleDrive_1imhT3ab:\"\n",
		"[GoogleDrive]\ntype = combine\nupstreams = \"Meu Drive=_GoogleDrive:\" \"Computadores=_GoogleDrive_computadores:\""} {
		if !strings.Contains(string(conf), want) {
			t.Fatalf("rclone.conf sem %q:\n%s", want, conf)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "escrita-GoogleDrive")); err != nil {
		t.Fatal("a pasta visível deveria liberar escrita")
	}
	// sem computadores, a pasta ainda tem "Meu Drive"
	st.Pastas = nil
	_ = labCloudWriteConf(st)
	conf, _ = os.ReadFile(filepath.Join(dir, "rclone.conf"))
	if !strings.Contains(string(conf), "upstreams = \"Meu Drive=_GoogleDrive:\"\n") || strings.Contains(string(conf), "alias") {
		t.Fatalf("sem computadores o conf deveria ter só Meu Drive:\n%s", conf)
	}
}

// A busca automática (sem a tela) grava os computadores e o rclone.conf no formato novo.
func TestLabCloudAutoRun(t *testing.T) {
	prov := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"files":[{"id":"1imhT3abcdefghijk","name":"PSD"},{"id":"2otherFolderIdxyz","name":"Fotos","parents":["1imhT3abcdefghijk"]}]}`))
	}))
	defer prov.Close()
	dir := t.TempDir()
	t.Setenv("FILEBROWSER_LAB_CLOUD_DIR", dir)
	t.Setenv("FILEBROWSER_LAB_CLOUD_DRIVE_FILES", prov.URL)
	exp := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	tok := `{"access_token":"AT","token_type":"Bearer","refresh_token":"RT","expiry":"` + exp + `"}`
	_ = labCloudSetSection("drive", map[string]string{"type": "drive", "token": tok})
	st := labCloudState{Apps: map[string]labCloudApp{}, Conns: map[string]labCloudInfo{"drive": {ReadOnly: true}}}
	if err := labCloudSave(st); err != nil {
		t.Fatal(err)
	}
	// o login já está no conf (como o rclone o deixa); a rotina lê o token de lá
	conf, _ := os.ReadFile(filepath.Join(dir, "rclone.conf"))
	if !strings.Contains(string(conf), "[_GoogleDrive]") {
		t.Fatalf("conf inicial sem a seção oculta:\n%s", conf)
	}
	labCloudAutoRun()
	conf, _ = os.ReadFile(filepath.Join(dir, "rclone.conf"))
	if !strings.Contains(string(conf), "\"PSD=_GoogleDrive_1imhT3ab:\"") || !strings.Contains(string(conf), "\"Computadores=_GoogleDrive_computadores:\"") {
		t.Fatalf("a busca automática não gravou o computador:\n%s", conf)
	}
	if got := labCloudLoad().Pastas["drive"]; len(got) != 1 || got[0].Name != "PSD" {
		t.Fatalf("estado sem o computador: %+v", got)
	}
}
