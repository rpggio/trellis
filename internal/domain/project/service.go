package project

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ganot/threds-mcp/internal/repository"
	"github.com/google/uuid"
)

// Service handles project operations.
type Service struct {
	repo   Repository
	logger *slog.Logger
}

// NewService creates a new project service.
func NewService(repo Repository, logger *slog.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

// CreateRequest defines project creation inputs.
type CreateRequest struct {
	ID          string
	Name        string
	Description string
}

// Create creates a new project.
func (s *Service) Create(ctx context.Context, tenantID string, req CreateRequest) (*Project, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, ErrInvalidInput
	}

	id := req.ID
	if strings.TrimSpace(id) == "" {
		id = uuid.NewString()
	}

	proj := &Project{
		ID:          id,
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Tick:        0,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.Create(ctx, tenantID, proj); err != nil {
		return nil, fmt.Errorf("creating project: %w", err)
	}

	return proj, nil
}

// Get fetches a project by ID.
func (s *Service) Get(ctx context.Context, tenantID, id string) (*Project, error) {
	proj, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("getting project: %w", err)
	}
	return proj, nil
}

// GetDefault returns the default project, creating one if missing.
func (s *Service) GetDefault(ctx context.Context, tenantID string) (*Project, error) {
	proj, err := s.repo.GetDefault(ctx, tenantID)
	if err == nil {
		return proj, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("getting default project: %w", err)
	}

	return s.Create(ctx, tenantID, CreateRequest{
		Name:        "Default Project",
		Description: "",
	})
}

// List returns project summaries.
func (s *Service) List(ctx context.Context, tenantID string) ([]ProjectSummary, error) {
	return s.repo.List(ctx, tenantID)
}
