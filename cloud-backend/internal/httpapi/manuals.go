package httpapi

import (
	"net/http"

	"mechazone/cloud-backend/internal/ledger"
)

// listManuals returns ingested workshop books so the bay can pin one when decode misses the body.
func (s *Server) listManuals(w http.ResponseWriter, r *http.Request) {
	rows, err := s.store.ListManuals(r.Context())
	if err != nil {
		s.log.Error("manuals", "err", err)
		writeError(w, http.StatusInternalServerError, "manual catalog failed")
		return
	}
	if rows == nil {
		rows = []ledger.ManualCatalog{}
	}
	writeJSON(w, http.StatusOK, rows)
}
