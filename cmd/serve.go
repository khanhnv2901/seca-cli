package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/khanhnv2901/seca-cli/internal/api"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run SECA-CLI as a REST API service",
	RunE: func(cmd *cobra.Command, args []string) error {
		appCtx := getAppContext(cmd)
		addr, _ := cmd.Flags().GetString("addr")
		authToken, _ := cmd.Flags().GetString("auth-token")
		telemetryLimit, _ := cmd.Flags().GetInt("telemetry-limit")
		shutdownTimeout, _ := cmd.Flags().GetDuration("shutdown-timeout")
		corsOrigins, _ := cmd.Flags().GetStringSlice("cors-origins")
		rateLimit, _ := cmd.Flags().GetInt("rate-limit")
		rateBurst, _ := cmd.Flags().GetInt("rate-burst")

		// Initialize structured logger
		logger, err := zap.NewProduction()
		if err != nil {
			return fmt.Errorf("failed to create logger: %w", err)
		}
		defer func() {
			if err := logger.Sync(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to sync logger: %v\n", err)
			}
		}()

		jobManager := api.NewJobManager()
		runner, err := newCliCheckRunner()
		if err != nil {
			return err
		}

		server := api.NewServer(api.Config{
			Engagements:    &engagementAPIService{appCtx: appCtx},
			Results:        &resultsAPIService{appCtx: appCtx},
			Telemetry:      &telemetryAPIService{appCtx: appCtx},
			Health:         &healthAPIService{appCtx: appCtx},
			Jobs:           &jobAPIService{manager: jobManager, runner: runner},
			AuthToken:      authToken,
			TelemetryLimit: telemetryLimit,
			Logger:         logger,
			CORSOrigins:    corsOrigins,
			RateLimit:      rateLimit,
			RateBurst:      rateBurst,
		})

		httpServer := &http.Server{
			Addr:         addr,
			Handler:      server,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		}

		// Channel to listen for errors from the server
		serverErrors := make(chan error, 1)

		// Start server in a goroutine
		go func() {
			fmt.Printf("%s API server listening on %s (results dir: %s)\n", colorInfo("→"), addr, appCtx.ResultsDir)
			fmt.Printf("%s Press Ctrl+C to gracefully shutdown\n", colorInfo("→"))
			serverErrors <- httpServer.ListenAndServe()
		}()

		// Channel to listen for interrupt signals
		shutdown := make(chan os.Signal, 1)
		signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

		// Block until we receive a signal or an error
		select {
		case err := <-serverErrors:
			if !errors.Is(err, http.ErrServerClosed) {
				return fmt.Errorf("server error: %w", err)
			}
		case sig := <-shutdown:
			fmt.Printf("\n%s Received signal %v, initiating graceful shutdown...\n", colorInfo("→"), sig)

			// Create context with timeout for shutdown
			ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
			defer cancel()

			// Attempt graceful shutdown
			if err := httpServer.Shutdown(ctx); err != nil {
				// Force close if graceful shutdown fails
				if closeErr := httpServer.Close(); closeErr != nil {
					return fmt.Errorf("failed to gracefully shutdown server: %w (close error: %v)", err, closeErr)
				}
				return fmt.Errorf("failed to gracefully shutdown server: %w", err)
			}

			fmt.Printf("%s Server shutdown complete\n", colorInfo("✓"))
		}

		return nil
	},
}

func init() {
	serveCmd.Flags().String("addr", "127.0.0.1:8080", "Address for the API server")
	serveCmd.Flags().String("auth-token", "", "Optional shared secret for API requests")
	serveCmd.Flags().Int("telemetry-limit", 10, "Default telemetry entries to return")
	serveCmd.Flags().Duration("shutdown-timeout", 30*time.Second, "Graceful shutdown timeout")
	serveCmd.Flags().StringSlice("cors-origins", []string{}, "Allowed CORS origins (empty = allow all)")
	serveCmd.Flags().Int("rate-limit", 10, "Rate limit per IP (requests/second, 0 = disabled)")
	serveCmd.Flags().Int("rate-burst", 20, "Rate limit burst size")
	rootCmd.AddCommand(serveCmd)
}

type engagementAPIService struct {
	appCtx *AppContext
}

func (s *engagementAPIService) ListEngagements(ctx context.Context) ([]api.Engagement, error) {
	items := loadEngagements()
	resp := make([]api.Engagement, 0, len(items))
	for _, e := range items {
		resp = append(resp, convertEngagement(e))
	}
	return resp, nil
}

func (s *engagementAPIService) GetEngagement(ctx context.Context, id string) (*api.Engagement, error) {
	eng, err := findEngagementByID(id)
	if err != nil {
		return nil, err
	}
	result := convertEngagement(*eng)
	return &result, nil
}

func (s *engagementAPIService) CreateEngagement(ctx context.Context, req api.EngagementCreateRequest) (*api.Engagement, error) {
	if req.Name == "" || req.Owner == "" {
		return nil, fmt.Errorf("name and owner are required")
	}
	if !req.ROEAgree {
		return nil, fmt.Errorf("roe_agree must be true")
	}
	engagement := Engagement{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Name:      req.Name,
		Owner:     req.Owner,
		ROE:       req.ROE,
		ROEAgree:  req.ROEAgree,
		CreatedAt: time.Now(),
	}
	if len(req.Scope) > 0 {
		normalized, err := normalizeScopeEntries(engagement.ID, req.Scope)
		if err != nil {
			return nil, err
		}
		engagement.Scope = normalized
	}
	list := loadEngagements()
	list = append(list, engagement)
	saveEngagements(list)
	result := convertEngagement(engagement)
	return &result, nil
}

type resultsAPIService struct {
	appCtx *AppContext
}

