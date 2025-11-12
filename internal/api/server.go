package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
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

type Config struct {
	Engagements    EngagementService
	Results        ResultsService
	Telemetry      TelemetryService
	Health         HealthService
	AuthToken      string
	TelemetryLimit int
}

type Server struct {
	cfg Config
	mux *http.ServeMux
}

func NewServer(cfg Config) *Server {
	srv := &Server{cfg: cfg, mux: http.NewServeMux()}
	srv.routes()
	return srv
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.Handle("/api/health", s.withAuth(http.HandlerFunc(s.handleHealth)))
	s.mux.Handle("/api/engagements", s.withAuth(http.HandlerFunc(s.handleEngagements)))
	s.mux.Handle("/api/engagements/", s.withAuth(http.HandlerFunc(s.handleEngagementByID)))
	s.mux.Handle("/api/results/", s.withAuth(http.HandlerFunc(s.handleResults)))
	s.mux.Handle("/api/telemetry/", s.withAuth(http.HandlerFunc(s.handleTelemetry)))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if s.cfg.Health != nil {
		if err := s.cfg.Health.Check(r.Context()); err != nil {
			writeError(w, http.StatusInternalServerError, err)
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
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var req EngagementCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		created, err := s.cfg.Engagements.CreateEngagement(r.Context(), req)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, created)
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleEngagementByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/engagements/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	eng, err := s.cfg.Engagements.GetEngagement(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, eng)
}

func (s *Server) handleResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/results/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	data, err := s.cfg.Results.GetResults(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (s *Server) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/telemetry/")
	if id == "" {
		http.NotFound(w, r)
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
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, records)
}

func (s *Server) withAuth(next http.Handler) http.Handler {
	if s.cfg.AuthToken == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Auth-Token")
		if token != s.cfg.AuthToken {
			writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func methodNotAllowed(w http.ResponseWriter) {
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}
