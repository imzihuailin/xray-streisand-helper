package webui

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerSecurityAndToken(t *testing.T) {
	stopped := make(chan struct{})
	h := (Server{}).Handler("vless://example", "secret", func() { close(stopped) })

	req := httptest.NewRequest(http.MethodGet, "/?token=wrong", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("wrong token status %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/?token=secret", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Security-Policy") == "" || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("missing page or security headers")
	}

	req = httptest.NewRequest(http.MethodPost, "/stop?token=secret", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("stop status %d", rec.Code)
	}
	<-stopped
}
