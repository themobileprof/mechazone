package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"mechazone/cloud-backend/internal/auth"
)

type ctxKey int

const principalKey ctxKey = 1

func withPrincipal(ctx context.Context, p auth.Principal) context.Context {
	return context.WithValue(ctx, principalKey, p)
}

func principalFrom(ctx context.Context) (auth.Principal, bool) {
	p, ok := ctx.Value(principalKey).(auth.Principal)
	return p, ok
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "malformed login")
		return
	}
	p, err := s.store.Authenticate(r.Context(), in.Email, in.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	raw, err := s.store.CreateSession(r.Context(), p.UserID)
	if err != nil {
		s.log.Error("create session", "err", err)
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}
	auth.SetSessionCookie(w, raw, cookieSecure())
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	_ = s.store.RevokeToken(r.Context(), auth.TokenFromRequest(r))
	auth.ClearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "signed_out"})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	p, ok := principalFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := s.store.PrincipalByToken(r.Context(), auth.TokenFromRequest(r))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "not authenticated")
			return
		}
		next(w, r.WithContext(withPrincipal(r.Context(), p)))
	}
}

func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		p, _ := principalFrom(r.Context())
		if p.Role != "super_admin" {
			writeError(w, http.StatusForbidden, "super admin only")
			return
		}
		next(w, r)
	})
}

// requireTechnician rejects super-admin cookies on bay routes (history, ingest, playbook).
func (s *Server) requireTechnician(next http.HandlerFunc) http.HandlerFunc {
	return s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		p, _ := principalFrom(r.Context())
		if p.Role != "technician" || p.TechnicianID == "" {
			writeError(w, http.StatusForbidden, "technician login required")
			return
		}
		next(w, r)
	})
}

func cookieSecure() bool {
	return strings.EqualFold(os.Getenv("COOKIE_SECURE"), "true")
}
