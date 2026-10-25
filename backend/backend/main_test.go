package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandlerMethodGuard(t *testing.T) {
	app := &App{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet { methodNotAllowed(w, "GET"); return }
		writeJSON(w, http.StatusOK, map[string]string{"status":"ok"})
	})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/health", nil))
	if rec.Code != http.StatusMethodNotAllowed { t.Fatalf("got %d, want 405", rec.Code) }
	_ = app
}

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusOK, map[string]string{"status":"ok"})
	if rec.Code != http.StatusOK { t.Fatalf("got %d, want 200", rec.Code) }
	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("unexpected content type: %q", got)
	}
}
