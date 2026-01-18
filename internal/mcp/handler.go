package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ganot/threds-mcp/internal/domain/activity"
	"github.com/ganot/threds-mcp/internal/domain/project"
	"github.com/ganot/threds-mcp/internal/domain/record"
	"github.com/ganot/threds-mcp/internal/domain/session"
)

// ProjectService defines project operations needed by MCP.
type ProjectService interface {
	Create(ctx context.Context, tenantID string, req project.CreateRequest) (*project.Project, error)
	List(ctx context.Context, tenantID string) ([]project.ProjectSummary, error)
	Get(ctx context.Context, tenantID, id string) (*project.Project, error)
	GetDefault(ctx context.Context, tenantID string) (*project.Project, error)
}

// RecordService defines record operations needed by MCP.
type RecordService interface {
	Create(ctx context.Context, tenantID string, req record.CreateRequest) (*record.Record, error)
	Update(ctx context.Context, tenantID string, req record.UpdateRequest) (*record.Record, *record.ConflictInfo, error)
	Transition(ctx context.Context, tenantID string, req record.TransitionRequest) (*record.Record, error)
	Get(ctx context.Context, tenantID, id string) (*record.Record, error)
	GetRef(ctx context.Context, tenantID, id string) (record.RecordRef, error)
	List(ctx context.Context, tenantID string, opts record.ListRecordsOptions) ([]record.RecordRef, error)
	Search(ctx context.Context, tenantID, projectID, query string, opts record.SearchOptions) ([]record.SearchResult, error)
}

// SessionService defines session operations needed by MCP.
type SessionService interface {
	Activate(ctx context.Context, tenantID string, req session.ActivateRequest) (*session.ActivateResult, error)
	SyncSession(ctx context.Context, tenantID, sessionID string) (*session.SyncResult, error)
	SaveSession(ctx context.Context, tenantID, sessionID string) error
	CloseSession(ctx context.Context, tenantID, sessionID string) error
	BranchSession(ctx context.Context, tenantID, sourceSessionID, focusRecordID string) (*session.Session, error)
	GetActiveSessionsForRecord(ctx context.Context, tenantID, recordID string) ([]session.SessionInfo, error)
	ListActiveSessions(ctx context.Context, tenantID, projectID string) ([]session.SessionInfo, error)
}

// ActivityService defines activity operations needed by MCP.
type ActivityService interface {
	GetRecentActivity(ctx context.Context, tenantID string, opts activity.ListActivityOptions) ([]activity.ActivityEntry, error)
}

// Handler dispatches MCP commands.
type Handler struct {
	projects ProjectService
	records  RecordService
	sessions SessionService
	activity ActivityService
}

// NewHandler creates a new MCP handler.
func NewHandler(projects ProjectService, records RecordService, sessions SessionService, activitySvc ActivityService) *Handler {
	return &Handler{
		projects: projects,
		records:  records,
		sessions: sessions,
		activity: activitySvc,
	}
}

