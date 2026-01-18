package session

import (
	"context"

	"github.com/ganot/threds-mcp/internal/domain/project"
	"github.com/ganot/threds-mcp/internal/domain/record"
)

// RecordRepository provides record retrieval for context loading.
type RecordRepository interface {
	Get(ctx context.Context, tenantID, id string) (*record.Record, error)
	GetChildren(ctx context.Context, tenantID, parentID string) ([]record.Record, error)
	GetChildrenRefs(ctx context.Context, tenantID, parentID string) ([]record.RecordRef, error)
}

// SessionRepository provides persistence for sessions.
type SessionRepository interface {
	Create(ctx context.Context, tenantID string, sess *Session) error
	Get(ctx context.Context, tenantID, id string) (*Session, error)
	Update(ctx context.Context, tenantID string, sess *Session) error
	Close(ctx context.Context, tenantID, id string) error
	ListActive(ctx context.Context, tenantID, projectID string) ([]SessionInfo, error)
	GetByRecordID(ctx context.Context, tenantID, recordID string) ([]SessionInfo, error)
	AddActivation(ctx context.Context, sessionID, recordID string, tick int64) error
}

// ProjectRepository provides project access for tick data.
type ProjectRepository interface {
	Get(ctx context.Context, tenantID, id string) (*project.Project, error)
}
