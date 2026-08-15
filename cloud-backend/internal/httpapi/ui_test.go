package httpapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestWithUIServesIndexAndLeavesAPI(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>bay</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	api := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	h := withUI(dir, api)

	ui := httptest.NewRecorder()
	h.ServeHTTP(ui, httptest.NewRequest(http.MethodGet, "/", nil))
	if ui.Code != http.StatusOK || ui.Body.String() != "<html>bay</html>" {
		t.Fatalf("ui: %d %q", ui.Code, ui.Body.String())
	}

	spa := httptest.NewRecorder()
	h.ServeHTTP(spa, httptest.NewRequest(http.MethodGet, "/bay", nil))
	if spa.Code != http.StatusOK || spa.Body.String() != "<html>bay</html>" {
		t.Fatalf("spa: %d %q", spa.Code, spa.Body.String())
	}

	apiRec := httptest.NewRecorder()
	h.ServeHTTP(apiRec, httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil))
	if apiRec.Code != http.StatusTeapot {
		t.Fatalf("api: %d", apiRec.Code)
	}
}

func TestWithUIEmptyDirIsPassthrough(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	h := withUI("", api)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("got %d", rec.Code)
	}
}
