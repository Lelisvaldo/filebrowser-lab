package web

// Arquivos compactados como pastas (filebrowser-lab): listar, ver, adicionar, apagar, renomear, editar,
// criar e extrair com o 7-Zip. Parte do remendo do lab (não existe no Quantum original).

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gtsteffaniak/filebrowser/backend/internal/state"
	"github.com/gtsteffaniak/filebrowser/backend/internal/utils"
	"github.com/gtsteffaniak/filebrowser/backend/pkg/indexing"
)

const (
	labArchiveTimeout = 20 * time.Minute
	labArchiveMaxEdit = int64(8) << 30   // acima disso, editar por dentro não é prático (cópia temporária)
	labArchiveMaxOut  = int64(100) << 30 // limite de extração (proteção contra "bomba" de compressão)
)

var (
	labArchiveEditMu sync.Mutex
	labEditableExt   = map[string]string{".zip": "zip", ".7z": "7z", ".tar": "tar"}
)

func labSevenBin() (string, error) {
	for _, n := range []string{"7zz", "7z", "7za"} {
		if p, err := exec.LookPath(n); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("o 7-Zip não está instalado nesta imagem")
}

// labReal resolve (fonte, caminho) para o caminho real, checando a permissão pedida ("view", "download", "modify", "create").
func labReal(d *Context, source, p, need string) (string, int, error) {
	if d.Share.Hash != "" {
		return "", http.StatusForbidden, fmt.Errorf("não permitido em links compartilhados")
	}
	perms, err := effectiveFilePerms(d, source)
	if err != nil {
		return "", http.StatusForbidden, err
	}
	ok := map[string]bool{"view": perms.View, "download": perms.Download, "modify": perms.Modify, "create": perms.Create}[need]
	if !ok {
		return "", http.StatusForbidden, fmt.Errorf("sem permissão (%s) nesta fonte", need)
	}
	clean, err := utils.SanitizePath(p)
	if err != nil {
		return "", http.StatusBadRequest, err
	}
	idx := indexing.GetIndex(source)
	if idx == nil {
		return "", http.StatusNotFound, fmt.Errorf("fonte %s não encontrada", source)
	}
	scope, err := d.User.GetScopeForSourceName(source)
	if err != nil {
		return "", http.StatusForbidden, err
	}
	full := utils.JoinPathAsUnix(scope, clean)
	if !state.AccessPermitted(idx.Path, utils.IndexPathFromNormalized(full, true), d.User.Username) {
		return "", http.StatusForbidden, fmt.Errorf("acesso negado a %s", p)
	}
	real, _, err := idx.GetRealPath(full)
	if err != nil {
		return "", http.StatusNotFound, fmt.Errorf("caminho não encontrado: %v", err)
	}
	return real, 0, nil
}

// labInner normaliza um caminho de dentro do compactado: sem "..", sem barra inicial, com "/".
func labInner(p string) (string, error) {
	p = strings.ReplaceAll(p, "\\", "/")
	if strings.ContainsRune(p, 0) {
		return "", fmt.Errorf("caminho inválido")
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." {
			return "", fmt.Errorf("caminho inválido")
		}
	}
	c := strings.TrimPrefix(path.Clean("/"+p), "/")
	return c, nil
}

func labPassword(r *http.Request) string {
	v, err := url.QueryUnescape(r.Header.Get("X-Archive-Password"))
	if err != nil {
		return ""
	}
	return v
}

func labRun(ctx context.Context, dir string, args ...string) ([]byte, error) {
	bin, err := labSevenBin()
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return out.Bytes(), fmt.Errorf("7-Zip: %s", labLastLine(out.String()))
	}
	return out.Bytes(), nil
}

func labLastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if l := strings.TrimSpace(lines[i]); l != "" && !strings.HasPrefix(l, "Everything is Ok") {
			return l
		}
	}
	return "erro desconhecido"
}

func labPwArg(pw string) []string {
	if pw == "" {
		return nil
	}
	return []string{"-p" + pw}
}

type labEntry struct {
	Name     string `json:"name"`
	Path     string `json:"path"` // caminho dentro do compactado
	Dir      bool   `json:"dir"`
	Size     int64  `json:"size"`
	Modified string `json:"modified,omitempty"`
}

type labRaw struct {
	path, mod  string
	dir, enc   bool
	link       bool
	size       int64
}

