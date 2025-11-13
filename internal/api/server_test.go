package api

import (
	"context"
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

// Mock services for testing
type mockHealthService struct {
	err error
}

func (m *mockHealthService) Check(ctx context.Context) error {
	return m.err
}

func TestNewServer(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cfg := Config{
		Logger: logger,
		Health: &mockHealthService{},
	}

	srv := NewServer(cfg)
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
	if srv.mux == nil {
		t.Error("expected mux to be initialized")
	}
	if srv.limiters == nil {
		t.Error("expected limiters to be initialized")
	}
}

func TestServer_HandleHealth(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name       string
		healthErr  error
		wantStatus int
	}{
		{
			name:       "healthy",
			healthErr:  nil,
			wantStatus: http.StatusOK,
		},
		{
			name:       "unhealthy",
			healthErr:  errors.New("database down"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Logger: logger,
				Health: &mockHealthService{err: tt.healthErr},
			}
			srv := NewServer(cfg)

			req := httptest.NewRequest("GET", "/api/health", nil)
			rr := httptest.NewRecorder()

			srv.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rr.Code)
			}
		})
	}
}

func TestServer_WithCORS(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name          string
		corsOrigins   []string
		requestOrigin string
		wantAllow     bool
	}{
		{
			name:          "allow all origins",
			corsOrigins:   []string{},
			requestOrigin: "https://example.com",
			wantAllow:     true,
		},
		{
			name:          "allow specific origin",
			corsOrigins:   []string{"https://example.com"},
			requestOrigin: "https://example.com",
			wantAllow:     true,
		},
		{
			name:          "deny unlisted origin",
			corsOrigins:   []string{"https://example.com"},
			requestOrigin: "https://evil.com",
			wantAllow:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Logger:      logger,
				CORSOrigins: tt.corsOrigins,
				Health:      &mockHealthService{},
			}
			srv := NewServer(cfg)

			req := httptest.NewRequest("OPTIONS", "/api/health", nil)
			req.Header.Set("Origin", tt.requestOrigin)
			rr := httptest.NewRecorder()

			srv.ServeHTTP(rr, req)

			allowOrigin := rr.Header().Get("Access-Control-Allow-Origin")
			if tt.wantAllow {
				if allowOrigin == "" {
					t.Error("expected CORS headers to be set")
				}
			} else {
				if allowOrigin == tt.requestOrigin {
					t.Error("expected origin to be denied")
				}
			}
		})
	}
}

func TestServer_WithAuth(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name         string
		authToken    string
		requestToken string
		wantStatus   int
	}{
		{
			name:         "valid token",
			authToken:    "secret123",
			requestToken: "secret123",
			wantStatus:   http.StatusOK,
		},
		{
			name:         "invalid token",
			authToken:    "secret123",
			requestToken: "wrong",
			wantStatus:   http.StatusUnauthorized,
		},
		{
			name:         "missing token",
			authToken:    "secret123",
			requestToken: "",
			wantStatus:   http.StatusUnauthorized,
		},
		{
			name:         "no auth required",
			authToken:    "",
			requestToken: "",
			wantStatus:   http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Logger:    logger,
				AuthToken: tt.authToken,
				Health:    &mockHealthService{},
			}
			srv := NewServer(cfg)

			req := httptest.NewRequest("GET", "/api/health", nil)
			if tt.requestToken != "" {
				req.Header.Set("X-Auth-Token", tt.requestToken)
			}
			rr := httptest.NewRecorder()

			srv.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rr.Code)
			}
		})
	}
}

func TestNewRateLimiterMap(t *testing.T) {
	rlm := newRateLimiterMap()
	if rlm == nil {
		t.Fatal("expected non-nil rate limiter map")
	}
	if rlm.limiters == nil {
		t.Error("expected limiters map to be initialized")
	}
}

func TestRateLimiterMap_GetLimiter(t *testing.T) {
	rlm := newRateLimiterMap()

	limiter1 := rlm.getLimiter("192.168.1.1", 10, 20)
	if limiter1 == nil {
		t.Fatal("expected non-nil limiter")
	}

	limiter2 := rlm.getLimiter("192.168.1.1", 10, 20)
	if limiter2 != limiter1 {
		t.Error("expected same limiter for same IP")
	}

	limiter3 := rlm.getLimiter("192.168.1.2", 10, 20)
	if limiter3 == limiter1 {
		t.Error("expected different limiter for different IP")
	}
}
