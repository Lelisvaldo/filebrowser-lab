package web

import (
	"encoding/json"
	"net/http"

	"github.com/gtsteffaniak/filebrowser/backend/internal/adapters/fs/fileutils"
)

// labJobHandler devolve o andamento de um mover/copiar iniciado pelo próprio usuário (id aleatório gerado pela interface).
// @Summary Progress of a move/copy job
// @Tags Resources
// @Param id query string true "Job id sent in the PATCH /api/resources body"
// @Success 200 {object} fileutils.ProgressSnapshot
// @Router /api/lab/job [get]
func labJobHandler(w http.ResponseWriter, r *http.Request, d *Context) (int, error) {
	p := fileutils.GetJob(r.URL.Query().Get("id"), d.User.ID)
	if p == nil {
		return http.StatusNotFound, nil
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	return http.StatusOK, json.NewEncoder(w).Encode(p.Snapshot())
}
