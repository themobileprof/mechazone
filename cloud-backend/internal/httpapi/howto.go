package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"mechazone/cloud-backend/internal/ledger"
)

func (s *Server) listPublishedHowTos(w http.ResponseWriter, r *http.Request) {
	guides, err := s.store.ListHowToGuides(r.Context(), true)
	if err != nil {
		s.log.Error("list howto", "err", err)
		writeError(w, http.StatusInternalServerError, "list how-to failed")
		return
	}
	writeJSON(w, http.StatusOK, guides)
}

func (s *Server) adminListHowTos(w http.ResponseWriter, r *http.Request) {
	guides, err := s.store.ListHowToGuides(r.Context(), false)
	if err != nil {
		s.log.Error("admin howto", "err", err)
		writeError(w, http.StatusInternalServerError, "list how-to failed")
		return
	}
	writeJSON(w, http.StatusOK, guides)
}

func (s *Server) adminListActions(w http.ResponseWriter, r *http.Request) {
	actions, err := s.store.ListPlaybookActions(r.Context())
	if err != nil {
		s.log.Error("admin actions", "err", err)
		writeError(w, http.StatusInternalServerError, "list actions failed")
		return
	}
	writeJSON(w, http.StatusOK, actions)
}

func decodeHowToIn(w http.ResponseWriter, r *http.Request) (ledger.HowToIn, bool) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 200<<10))
	if err != nil {
		writeError(w, http.StatusBadRequest, "unreadable body")
		return ledger.HowToIn{}, false
	}
	var in ledger.HowToIn
	if err := json.Unmarshal(body, &in); err != nil {
		writeError(w, http.StatusBadRequest, "malformed how-to")
		return ledger.HowToIn{}, false
	}
	return in, true
}

func (s *Server) adminCreateHowTo(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeHowToIn(w, r)
	if !ok {
		return
	}
	g, err := s.store.CreateHowToGuide(r.Context(), in)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, g)
}

func (s *Server) adminUpdateHowTo(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeHowToIn(w, r)
	if !ok {
		return
	}
	g, err := s.store.UpdateHowToGuide(r.Context(), r.PathValue("id"), in)
	if err != nil {
		status := http.StatusUnprocessableEntity
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (s *Server) adminDeleteHowTo(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteHowToGuide(r.Context(), r.PathValue("id")); err != nil {
		status := http.StatusUnprocessableEntity
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
