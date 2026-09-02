package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"mechazone/cloud-backend/internal/importreport"
	"mechazone/cloud-backend/internal/ledger"
	"mechazone/cloud-backend/internal/pii"
)

func (s *Server) attachImportedReport(w http.ResponseWriter, r *http.Request) {
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
	if strings.TrimSpace(s.cfg.ImportDir) == "" {
		writeError(w, http.StatusInternalServerError, "import storage is not configured")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, importreport.MaxBytes+1<<20)
	if err := r.ParseMultipartForm(importreport.MaxBytes + 1<<20); err != nil {
		if strings.Contains(err.Error(), "too large") {
			writeError(w, http.StatusRequestEntityTooLarge, "file exceeds 8 MB")
			return
		}
		writeError(w, http.StatusBadRequest, "expected a multipart file upload")
		return
	}

	fh, hdr, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer fh.Close()
	data, err := io.ReadAll(io.LimitReader(fh, importreport.MaxBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "unreadable file")
		return
	}
	if len(data) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "empty file")
		return
	}
	if len(data) > importreport.MaxBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "file exceeds 8 MB")
		return
	}

	head := data
	if len(head) > 512 {
		head = head[:512]
	}
	ct, ext, err := importreport.Sniff(head, hdr.Filename)
	if err != nil {
		writeError(w, http.StatusUnsupportedMediaType, err.Error())
		return
	}
	source, err := importreport.NormalizeSource(r.FormValue("source"))
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	note := strings.TrimSpace(r.FormValue("note"))
	if len(note) > 2000 {
		writeError(w, http.StatusUnprocessableEntity, "note is too long")
		return
	}
	if err := pii.RejectCloudPII([]byte(`{"note":` + strconv.Quote(note) + `}`)); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	mileage, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("mileage_km")))
	if mileage < 0 {
		mileage = 0
	}
	codes := importreport.ParseCodes(r.FormValue("codes"))
	hostOS := importreport.HostOS(r.FormValue("host_os"), r.UserAgent())
	original := importreport.SanitizeFilename(hdr.Filename)

	ff, err := json.Marshal(map[string]any{
		"imported":      true,
		"source":        source,
		"original_name": original,
		"note":          note,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record import metadata")
		return
	}

	_, _, hit, err := s.store.CachedVIN(r.Context(), v)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "vin cache read failed")
		return
	}
	if !hit {
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		dec, decErr := s.vins.Decode(ctx, v)
		cancel()
		if decErr != nil {
			s.log.Warn("vpic on import", "err", decErr, "vin", v)
		} else if err := s.store.SaveVINDecode(r.Context(), dec); err != nil {
			s.log.Warn("vin cache write on import", "err", err)
		}
	}
	if err := s.store.EnsureVehicle(r.Context(), v, r.FormValue("make_hint"), r.FormValue("model_hint"), 0, "imported_report"); err != nil {
		s.log.Error("ensure vehicle import", "err", err)
		writeError(w, http.StatusInternalServerError, "vehicle persist failed")
		return
	}

	in := ledger.SessionIngest{
		VIN:          v,
		ShopID:       p.ShopID,
		TechnicianID: p.TechnicianID,
		MileageKM:    mileage,
		AdapterType:  "imported_report",
		HostOS:       hostOS,
		Protocol:     "file_import",
		ActiveCodes:  codes,
		FreezeFrame:  ff,
	}
	meta := ledger.JobImport{
		Source:       source,
		OriginalName: original,
		ContentType:  ct,
		Note:         note,
	}
	sess, saved, err := s.store.InsertImportedReport(r.Context(), in, meta, data, s.cfg.ImportDir, ext)
	if err != nil {
		s.log.Error("import", "err", err)
		writeError(w, http.StatusInternalServerError, "import persist failed")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"session": sess,
		"import":  saved,
	})
}

func (s *Server) downloadImportedReport(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "session id required")
		return
	}
	p, ok := principalFrom(r.Context())
	if !ok || p.TechnicianID == "" {
		writeError(w, http.StatusForbidden, "technician login required")
		return
	}
	sess, err := s.store.SessionByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	if !ledger.InShopScope(p.ShopID, sess.ShopID, p.TechnicianID, sess.TechnicianID) {
		writeError(w, http.StatusForbidden, "not this shop's job")
		return
	}
	row, err := s.store.ImportBySession(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "import not found")
		return
	}
	full, err := importreport.ResolveStorage(s.cfg.ImportDir, row.StoragePath)
	if err != nil {
		writeError(w, http.StatusNotFound, "import file missing")
		return
	}
	if _, err := os.Stat(full); err != nil {
		writeError(w, http.StatusNotFound, "import file missing")
		return
	}
	w.Header().Set("Content-Type", row.ContentType)
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Disposition", `attachment; filename="`+importreport.SanitizeFilename(row.OriginalName)+`"`)
	http.ServeFile(w, r, full)
}
