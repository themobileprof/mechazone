package httpapi

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// withUI serves client/dist for non-API paths so an installed laptop uses one port.
func withUI(uiDir string, api http.Handler) http.Handler {
	if strings.TrimSpace(uiDir) == "" {
		return api
	}
	root := http.Dir(uiDir)
	files := http.FileServer(root)
	index := filepath.Join(uiDir, "index.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || strings.HasPrefix(r.URL.Path, "/api/") {
			api.ServeHTTP(w, r)
			return
		}
		rel := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if rel == "." || rel == "" {
			http.ServeFile(w, r, index)
			return
		}
		f, err := root.Open(rel)
		if err != nil {
			if _, statErr := os.Stat(index); statErr == nil {
				http.ServeFile(w, r, index)
				return
			}
			http.NotFound(w, r)
			return
		}
		_ = f.Close()
		files.ServeHTTP(w, r)
	})
}
