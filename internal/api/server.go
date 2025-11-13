package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type Engagement struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Owner     string    `json:"owner"`
	Scope     []string  `json:"scope,omitempty"`
	ROE       string    `json:"roe,omitempty"`
	ROEAgree  bool      `json:"roe_agree"`
	CreatedAt time.Time `json:"created_at"`
}

type EngagementCreateRequest struct {
	Name     string   `json:"name"`
	Owner    string   `json:"owner"`
	ROE      string   `json:"roe"`
	ROEAgree bool     `json:"roe_agree"`
	Scope    []string `json:"scope"`
}

type TelemetryRecord struct {
	Timestamp           time.Time `json:"timestamp"`
	Command             string    `json:"command"`
	EngagementID        string    `json:"engagement_id"`
	TargetCount         int       `json:"target_count"`
	SuccessCount        int       `json:"success_count"`
	ErrorCount          int       `json:"error_count"`
	SuccessRate         float64   `json:"success_rate"`
	DurationSeconds     float64   `json:"duration_seconds"`
	AvgDurationPerCheck float64   `json:"avg_duration_per_check"`
}

type EngagementService interface {
	ListEngagements(ctx context.Context) ([]Engagement, error)
	GetEngagement(ctx context.Context, id string) (*Engagement, error)
	CreateEngagement(ctx context.Context, req EngagementCreateRequest) (*Engagement, error)
}

type ResultsService interface {
	GetResults(ctx context.Context, id string) ([]byte, error)
}

type TelemetryService interface {
	GetTelemetry(ctx context.Context, id string, limit int) ([]TelemetryRecord, error)
}

type HealthService interface {
	Check(ctx context.Context) error
}

type JobService interface {
	StartJob(ctx context.Context, req JobRequest) (*Job, error)
	GetJob(ctx context.Context, id string) (*Job, error)
	ListJobs(ctx context.Context, limit int) ([]Job, error)
	Subscribe() (chan Job, func())
}

type Config struct {
	Engagements    EngagementService
	Results        ResultsService
	Telemetry      TelemetryService
	Health         HealthService
	Jobs           JobService
	AuthToken      string
	TelemetryLimit int
	Logger         *zap.Logger
	CORSOrigins    []string // Allowed CORS origins (empty = allow all)
	RateLimit      int      // Requests per second per IP (0 = disabled)
	RateBurst      int      // Burst size for rate limiter
}

type Server struct {
	cfg      Config
	mux      *http.ServeMux
	limiters *rateLimiterMap
}

func NewServer(cfg Config) *Server {
	srv := &Server{
		cfg:      cfg,
		mux:      http.NewServeMux(),
		limiters: newRateLimiterMap(),
	}
	srv.routes()
	return srv
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Apply middleware chain: CORS -> RateLimit -> Logging -> Auth -> Handler
	handler := s.withLogging(s.withRateLimit(s.withCORS(s.mux)))
	handler.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.Handle("/api/health", s.withAuth(http.HandlerFunc(s.handleHealth)))
	s.mux.Handle("/api/engagements", s.withAuth(http.HandlerFunc(s.handleEngagements)))
	s.mux.Handle("/api/engagements/", s.withAuth(http.HandlerFunc(s.handleEngagementByID)))
	s.mux.Handle("/api/results/", s.withAuth(http.HandlerFunc(s.handleResults)))
	s.mux.Handle("/api/telemetry/", s.withAuth(http.HandlerFunc(s.handleTelemetry)))
	s.mux.Handle("/api/jobs", s.withAuth(http.HandlerFunc(s.handleJobs)))
	s.mux.Handle("/api/jobs/", s.withAuth(http.HandlerFunc(s.handleJobByID)))
	s.mux.Handle("/api/jobs-stream", s.withAuth(http.HandlerFunc(s.handleJobStream)))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}
	if s.cfg.Health != nil {
		if err := s.cfg.Health.Check(r.Context()); err != nil {
			s.writeError(w, http.StatusInternalServerError, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleEngagements(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := s.cfg.Engagements.ListEngagements(r.Context())
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		r.Body = http.MaxBytesReader(w, r.Body, 1048576) // 1MB limit
		var req EngagementCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, err)
			return
		}
		created, err := s.cfg.Engagements.CreateEngagement(r.Context(), req)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, created)
	default:
		s.methodNotAllowed(w)
	}
}

