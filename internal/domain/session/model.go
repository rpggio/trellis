package session

import (
	"time"

	"github.com/ganot/threds-mcp/internal/domain/record"
)

// SessionStatus represents the lifecycle status of a session
type SessionStatus string

const (
	StatusActive SessionStatus = "active"
	StatusStale  SessionStatus = "stale"
	StatusClosed SessionStatus = "closed"
)

// Session represents a chat session tracking activated records
type Session struct {
	ID            string        `json:"id"`
	TenantID      string        `json:"tenant_id"`
	ProjectID     string        `json:"project_id"`
	Status        SessionStatus `json:"status"`
	FocusRecord   *string       `json:"focus_record,omitempty"`
	ParentSession *string       `json:"parent_session,omitempty"`
	LastSyncTick  int64         `json:"last_sync_tick"`
	CreatedAt     time.Time     `json:"created_at"`
	LastActivity  time.Time     `json:"last_activity"`
	ClosedAt      *time.Time    `json:"closed_at,omitempty"`
	ActiveRecords []string      `json:"active_records"`
}

// ContextBundle contains everything needed to reason with a record
type ContextBundle struct {
	Target        record.Record      `json:"target"`
	Parent        *record.Record     `json:"parent,omitempty"`
	OpenChildren  []record.Record    `json:"open_children"`
	OtherChildren []record.RecordRef `json:"other_children"`
	Grandchildren []record.RecordRef `json:"grandchildren"`
	Warnings      []string           `json:"warnings,omitempty"`
}

// ConflictInfo describes a conflict detected during activation or update
type ConflictInfo struct {
	ConflictType   string         `json:"conflict_type"` // "activation" or "update"
	OtherSessionID string         `json:"other_session_id,omitempty"`
	LocalVersion   *record.Record `json:"local_version,omitempty"`
	RemoteVersion  *record.Record `json:"remote_version,omitempty"`
	Message        string         `json:"message"`
}

// SessionInfo provides information about an active session
type SessionInfo struct {
	SessionID    string    `json:"session_id"`
	FocusRecord  *string   `json:"focus_record,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	LastActivity time.Time `json:"last_activity"`
}
