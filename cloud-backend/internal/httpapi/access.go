package httpapi

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"mechazone/cloud-backend/internal/ledger"
)

type requestLimiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

func newRequestLimiter() *requestLimiter {
	return &requestLimiter{hits: map[string][]time.Time{}}
}

func (l *requestLimiter) allow(key string, n int, window time.Duration) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	keep := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if now.Sub(t) < window {
			keep = append(keep, t)
		}
	}
	if len(keep) >= n {
		l.hits[key] = keep
		return false
	}
	l.hits[key] = append(keep, now)
	return true
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// createAccessRequest is the landing ticket. Super admin issues logins; this is not signup.
func (s *Server) createAccessRequest(w http.ResponseWriter, r *http.Request) {
	if !s.accessLimit.allow(clientIP(r), 5, time.Hour) {
		writeError(w, http.StatusTooManyRequests, "too many requests from this network — try again later")
		return
	}
	var in ledger.CreateAccessRequestInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "malformed request")
		return
	}
	row, err := s.store.CreateAccessRequest(r.Context(), in)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":          "queued",
		"already_queued":  row.AlreadyQueued,
		"id":              row.ID,
		"contact_email":   row.ContactEmail,
	})
}

func (s *Server) listAccessRequests(w http.ResponseWriter, r *http.Request) {
	rows, err := s.store.ListAccessRequests(r.Context())
	if err != nil {
		s.log.Error("list access requests", "err", err)
		writeError(w, http.StatusInternalServerError, "list access requests failed")
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) setAccessRequestStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id required")
		return
	}
	var in struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "malformed status")
		return
	}
	row, err := s.store.SetAccessRequestStatus(r.Context(), id, in.Status)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, row)
}
