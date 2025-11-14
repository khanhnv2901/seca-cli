package application

import (
	"fmt"

	auditapp "github.com/khanhnv2901/seca-cli/internal/application/audit"
	checkapp "github.com/khanhnv2901/seca-cli/internal/application/check"
	engagementapp "github.com/khanhnv2901/seca-cli/internal/application/engagement"
	"github.com/khanhnv2901/seca-cli/internal/domain/audit"
	"github.com/khanhnv2901/seca-cli/internal/domain/check"
	"github.com/khanhnv2901/seca-cli/internal/domain/engagement"
	"github.com/khanhnv2901/seca-cli/internal/infrastructure/persistence/json"
)

// Container holds all application services and repositories
// This is a simple dependency injection container
type Container struct {
	// Repositories
	EngagementRepo engagement.Repository
	CheckRunRepo   check.Repository
	AuditRepo      audit.Repository

	// Services
	EngagementService *engagementapp.Service
	CheckOrchestrator *checkapp.Orchestrator
	AuditService      *auditapp.Service
}

// NewContainer creates a new application service container
func NewContainer(dataDir, resultsDir string) (*Container, error) {
	// Initialize repositories
	engagementRepo, err := json.NewEngagementRepository(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create engagement repository: %w", err)
	}

	checkRunRepo, err := json.NewCheckRunRepository(resultsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create check run repository: %w", err)
	}

	auditRepo, err := json.NewAuditRepository(resultsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create audit repository: %w", err)
	}

	// Initialize services
	engagementService := engagementapp.NewService(engagementRepo)
	checkOrchestrator := checkapp.NewOrchestrator(engagementRepo, checkRunRepo, auditRepo)
	auditService := auditapp.NewService(auditRepo)

	return &Container{
		EngagementRepo:    engagementRepo,
		CheckRunRepo:      checkRunRepo,
		AuditRepo:         auditRepo,
		EngagementService: engagementService,
		CheckOrchestrator: checkOrchestrator,
		AuditService:      auditService,
	}, nil
}
