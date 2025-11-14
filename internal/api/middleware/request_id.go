package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	// RequestIDKey is the context key for request ID
	RequestIDKey ContextKey = "request_id"
)

// RequestID is a middleware that adds a unique request ID to each request.
// It first checks if the client provided an X-Request-ID header, otherwise generates a new one.
// The request ID is added to the response headers and stored in the request context.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		// Add request ID to response headers
		w.Header().Set("X-Request-ID", requestID)

		// Add request ID to context
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)

		// Call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// generateRequestID generates a random 16-character hexadecimal request ID
func generateRequestID() string {
	b := make([]byte, 8) // 8 bytes = 16 hex characters
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if random generation fails
		return "fallback-id"
	}
	return hex.EncodeToString(b)
}

// GetRequestID retrieves the request ID from the context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}
