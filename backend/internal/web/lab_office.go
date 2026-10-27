package web

// Documentos do Office de qualquer geração (filebrowser-lab): o LibreOffice (headless, dentro da imagem)
// converte para PDF (prévia) ou para outro formato (cópia ao lado do original). Parte do remendo do lab.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gtsteffaniak/filebrowser/backend/pkg/settings"
)

const (
	labOfficeTimeout = 2 * time.Minute
	labOfficeMaxIn   = int64(80) << 20 // acima disso a conversão não compensa
	labOfficeKeep    = 14 * 24 * time.Hour
)

// extensão -> família (decide para quais formatos dá para converter)
var labOfficeKind = map[string]string{
	".doc": "w", ".dot": "w", ".docx": "w", ".docm": "w", ".dotx": "w", ".odt": "w", ".ott": "w", ".rtf": "w", ".wpd": "w", ".wps": "w",
	".xls": "c", ".xlt": "c", ".xlsx": "c", ".xlsm": "c", ".xltx": "c", ".xlsb": "c", ".ods": "c", ".ots": "c", ".csv": "c",
	".ppt": "i", ".pps": "i", ".pot": "i", ".pptx": "i", ".pptm": "i", ".ppsx": "i", ".potx": "i", ".odp": "i", ".otp": "i",
}

var labOfficeTargets = map[string][]string{
	"w": {"pdf", "docx", "odt", "rtf", "txt"},
	"c": {"pdf", "xlsx", "ods", "csv"},
	"i": {"pdf", "pptx", "odp"},
}

var labOfficeSem = make(chan struct{}, 2)

