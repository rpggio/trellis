package mcp

import (
	"time"

	"github.com/ganot/threds-mcp/internal/domain/activity"
	"github.com/ganot/threds-mcp/internal/domain/project"
	"github.com/ganot/threds-mcp/internal/domain/record"
	"github.com/ganot/threds-mcp/internal/domain/session"
)

type CreateProjectParams struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type GetProjectParams struct {
	ID string `json:"id,omitempty"`
}

type GetProjectOverviewParams struct {
	ProjectID string `json:"project_id,omitempty"`
}

type SearchRecordsParams struct {
	ProjectID string               `json:"project_id,omitempty"`
	Query     string               `json:"query"`
	States    []record.RecordState `json:"states,omitempty"`
	Types     []string             `json:"types,omitempty"`
	Limit     int                  `json:"limit,omitempty"`
	Offset    int                  `json:"offset,omitempty"`
}

type ListRecordsParams struct {
	ProjectID string               `json:"project_id,omitempty"`
	ParentID  *string              `json:"parent_id,omitempty"`
	States    []record.RecordState `json:"states,omitempty"`
	Types     []string             `json:"types,omitempty"`
	Limit     int                  `json:"limit,omitempty"`
	Offset    int                  `json:"offset,omitempty"`
}

type GetRecordRefParams struct {
	ID string `json:"id"`
}

type ActivateParams struct {
	ID string `json:"id"`
}

type SyncSessionParams struct {
	SessionID string `json:"session_id,omitempty"`
}

type CreateRecordParams struct {
	ParentID *string            `json:"parent_id,omitempty"`
	Type     string             `json:"type"`
	Title    string             `json:"title"`
	Summary  string             `json:"summary"`
	Body     string             `json:"body"`
	State    record.RecordState `json:"state,omitempty"`
	Related  []string           `json:"related,omitempty"`
}

type UpdateRecordParams struct {
	ID        string   `json:"id"`
	SessionID string   `json:"session_id,omitempty"`
	Title     *string  `json:"title,omitempty"`
	Summary   *string  `json:"summary,omitempty"`
	Body      *string  `json:"body,omitempty"`
	Related   []string `json:"related,omitempty"`
	Force     bool     `json:"force,omitempty"`
}

type TransitionParams struct {
	ID         string             `json:"id"`
	ToState    record.RecordState `json:"to_state"`
	Reason     *string            `json:"reason,omitempty"`
	ResolvedBy *string            `json:"resolved_by,omitempty"`
}

type SaveSessionParams struct {
	SessionID string `json:"session_id,omitempty"`
}

type CloseSessionParams struct {
	SessionID string `json:"session_id,omitempty"`
}

type BranchSessionParams struct {
	SessionID   string `json:"session_id"`
	FocusRecord string `json:"focus_record,omitempty"`
}

type GetRecordHistoryParams struct {
	ID    string `json:"id"`
	Since string `json:"since,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

type GetRecordDiffParams struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to,omitempty"`
}

type GetActiveSessionsParams struct {
	RecordID string `json:"record_id"`
}

type GetRecentActivityParams struct {
	ProjectID string                  `json:"project_id,omitempty"`
	Limit     int                     `json:"limit,omitempty"`
	Since     string                  `json:"since,omitempty"`
	Types     []activity.ActivityType `json:"types,omitempty"`
	RecordID  *string                 `json:"record_id,omitempty"`
}

type ProjectSummaryResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	Tick         int64  `json:"tick"`
	OpenSessions int    `json:"open_sessions"`
	OpenRecords  int    `json:"open_records"`
}

type ListProjectsResponse struct {
	Projects []ProjectSummaryResponse `json:"projects"`
}

type SearchRecordsResponse struct {
	Results []record.SearchResult `json:"results"`
}

type ListRecordsResponse struct {
	Records []record.RecordRef `json:"records"`
}

type GetRecordHistoryResponse struct {
	History []RecordHistoryEntry `json:"history"`
}

type GetActiveSessionsResponse struct {
	Sessions []ActiveSessionStatus `json:"sessions"`
}

type GetRecentActivityResponse struct {
	Activity []ActivityEntryResponse `json:"activity"`
}

type ProjectOverviewResponse struct {
	Project      project.Project        `json:"project"`
	OpenSessions []ProjectSessionStatus `json:"open_sessions"`
	RootRecords  []record.RecordRef     `json:"root_records"`
}

type ProjectSessionStatus struct {
	ID           string    `json:"id"`
	FocusRecord  string    `json:"focus_record,omitempty"`
	LastActivity time.Time `json:"last_activity"`
	LastSyncTick int64     `json:"last_sync_tick"`
	TickGap      int64     `json:"tick_gap"`
	Warning      string    `json:"warning,omitempty"`
}

type CreateRecordResponse struct {
	Record        record.Record `json:"record"`
	AutoActivated bool          `json:"auto_activated"`
}

type UpdateRecordResponse struct {
	Record   *record.Record        `json:"record,omitempty"`
	Conflict *RecordConflictResult `json:"conflict,omitempty"`
}

type RecordConflictResult struct {
	Message      string        `json:"message"`
	OtherVersion record.Record `json:"other_version"`
}

type ActivateResponse struct {
	SessionID string                `json:"session_id"`
	Context   session.ContextBundle `json:"context"`
	Warnings  []string              `json:"warnings,omitempty"`
}

type SyncSessionResponse struct {
	SessionID     string                `json:"session_id"`
	Staleness     int64                 `json:"staleness"`
	SessionStatus session.SessionStatus `json:"session_status"`
	Warning       string                `json:"warning,omitempty"`
}

type BranchSessionResponse struct {
	Session session.Session `json:"session"`
}

type RecordHistoryEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	SessionID  string    `json:"session_id,omitempty"`
	ChangeType string    `json:"change_type"`
	Summary    string    `json:"summary"`
	Diff       string    `json:"diff,omitempty"`
}

type RecordDiffResponse struct {
	FromVersion record.Record `json:"from_version"`
	ToVersion   record.Record `json:"to_version"`
	Diff        RecordDiff    `json:"diff"`
}

type RecordDiff struct {
	Title   *FieldDiff      `json:"title,omitempty"`
	Summary *FieldDiff      `json:"summary,omitempty"`
	Body    string          `json:"body,omitempty"`
	State   *StateFieldDiff `json:"state,omitempty"`
}

type FieldDiff struct {
	Old string `json:"old"`
	New string `json:"new"`
}

type StateFieldDiff struct {
	Old record.RecordState `json:"old"`
	New record.RecordState `json:"new"`
}

type ActiveSessionStatus struct {
	SessionID    string    `json:"session_id"`
	LastActivity time.Time `json:"last_activity"`
	IsCurrent    bool      `json:"is_current"`
}

type ActivityEntryResponse struct {
	Timestamp time.Time             `json:"timestamp"`
	Type      activity.ActivityType `json:"type"`
	SessionID string                `json:"session_id,omitempty"`
	RecordID  *string               `json:"record_id,omitempty"`
	Summary   string                `json:"summary"`
	Details   string                `json:"details,omitempty"`
}