func labList(ctx context.Context, archive, pw string) ([]labRaw, error) {
	args := append([]string{"l", "-slt", "-ba", "-sccUTF-8", "-spd"}, labPwArg(pw)...)
	args = append(args, "--", archive)
	out, err := labRun(ctx, "", args...)
	if err != nil {
		return nil, err
	}
	var res []labRaw
	var cur labRaw
	have := false
	flush := func() {
		if have && cur.path != "" {
			res = append(res, cur)
		}
		cur, have = labRaw{}, false
	}
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		k, v, ok := strings.Cut(line, " = ")
		if !ok {
			continue
		}
		have = true
		switch k {
		case "Path":
			cur.path = v
		case "Folder":
			cur.dir = v == "+"
		case "Size":
			fmt.Sscan(v, &cur.size)
		case "Modified":
			cur.mod = v
		case "Encrypted":
			cur.enc = v == "+"
		case "Symbolic Link", "Hard Link", "Link":
			cur.link = true
		case "Attributes":
			a := strings.TrimSpace(v)
			if strings.HasPrefix(a, "D") || (len(a) >= 10 && a[0] == 'd') {
				cur.dir = true
			}
			if len(a) >= 10 && a[0] == 'l' { // modo Unix de atalho (lrwxrwxrwx)
				cur.link = true
			}
		}
	}
	flush()
	return res, nil
}

func labTime(s string) string {
	for _, f := range []string{"2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05"} {
		if t, err := time.ParseInLocation(f, s, time.UTC); err == nil {
			return t.Format(time.RFC3339)
		}
	}
	return ""
}

func labJSON(w http.ResponseWriter, v any) (int, error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
	return 0, nil
}

