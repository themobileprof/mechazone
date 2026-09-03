package httpapi

import (
	"net/http"
	"path/filepath"
	"strings"
)

// figureImage serves a retrieved manual figure. Paths must stay under the ingested tree.
func (s *Server) figureImage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id required")
		return
	}
	path, err := s.store.FigureImagePath(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "figure not found")
		return
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	case ".webp":
		w.Header().Set("Content-Type", "image/webp")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	http.ServeFile(w, r, path)
}
