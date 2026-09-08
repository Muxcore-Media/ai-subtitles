package internal

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	mediaevents "github.com/Muxcore-Media/contracts-media/events"
)

func (m *Module) registerHTTP(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/generate", m.handleGenerate)
	mux.HandleFunc("POST /v1/sync", m.handleSync)
	mux.HandleFunc("POST /v1/from-search-failed", m.handleFromFailed)
	mux.HandleFunc("GET /v1/jobs", m.handleJobs)
}

func (m *Module) handleGenerate(w http.ResponseWriter, r *http.Request) {
	var in GenerateInput
	if !readJSON(w, r, &in) {
		return
	}
	got, err := generateSubtitles(in)
	if err != nil {
		http.Error(w, jsonErr(err), http.StatusBadRequest)
		return
	}
	m.rememberGenerate(got)
	writeJSON(w, got)
}

func (m *Module) handleSync(w http.ResponseWriter, r *http.Request) {
	var in SyncInput
	if !readJSON(w, r, &in) {
		return
	}
	got, err := syncSubtitles(in)
	if err != nil {
		http.Error(w, jsonErr(err), http.StatusBadRequest)
		return
	}
	m.rememberSync(got)
	writeJSON(w, got)
}

func (m *Module) handleFromFailed(w http.ResponseWriter, r *http.Request) {
	var body struct {
		mediaevents.SubtitleSearchFailedPayload
		DurationSec    float64 `json:"duration_sec"`
		TranscriptHint string  `json:"transcript_hint"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	m.cfgMu.RLock()
	auto := m.autoGenerate
	m.cfgMu.RUnlock()
	if !auto {
		http.Error(w, `{"error":"auto_generate disabled"}`, http.StatusConflict)
		return
	}
	got, err := generateFromSearchFailed(body.SubtitleSearchFailedPayload, body.DurationSec, body.TranscriptHint)
	if err != nil {
		http.Error(w, jsonErr(err), http.StatusBadRequest)
		return
	}
	m.rememberGenerate(got)
	writeJSON(w, got)
}

func (m *Module) handleJobs(w http.ResponseWriter, _ *http.Request) {
	m.cfgMu.RLock()
	defer m.cfgMu.RUnlock()
	writeJSON(w, map[string]any{"generated": m.jobs, "synced": m.syncs})
}

func (m *Module) handleMesh(ctx context.Context, method string, payload []byte) ([]byte, error) {
	switch method {
	case "Generate":
		var in GenerateInput
		if err := json.Unmarshal(payload, &in); err != nil {
			return nil, err
		}
		got, err := generateSubtitles(in)
		if err != nil {
			return nil, err
		}
		m.rememberGenerate(got)
		return json.Marshal(got)
	case "Sync":
		var in SyncInput
		if err := json.Unmarshal(payload, &in); err != nil {
			return nil, err
		}
		got, err := syncSubtitles(in)
		if err != nil {
			return nil, err
		}
		m.rememberSync(got)
		return json.Marshal(got)
	case "Jobs":
		m.cfgMu.RLock()
		defer m.cfgMu.RUnlock()
		return json.Marshal(map[string]any{"generated": m.jobs, "synced": m.syncs})
	default:
		return nil, errUnknownMethod(method)
	}
}

func readJSON(w http.ResponseWriter, r *http.Request, dest any) bool {
	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		http.Error(w, `{"error":"read body"}`, http.StatusBadRequest)
		return false
	}
	if err := json.Unmarshal(body, dest); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func jsonErr(err error) string {
	b, _ := json.Marshal(map[string]string{"error": err.Error()})
	return string(b)
}

func errUnknownMethod(method string) error {
	return fmtError("unknown method %s", method)
}
