package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"mechazone/cloud-backend/internal/ledger"
)

func (s *Server) createShop(w http.ResponseWriter, r *http.Request) {
	var in ledger.CreateShopInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "malformed shop")
		return
	}
	shop, err := s.store.CreateShop(r.Context(), in)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, shop)
}

func (s *Server) listShops(w http.ResponseWriter, r *http.Request) {
	shops, err := s.store.ListShops(r.Context())
	if err != nil {
		s.log.Error("list shops", "err", err)
		writeError(w, http.StatusInternalServerError, "list shops failed")
		return
	}
	writeJSON(w, http.StatusOK, shops)
}

func (s *Server) createTechnician(w http.ResponseWriter, r *http.Request) {
	var in ledger.CreateTechnicianInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "malformed technician")
		return
	}
	tech, err := s.store.CreateTechnician(r.Context(), in)
	if err != nil {
		status := http.StatusUnprocessableEntity
		if strings.Contains(err.Error(), "already") {
			status = http.StatusConflict
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, tech)
}

func (s *Server) listTechnicians(w http.ResponseWriter, r *http.Request) {
	techs, err := s.store.ListTechnicians(r.Context())
	if err != nil {
		s.log.Error("list technicians", "err", err)
		writeError(w, http.StatusInternalServerError, "list technicians failed")
		return
	}
	writeJSON(w, http.StatusOK, techs)
}