// Handle dispatches MCP requests to domain services.
func (h *Handler) Handle(ctx context.Context, tenantID, sessionID, method string, params json.RawMessage) (any, error) {
	switch method {
	case "create_project":
		var req CreateProjectParams
		if err := decodeParams(params, &req); err != nil {
			return nil, err
		}
		return h.projects.Create(ctx, tenantID, project.CreateRequest{
			ID:          req.ID,
			Name:        req.Name,
			Description: req.Description,
		})
	case "list_projects":
		projects, err := h.projects.List(ctx, tenantID)
		if err != nil {
			return nil, mapError(err)
		}
		resp := make([]ProjectSummaryResponse, 0, len(projects))
		for _, proj := range projects {
			resp = append(resp, ProjectSummaryResponse{
				ID:           proj.ID,
				Name:         proj.Name,
				Description:  proj.Description,
				Tick:         proj.Tick,
				OpenSessions: proj.ActiveSessions,
				OpenRecords:  proj.OpenRecords,
			})
		}
		return resp, nil
	case "get_project":
		var req GetProjectParams
		if err := decodeParams(params, &req); err != nil {
			return nil, err
		}
		if req.ID == "" {
			return h.projects.GetDefault(ctx, tenantID)
		}
		return h.projects.Get(ctx, tenantID, req.ID)
	case "get_project_overview":
		var req GetProjectOverviewParams
		if err := decodeParams(params, &req); err != nil {
			return nil, err
		}
		proj, err := h.getProjectOrDefault(ctx, tenantID, req.ProjectID)
		if err != nil {
			return nil, err
		}
		rootID := ""
		rootRecords, err := h.records.List(ctx, tenantID, record.ListRecordsOptions{
			ProjectID: proj.ID,
			ParentID:  &rootID,
		})
		if err != nil {
			return nil, mapError(err)
		}
		sessions, err := h.sessions.ListActiveSessions(ctx, tenantID, proj.ID)
		if err != nil {
			return nil, mapError(err)
		}
		openSessions := make([]ProjectSessionStatus, 0, len(sessions))
		for _, sess := range sessions {
			tickGap := proj.Tick - sess.LastSyncTick
			warning := ""
			if tickGap > 0 {
				warning = fmt.Sprintf("%d writes have occurred since last sync", tickGap)
			}
			openSessions = append(openSessions, ProjectSessionStatus{
				ID:           sess.SessionID,
				FocusRecord:  stringValue(sess.FocusRecord),
				LastActivity: sess.LastActivity,
				LastSyncTick: sess.LastSyncTick,
				TickGap:      tickGap,
				Warning:      warning,
			})
		}
		return ProjectOverviewResponse{
			Project:      *proj,
			OpenSessions: openSessions,
			RootRecords:  rootRecords,
		}, nil
	case "search_records":
		var req SearchRecordsParams
		if err := decodeParams(params, &req); err != nil {
			return nil, err
		}
		proj, err := h.getProjectOrDefault(ctx, tenantID, req.ProjectID)
		if err != nil {
			return nil, err
		}
		return h.records.Search(ctx, tenantID, proj.ID, req.Query, record.SearchOptions{
			States: req.States,
			Types:  req.Types,
			Limit:  req.Limit,
			Offset: req.Offset,
		})
	case "list_records":
		var req ListRecordsParams
		if err := decodeParams(params, &req); err != nil {
			return nil, err
		}
		proj, err := h.getProjectOrDefault(ctx, tenantID, req.ProjectID)
		if err != nil {
			return nil, err
		}
		return h.records.List(ctx, tenantID, record.ListRecordsOptions{
			ProjectID: proj.ID,
			ParentID:  req.ParentID,
			States:    req.States,
			Types:     req.Types,
			Limit:     req.Limit,
			Offset:    req.Offset,
		})
	case "get_record_ref":
		var req GetRecordRefParams
		if err := decodeParams(params, &req); err != nil {
			return nil, err
		}
		return h.records.GetRef(ctx, tenantID, req.ID)
	case "activate":
		var req ActivateParams
		if err := decodeParams(params, &req); err != nil {
			return nil, err
		}
		result, err := h.sessions.Activate(ctx, tenantID, session.ActivateRequest{
			SessionID: sessionID,
			RecordID:  req.ID,
		})
		if err != nil {
			return nil, mapError(err)
		}
		return ActivateResponse{
			SessionID: result.SessionID,
			Context:   result.Context,
			Warnings:  result.Warnings,
		}, nil
	case "sync_session":
		var req SyncSessionParams
		if err := decodeParams(params, &req); err != nil {
			return nil, err
		}
		currentSessionID := sessionID
		if currentSessionID == "" {
			currentSessionID = req.SessionID
		}
		result, err := h.sessions.SyncSession(ctx, tenantID, currentSessionID)
		if err != nil {
			return nil, mapError(err)
		}
		warning := ""
		if result.TickGap > 0 {
			warning = fmt.Sprintf("%d writes occurred since last sync", result.TickGap)
		}
		return SyncSessionResponse{
			SessionID:     result.SessionID,
			Staleness:     result.TickGap,
			SessionStatus: result.Status,
			Warning:       warning,
		}, nil
	case "create_record":
		var req CreateRecordParams
		if err := decodeParams(params, &req); err != nil {
			return nil, err
		}
		proj, err := h.getProjectOrDefault(ctx, tenantID, "")
		if err != nil {
			return nil, err
		}
		rec, err := h.records.Create(ctx, tenantID, record.CreateRequest{
			SessionID: sessionID,
			ProjectID: proj.ID,
			ParentID:  req.ParentID,
			Type:      req.Type,
			Title:     req.Title,
			Summary:   req.Summary,
			Body:      req.Body,
			State:     req.State,
			Related:   req.Related,
		})
		if err != nil {
			return nil, mapError(err)
		}
		return CreateRecordResponse{
			Record:        *rec,
			AutoActivated: sessionID != "",
		}, nil
	case "update_record":
		var req UpdateRecordParams
		if err := decodeParams(params, &req); err != nil {
			return nil, err
		}
		rec, conflict, err := h.records.Update(ctx, tenantID, record.UpdateRequest{
			SessionID: sessionID,
			ID:        req.ID,
			Title:     req.Title,
			Summary:   req.Summary,
			Body:      req.Body,
			Related:   req.Related,
			Force:     req.Force,
		})
		if err != nil {
			return nil, mapError(err)
		}
		resp := UpdateRecordResponse{Record: rec}
		if conflict != nil && conflict.RemoteVersion != nil {
			resp.Record = nil
			resp.Conflict = &RecordConflictResult{
				Message:      conflict.Message,
				OtherVersion: *conflict.RemoteVersion,
			}
		}
		return resp, nil
	case "transition":
		var req TransitionParams
		if err := decodeParams(params, &req); err != nil {
			return nil, err
		}
		rec, err := h.records.Transition(ctx, tenantID, record.TransitionRequest{
			SessionID:  sessionID,
			ID:         req.ID,
			ToState:    req.ToState,
			Reason:     req.Reason,
			ResolvedBy: req.ResolvedBy,
		})
		if err != nil {
			return nil, mapError(err)
		}
		return rec, nil
	case "save_session":
		var req SaveSessionParams
		if err := decodeParams(params, &req); err != nil {
			return nil, err
		}
		currentSessionID := sessionID
		if currentSessionID == "" {
			currentSessionID = req.SessionID
		}
		if err := h.sessions.SaveSession(ctx, tenantID, currentSessionID); err != nil {
			return nil, mapError(err)
		}
		return map[string]string{"status": "ok"}, nil
	case "close_session":
		var req CloseSessionParams
		if err := decodeParams(params, &req); err != nil {
			return nil, err
		}
		currentSessionID := sessionID
		if currentSessionID == "" {
			currentSessionID = req.SessionID
		}
		if err := h.sessions.CloseSession(ctx, tenantID, currentSessionID); err != nil {
			return nil, mapError(err)
		}
		return map[string]string{"status": "closed"}, nil
	case "branch_session":
		var req BranchSessionParams
		if err := decodeParams(params, &req); err != nil {
			return nil, err
		}
		focus := req.FocusRecord
		sess, err := h.sessions.BranchSession(ctx, tenantID, req.SessionID, focus)
		if err != nil {
			return nil, mapError(err)
		}
		return BranchSessionResponse{Session: *sess}, nil
	case "get_record_history":
		var req GetRecordHistoryParams
		if err := decodeParams(params, &req); err != nil {
			return nil, err
		}
		entries, err := h.activity.GetRecentActivity(ctx, tenantID, activity.ListActivityOptions{
			RecordID: &req.ID,
			Limit:    req.Limit,
		})
		if err != nil {
			return nil, mapError(err)
		}
		resp := make([]RecordHistoryEntry, 0, len(entries))
		for _, entry := range entries {
			resp = append(resp, RecordHistoryEntry{
				Timestamp:  entry.CreatedAt,
				SessionID:  stringValue(entry.SessionID),
				ChangeType: string(entry.ActivityType),
				Summary:    entry.Summary,
			})
		}
		return resp, nil
	case "get_record_diff":
		var req GetRecordDiffParams
		if err := decodeParams(params, &req); err != nil {
			return nil, err
		}
		current, err := h.records.Get(ctx, tenantID, req.ID)
		if err != nil {
			return nil, mapError(err)
		}
		return RecordDiffResponse{
			FromVersion: *current,
			ToVersion:   *current,
			Diff:        RecordDiff{},
		}, nil
	case "get_active_sessions":
		var req GetActiveSessionsParams
		if err := decodeParams(params, &req); err != nil {
			return nil, err
		}
		sessions, err := h.sessions.GetActiveSessionsForRecord(ctx, tenantID, req.RecordID)
		if err != nil {
			return nil, mapError(err)
		}
		resp := make([]ActiveSessionStatus, 0, len(sessions))
		for _, sess := range sessions {
			resp = append(resp, ActiveSessionStatus{
				SessionID:    sess.SessionID,
				LastActivity: sess.LastActivity,
				IsCurrent:    sess.SessionID == sessionID,
			})
		}
		return resp, nil
	case "get_recent_activity":
		var req GetRecentActivityParams
		if err := decodeParams(params, &req); err != nil {
			return nil, err
		}
		opts := activity.ListActivityOptions{
			ProjectID: req.ProjectID,
			RecordID:  req.RecordID,
			Limit:     req.Limit,
		}
		entries, err := h.activity.GetRecentActivity(ctx, tenantID, opts)
		if err != nil {
			return nil, mapError(err)
		}
		resp := make([]ActivityEntryResponse, 0, len(entries))
		for _, entry := range entries {
			resp = append(resp, ActivityEntryResponse{
				Timestamp: entry.CreatedAt,
				Type:      entry.ActivityType,
				SessionID: stringValue(entry.SessionID),
				RecordID:  entry.RecordID,
				Summary:   entry.Summary,
				Details:   entry.Details,
			})
		}
		return resp, nil
	default:
		return nil, fmt.Errorf("unknown method: %s", method)
	}
}

func decodeParams(params json.RawMessage, out any) error {
	if len(params) == 0 {
		return nil
	}
	return json.Unmarshal(params, out)
}

func (h *Handler) getProjectOrDefault(ctx context.Context, tenantID, projectID string) (*project.Project, error) {
	if projectID == "" {
		return h.projects.GetDefault(ctx, tenantID)
	}
	return h.projects.Get(ctx, tenantID, projectID)
}

func mapError(err error) error {
	if apiErr := MapError(err); apiErr != nil {
		return apiErr
	}
	return err
}

func stringValue(val *string) string {
	if val == nil {
		return ""
	}
	return *val
}
