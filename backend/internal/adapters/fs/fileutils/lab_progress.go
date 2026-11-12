package fileutils

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Progresso de mover/copiar (lab): o PATCH /api/resources é uma chamada só e não informa nada até terminar.
// Aqui o servidor conta os bytes copiados do item em andamento e a interface consulta (GET /api/lab/job).
// Mover na mesma partição é só renomear (instantâneo, sem bytes); só cópia entre partições e "copiar" têm progresso.

// Progress é o andamento de uma operação (um PATCH com um ou mais itens).
type Progress struct {
	UserID    uint64
	Items     int
	StartedAt time.Time

	mu       sync.Mutex
	item     int    // item em andamento (1..Items)
	name     string // nome do item em andamento
	curTotal int64
	finished bool

	curDone atomic.Int64
}

// ProgressSnapshot é o que a interface recebe.
type ProgressSnapshot struct {
	Items     int    `json:"items"`
	Item      int    `json:"item"`
	Name      string `json:"name"`
	Total     int64  `json:"total"`
	Done      int64  `json:"done"`
	Finished  bool   `json:"finished"`
	ElapsedMs int64  `json:"elapsedMs"`
}

var (
	labJobsMu sync.Mutex
	labJobs   = map[string]*Progress{}
	labSrcs   = map[string]*Progress{} // caminho de origem em cópia -> operação
)

// StartJob registra uma operação para consulta. O id vem do cliente (aleatório); guarda por 5 min depois de terminar.
func StartJob(id string, userID uint64, items int) *Progress {
	if id == "" || len(id) > 64 {
		return nil
	}
	p := &Progress{UserID: userID, Items: items, StartedAt: time.Now()}
	labJobsMu.Lock()
	labJobs[id] = p
	labJobsMu.Unlock()
	return p
}

// GetJob devolve a operação só ao usuário que a iniciou.
func GetJob(id string, userID uint64) *Progress {
	labJobsMu.Lock()
	defer labJobsMu.Unlock()
	p := labJobs[id]
	if p == nil || p.UserID != userID {
		return nil
	}
	return p
}

// Finish marca como terminada e agenda a limpeza do registro.
func (p *Progress) Finish(id string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.finished = true
	p.mu.Unlock()
	time.AfterFunc(5*time.Minute, func() {
		labJobsMu.Lock()
		delete(labJobs, id)
		labJobsMu.Unlock()
	})
}

// BeginItem começa a contar o item (soma o tamanho da origem) e registra a origem para a cópia reportar os bytes.
func (p *Progress) BeginItem(src string) {
	if p == nil {
		return
	}
	total := SizeOf(src)
	p.mu.Lock()
	p.item++
	p.name = filepath.Base(src)
	p.curTotal = total
	p.mu.Unlock()
	p.curDone.Store(0)
	labJobsMu.Lock()
	labSrcs[filepath.Clean(src)] = p
	labJobsMu.Unlock()
}

// EndItem encerra o item: a barra do item vai a 100% mesmo que tenha sido só um renomear.
func (p *Progress) EndItem(src string) {
	if p == nil {
		return
	}
	labJobsMu.Lock()
	delete(labSrcs, filepath.Clean(src))
	labJobsMu.Unlock()
	p.mu.Lock()
	p.curDone.Store(p.curTotal)
	p.mu.Unlock()
}

// Snapshot lê o andamento.
func (p *Progress) Snapshot() ProgressSnapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	done := p.curDone.Load()
	if done > p.curTotal {
		done = p.curTotal
	}
	return ProgressSnapshot{Items: p.Items, Item: p.item, Name: p.name, Total: p.curTotal, Done: done, Finished: p.finished, ElapsedMs: time.Since(p.StartedAt).Milliseconds()}
}

// progressFor acha a operação dona deste caminho (a origem registrada ou uma pasta acima dele).
func progressFor(path string) *Progress {
	labJobsMu.Lock()
	defer labJobsMu.Unlock()
	if len(labSrcs) == 0 {
		return nil
	}
	c := filepath.Clean(path)
	for src, p := range labSrcs {
		if c == src || strings.HasPrefix(c, src+string(os.PathSeparator)) {
			return p
		}
	}
	return nil
}

// SizeOf soma o tamanho dos arquivos de um caminho (arquivo ou pasta).
func SizeOf(path string) int64 {
	var n int64
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err == nil && info.Mode().IsRegular() {
			n += info.Size()
		}
		return nil
	})
	return n
}

type countWriter struct {
	w io.Writer
	p *Progress
}

func (c *countWriter) Write(b []byte) (int, error) {
	n, err := c.w.Write(b)
	c.p.curDone.Add(int64(n))
	return n, err
}

// copyWithProgress copia em blocos de 1 MiB contando os bytes (esconde ReadFrom/copy_file_range de propósito: não há como contar).
func copyWithProgress(dst io.Writer, src io.Reader, p *Progress) error {
	_, err := io.CopyBuffer(&countWriter{w: dst, p: p}, struct{ io.Reader }{src}, make([]byte, 1<<20))
	return err
}