// GET /api/lab/archive/list?source=&path=&dir=
func labArchiveListHandler(w http.ResponseWriter, r *http.Request, d *Context) (int, error) {
	q := r.URL.Query()
	real, st, err := labReal(d, q.Get("source"), q.Get("path"), "view")
	if err != nil {
		return st, err
	}
	inner, err := labInner(q.Get("dir"))
	if err != nil {
		return http.StatusBadRequest, err
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	raws, err := labList(ctx, real, labPassword(r))
	if err != nil {
		return http.StatusBadRequest, err
	}
	prefix := ""
	if inner != "" {
		prefix = inner + "/"
	}
	seen := map[string]*labEntry{}
	var order []string
	encrypted := false
	for _, e := range raws {
		p, perr := labInner(e.path)
		if perr != nil || p == "" {
			continue
		}
		if e.enc {
			encrypted = true
		}
		if !strings.HasPrefix(p, prefix) || p == strings.TrimSuffix(prefix, "/") {
			continue
		}
		rest := strings.TrimPrefix(p, prefix)
		seg, more, hasMore := strings.Cut(rest, "/")
		_ = more
		isDir := e.dir || hasMore
		it, ok := seen[seg]
		if !ok {
			it = &labEntry{Name: seg, Path: prefix + seg, Dir: isDir}
			seen[seg] = it
			order = append(order, seg)
		}
		if isDir {
			it.Dir = true
		}
		if !hasMore && !e.dir {
			it.Size = e.size
			it.Modified = labTime(e.mod)
		} else {
			it.Size += e.size
			if t := labTime(e.mod); t > it.Modified {
				it.Modified = t
			}
		}
	}
	items := make([]labEntry, 0, len(order))
	for _, k := range order {
		items = append(items, *seen[k])
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Dir != items[j].Dir {
			return items[i].Dir
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	_, editable := labEditableExt[strings.ToLower(filepath.Ext(real))]
	perms, _ := effectiveFilePerms(d, q.Get("source"))
	return labJSON(w, map[string]any{"items": items, "encrypted": encrypted, "editable": editable && perms.Modify})
}

// GET /api/lab/archive/file?source=&path=&inner=&download=1
func labArchiveFileHandler(w http.ResponseWriter, r *http.Request, d *Context) (int, error) {
	q := r.URL.Query()
	real, st, err := labReal(d, q.Get("source"), q.Get("path"), "download")
	if err != nil {
		return st, err
	}
	inner, err := labInner(q.Get("inner"))
	if err != nil || inner == "" {
		return http.StatusBadRequest, fmt.Errorf("item inválido")
	}
	bin, err := labSevenBin()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	ctx, cancel := context.WithTimeout(r.Context(), labArchiveTimeout)
	defer cancel()
	args := append([]string{"x", "-so", "-bd", "-spd", "-sccUTF-8"}, labPwArg(labPassword(r))...)
	args = append(args, "--", real, inner)
	cmd := exec.CommandContext(ctx, bin, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return http.StatusInternalServerError, err
	}
	name := path.Base(inner)
	ctype := mime.TypeByExtension(strings.ToLower(path.Ext(name)))
	if ctype == "" {
		ctype = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ctype)
	disp := "inline"
	if q.Get("download") == "1" {
		disp = "attachment"
	}
	w.Header().Set("Content-Disposition", disp+"; filename*=UTF-8''"+url.PathEscape(name))
	w.Header().Set("Cache-Control", "private, no-store")
	n, _ := io.Copy(w, stdout)
	if err := cmd.Wait(); err != nil && n == 0 {
		return http.StatusBadRequest, fmt.Errorf("7-Zip: %s", labLastLine(stderr.String()))
	}
	return 0, nil
}

// withEdit copia o compactado, aplica fn na cópia e troca o original só se tudo deu certo.
func labWithEdit(ctx context.Context, real string, fn func(copyPath string) error) error {
	ext := strings.ToLower(filepath.Ext(real))
	if _, ok := labEditableExt[ext]; !ok {
		return fmt.Errorf("este formato é só leitura: extraia, ou crie um .zip/.7z novo")
	}
	fi, err := os.Stat(real)
	if err != nil {
		return err
	}
	if fi.Size() > labArchiveMaxEdit {
		return fmt.Errorf("compactado grande demais para editar por dentro (limite %d GB)", labArchiveMaxEdit>>30)
	}
	labArchiveEditMu.Lock()
	defer labArchiveEditMu.Unlock()
	tmp, err := os.CreateTemp(filepath.Dir(real), ".lab-edit-*"+ext)
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	src, err := os.Open(real)
	if err != nil {
		tmp.Close()
		return err
	}
	_, err = io.Copy(tmp, src)
	src.Close()
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	if err := fn(tmpName); err != nil {
		return err
	}
	_ = os.Chmod(tmpName, fi.Mode().Perm())
	return os.Rename(tmpName, real)
}

// POST /api/lab/archive/edit  {source,path,op: delete|rename|mkdir|save, ...}
type labEditReq struct {
	Source string   `json:"source"`
	Path   string   `json:"path"`
	Op     string   `json:"op"`
	Items  []string `json:"items"`   // delete
	From   string   `json:"from"`    // rename
	To     string   `json:"to"`      // rename
	Dir    string   `json:"dir"`     // mkdir: pasta pai; save: n/a
	Name   string   `json:"name"`    // mkdir
	Inner  string   `json:"inner"`   // save
	Text   string   `json:"content"` // save
}

func labArchiveEditHandler(w http.ResponseWriter, r *http.Request, d *Context) (int, error) {
	var req labEditReq
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<20)).Decode(&req); err != nil {
		return http.StatusBadRequest, fmt.Errorf("corpo inválido: %v", err)
	}
	real, st, err := labReal(d, req.Source, req.Path, "modify")
	if err != nil {
		return st, err
	}
	pw := labPassword(r)
	ctx, cancel := context.WithTimeout(r.Context(), labArchiveTimeout)
	defer cancel()
	err = labWithEdit(ctx, real, func(cp string) error {
		switch req.Op {
		case "delete":
			if len(req.Items) == 0 {
				return fmt.Errorf("nada para apagar")
			}
			args := append([]string{"d", "-bd", "-spd", "-sccUTF-8"}, labPwArg(pw)...)
			args = append(args, "--", cp)
			for _, it := range req.Items {
				p, e := labInner(it)
				if e != nil || p == "" {
					return fmt.Errorf("item inválido")
				}
				args = append(args, p)
			}
			_, e := labRun(ctx, "", args...)
			return e
		case "rename":
			from, e1 := labInner(req.From)
			to, e2 := labInner(req.To)
			if e1 != nil || e2 != nil || from == "" || to == "" {
				return fmt.Errorf("nome inválido")
			}
			args := append([]string{"rn", "-bd", "-spd", "-sccUTF-8"}, labPwArg(pw)...)
			args = append(args, "--", cp, from, to)
			_, e := labRun(ctx, "", args...)
			return e
		case "mkdir", "save":
			tmpDir, e := os.MkdirTemp("", "lab-arc-*")
			if e != nil {
				return e
			}
			defer os.RemoveAll(tmpDir)
			var rel string
			if req.Op == "mkdir" {
				parent, e1 := labInner(req.Dir)
				nm, e2 := labInner(req.Name)
				if e1 != nil || e2 != nil || nm == "" || strings.Contains(nm, "/") {
					return fmt.Errorf("nome inválido")
				}
				rel = path.Join(parent, nm)
				if e := os.MkdirAll(filepath.Join(tmpDir, filepath.FromSlash(rel)), 0o755); e != nil {
					return e
				}
			} else {
				in, e1 := labInner(req.Inner)
				if e1 != nil || in == "" {
					return fmt.Errorf("arquivo inválido")
				}
				rel = in
				full := filepath.Join(tmpDir, filepath.FromSlash(rel))
				if e := os.MkdirAll(filepath.Dir(full), 0o755); e != nil {
					return e
				}
				if e := os.WriteFile(full, []byte(req.Text), 0o644); e != nil {
					return e
				}
			}
			top := strings.SplitN(rel, "/", 2)[0]
			args := append([]string{"a", "-bd", "-spd", "-sccUTF-8"}, labPwArg(pw)...)
			args = append(args, "--", cp, top)
			_, e = labRun(ctx, tmpDir, args...)
			return e
		}
		return fmt.Errorf("operação desconhecida")
	})
	if err != nil {
		return http.StatusBadRequest, err
	}
	return labJSON(w, map[string]string{"status": "ok"})
}

// POST /api/lab/archive/add  (multipart: source, path, dir, rel[] ..., file[] ...)
func labArchiveAddHandler(w http.ResponseWriter, r *http.Request, d *Context) (int, error) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return http.StatusBadRequest, fmt.Errorf("envio inválido: %v", err)
	}
	defer r.MultipartForm.RemoveAll()
	real, st, err := labReal(d, r.FormValue("source"), r.FormValue("path"), "modify")
	if err != nil {
		return st, err
	}
	dir, err := labInner(r.FormValue("dir"))
	if err != nil {
		return http.StatusBadRequest, err
	}
	files := r.MultipartForm.File["file"]
	rels := r.MultipartForm.Value["rel"]
	if len(files) == 0 {
		return http.StatusBadRequest, fmt.Errorf("nenhum arquivo enviado")
	}
	tmpDir, err := os.MkdirTemp("", "lab-arc-*")
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer os.RemoveAll(tmpDir)
	tops := map[string]bool{}
	for i, fh := range files {
		name := fh.Filename
		if i < len(rels) && rels[i] != "" {
			name = rels[i]
		}
		in, e := labInner(name)
		if e != nil || in == "" {
			return http.StatusBadRequest, fmt.Errorf("nome de arquivo inválido")
		}
		rel := path.Join(dir, in)
		full := filepath.Join(tmpDir, filepath.FromSlash(rel))
		if e := os.MkdirAll(filepath.Dir(full), 0o755); e != nil {
			return http.StatusInternalServerError, e
		}
		src, e := fh.Open()
		if e != nil {
			return http.StatusBadRequest, e
		}
		dst, e := os.Create(full)
		if e != nil {
			src.Close()
			return http.StatusInternalServerError, e
		}
		_, e = io.Copy(dst, src)
		src.Close()
		dst.Close()
		if e != nil {
			return http.StatusInternalServerError, e
		}
		tops[strings.SplitN(rel, "/", 2)[0]] = true
	}
	pw := labPassword(r)
	ctx, cancel := context.WithTimeout(r.Context(), labArchiveTimeout)
	defer cancel()
	err = labWithEdit(ctx, real, func(cp string) error {
		args := append([]string{"a", "-bd", "-spd", "-sccUTF-8"}, labPwArg(pw)...)
		args = append(args, "--", cp)
		for t := range tops {
			args = append(args, t)
		}
		_, e := labRun(ctx, tmpDir, args...)
		return e
	})
	if err != nil {
		return http.StatusBadRequest, err
	}
	return labJSON(w, map[string]any{"status": "ok", "added": len(files)})
}

