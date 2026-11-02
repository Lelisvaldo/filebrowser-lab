package files

// Lixeira (filebrowser-lab, HL-129): em vez de apagar de vez, o "Apagar" move o item para <fonte>/.lixeira/<id>/,
// no mesmo disco (mover é instantâneo), e guarda quem apagou e de onde veio num <id>.json. Itens vencem em
// FILEBROWSER_LAB_TRASH_DAYS dias (padrão 30). Pendrives e Nuvens ficam de fora (FILEBROWSER_LAB_TRASH_OFF):
// o pendrive apaga direto e a nuvem usa a lixeira do próprio provedor.

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gtsteffaniak/filebrowser/backend/pkg/indexing"
	"github.com/gtsteffaniak/filebrowser/backend/pkg/settings"
	"github.com/gtsteffaniak/go-logger/logger"
)

const LabTrashDir = ".lixeira"

var labTrashID = regexp.MustCompile(`^[0-9]{10,16}-[0-9a-f]{8}$`)

type TrashItem struct {
	ID        string `json:"id"`
	Source    string `json:"source"`
	Path      string `json:"path"` // caminho original dentro da fonte, ex.: /Filmes/a.txt
	Name      string `json:"name"`
	IsDir     bool   `json:"isDir"`
	Size      int64  `json:"size"` // -1: não calculado (pasta grande)
	DeletedAt int64  `json:"deletedAt"`
	By        string `json:"by"`
	ExpiresAt int64  `json:"expiresAt"`
}

func LabTrashDays() int {
	if n, err := strconv.Atoi(os.Getenv("FILEBROWSER_LAB_TRASH_DAYS")); err == nil && n > 0 {
		return n
	}
	return 30
}

// fontes sem lixeira (por nome, sem diferenciar maiúsculas); FILEBROWSER_LAB_TRASH=off desliga em todas
func LabTrashEnabled(source string) bool {
	if strings.EqualFold(os.Getenv("FILEBROWSER_LAB_TRASH"), "off") {
		return false
	}
	off := os.Getenv("FILEBROWSER_LAB_TRASH_OFF")
	if off == "" {
		off = "Pendrives,Nuvens"
	}
	for _, n := range strings.Split(off, ",") {
		if strings.EqualFold(strings.TrimSpace(n), source) {
			return false
		}
	}
	return true
}

func labTrashRoot(root string) string { return filepath.Join(root, LabTrashDir) }

func inLabTrash(root, abs string) bool {
	rel, err := filepath.Rel(labTrashRoot(root), abs)
	return err == nil && !strings.HasPrefix(rel, "..")
}

// tamanho: arquivo direto; pasta somando, com limite de 3 s (depois disso fica -1)
func labTrashSize(abs string, isDir bool) int64 {
	st, err := os.Lstat(abs)
	if err != nil {
		return -1
	}
	if !isDir {
		return st.Size()
	}
	deadline := time.Now().Add(3 * time.Second)
	var total int64
	stop := fmt.Errorf("tempo")
	err = filepath.WalkDir(abs, func(p string, d os.DirEntry, err error) error {
		if time.Now().After(deadline) {
			return stop
		}
		if err == nil && !d.IsDir() {
			if fi, e := d.Info(); e == nil {
				total += fi.Size()
			}
		}
		return nil
	})
	if err != nil {
		return -1
	}
	return total
}

// TrashMove move abs (dentro de root) para a lixeira da fonte. rel é o caminho dentro da fonte.
func TrashMove(root, source, rel, abs string, isDir bool, actor string) (*TrashItem, error) {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	id := fmt.Sprintf("%d-%s", time.Now().UnixMilli(), hex.EncodeToString(b))
	itemDir := filepath.Join(labTrashRoot(root), id)
	if err := os.MkdirAll(itemDir, 0o755); err != nil {
		return nil, fmt.Errorf("não foi possível criar a lixeira: %w", err)
	}
	it := TrashItem{ID: id, Source: source, Path: "/" + strings.Trim(rel, "/"), Name: filepath.Base(abs), IsDir: isDir,
		Size: labTrashSize(abs, isDir), DeletedAt: time.Now().Unix(), By: actor}
	if err := os.Rename(abs, filepath.Join(itemDir, it.Name)); err != nil {
		_ = os.Remove(itemDir)
		return nil, fmt.Errorf("não foi possível mover para a lixeira: %w", err)
	}
	mb, _ := json.Marshal(it)
	if err := os.WriteFile(filepath.Join(labTrashRoot(root), id+".json"), mb, 0o644); err != nil {
		logger.Errorf("lixeira: não gravei os dados de %s: %v", id, err)
	}
	return &it, nil
}

func labTrashLoad(root, source, id string) (*TrashItem, error) {
	if !labTrashID.MatchString(id) {
		return nil, fmt.Errorf("item inválido")
	}
	b, err := os.ReadFile(filepath.Join(labTrashRoot(root), id+".json"))
	if err != nil {
		return nil, fmt.Errorf("item não encontrado na lixeira")
	}
	var it TrashItem
	if json.Unmarshal(b, &it) != nil || it.ID != id {
		return nil, fmt.Errorf("item da lixeira corrompido")
	}
	it.Source = source
	it.ExpiresAt = it.DeletedAt + int64(LabTrashDays())*86400
	return &it, nil
}

