package activity

import "time"

// ActivityType represents the type of activity event
type ActivityType string

const (
	TypeRecordCreated      ActivityType = "record_created"
	TypeRecordUpdated      ActivityType = "record_updated"
	TypeStateTransition    ActivityType = "state_transition"
	TypeSessionStarted     ActivityType = "session_started"
	TypeSessionSaved       ActivityType = "session_saved"
	TypeSessionClosed      ActivityType = "session_closed"
	TypeSessionBranched    ActivityType = "session_branched"
	TypeActivation         ActivityType = "activation"
	TypeConflictDetected   ActivityType = "conflict_detected"
	TypeConflictResolved   ActivityType = "conflict_resolved"
)

// ActivityEntry represents an event in the activity log
type ActivityEntry struct {
	ID           int64        `json:"id"`
	TenantID     string       `json:"tenant_id"`
	ProjectID    string       `json:"project_id"`
	SessionID    *string      `json:"session_id,omitempty"`
	RecordID     *string      `json:"record_id,omitempty"`
	ActivityType ActivityType `json:"type"`
	Summary      string       `json:"summary"`
	Details      string       `json:"details,omitempty"`  // JSON string
	CreatedAt    time.Time    `json:"created_at"`
	Tick         int64        `json:"tick"`
}