func (s *resultsAPIService) GetResults(ctx context.Context, id string) ([]byte, error) {
	path, err := resolveResultsPath(s.appCtx.ResultsDir, id, "http_results.json")
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

type telemetryAPIService struct {
	appCtx *AppContext
}

func (s *telemetryAPIService) GetTelemetry(ctx context.Context, id string, limit int) ([]api.TelemetryRecord, error) {
	records, err := loadTelemetryHistory(s.appCtx.ResultsDir, id, limit)
	if err != nil {
		return nil, err
	}
	resp := make([]api.TelemetryRecord, 0, len(records))
	for _, rec := range records {
		resp = append(resp, api.TelemetryRecord{
			Timestamp:           rec.Timestamp,
			Command:             rec.Command,
			EngagementID:        rec.EngagementID,
			TargetCount:         rec.TargetCount,
			SuccessCount:        rec.SuccessCount,
			ErrorCount:          rec.ErrorCount,
			SuccessRate:         rec.SuccessRate,
			DurationSeconds:     rec.DurationSeconds,
			AvgDurationPerCheck: rec.AvgDurationPerCheck,
		})
	}
	return resp, nil
}

type healthAPIService struct {
	appCtx *AppContext
}

func (s *healthAPIService) Check(ctx context.Context) error {
	if s.appCtx.ResultsDir == "" {
		return fmt.Errorf("results directory not configured")
	}
	return nil
}

func convertEngagement(e Engagement) api.Engagement {
	return api.Engagement{
		ID:        e.ID,
		Name:      e.Name,
		Owner:     e.Owner,
		Scope:     append([]string(nil), e.Scope...),
		ROE:       e.ROE,
		ROEAgree:  e.ROEAgree,
		CreatedAt: e.CreatedAt,
	}
}

type jobAPIService struct {
	manager *api.JobManager
	runner  jobRunner
}

type jobRunner interface {
	RunHTTP(ctx context.Context, engagementID string) error
}

func (s *jobAPIService) StartJob(ctx context.Context, req api.JobRequest) (*api.Job, error) {
	jobType := strings.ToLower(strings.TrimSpace(req.Type))
	if jobType == "" {
		jobType = "http"
	}
	if req.EngagementID == "" {
		return nil, fmt.Errorf("engagement_id required")
	}
	if err := validateEngagementID(req.EngagementID); err != nil {
		return nil, fmt.Errorf("invalid engagement_id: %w", err)
	}
	if jobType != "http" {
		return nil, fmt.Errorf("unsupported job type %s", req.Type)
	}
	if _, err := findEngagementByID(req.EngagementID); err != nil {
		return nil, err
	}
	job := s.manager.CreateJob(jobType, req.EngagementID)
	go s.execute(job, req)
	return job, nil
}

func (s *jobAPIService) execute(job *api.Job, req api.JobRequest) {
	now := time.Now()
	s.manager.UpdateJob(job.ID, func(j *api.Job) {
		j.Status = "running"
		j.StartedAt = &now
	})
	// Set reasonable timeout for job execution (90 seconds)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	if err := s.runner.RunHTTP(ctx, req.EngagementID); err != nil {
		errTime := time.Now()
		s.manager.UpdateJob(job.ID, func(j *api.Job) {
			j.Status = "error"
			j.Error = err.Error()
			j.FinishedAt = &errTime
		})
		return
	}
	doneTime := time.Now()
	s.manager.UpdateJob(job.ID, func(j *api.Job) {
		j.Status = "done"
		j.FinishedAt = &doneTime
	})
}

func (s *jobAPIService) GetJob(ctx context.Context, id string) (*api.Job, error) {
	job := s.manager.GetJob(id)
	if job == nil {
		return nil, fmt.Errorf("job not found")
	}
	return job, nil
}

func (s *jobAPIService) ListJobs(ctx context.Context, limit int) ([]api.Job, error) {
	jobs := s.manager.ListJobs(limit)
	return jobs, nil
}

func (s *jobAPIService) Subscribe() (chan api.Job, func()) {
	return s.manager.Subscribe()
}

type cliCheckRunner struct {
	executable string
}

func newCliCheckRunner() (*cliCheckRunner, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return &cliCheckRunner{executable: exe}, nil
}

func (r *cliCheckRunner) RunHTTP(ctx context.Context, engagementID string) error {
	if err := validateEngagementID(engagementID); err != nil {
		return err
	}
	args := []string{"check", "http", "--id", engagementID, "--roe-confirm", "--progress=false"}
	cmd := exec.CommandContext(ctx, r.executable, args...) // #nosec G204 -- executable is trusted binary and args are fixed with validated engagement ID.

	// Create limited buffers (1MB max each) to prevent memory exhaustion
	// If output exceeds this, command will block until buffer space is available
	const maxBufferSize = 1 * 1024 * 1024 // 1MB
	stdoutBuf := &limitedBuffer{max: maxBufferSize}
	stderrBuf := &limitedBuffer{max: maxBufferSize}

	cmd.Stdout = io.MultiWriter(os.Stdout, stdoutBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, stderrBuf)

	return cmd.Run()
}

// limitedBuffer implements io.Writer with a size limit to prevent unbounded memory growth
type limitedBuffer struct {
	buf bytes.Buffer
	max int
}

func (lb *limitedBuffer) Write(p []byte) (n int, err error) {
	// If buffer is at max capacity, discard oldest data
	if lb.buf.Len()+len(p) > lb.max {
		// Keep only the last max-len(p) bytes
		keep := lb.max - len(p)
		if keep > 0 {
			data := lb.buf.Bytes()
			lb.buf.Reset()
			lb.buf.Write(data[len(data)-keep:])
		} else {
			lb.buf.Reset()
		}
	}
	return lb.buf.Write(p)
}