// TrashGet devolve os dados de um item da lixeira
func TrashGet(root, source, id string) (*TrashItem, error) { return labTrashLoad(root, source, id) }

// TrashList lista a lixeira de uma fonte (mais novos primeiro)
func TrashList(root, source string) []TrashItem {
	ents, err := os.ReadDir(labTrashRoot(root))
	if err != nil {
		return nil
	}
	var out []TrashItem
	for _, e := range ents {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".json") {
			continue
		}
		if it, err := labTrashLoad(root, source, strings.TrimSuffix(n, ".json")); err == nil {
			out = append(out, *it)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DeletedAt > out[j].DeletedAt })
	return out
}

// TrashRestore devolve o item ao lugar de origem (com " (1)" se já houver algo lá) e devolve o caminho final dentro da fonte
func TrashRestore(root, source, id string) (string, error) {
	it, err := labTrashLoad(root, source, id)
	if err != nil {
		return "", err
	}
	from := filepath.Join(labTrashRoot(root), id, it.Name)
	if _, err := os.Lstat(from); err != nil {
		return "", fmt.Errorf("o arquivo não está mais na lixeira")
	}
	rel := "/" + strings.Trim(it.Path, "/")
	target := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	final := target
	for i := 2; ; i++ {
		if _, err := os.Lstat(final); os.IsNotExist(err) {
			break
		}
		ext := ""
		base := filepath.Base(target)
		if !it.IsDir {
			ext = filepath.Ext(base)
			base = strings.TrimSuffix(base, ext)
		}
		final = filepath.Join(filepath.Dir(target), fmt.Sprintf("%s (%d)%s", base, i, ext))
		if i > 999 {
			return "", fmt.Errorf("nome indisponível no destino")
		}
	}
	if err := os.Rename(from, final); err != nil {
		return "", err
	}
	_ = os.RemoveAll(filepath.Join(labTrashRoot(root), id))
	_ = os.Remove(filepath.Join(labTrashRoot(root), id+".json"))
	relFinal, _ := filepath.Rel(root, final)
	return "/" + filepath.ToSlash(relFinal), nil
}

// TrashPurge apaga de vez um item da lixeira
func TrashPurge(root, id string) error {
	if !labTrashID.MatchString(id) {
		return fmt.Errorf("item inválido")
	}
	if err := os.RemoveAll(filepath.Join(labTrashRoot(root), id)); err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(labTrashRoot(root), id+".json"))
	return nil
}

// TrashPurgeExpired apaga o que passou do prazo (e restos sem dados); devolve quantos itens saíram
func TrashPurgeExpired(root, source string, now time.Time) int {
	n := 0
	for _, it := range TrashList(root, source) {
		if now.Unix() >= it.ExpiresAt {
			if TrashPurge(root, it.ID) == nil {
				n++
			}
		}
	}
	// pastas de itens sem o .json (gravação interrompida) com mais de 1 dia
	if ents, err := os.ReadDir(labTrashRoot(root)); err == nil {
		for _, e := range ents {
			if !e.IsDir() || !labTrashID.MatchString(e.Name()) {
				continue
			}
			if _, err := os.Stat(filepath.Join(labTrashRoot(root), e.Name()+".json")); err == nil {
				continue
			}
			if fi, err := e.Info(); err == nil && now.Sub(fi.ModTime()) > 24*time.Hour {
				_ = os.RemoveAll(filepath.Join(labTrashRoot(root), e.Name()))
			}
		}
	}
	return n
}

// ---- ligação com as fontes configuradas ----

type TrashSource struct{ Name, Root string }

func TrashSources() []TrashSource {
	var out []TrashSource
	for _, s := range settings.Config.Server.SourceMap {
		if s.Name != "" && s.Path != "" && LabTrashEnabled(s.Name) {
			out = append(out, TrashSource{Name: s.Name, Root: s.Path})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func TrashSourceRoot(name string) (string, bool) {
	if !LabTrashEnabled(name) {
		return "", false
	}
	for _, s := range TrashSources() {
		if s.Name == name {
			return s.Root, true
		}
	}
	return "", false
}

// refresca o índice depois de restaurar
func TrashRefresh(source, finalRel string, isDir bool) {
	if idx := indexing.GetIndex(source); idx != nil {
		_ = RefreshIndex(source, filepath.Dir(finalRel), true, false)
		_ = RefreshIndex(source, finalRel, isDir, isDir)
	}
}

var labTrashJanitorOnce = make(chan struct{}, 1)

// StartTrashJanitor: limpa o que venceu 60 s depois de iniciar e a cada 6 horas
func StartTrashJanitor() {
	select {
	case labTrashJanitorOnce <- struct{}{}:
	default:
		return
	}
	go func() {
		time.Sleep(60 * time.Second)
		for {
			for _, s := range TrashSources() {
				if n := TrashPurgeExpired(s.Root, s.Name, time.Now()); n > 0 {
					logger.Infof("lixeira: %d item(ns) vencido(s) apagado(s) em %s", n, s.Name)
				}
			}
			time.Sleep(6 * time.Hour)
		}
	}()
}