func labOfficeBin() (string, error) {
	for _, n := range []string{"soffice", "libreoffice"} {
		if p, err := exec.LookPath(n); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("o LibreOffice não está instalado nesta imagem")
}

func labCopy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// labOfficeConvert converte "real" para o formato "to" e devolve o caminho de um arquivo temporário
// (o chamador move ou apaga o diretório devolvido em "tmp").
func labOfficeConvert(ctx context.Context, real, to string) (result string, tmp string, err error) {
	bin, err := labOfficeBin()
	if err != nil {
		return "", "", err
	}
	fi, err := os.Stat(real)
	if err != nil || fi.IsDir() {
		return "", "", fmt.Errorf("arquivo não encontrado")
	}
	if fi.Size() > labOfficeMaxIn {
		return "", "", fmt.Errorf("arquivo grande demais para converter (limite %d MB)", labOfficeMaxIn>>20)
	}
	select {
	case labOfficeSem <- struct{}{}:
		defer func() { <-labOfficeSem }()
	case <-ctx.Done():
		return "", "", ctx.Err()
	}
	tmp, err = os.MkdirTemp("", "lo-")
	if err != nil {
		return "", "", err
	}
	ok := false
	defer func() {
		if !ok {
			os.RemoveAll(tmp)
		}
	}()
	in := filepath.Join(tmp, "in"+strings.ToLower(filepath.Ext(real)))
	if err := labCopy(real, in); err != nil {
		return "", "", err
	}
	outDir := filepath.Join(tmp, "out")
	if err := os.Mkdir(outDir, 0o700); err != nil {
		return "", "", err
	}
	prof := filepath.Join(tmp, "profile")
	cctx, cancel := context.WithTimeout(ctx, labOfficeTimeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, bin, "--headless", "--norestore", "--nolockcheck", "--nodefault", "--nologo", "--nofirststartwizard",
		"-env:UserInstallation=file://"+prof, "--convert-to", to, "--outdir", outDir, in)
	cmd.Dir = tmp
	cmd.Env = append(os.Environ(), "HOME="+tmp)
	out, runErr := cmd.CombinedOutput()
	res := filepath.Join(outDir, "in."+to)
	if _, e := os.Stat(res); e != nil {
		if cctx.Err() != nil {
			return "", "", fmt.Errorf("a conversão demorou demais")
		}
		msg := labLastLine(string(out))
		if runErr != nil && msg == "erro desconhecido" {
			msg = runErr.Error()
		}
		return "", "", fmt.Errorf("LibreOffice: %s", msg)
	}
	ok = true
	return res, tmp, nil
}

func labOfficeCacheDir() string {
	d := settings.Config.Server.CacheDir
	if d == "" {
		d = os.TempDir()
	}
	return filepath.Join(d, "office")
}

func labOfficeSweep(dir string) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range ents {
		if info, err := e.Info(); err == nil && time.Since(info.ModTime()) > labOfficeKeep {
			os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}

// GET /api/lab/office/pdf?source=&path=
func labOfficePDFHandler(w http.ResponseWriter, r *http.Request, d *Context) (int, error) {
	q := r.URL.Query()
	real, st, err := labReal(d, q.Get("source"), q.Get("path"), "view")
	if err != nil {
		return st, err
	}
	if _, ok := labOfficeKind[strings.ToLower(filepath.Ext(real))]; !ok {
		return http.StatusBadRequest, fmt.Errorf("este tipo de arquivo não é de escritório")
	}
	fi, err := os.Stat(real)
	if err != nil {
		return http.StatusNotFound, err
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%d|%d", real, fi.Size(), fi.ModTime().UnixNano())))
	dir := labOfficeCacheDir()
	cached := filepath.Join(dir, hex.EncodeToString(sum[:16])+".pdf")
	if _, err := os.Stat(cached); err != nil {
		res, tmp, err := labOfficeConvert(r.Context(), real, "pdf")
		if err != nil {
			return http.StatusBadRequest, err
		}
		defer os.RemoveAll(tmp)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return http.StatusInternalServerError, err
		}
		part := cached + ".part"
		if err := labCopy(res, part); err != nil {
			return http.StatusInternalServerError, err
		}
		if err := os.Rename(part, cached); err != nil {
			return http.StatusInternalServerError, err
		}
		go labOfficeSweep(dir)
	}
	f, err := os.Open(cached)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer f.Close()
	cfi, _ := f.Stat()
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "inline")
	http.ServeContent(w, r, "preview.pdf", cfi.ModTime(), f)
	return 0, nil
}

type labOfficeReq struct {
	Source string `json:"source"`
	Path   string `json:"path"`
	To     string `json:"to"`
}

// POST /api/lab/office/convert {source, path, to}: grava uma cópia no formato novo, ao lado do original.
func labOfficeConvertHandler(w http.ResponseWriter, r *http.Request, d *Context) (int, error) {
	var req labOfficeReq
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		return http.StatusBadRequest, fmt.Errorf("corpo inválido: %v", err)
	}
	real, st, err := labReal(d, req.Source, req.Path, "download")
	if err != nil {
		return st, err
	}
	kind, ok := labOfficeKind[strings.ToLower(filepath.Ext(real))]
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("este tipo de arquivo não é de escritório")
	}
	allowed := false
	for _, t := range labOfficeTargets[kind] {
		if t == req.To {
			allowed = true
		}
	}
	if !allowed {
		return http.StatusBadRequest, fmt.Errorf("conversão para %q não é permitida", req.To)
	}
	parent := path.Dir(strings.ReplaceAll(req.Path, "\\", "/"))
	destDir, st, err := labReal(d, req.Source, parent, "create")
	if err != nil {
		return st, err
	}
	res, tmp, err := labOfficeConvert(r.Context(), real, req.To)
	if err != nil {
		return http.StatusBadRequest, err
	}
	defer os.RemoveAll(tmp)
	base := strings.TrimSuffix(filepath.Base(real), filepath.Ext(real))
	name := base + "." + req.To
	for i := 1; ; i++ {
		if _, err := os.Stat(filepath.Join(destDir, name)); os.IsNotExist(err) {
			break
		}
		name = fmt.Sprintf("%s (%d).%s", base, i, req.To)
		if i > 999 {
			return http.StatusConflict, fmt.Errorf("nomes demais em uso")
		}
	}
	if err := labCopy(res, filepath.Join(destDir, name)); err != nil {
		return http.StatusInternalServerError, err
	}
	return labJSON(w, map[string]string{"status": "ok", "name": name})
}
