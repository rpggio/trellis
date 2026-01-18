package repository

import (
	"context"

	"github.com/ganot/threds-mcp/internal/domain/activity"
	"github.com/ganot/threds-mcp/internal/domain/project"
	"github.com/ganot/threds-mcp/internal/domain/record"
	"github.com/ganot/threds-mcp/internal/domain/session"
)

// ProjectRepository manages project persistence
type ProjectRepository interface {
	Create(ctx context.Context, tenantID string, proj *project.Project) error
	Get(ctx context.Context, tenantID, id string) (*project.Project, error)
	GetDefault(ctx context.Context, tenantID string) (*project.Project, error)
	List(ctx context.Context, tenantID string) ([]project.ProjectSummary, error)
	IncrementTick(ctx context.Context, tenantID, projectID string) (int64, error)
}

// RecordRepository manages record persistence
type RecordRepository interface {
	Create(ctx context.Context, tenantID string, rec *record.Record) error
	Get(ctx context.Context, tenantID, id string) (*record.Record, error)
	Update(ctx context.Context, tenantID string, rec *record.Record, expectedTick int64) error
	Delete(ctx context.Context, tenantID, id string) error
	List(ctx context.Context, tenantID string, opts ListRecordsOptions) ([]record.RecordRef, error)
	GetChildren(ctx context.Context, tenantID, parentID string) ([]record.Record, error)
	GetChildrenRefs(ctx context.Context, tenantID, parentID string) ([]record.RecordRef, error)
	GetRelated(ctx context.Context, tenantID, recordID string) ([]string, error)
	AddRelation(ctx context.Context, fromRecordID, toRecordID string) error
}

// ListRecordsOptions provides filtering options for listing records
type ListRecordsOptions struct {
	ProjectID string
	ParentID  *string
	States    []record.RecordState
	Types     []string
	Limit     int
	Offset    int
}

// SessionRepository manages session persistence
type SessionRepository interface {
	Create(ctx context.Context, tenantID string, sess *session.Session) error
	Get(ctx context.Context, tenantID, id string) (*session.Session, error)
	Update(ctx context.Context, tenantID string, sess *session.Session) error
	Close(ctx context.Context, tenantID, id string) error
	ListActive(ctx context.Context, tenantID, projectID string) ([]session.SessionInfo, error)
	GetByRecordID(ctx context.Context, tenantID, recordID string) ([]session.SessionInfo, error)
	AddActivation(ctx context.Context, sessionID, recordID string, tick int64) error
	GetActivations(ctx context.Context, sessionID string) ([]string, error)
	GetActivationTick(ctx context.Context, sessionID, recordID string) (int64, error)
}

// ActivityRepository manages activity log persistence
type ActivityRepository interface {
	Log(ctx context.Context, tenantID string, entry *activity.ActivityEntry) error
	List(ctx context.Context, tenantID string, opts ListActivityOptions) ([]activity.ActivityEntry, error)
}

// ListActivityOptions provides filtering options for listing activity
type ListActivityOptions struct {
	ProjectID    string
	RecordID     *string
	SessionID    *string
	ActivityType *activity.ActivityType
	Limit        int
	Offset       int
}

// SearchRepository manages full-text search
type SearchRepository interface {
	Search(ctx context.Context, tenantID, projectID, query string, opts SearchOptions) ([]record.SearchResult, error)
}

// SearchOptions provides filtering options for search
type SearchOptions struct {
	States []record.RecordState
	Types  []string
	Limit  int
	Offset int
}
