package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/khanhnv2901/seca-cli/internal/api"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run SECA-CLI as a REST API service",
	RunE: func(cmd *cobra.Command, args []string) error {
		appCtx := getAppContext(cmd)
		addr, _ := cmd.Flags().GetString("addr")
		authToken, _ := cmd.Flags().GetString("auth-token")
		telemetryLimit, _ := cmd.Flags().GetInt("telemetry-limit")

		server := api.NewServer(api.Config{
			Engagements:    &engagementAPIService{appCtx: appCtx},
			Results:        &resultsAPIService{appCtx: appCtx},
			Telemetry:      &telemetryAPIService{appCtx: appCtx},
			Health:         &healthAPIService{appCtx: appCtx},
			AuthToken:      authToken,
			TelemetryLimit: telemetryLimit,
		})

		httpServer := &http.Server{
			Addr:    addr,
			Handler: server,
		}

		fmt.Printf("%s API server listening on %s (results dir: %s)\n", colorInfo("→"), addr, appCtx.ResultsDir)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	},
}

func init() {
	serveCmd.Flags().String("addr", "127.0.0.1:8080", "Address for the API server")
	serveCmd.Flags().String("auth-token", "", "Optional shared secret for API requests")
	serveCmd.Flags().Int("telemetry-limit", 10, "Default telemetry entries to return")
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
	path := filepath.Join(s.appCtx.ResultsDir, id, "results.json")
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
