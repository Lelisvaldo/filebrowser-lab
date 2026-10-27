package web

// Nuvens (filebrowser-lab, HL-112): login pela interface (OAuth no navegador, ou usuário e senha no MEGA).
// O servidor guarda o token numa configuração de rclone; o container "rclone-mount" vigia esse arquivo e
// monta cada nuvem em /mnt/cloud/<nome>. Só administradores. Ligada quando FILEBROWSER_LAB_CLOUD_DIR existe.
//
//	FILEBROWSER_LAB_CLOUD_DIR     pasta compartilhada com o rclone-mount (rclone.conf, apps.json, estado)
//	FILEBROWSER_LAB_CLOUD_ORIGIN  endereço público (ex.: https://filebrowser.lab); o retorno é <origin>/api/lab/cloud/callback

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type labCloudProv struct {
	ID, Name       string
	Kind           string // "oauth" ou "senha"
	AuthURL, Token string
	Scope          string
	Extra          url.Values // parâmetros extras do pedido de login
	InfoURL        string
	RcloneType     string
	RcloneExtra    map[string]string
}

var labCloudProvs = []labCloudProv{
	{ID: "onedrive", Name: "OneDrive", Kind: "oauth", RcloneType: "onedrive",
		AuthURL: "https://login.microsoftonline.com/common/oauth2/v2.0/authorize", Token: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		Scope: "Files.ReadWrite.All offline_access User.Read", InfoURL: "https://graph.microsoft.com/v1.0/me/drive"},
	{ID: "drive", Name: "Google Drive", Kind: "oauth", RcloneType: "drive", RcloneExtra: map[string]string{"scope": "drive"},
		AuthURL: "https://accounts.google.com/o/oauth2/v2/auth", Token: "https://oauth2.googleapis.com/token",
		Scope: "https://www.googleapis.com/auth/drive", Extra: url.Values{"access_type": {"offline"}, "prompt": {"consent"}},
		InfoURL: "https://www.googleapis.com/drive/v3/about?fields=user"},
	{ID: "dropbox", Name: "Dropbox", Kind: "oauth", RcloneType: "dropbox",
		AuthURL: "https://www.dropbox.com/oauth2/authorize", Token: "https://api.dropboxapi.com/oauth2/token",
		Extra: url.Values{"token_access_type": {"offline"}}, InfoURL: "https://api.dropboxapi.com/2/users/get_current_account"},
	{ID: "mega", Name: "MEGA", Kind: "senha", RcloneType: "mega"},
}

func labCloudProvByID(id string) *labCloudProv {
	for i := range labCloudProvs {
		if labCloudProvs[i].ID == id {
			return &labCloudProvs[i]
		}
	}
	return nil
}

func (p *labCloudProv) override(field, def string) string {
	if v := os.Getenv("FILEBROWSER_LAB_CLOUD_" + strings.ToUpper(p.ID) + "_" + field); v != "" {
		return v
	}
	return def
}

func labCloudDir() string { return os.Getenv("FILEBROWSER_LAB_CLOUD_DIR") }

// Endereço público do site: FILEBROWSER_LAB_CLOUD_ORIGIN, ou o do próprio pedido (atrás do proxy, usa X-Forwarded-*).
func labCloudOrigin(r *http.Request) string {
	if o := strings.TrimRight(os.Getenv("FILEBROWSER_LAB_CLOUD_ORIGIN"), "/"); o != "" {
		return o
	}
	proto := r.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		proto = "http"
		if r.TLS != nil {
			proto = "https"
		}
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	return proto + "://" + host
}

// ---- arquivos de estado (tudo em 600, gravação atômica) ----

var labCloudMu sync.Mutex

type labCloudApp struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

type labCloudInfo struct {
	Account   string `json:"account,omitempty"`
	ReadOnly  bool   `json:"readOnly"`
	Connected string `json:"connected,omitempty"`
}

type labCloudState struct {
	Apps  map[string]labCloudApp  `json:"apps"`
	Conns map[string]labCloudInfo `json:"conns"`
}

