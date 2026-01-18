package project

import "context"

// Repository provides persistence for projects.
type Repository interface {
	Create(ctx context.Context, tenantID string, proj *Project) error
	Get(ctx context.Context, tenantID, id string) (*Project, error)
	GetDefault(ctx context.Context, tenantID string) (*Project, error)
	List(ctx context.Context, tenantID string) ([]ProjectSummary, error)
	IncrementTick(ctx context.Context, tenantID, projectID string) (int64, error)
}
