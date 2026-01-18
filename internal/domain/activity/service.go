package activity

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Service handles activity log operations.
type Service struct {
	repo   Repository
	logger *slog.Logger
}

// NewService creates a new activity service.
func NewService(repo Repository, logger *slog.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

// LogActivity logs an activity entry with the current timestamp if missing.
func (s *Service) LogActivity(ctx context.Context, tenantID string, entry *ActivityEntry) error {
	if entry == nil {
		return ErrInvalidInput
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	if err := s.repo.Log(ctx, tenantID, entry); err != nil {
		return fmt.Errorf("logging activity: %w", err)
	}
	return nil
}

// GetRecentActivity lists activity entries with filtering.
func (s *Service) GetRecentActivity(ctx context.Context, tenantID string, opts ListActivityOptions) ([]ActivityEntry, error) {
	return s.repo.List(ctx, tenantID, opts)
}