func labCloudWrite(name string, data []byte) error {
	p := filepath.Join(labCloudDir(), name)
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func labCloudLoad() labCloudState {
	st := labCloudState{Apps: map[string]labCloudApp{}, Conns: map[string]labCloudInfo{}}
	if b, err := os.ReadFile(filepath.Join(labCloudDir(), "estado.json")); err == nil {
		_ = json.Unmarshal(b, &st)
	}
	if st.Apps == nil {
		st.Apps = map[string]labCloudApp{}
	}
	if st.Conns == nil {
		st.Conns = map[string]labCloudInfo{}
	}
	return st
}

func labCloudSave(st labCloudState) error {
	b, _ := json.MarshalIndent(st, "", "  ")
	if err := labCloudWrite("estado.json", b); err != nil {
		return err
	}
	return labCloudWriteConf(st)
}

// Seções do rclone.conf guardadas à parte (token e chaves ficam só aqui e no rclone.conf).
func labCloudSections() map[string]map[string]string {
	out := map[string]map[string]string{}
	b, err := os.ReadFile(filepath.Join(labCloudDir(), "remotos.json"))
	if err == nil {
		_ = json.Unmarshal(b, &out)
	}
	return out
}

func labCloudWriteConf(st labCloudState) error {
	secs := labCloudSections()
	var sb strings.Builder
	for _, p := range labCloudProvs {
		kv, ok := secs[p.ID]
		if !ok {
			continue
		}
		if _, on := st.Conns[p.ID]; !on {
			continue
		}
		sb.WriteString("[" + p.ID + "]\n")
		keys := make([]string, 0, len(kv))
		for k := range kv {
			keys = append(keys, k)
		}
		sortStrings(keys)
		for _, k := range keys {
			sb.WriteString(k + " = " + kv[k] + "\n")
		}
		sb.WriteString("\n")
	}
	if err := labCloudWrite("rclone.conf", []byte(sb.String())); err != nil {
		return err
	}
	// um arquivo "escrita-<nome>" libera escrita; sem ele a nuvem fica só leitura
	for _, p := range labCloudProvs {
		flag := filepath.Join(labCloudDir(), "escrita-"+p.ID)
		if c, on := st.Conns[p.ID]; on && !c.ReadOnly {
			_ = os.WriteFile(flag, nil, 0o600)
		} else {
			_ = os.Remove(flag)
		}
	}
	return nil
}

func labCloudSetSection(id string, kv map[string]string) error {
	secs := labCloudSections()
	if kv == nil {
		delete(secs, id)
	} else {
		secs[id] = kv
	}
	b, _ := json.Marshal(secs)
	return labCloudWrite("remotos.json", b)
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// ---- estados de login em andamento (uso único, 10 minutos) ----

type labCloudPending struct {
	prov string
	exp  time.Time
}

var (
	labCloudPendMu sync.Mutex
	labCloudPend   = map[string]labCloudPending{}
)

func labCloudNewState(prov string) string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	s := hex.EncodeToString(b)
	labCloudPendMu.Lock()
	defer labCloudPendMu.Unlock()
	for k, v := range labCloudPend {
		if time.Now().After(v.exp) {
			delete(labCloudPend, k)
		}
	}
	labCloudPend[s] = labCloudPending{prov: prov, exp: time.Now().Add(10 * time.Minute)}
	return s
}

func labCloudTakeState(s string) (string, bool) {
	labCloudPendMu.Lock()
	defer labCloudPendMu.Unlock()
	v, ok := labCloudPend[s]
	delete(labCloudPend, s)
	if !ok || time.Now().After(v.exp) {
		return "", false
	}
	return v.prov, true
}

// ---- senha do rclone (ofuscação padrão, necessária no MEGA) ----

var labCloudObscureKey = []byte{0x9c, 0x93, 0x5b, 0x48, 0x73, 0x0a, 0x55, 0x4d, 0x6b, 0xfd, 0x7c, 0x63, 0xc8, 0x86, 0xa9, 0x2b,
	0xd3, 0x90, 0x19, 0x8e, 0xb8, 0x12, 0x8a, 0xfb, 0xf4, 0xde, 0x16, 0x2b, 0x8b, 0x95, 0xf6, 0x38}

func labCloudObscure(plain string) (string, error) {
	block, err := aes.NewCipher(labCloudObscureKey)
	if err != nil {
		return "", err
	}
	buf := make([]byte, aes.BlockSize+len(plain))
	iv := buf[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}
	cipher.NewCTR(block, iv).XORKeyStream(buf[aes.BlockSize:], []byte(plain))
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// ---- chamadas aos provedores ----

var labCloudHTTP = &http.Client{Timeout: 30 * time.Second}

type labCloudTok struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

func labCloudExchange(p *labCloudProv, app labCloudApp, code, redirect string) (*labCloudTok, error) {
	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {redirect},
		"client_id": {app.ClientID}, "client_secret": {app.ClientSecret}}
	resp, err := labCloudHTTP.PostForm(p.override("TOKEN", p.Token), form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var t labCloudTok
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&t); err != nil {
		return nil, fmt.Errorf("resposta inválida do %s", p.Name)
	}
	if t.AccessToken == "" {
		msg := t.Error
		if t.ErrorDesc != "" {
			msg += ": " + t.ErrorDesc
		}
		return nil, fmt.Errorf("o %s recusou o login (%s)", p.Name, labCloudShort(msg))
	}
	return &t, nil
}

var labCloudNL = regexp.MustCompile(`\s+`)

func labCloudShort(s string) string {
	s = labCloudNL.ReplaceAllString(s, " ")
	if len(s) > 160 {
		s = s[:160]
	}
	return s
}

// conta (e, no OneDrive, o drive_id) para mostrar na tela e preencher o rclone.conf
func labCloudWho(p *labCloudProv, access string) (account string, extra map[string]string) {
	extra = map[string]string{}
	u := p.override("INFO", p.InfoURL)
	if u == "" {
		return "", extra
	}
	method := http.MethodGet
	if p.ID == "dropbox" {
		method = http.MethodPost
	}
	req, err := http.NewRequest(method, u, nil)
	if err != nil {
		return "", extra
	}
	req.Header.Set("Authorization", "Bearer "+access)
	resp, err := labCloudHTTP.Do(req)
	if err != nil {
		return "", extra
	}
	defer resp.Body.Close()
	var m map[string]any
	if json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&m) != nil {
		return "", extra
	}
	str := func(v any) string { s, _ := v.(string); return s }
	switch p.ID {
	case "onedrive":
		extra["drive_id"] = str(m["id"])
		extra["drive_type"] = str(m["driveType"])
		if o, ok := m["owner"].(map[string]any); ok {
			if usr, ok := o["user"].(map[string]any); ok {
				account = str(usr["displayName"])
			}
		}
	case "drive":
		if usr, ok := m["user"].(map[string]any); ok {
			account = str(usr["emailAddress"])
		}
	case "dropbox":
		account = str(m["email"])
	}
	return account, extra
}