// POST /api/lab/archive/create {source, dir, name, items:[nomes na pasta dir]}
type labCreateReq struct {
	Source   string   `json:"source"`
	Dir      string   `json:"dir"`
	Name     string   `json:"name"`
	Items    []string `json:"items"`
	Password string   `json:"password"`
}

func labArchiveCreateHandler(w http.ResponseWriter, r *http.Request, d *Context) (int, error) {
	var req labCreateReq
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		return http.StatusBadRequest, fmt.Errorf("corpo inválido: %v", err)
	}
	realDir, st, err := labReal(d, req.Source, req.Dir, "create")
	if err != nil {
		return st, err
	}
	if _, st, err := labReal(d, req.Source, req.Dir, "download"); err != nil {
		return st, err
	}
	typ, ok := labEditableExt[strings.ToLower(filepath.Ext(req.Name))]
	if !ok || req.Name == "" || strings.ContainsAny(req.Name, "/\\") || len(req.Items) == 0 {
		return http.StatusBadRequest, fmt.Errorf("use um nome terminando em .zip, .7z ou .tar")
	}
	target := filepath.Join(realDir, req.Name)
	if _, err := os.Stat(target); err == nil {
		return http.StatusConflict, fmt.Errorf("já existe um arquivo com esse nome")
	}
	args := []string{"a", "-t" + typ, "-bd", "-spd", "-sccUTF-8"}
	if typ != "tar" {
		args = append(args, labPwArg(req.Password)...)
	}
	tmp, err := os.CreateTemp(realDir, ".lab-new-*."+typ)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	tmpName := tmp.Name()
	tmp.Close()
	os.Remove(tmpName) // o 7-Zip cria o arquivo; só queríamos um nome livre na mesma pasta
	defer os.Remove(tmpName)
	args = append(args, "--", tmpName)
	for _, it := range req.Items {
		n, e := labInner(it)
		if e != nil || n == "" || strings.Contains(n, "/") {
			return http.StatusBadRequest, fmt.Errorf("item inválido")
		}
		if _, e := os.Lstat(filepath.Join(realDir, n)); e != nil {
			return http.StatusNotFound, fmt.Errorf("não achei %s", n)
		}
		args = append(args, n)
	}
	ctx, cancel := context.WithTimeout(r.Context(), labArchiveTimeout)
	defer cancel()
	if _, err := labRun(ctx, realDir, args...); err != nil {
		return http.StatusBadRequest, err
	}
	if err := os.Rename(tmpName, target); err != nil {
		return http.StatusInternalServerError, err
	}
	return labJSON(w, map[string]string{"status": "ok", "name": req.Name})
}