func (s *Server) handleEngagementByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/engagements/")
	if id == "" {
		s.writeError(w, http.StatusNotFound, errors.New("engagement ID required"))
		return
	}
	eng, err := s.cfg.Engagements.GetEngagement(r.Context(), id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, eng)
}

func (s *Server) handleResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/results/")
	if id == "" {
		s.writeError(w, http.StatusNotFound, errors.New("engagement ID required"))
		return
	}
	data, err := s.cfg.Results.GetResults(r.Context(), id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, err)
		return
	}
	// Write raw JSON data (already formatted from file)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil && s.cfg.Logger != nil {
		s.cfg.Logger.Error("failed to write response", zap.Error(err))
	}
}

func (s *Server) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/telemetry/")
	if id == "" {
		s.writeError(w, http.StatusNotFound, errors.New("engagement ID required"))
		return
	}
	limit := s.cfg.TelemetryLimit
	if limit <= 0 {
		limit = 10
	}
	if q := r.URL.Query().Get("limit"); q != "" {
		if parsed, err := strconv.Atoi(q); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	records, err := s.cfg.Telemetry.GetTelemetry(r.Context(), id, limit)
	if err != nil {
		s.writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, records)
}

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Jobs == nil {
		s.writeError(w, http.StatusNotFound, errors.New("job service not available"))
		return
	}
	switch r.Method {
	case http.MethodGet:
		limit := 25
		if q := r.URL.Query().Get("limit"); q != "" {
			if parsed, err := strconv.Atoi(q); err == nil && parsed > 0 {
				limit = parsed
			}
		}
		jobs, err := s.cfg.Jobs.ListJobs(r.Context(), limit)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, jobs)
	case http.MethodPost:
		r.Body = http.MaxBytesReader(w, r.Body, 1048576) // 1MB limit
		var req JobRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, err)
			return
		}
		job, err := s.cfg.Jobs.StartJob(r.Context(), req)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusAccepted, job)
	default:
		s.methodNotAllowed(w)
	}
}