// ---- API (só administradores) ----

type labCloudItem struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	HasApp    bool   `json:"hasApp"`
	Connected bool   `json:"connected"`
	Account   string `json:"account,omitempty"`
	ReadOnly  bool   `json:"readOnly"`
	Since     string `json:"since,omitempty"`
}

func labCloudOn() error {
	if labCloudDir() == "" {
		return fmt.Errorf("as nuvens não estão ligadas neste servidor")
	}
	return nil
}

// GET /api/lab/cloud
func labCloudListHandler(w http.ResponseWriter, r *http.Request, d *Context) (int, error) {
	if err := labCloudOn(); err != nil {
		return http.StatusNotFound, err
	}
	labCloudMu.Lock()
	st := labCloudLoad()
	labCloudMu.Unlock()
	out := make([]labCloudItem, 0, len(labCloudProvs))
	for _, p := range labCloudProvs {
		c, on := st.Conns[p.ID]
		_, app := st.Apps[p.ID]
		out = append(out, labCloudItem{ID: p.ID, Name: p.Name, Kind: p.Kind, HasApp: app || p.Kind == "senha",
			Connected: on, Account: c.Account, ReadOnly: !on || c.ReadOnly, Since: c.Connected})
	}
	return labJSON(w, map[string]any{"items": out, "callback": labCloudOrigin(r) + "/api/lab/cloud/callback"})
}

