package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"mechazone/cloud-backend/internal/ai"
	"mechazone/cloud-backend/internal/pii"
	"mechazone/cloud-backend/internal/vin"
)

// buildPlaybook fuses this shop's jobs and retrieved manuals with the posted scan. Scope IDs from the cookie.
func (s *Server) buildPlaybook(w http.ResponseWriter, r *http.Request) {
	req, ok := s.decodePlaybookRequest(w, r)
	if !ok {
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

// askPlaybook answers a technician question about one playbook step using the same fuse context.
func (s *Server) askPlaybook(w http.ResponseWriter, r *http.Request) {
	if s.fuser == nil {
		writeError(w, http.StatusServiceUnavailable, "playbook AI is not configured")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "unreadable body")
		return
	}
	if err := pii.RejectCloudPII(body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	var req ai.AskRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed playbook ask")
		return
	}
	if _, err := vin.Normalize(req.VIN); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	p, ok := principalFrom(r.Context())
	if !ok || p.TechnicianID == "" {
		writeError(w, http.StatusForbidden, "technician login required")
		return
	}
	req.ShopID = p.ShopID
	req.TechnicianID = p.TechnicianID
	reply, err := s.fuser.Ask(r.Context(), req)
	if err != nil {
		s.log.Error("playbook ask", "err", err)
		status := http.StatusBadGateway
		msg := err.Error()
		if strings.Contains(msg, "required") || strings.Contains(msg, "too long") {
			status = http.StatusUnprocessableEntity
		}
		writeError(w, status, msg)
		return
	}
	writeJSON(w, http.StatusOK, reply)
}

func (s *Server) decodePlaybookRequest(w http.ResponseWriter, r *http.Request) (ai.Request, bool) {
	if s.fuser == nil {
		writeError(w, http.StatusServiceUnavailable, "playbook AI is not configured")
		return ai.Request{}, false
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "unreadable body")
		return ai.Request{}, false
	}
	if err := pii.RejectCloudPII(body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return ai.Request{}, false
	}
	var req ai.Request
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed playbook request")
		return ai.Request{}, false
	}
	if _, err := vin.Normalize(req.VIN); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return ai.Request{}, false
	}
	p, ok := principalFrom(r.Context())
	if !ok || p.TechnicianID == "" {
		writeError(w, http.StatusForbidden, "technician login required")
		return ai.Request{}, false
	}
	req.ShopID = p.ShopID
	req.TechnicianID = p.TechnicianID
	return req, true
}
