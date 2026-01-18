package activity

import "context"

// Repository provides persistence operations for activity entries.
type Repository interface {
	Log(ctx context.Context, tenantID string, entry *ActivityEntry) error
	List(ctx context.Context, tenantID string, opts ListActivityOptions) ([]ActivityEntry, error)
}
