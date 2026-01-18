package record

import (
	"context"

	"github.com/ganot/threds-mcp/internal/domain/activity"
)

// RecordRepository provides persistence for records.
type RecordRepository interface {
	Create(ctx context.Context, tenantID string, rec *Record) error
	Get(ctx context.Context, tenantID, id string) (*Record, error)
	Update(ctx context.Context, tenantID string, rec *Record, expectedTick int64) error
	Delete(ctx context.Context, tenantID, id string) error
	List(ctx context.Context, tenantID string, opts ListRecordsOptions) ([]RecordRef, error)
	GetChildren(ctx context.Context, tenantID, parentID string) ([]Record, error)
	GetChildrenRefs(ctx context.Context, tenantID, parentID string) ([]RecordRef, error)
	GetRelated(ctx context.Context, tenantID, recordID string) ([]string, error)
	AddRelation(ctx context.Context, fromRecordID, toRecordID string) error
}

// SessionRepository provides session activation data for records.
type SessionRepository interface {
	AddActivation(ctx context.Context, sessionID, recordID string, tick int64) error
	GetActivations(ctx context.Context, sessionID string) ([]string, error)
	GetActivationTick(ctx context.Context, sessionID, recordID string) (int64, error)
}

// ProjectRepository provides project tick operations.
type ProjectRepository interface {
	IncrementTick(ctx context.Context, tenantID, projectID string) (int64, error)
}

// ActivityRepository logs record activities.
type ActivityRepository interface {
	Log(ctx context.Context, tenantID string, entry *activity.ActivityEntry) error
}

// SearchRepository performs full-text search.
type SearchRepository interface {
	Search(ctx context.Context, tenantID, projectID, query string, opts SearchOptions) ([]SearchResult, error)
}
