package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap/zaptest"
)

func TestWriteJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusCreated, map[string]string{"status": "ok"})

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected application/json content-type, got %s", got)
	}
	if !strings.Contains(rr.Body.String(), `"status":"ok"`) {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}

func TestWriteErrorInternal(t *testing.T) {
	logger := zaptest.NewLogger(t)
	s := &Server{cfg: Config{Logger: logger}}

	rr := httptest.NewRecorder()
	s.writeError(rr, http.StatusInternalServerError, errors.New("boom"))

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "internal server error") {
		t.Fatalf("expected sanitized message, got %s", rr.Body.String())
	}
}

func TestWriteErrorClient(t *testing.T) {
	s := &Server{}
	rr := httptest.NewRecorder()
	s.writeError(rr, http.StatusBadRequest, errors.New("bad input"))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "bad input") {
		t.Fatalf("expected original error message, got %s", rr.Body.String())
	}
}

func TestMethodNotAllowed(t *testing.T) {
	s := &Server{}
	rr := httptest.NewRecorder()
	s.methodNotAllowed(rr)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", rr.Code)
	}
}

func TestWriteStreamChunk(t *testing.T) {
	s := &Server{}
	rr := httptest.NewRecorder()
	if !s.writeStreamChunk(rr, []byte("hello")) {
		t.Fatal("expected writeStreamChunk to succeed")
	}
	if rr.Body.String() != "hello" {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}

	if s.writeStreamChunk(&failingWriter{}, []byte("fail")) {
		t.Fatalf("expected writeStreamChunk to fail")
	}
}

type failingWriter struct{}

func (f *failingWriter) Header() http.Header { return http.Header{} }
func (f *failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}
func (f *failingWriter) WriteHeader(statusCode int) {}