// PUT /api/lab/cloud/app?id=   {clientId, clientSecret}
func labCloudAppHandler(w http.ResponseWriter, r *http.Request, d *Context) (int, error) {
	if err := labCloudOn(); err != nil {
		return http.StatusNotFound, err
	}
	p := labCloudProvByID(r.URL.Query().Get("id"))
	if p == nil || p.Kind != "oauth" {
		return http.StatusBadRequest, fmt.Errorf("provedor inválido")
	}
	var app labCloudApp
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&app); err != nil {
		return http.StatusBadRequest, fmt.Errorf("corpo inválido")
	}
	app.ClientID, app.ClientSecret = strings.TrimSpace(app.ClientID), strings.TrimSpace(app.ClientSecret)
	if app.ClientID == "" || app.ClientSecret == "" {
		return http.StatusBadRequest, fmt.Errorf("informe o ID e o segredo do aplicativo")
	}
	labCloudMu.Lock()
	defer labCloudMu.Unlock()
	st := labCloudLoad()
	st.Apps[p.ID] = app
	if err := labCloudSave(st); err != nil {
		return http.StatusInternalServerError, err
	}
	return labJSON(w, map[string]string{"status": "ok"})
}

// POST /api/lab/cloud/start?id=   -> {url}
func labCloudStartHandler(w http.ResponseWriter, r *http.Request, d *Context) (int, error) {
	if err := labCloudOn(); err != nil {
		return http.StatusNotFound, err
	}
	p := labCloudProvByID(r.URL.Query().Get("id"))
	if p == nil || p.Kind != "oauth" {
		return http.StatusBadRequest, fmt.Errorf("provedor inválido")
	}
	labCloudMu.Lock()
	app, ok := labCloudLoad().Apps[p.ID]
	labCloudMu.Unlock()
	if !ok {
		return http.StatusPreconditionFailed, fmt.Errorf("cadastre primeiro o aplicativo do %s (botão da chave)", p.Name)
	}
	q := url.Values{"client_id": {app.ClientID}, "response_type": {"code"}, "redirect_uri": {labCloudOrigin(r) + "/api/lab/cloud/callback"},
		"state": {labCloudNewState(p.ID)}}
	if p.Scope != "" {
		q.Set("scope", p.Scope)
	}
	for k, v := range p.Extra {
		q[k] = v
	}
	return labJSON(w, map[string]string{"url": p.override("AUTH", p.AuthURL) + "?" + q.Encode()})
}

// GET /api/lab/cloud/callback?code=&state=   (volta do provedor; a prova é o "state" de uso único)
func labCloudCallbackHandler(w http.ResponseWriter, r *http.Request, d *Context) (int, error) {
	done := func(ok bool, msg string) (int, error) {
		q := url.Values{"nuvem": {"ok"}}
		if !ok {
			q = url.Values{"nuvem": {"erro"}, "msg": {msg}}
		}
		http.Redirect(w, r, "/admin/servicos?"+q.Encode(), http.StatusFound)
		return 0, nil
	}
	if labCloudOn() != nil {
		return http.StatusNotFound, fmt.Errorf("nuvens desligadas")
	}
	id, ok := labCloudTakeState(r.URL.Query().Get("state"))
	if !ok {
		return done(false, "pedido de login vencido ou inválido; tente de novo")
	}
	if e := r.URL.Query().Get("error"); e != "" {
		return done(false, "login cancelado ("+labCloudShort(e)+")")
	}
	p := labCloudProvByID(id)
	labCloudMu.Lock()
	defer labCloudMu.Unlock()
	st := labCloudLoad()
	app := st.Apps[id]
	tok, err := labCloudExchange(p, app, r.URL.Query().Get("code"), labCloudOrigin(r)+"/api/lab/cloud/callback")
	if err != nil {
		return done(false, err.Error())
	}
	account, extra := labCloudWho(p, tok.AccessToken)
	exp := time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second).UTC().Format(time.RFC3339)
	if tok.ExpiresIn == 0 {
		exp = time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	}
	tt := tok.TokenType
	if tt == "" {
		tt = "Bearer"
	}
	tj, _ := json.Marshal(map[string]string{"access_token": tok.AccessToken, "token_type": tt, "refresh_token": tok.RefreshToken, "expiry": exp})
	kv := map[string]string{"type": p.RcloneType, "client_id": app.ClientID, "client_secret": app.ClientSecret, "token": string(tj)}
	for k, v := range p.RcloneExtra {
		kv[k] = v
	}
	for k, v := range extra {
		if v != "" {
			kv[k] = v
		}
	}
	if err := labCloudSetSection(id, kv); err != nil {
		return done(false, "não consegui guardar o token")
	}
	st.Conns[id] = labCloudInfo{Account: account, ReadOnly: true, Connected: time.Now().Format(time.RFC3339)}
	if err := labCloudSave(st); err != nil {
		return done(false, "não consegui gravar a configuração")
	}
	return done(true, "")
}

