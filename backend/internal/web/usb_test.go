package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestUploadSessionAnyUnder(t *testing.T) {
	var r uploadSessionRegistry
	if r.anyUnder("/srv/usb/SANDISK/") {
		t.Fatal("registro vazio não deveria ter envio")
	}
	if err := r.acquire("/srv/usb/SANDISK/foto.jpg", "s1"); err != nil {
		t.Fatal(err)
	}
	if !r.anyUnder("/srv/usb/SANDISK/") {
		t.Fatal("deveria achar o envio dentro do pendrive")
	}
	if r.anyUnder("/srv/usb/OUTRO/") {
		t.Fatal("não deveria achar envio em outro pendrive")
	}
	r.expireForTest("/srv/usb/SANDISK/foto.jpg")
	if r.anyUnder("/srv/usb/SANDISK/") {
		t.Fatal("envio expirado não conta")
	}
}

func TestUSBUUIDPattern(t *testing.T) {
	for _, ok := range []string{"28C1-E530", "0123456789abcdef", "a1b2c3d4-e5f6-7890-abcd-ef1234567890"} {
		if !usbUUIDPattern.MatchString(ok) {
			t.Errorf("%q deveria ser aceito", ok)
		}
	}
	for _, bad := range []string{"", "../etc", "28C1/E530", "abc", "28C1-E530; rm -rf /"} {
		if usbUUIDPattern.MatchString(bad) {
			t.Errorf("%q deveria ser recusado", bad)
		}
	}
}

func TestUSBDevicesParsesManagerReply(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/usb" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`[{"uuid":"28C1-E530","mountpoint":"/mnt/usb/SANDISK","ejectable":true}]`))
	}))
	defer srv.Close()
	t.Setenv("FILEBROWSER_LAB_USB_URL", srv.URL+"/")
	raw, list, err := usbDevices()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].UUID != "28C1-E530" || !list[0].Ejectable || len(raw) == 0 {
		t.Fatalf("resposta inesperada: %+v", list)
	}
	_ = time.Second
}

func TestUSBBaseURLEmptyDisablesIntegration(t *testing.T) {
	t.Setenv("FILEBROWSER_LAB_USB_URL", "")
	if usbBaseURL() != "" {
		t.Fatal("sem variável a integração deve ficar desligada")
	}
}
