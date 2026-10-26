package web

// filebrowser-lab: ponte com o usb-manager do servidor (lista e remoção segura de pendrives).
//
// Ligada só quando FILEBROWSER_LAB_USB_URL está definida (ex.: http://usb-manager.local:8780).
// Sem ela, as rotas respondem 404 e a interface esconde a seção de pendrives.
//
//	FILEBROWSER_LAB_USB_URL          endereço do usb-manager
//	FILEBROWSER_LAB_USB_ORIGIN       Origin enviado no POST de remoção (padrão http://localhost)
//	FILEBROWSER_LAB_USB_HOST_ROOT    onde o host monta os pendrives (padrão /mnt/usb)
//	FILEBROWSER_LAB_USB_MOUNT_ROOT   onde essa pasta aparece neste container (padrão /srv/usb)
//	FILEBROWSER_LAB_USB_SOURCE       nome da fonte que exibe os pendrives (padrão Pendrives)

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

var usbUUIDPattern = regexp.MustCompile(`^[0-9A-Fa-f-]{8,64}$`)

func usbEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func usbBaseURL() string {
	return strings.TrimRight(strings.TrimSpace(os.Getenv("FILEBROWSER_LAB_USB_URL")), "/")
}

type usbDevice struct {
	UUID       string `json:"uuid"`
	Mountpoint string `json:"mountpoint"`
	Ejectable  bool   `json:"ejectable"`
}

// usbDevices consulta o usb-manager e devolve o corpo bruto (para repassar) e a lista tipada.
func usbDevices() ([]byte, []usbDevice, error) {
	client := &http.Client{Timeout: 8 * time.Second}
	res, err := client.Get(usbBaseURL() + "/api/usb")
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("usb-manager respondeu %d", res.StatusCode)
	}
	var list []usbDevice
	if err = json.Unmarshal(raw, &list); err != nil {
		return nil, nil, err
	}
	return raw, list, nil
}

// usbAllowed: administradores ou quem enxerga a fonte dos pendrives.
func usbAllowed(d *Context) bool {
	if d.User == nil {
		return false
	}
	if d.User.Permissions.Admin {
		return true
	}
	_, err := d.User.GetScopeForSourceName(usbEnv("FILEBROWSER_LAB_USB_SOURCE", "Pendrives"))
	return err == nil
}

// usbListHandler lista os pendrives ligados.
func usbListHandler(w http.ResponseWriter, r *http.Request, d *Context) (int, error) {
	if usbBaseURL() == "" {
		return http.StatusNotFound, fmt.Errorf("integração de pendrives desligada")
	}
	if !usbAllowed(d) {
		return http.StatusForbidden, fmt.Errorf("sem acesso aos pendrives")
	}
	raw, _, err := usbDevices()
	if err != nil {
		return http.StatusBadGateway, fmt.Errorf("não foi possível consultar os pendrives: %w", err)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
	return 0, nil
}

// usbEjectHandler remove o pendrive com segurança, recusando enquanto houver envio em andamento nele.
func usbEjectHandler(w http.ResponseWriter, r *http.Request, d *Context) (int, error) {
	base := usbBaseURL()
	if base == "" {
		return http.StatusNotFound, fmt.Errorf("integração de pendrives desligada")
	}
	if !usbAllowed(d) {
		return http.StatusForbidden, fmt.Errorf("sem acesso aos pendrives")
	}
	uuid := r.PathValue("uuid")
	if !usbUUIDPattern.MatchString(uuid) {
		return http.StatusBadRequest, fmt.Errorf("identificador de pendrive inválido")
	}
	_, list, err := usbDevices()
	if err != nil {
		return http.StatusBadGateway, fmt.Errorf("não foi possível consultar os pendrives: %w", err)
	}
	var target *usbDevice
	for i := range list {
		if strings.EqualFold(list[i].UUID, uuid) {
			target = &list[i]
			break
		}
	}
	if target == nil {
		return http.StatusNotFound, fmt.Errorf("pendrive não encontrado")
	}
	hostRoot := strings.TrimRight(usbEnv("FILEBROWSER_LAB_USB_HOST_ROOT", "/mnt/usb"), "/")
	mountRoot := strings.TrimRight(usbEnv("FILEBROWSER_LAB_USB_MOUNT_ROOT", "/srv/usb"), "/")
	if strings.HasPrefix(target.Mountpoint, hostRoot+"/") {
		inside := mountRoot + strings.TrimPrefix(target.Mountpoint, hostRoot)
		if activeUploadSessions.anyUnder(inside + "/") {
			return http.StatusConflict, fmt.Errorf("há um envio em andamento para este pendrive; espere terminar")
		}
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, base+"/api/usb/"+url.PathEscape(uuid)+"/eject", nil)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	req.Header.Set("Origin", usbEnv("FILEBROWSER_LAB_USB_ORIGIN", "http://localhost"))
	client := &http.Client{Timeout: 190 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return http.StatusBadGateway, fmt.Errorf("falha ao falar com o usb-manager: %w", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<16))
	if res.StatusCode == http.StatusOK {
		return RenderJSON(w, r, map[string]any{"ok": true, "uuid": uuid})
	}
	msg := "não foi possível remover o pendrive"
	var parsed struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &parsed) == nil && parsed.Error != "" {
		msg = parsed.Error
	}
	if res.StatusCode == http.StatusConflict || strings.Contains(strings.ToLower(msg), "busy") {
		return http.StatusConflict, fmt.Errorf("pendrive em uso (busy); feche os arquivos abertos e tente de novo")
	}
	return res.StatusCode, fmt.Errorf("%s", msg)
}