// POST /api/lab/cloud/senha?id=mega   {user, pass}
func labCloudLoginHandler(w http.ResponseWriter, r *http.Request, d *Context) (int, error) {
	if err := labCloudOn(); err != nil {
		return http.StatusNotFound, err
	}
	p := labCloudProvByID(r.URL.Query().Get("id"))
	if p == nil || p.Kind != "senha" {
		return http.StatusBadRequest, fmt.Errorf("provedor inválido")
	}
	var req struct{ User, Pass string }
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil || req.User == "" || req.Pass == "" {
		return http.StatusBadRequest, fmt.Errorf("informe o email e a senha")
	}
	ob, err := labCloudObscure(req.Pass)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	labCloudMu.Lock()
	defer labCloudMu.Unlock()
	st := labCloudLoad()
	if err := labCloudSetSection(p.ID, map[string]string{"type": p.RcloneType, "user": req.User, "pass": ob}); err != nil {
		return http.StatusInternalServerError, err
	}
	st.Conns[p.ID] = labCloudInfo{Account: req.User, ReadOnly: true, Connected: time.Now().Format(time.RFC3339)}
	if err := labCloudSave(st); err != nil {
		return http.StatusInternalServerError, err
	}
	return labJSON(w, map[string]string{"status": "ok"})
}

// POST /api/lab/cloud/mode?id=&readOnly=true|false
func labCloudModeHandler(w http.ResponseWriter, r *http.Request, d *Context) (int, error) {
	if err := labCloudOn(); err != nil {
		return http.StatusNotFound, err
	}
	id := r.URL.Query().Get("id")
	labCloudMu.Lock()
	defer labCloudMu.Unlock()
	st := labCloudLoad()
	c, on := st.Conns[id]
	if !on {
		return http.StatusNotFound, fmt.Errorf("nuvem não conectada")
	}
	c.ReadOnly = r.URL.Query().Get("readOnly") != "false"
	st.Conns[id] = c
	if err := labCloudSave(st); err != nil {
		return http.StatusInternalServerError, err
	}
	return labJSON(w, map[string]string{"status": "ok"})
}

// DELETE /api/lab/cloud?id=
func labCloudDisconnectHandler(w http.ResponseWriter, r *http.Request, d *Context) (int, error) {
	if err := labCloudOn(); err != nil {
		return http.StatusNotFound, err
	}
	id := r.URL.Query().Get("id")
	if labCloudProvByID(id) == nil {
		return http.StatusBadRequest, fmt.Errorf("provedor inválido")
	}
	labCloudMu.Lock()
	defer labCloudMu.Unlock()
	st := labCloudLoad()
	delete(st.Conns, id)
	_ = labCloudSetSection(id, nil)
	if err := labCloudSave(st); err != nil {
		return http.StatusInternalServerError, err
	}
	return labJSON(w, map[string]string{"status": "ok"})
}
