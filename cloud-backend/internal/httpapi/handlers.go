package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"mechazone/cloud-backend/internal/ai"
	"mechazone/cloud-backend/internal/ledger"
	"mechazone/cloud-backend/internal/pii"
	"mechazone/cloud-backend/internal/vin"
)

func (s *Server) vehicleHistory(w http.ResponseWriter, r *http.Request) {
	v, err := pathVIN(r)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	p, ok := principalFrom(r.Context())
	if !ok || p.TechnicianID == "" {
		writeError(w, http.StatusForbidden, "technician login required")
		return
	}
	h, err := s.store.History(r.Context(), v, p.ShopID, p.TechnicianID)
	if err != nil {
		s.log.Error("history", "err", err, "vin", v)
		writeError(w, http.StatusInternalServerError, "ledger read failed")
		return
	}
	writeJSON(w, http.StatusOK, h)
}

func (s *Server) decodeVIN(w http.ResponseWriter, r *http.Request) {
	v, err := pathVIN(r)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	payload, source, hit, err := s.store.CachedVIN(r.Context(), v)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cache read failed")
		return
	}
	if hit {
		writeJSON(w, http.StatusOK, map[string]any{
			"vin":     v,
			"source":  source,
			"cached":  true,
			"payload": json.RawMessage(payload),
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	dec, err := s.vins.Decode(ctx, v)
	if err != nil {
		s.log.Warn("vpic", "err", err, "vin", v)
		writeError(w, http.StatusBadGateway, "vpic unavailable")
		return
	}
	if err := s.store.SaveVINDecode(r.Context(), dec); err != nil {
		s.log.Error("vin cache write", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to cache vin decode")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"vin":     dec.VIN,
		"make":    dec.Make,
		"model":   dec.Model,
		"year":    dec.Year,
		"source":  dec.Source,
		"cached":  false,
		"empty":   dec.Empty,
		"payload": dec.Raw,
	})
}

func (s *Server) lookupDTC(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(strings.TrimSpace(r.PathValue("code")))
	if code == "" {
		writeError(w, http.StatusBadRequest, "code required")
		return
	}
	d, err := s.store.LookupDTC(r.Context(), code)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "dtc lookup failed")
		return
	}
	class := ai.ClassifyCode(d.Code, d.Title)
	if d.Title == "" {
		writeJSON(w, http.StatusOK, map[string]any{
			"code":              d.Code,
			"category":          "manufacturer_or_unknown",
			"title":             "",
			"circuit_class":     class.Class,
			"circuit_reason":    class.Reason,
			"cloud_ai_reserved": true,
			"note":              "Not a seeded SAE P0xxx definition. Use this shop's jobs and retrieved manuals.",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"code":           d.Code,
		"category":       d.Category,
		"title":          d.Title,
		"source":         d.Source,
		"circuit_class":  class.Class,
		"circuit_reason": class.Reason,
	})
}

// ingestSession records a live (or queued) scan. Shop and technician IDs come from the cookie.
func (s *Server) ingestSession(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "unreadable body")
		return
	}
	if err := pii.RejectCloudPII(body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	var in ledger.SessionIngest
	if err := json.Unmarshal(body, &in); err != nil {
		writeError(w, http.StatusBadRequest, "malformed session payload")
		return
	}
	normalized, err := vin.Normalize(in.VIN)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	in.VIN = normalized
	p, ok := principalFrom(r.Context())
	if !ok || p.TechnicianID == "" {
		writeError(w, http.StatusForbidden, "technician login required")
		return
	}
	in.ShopID = p.ShopID
	in.TechnicianID = p.TechnicianID
	if in.AdapterType == "" || in.HostOS == "" || in.Protocol == "" {
		writeError(w, http.StatusUnprocessableEntity, "adapter_type, host_os, and protocol are required")
		return
	}
	if in.ActiveCodes == nil {
		in.ActiveCodes = []string{}
	}

	if err := s.store.EnsureVehicle(r.Context(), in.VIN, in.MakeHint, in.ModelHint, in.YearHint, "session"); err != nil {
		s.log.Error("ensure vehicle", "err", err)
		writeError(w, http.StatusInternalServerError, "vehicle persist failed")
		return
	}

	payload, _, hit, err := s.store.CachedVIN(r.Context(), in.VIN)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "vin cache read failed")
		return
	}
	if !hit {
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		dec, decErr := s.vins.Decode(ctx, in.VIN)
		cancel()
		if decErr != nil {
			s.log.Warn("vpic on ingest", "err", decErr, "vin", in.VIN)
		} else if err := s.store.SaveVINDecode(r.Context(), dec); err != nil {
			s.log.Warn("vin cache write on ingest", "err", err)
		}
		_ = payload
	}

	sess, err := s.store.InsertSession(r.Context(), in)
	if err != nil {
		s.log.Error("ingest", "err", err)
		writeError(w, http.StatusInternalServerError, "session persist failed")
		return
	}
	writeJSON(w, http.StatusAccepted, sess)
}

// closeout records what this shop did on this visit. Technician ID comes from the cookie.
func (s *Server) closeout(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "session id required")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		writeError(w, http.StatusBadRequest, "unreadable body")
		return
	}
	if err := pii.RejectCloudPII(body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	var c ledger.Closeout
	if err := json.Unmarshal(body, &c); err != nil {
		writeError(w, http.StatusBadRequest, "malformed closeout")
		return
	}
	if strings.TrimSpace(c.RootCause) == "" {
		writeError(w, http.StatusUnprocessableEntity, "root_cause_explanation is required")
		return
	}
	if c.Parts == nil {
		c.Parts = []string{}
	}
	p, ok := principalFrom(r.Context())
	if !ok || p.TechnicianID == "" {
		writeError(w, http.StatusForbidden, "technician login required")
		return
	}
	res, err := s.store.Closeout(r.Context(), id, p.TechnicianID, c)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "outcome") {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		s.log.Error("closeout", "err", err)
		writeError(w, http.StatusInternalServerError, "closeout failed")
		return
	}
	writeJSON(w, http.StatusOK, res)
}
