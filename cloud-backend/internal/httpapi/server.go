package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"mechazone/cloud-backend/internal/config"
	"mechazone/cloud-backend/internal/ledger"
	"mechazone/cloud-backend/internal/vin"
)

type Server struct {
	cfg    config.Config
	store  *ledger.Store
	vpic   *vin.Client
	log    *slog.Logger
}

func New(cfg config.Config, store *ledger.Store, vpic *vin.Client, log *slog.Logger) http.Handler {
	s := &Server{cfg: cfg, store: store, vpic: vpic, log: log}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /api/v1/vehicles/{vin}", s.vehicleHistory)
	mux.HandleFunc("POST /api/v1/vehicles/{vin}/decode", s.decodeVIN)
	mux.HandleFunc("GET /api/v1/dtcs/{code}", s.lookupDTC)
	mux.HandleFunc("POST /api/v1/sessions", s.ingestSession)
	mux.HandleFunc("POST /api/v1/sessions/{id}/closeout", s.closeout)
	return withCORS(withLog(log, mux))
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func withLog(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Info("http", "method", r.Method, "path", r.URL.Path, "dur_ms", time.Since(start).Milliseconds())
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func pathVIN(r *http.Request) (string, error) {
	return vin.Normalize(r.PathValue("vin"))
}