// POST /api/lab/archive/extract {source, path, toSource, dest, items?, password?}
type labExtractReq struct {
	Source   string   `json:"source"`
	Path     string   `json:"path"`
	ToSource string   `json:"toSource"`
	Dest     string   `json:"dest"`
	Items    []string `json:"items"`
	Password string   `json:"password"`
}

func labArchiveExtractHandler(w http.ResponseWriter, r *http.Request, d *Context) (int, error) {
	var req labExtractReq
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		return http.StatusBadRequest, fmt.Errorf("corpo inválido: %v", err)
	}
	if req.ToSource == "" {
		req.ToSource = req.Source
	}
	real, st, err := labReal(d, req.Source, req.Path, "download")
	if err != nil {
		return st, err
	}
	destReal, st, err := labReal(d, req.ToSource, req.Dest, "create")
	if err != nil {
		return st, err
	}
	if fi, e := os.Stat(destReal); e != nil || !fi.IsDir() {
		return http.StatusBadRequest, fmt.Errorf("a pasta de destino não existe")
	}
	ctx, cancel := context.WithTimeout(r.Context(), labArchiveTimeout)
	defer cancel()
	raws, err := labList(ctx, real, req.Password)
	if err != nil {
		return http.StatusBadRequest, err
	}
	var total int64
	for _, e := range raws {
		if e.link {
			return http.StatusBadRequest, fmt.Errorf("este compactado contém atalhos (links simbólicos); por segurança não é extraído")
		}
		total += e.size
	}
	if total > labArchiveMaxOut {
		return http.StatusBadRequest, fmt.Errorf("conteúdo grande demais para extrair (limite %d GB)", labArchiveMaxOut>>30)
	}
	args := append([]string{"x", "-y", "-aou", "-bd", "-spd", "-sccUTF-8", "-o" + destReal}, labPwArg(req.Password)...)
	args = append(args, "--", real)
	for _, it := range req.Items {
		p, e := labInner(it)
		if e != nil || p == "" {
			return http.StatusBadRequest, fmt.Errorf("item inválido")
		}
		args = append(args, p)
	}
	if _, err := labRun(ctx, "", args...); err != nil {
		return http.StatusBadRequest, err
	}
	return labJSON(w, map[string]string{"status": "ok"})
}
