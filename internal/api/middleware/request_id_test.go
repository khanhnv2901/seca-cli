package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestID(t *testing.T) {
	t.Run("generates request ID when not provided", func(t *testing.T) {
		handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := GetRequestID(r.Context())
			if requestID == "" {
				t.Error("expected request ID to be set in context")
			}

			// Verify it's a valid hex string
			if len(requestID) != 16 {
				t.Errorf("expected request ID length 16, got %d", len(requestID))
			}

			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		responseID := rec.Header().Get("X-Request-ID")
		if responseID == "" {
			t.Error("expected X-Request-ID header in response")
		}

		if len(responseID) != 16 {
			t.Errorf("expected X-Request-ID length 16, got %d", len(responseID))
		}
	})

	t.Run("uses client-provided request ID", func(t *testing.T) {
		expectedID := "client-request-123"
		var actualID string

		handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actualID = GetRequestID(r.Context())
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", expectedID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if actualID != expectedID {
			t.Errorf("expected request ID %q, got %q", expectedID, actualID)
		}

		responseID := rec.Header().Get("X-Request-ID")
		if responseID != expectedID {
			t.Errorf("expected X-Request-ID header %q, got %q", expectedID, responseID)
		}
	})

	t.Run("request ID is accessible in context", func(t *testing.T) {
		handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := GetRequestID(r.Context())
			if requestID == "" {
				t.Error("GetRequestID returned empty string")
			}
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
	})

	t.Run("GetRequestID returns empty string when not set", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		requestID := GetRequestID(req.Context())

		if requestID != "" {
			t.Errorf("expected empty string, got %q", requestID)
		}
	})

	t.Run("generates unique IDs for different requests", func(t *testing.T) {
		ids := make(map[string]bool)

		handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := GetRequestID(r.Context())
			ids[requestID] = true
			w.WriteHeader(http.StatusOK)
		}))

		// Make multiple requests
		for i := 0; i < 100; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}

		// All IDs should be unique
		if len(ids) != 100 {
			t.Errorf("expected 100 unique IDs, got %d", len(ids))
		}
	})
}

func TestGenerateRequestID(t *testing.T) {
	t.Run("generates non-empty ID", func(t *testing.T) {
		id := generateRequestID()
		if id == "" {
			t.Error("generateRequestID returned empty string")
		}
	})

	t.Run("generates different IDs", func(t *testing.T) {
		id1 := generateRequestID()
		id2 := generateRequestID()

		if id1 == id2 {
			t.Error("generateRequestID returned same ID twice")
		}
	})

	t.Run("generates 16-character hex string", func(t *testing.T) {
		id := generateRequestID()
		if len(id) != 16 {
			t.Errorf("expected length 16, got %d", len(id))
		}

		// Verify it's a valid hex string
		for _, c := range id {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("expected hex character, got %c", c)
			}
		}
	})
}
