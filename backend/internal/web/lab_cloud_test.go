package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	for _, want := range []string{"[onedrive]", "type = onedrive", "drive_id = drv1", "drive_type = personal", `"refresh_token":"RT"`} {
		if !strings.Contains(string(conf), want) {
			t.Fatalf("rclone.conf sem %q:\n%s", want, conf)
		}
	}
	if fi, _ := os.Stat(filepath.Join(dir, "rclone.conf")); fi.Mode().Perm() != 0o600 {
		t.Fatalf("permissão do rclone.conf = %v", fi.Mode().Perm())
	}
	if _, err := os.Stat(filepath.Join(dir, "escrita-onedrive")); err == nil {
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
	if _, err := os.Stat(filepath.Join(dir, "escrita-onedrive")); err != nil {
		t.Fatal("escrita deveria estar liberada")
	}
	w = httptest.NewRecorder()
	_, _ = labCloudDisconnectHandler(w, httptest.NewRequest("DELETE", "/lab/cloud?id=onedrive", nil), nil)
	conf, _ = os.ReadFile(filepath.Join(dir, "rclone.conf"))
	if strings.Contains(string(conf), "onedrive") {
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
	if !strings.Contains(string(conf), "type = mega") || strings.Contains(string(conf), "s3nha") {
		t.Fatalf("conf do mega errada (senha deve estar ofuscada):\n%s", conf)
	}
	t.Logf("%s", conf)
}