func (s *Server) handleJobByID(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Jobs == nil {
		s.writeError(w, http.StatusNotFound, errors.New("job service not available"))
		return
	}
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/jobs/")
	if id == "" {
		s.writeError(w, http.StatusNotFound, errors.New("job ID required"))
		return
	}
	job, err := s.cfg.Jobs.GetJob(r.Context(), id)
	if err != nil || job == nil {
		s.writeError(w, http.StatusNotFound, errors.New("job not found"))
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleJobStream(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Jobs == nil {
		s.writeError(w, http.StatusNotFound, errors.New("job service not available"))
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeError(w, http.StatusInternalServerError, errors.New("streaming unsupported"))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	updates, unsubscribe := s.cfg.Jobs.Subscribe()
	defer unsubscribe()
	ctx := r.Context()
	for {
		select {
		case job, ok := <-updates:
			if !ok {
				return
			}
			payload, _ := json.Marshal(job)
			w.Write([]byte("event: job\n"))
			w.Write([]byte("data: "))
			w.Write(payload)
			w.Write([]byte("\n\n"))
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}

func (s *Server) withRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting if disabled
		if s.cfg.RateLimit <= 0 {
			next.ServeHTTP(w, r)
			return
		}

		// Extract client IP (handle X-Forwarded-For for proxied requests)
		clientIP := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			// Use first IP in X-Forwarded-For chain
			if idx := strings.Index(forwarded, ","); idx > 0 {
				clientIP = strings.TrimSpace(forwarded[:idx])
			} else {
				clientIP = strings.TrimSpace(forwarded)
			}
		}
		// Remove port if present
		if idx := strings.LastIndex(clientIP, ":"); idx > 0 {
			clientIP = clientIP[:idx]
		}

		// Get or create limiter for this IP
		limiter := s.limiters.getLimiter(clientIP, s.cfg.RateLimit, s.cfg.RateBurst)

		// Check if request is allowed
		if !limiter.Allow() {
			if s.cfg.Logger != nil {
				s.cfg.Logger.Warn("rate_limit_exceeded",
					zap.String("client_ip", clientIP),
					zap.String("path", r.URL.Path),
				)
			}
			s.writeError(w, http.StatusTooManyRequests, errors.New("rate limit exceeded"))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Determine if origin is allowed
		allowOrigin := "*"
		if len(s.cfg.CORSOrigins) > 0 {
			// Check if origin is in whitelist
			allowed := false
			for _, allowedOrigin := range s.cfg.CORSOrigins {
				if allowedOrigin == origin {
					allowed = true
					allowOrigin = origin
					break
				}
			}
			if !allowed {
				allowOrigin = ""
			}
		}

		// Set CORS headers
		if allowOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Auth-Token")
			w.Header().Set("Access-Control-Max-Age", "3600")
		}

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process request
		next.ServeHTTP(lrw, r)

		// Log request details
		duration := time.Since(start)
		if s.cfg.Logger != nil {
			s.cfg.Logger.Info("http_request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote_addr", r.RemoteAddr),
				zap.Int("status", lrw.statusCode),
				zap.Duration("duration", duration),
				zap.Int64("bytes", lrw.bytesWritten),
			)
		}
	})
}

func (s *Server) withAuth(next http.Handler) http.Handler {
	if s.cfg.AuthToken == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Auth-Token")
		// Use constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.AuthToken)) != 1 {
			s.writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// loggingResponseWriter wraps http.ResponseWriter to capture status code and bytes written
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	n, err := lrw.ResponseWriter.Write(b)
	lrw.bytesWritten += int64(n)
	return n, err
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) writeError(w http.ResponseWriter, status int, err error) {
	// Sanitize error messages to prevent information disclosure
	msg := err.Error()

	// For 5xx errors, return generic message and log details server-side
	if status >= 500 {
		if s.cfg.Logger != nil {
			s.cfg.Logger.Error("internal_server_error",
				zap.Error(err),
				zap.Int("status", status),
			)
		}
		msg = "internal server error"
	}

	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) methodNotAllowed(w http.ResponseWriter) {
	s.writeError(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
}

// rateLimiterMap manages per-IP rate limiters with automatic cleanup
type rateLimiterMap struct {
	mu       sync.RWMutex
	limiters map[string]*ipLimiter
}

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func newRateLimiterMap() *rateLimiterMap {
	m := &rateLimiterMap{
		limiters: make(map[string]*ipLimiter),
	}
	// Start cleanup goroutine to remove stale limiters
	go m.cleanupLoop()
	return m
}

func (m *rateLimiterMap) getLimiter(ip string, rps, burst int) *rate.Limiter {
	m.mu.Lock()
	defer m.mu.Unlock()

	limiter, exists := m.limiters[ip]
	if !exists {
		limiter = &ipLimiter{
			limiter:  rate.NewLimiter(rate.Limit(rps), burst),
			lastSeen: time.Now(),
		}
		m.limiters[ip] = limiter
	} else {
		limiter.lastSeen = time.Now()
	}

	return limiter.limiter
}

// cleanupLoop removes limiters that haven't been used in 5 minutes
func (m *rateLimiterMap) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		for ip, limiter := range m.limiters {
			if time.Since(limiter.lastSeen) > 5*time.Minute {
				delete(m.limiters, ip)
			}
		}
		m.mu.Unlock()
	}
}
