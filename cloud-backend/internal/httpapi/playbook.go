package httpapi

import (
	"encoding/json"
	"net/http"

	"mechazone/cloud-backend/internal/ai"
	"mechazone/cloud-backend/internal/vin"
)

func (s *Server) buildPlaybook(w http.ResponseWriter, r *http.Request) {
	if s.fuser == nil {
		writeError(w, http.StatusServiceUnavailable, "playbook AI is not configured")
		return
	}
	var req ai.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed playbook request")
		return
	}
	if _, err := vin.Normalize(req.VIN); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	book, err := s.fuser.Build(r.Context(), req)
	if err != nil {
		s.log.Error("playbook", "err", err)
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, book)
}
