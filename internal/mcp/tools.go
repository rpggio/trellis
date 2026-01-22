package mcp

import (
	"context"
	"fmt"

	"github.com/rpggio/trellis/internal/domain/activity"
	"github.com/rpggio/trellis/internal/domain/project"
	"github.com/rpggio/trellis/internal/domain/record"
	"github.com/rpggio/trellis/internal/domain/session"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerTools adds all currently supported MCP tools to the server.
func registerTools(server *sdkmcp.Server, svc Services) {
	// Projects (3 tools)
	registerProjectTools(server, svc)

	// Orientation (4 tools)
	registerOrientationTools(server, svc)

	// Activation (2 tools)
	registerActivationTools(server, svc)

	// Mutations (3 tools)
	registerMutationTools(server, svc)

	// Session Lifecycle (2 tools)
	registerSessionTools(server, svc)

	// History/Conflict (4 tools)
	registerHistoryTools(server, svc)

	// Utilities (7 tools) - minimal implementation
	registerUtilityTools(server, svc)
}

// Helper functions
func getProjectOrDefault(ctx context.Context, svc ProjectService, tenantID, projectID string) (*project.Project, error) {
	if projectID == "" {
		return svc.GetDefault(ctx, tenantID)
	}
	return svc.Get(ctx, tenantID, projectID)
}

func stringValue(val *string) string {
	if val == nil {
		return ""
	}
	return *val
}

// Project tools
func registerProjectTools(server *sdkmcp.Server, svc Services) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "create_project",
		Description: "Create a new project (container) for records; returns the created project with its tick.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input CreateProjectParams) (*sdkmcp.CallToolResult, *project.Project, error) {
		tenantID := getTenantID(ctx)
		proj, err := svc.Projects.Create(ctx, tenantID, project.CreateRequest{
			ID:          input.ID,
			Name:        input.Name,
			Description: input.Description,
		})
		return nil, proj, mapError(err)
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "list_projects",
		Description: "List projects for the current tenant (summaries include tick and open counts).",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, _ struct{}) (*sdkmcp.CallToolResult, *ListProjectsResponse, error) {
		tenantID := getTenantID(ctx)
		projects, err := svc.Projects.List(ctx, tenantID)
		if err != nil {
			return nil, nil, mapError(err)
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
		return nil, &ListProjectsResponse{Projects: resp}, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "get_project",
		Description: "Get a project by id, or the default project if id is omitted.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input GetProjectParams) (*sdkmcp.CallToolResult, *project.Project, error) {
		tenantID := getTenantID(ctx)
		if input.ID == "" {
			proj, err := svc.Projects.GetDefault(ctx, tenantID)
			return nil, proj, mapError(err)
		}
		proj, err := svc.Projects.Get(ctx, tenantID, input.ID)
		return nil, proj, mapError(err)
	})
}

// Orientation tools
func registerOrientationTools(server *sdkmcp.Server, svc Services) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "get_project_overview",
		Description: "Cold-start orientation: project tick + open sessions (with tick-gap warnings) + root record refs.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input GetProjectOverviewParams) (*sdkmcp.CallToolResult, *ProjectOverviewResponse, error) {
		tenantID := getTenantID(ctx)
		proj, err := getProjectOrDefault(ctx, svc.Projects, tenantID, input.ProjectID)
		if err != nil {
			return nil, nil, mapError(err)
		}

		rootID := ""
		rootRecords, err := svc.Records.List(ctx, tenantID, record.ListRecordsOptions{
			ProjectID: proj.ID,
			ParentID:  &rootID,
		})
		if err != nil {
			return nil, nil, mapError(err)
		}

		sessions, err := svc.Sessions.ListActiveSessions(ctx, tenantID, proj.ID)
		if err != nil {
			return nil, nil, mapError(err)
		}

		openSessions := make([]ProjectSessionStatus, 0, len(sessions))
		for _, sess := range sessions {
			tickGap := proj.Tick - sess.LastSyncTick
			activeRecords := sess.ActiveRecords
			if activeRecords == nil {
				activeRecords = []string{}
			}
			openSessions = append(openSessions, ProjectSessionStatus{
				ID:            sess.SessionID,
				ActiveRecords: activeRecords,
				LastSyncTick:  sess.LastSyncTick,
				TickGap:       tickGap,
			})
		}

		// Ensure slices are never nil (marshal as [] not null)
		if rootRecords == nil {
			rootRecords = []record.RecordRef{}
		}

		return nil, &ProjectOverviewResponse{
			Project:      *proj,
			OpenSessions: openSessions,
			RootRecords:  rootRecords,
		}, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "search_records",
		Description: "Browse cheaply: full-text search returning RecordRef hits (use limit to control result size).",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input SearchRecordsParams) (*sdkmcp.CallToolResult, *SearchRecordsResponse, error) {
		tenantID := getTenantID(ctx)
		proj, err := getProjectOrDefault(ctx, svc.Projects, tenantID, input.ProjectID)
		if err != nil {
			return nil, nil, mapError(err)
		}
		results, err := svc.Records.Search(ctx, tenantID, proj.ID, input.Query, record.SearchOptions{
			States: input.States,
			Types:  input.Types,
			Limit:  input.Limit,
			Offset: input.Offset,
		})
		if err != nil {
			return nil, nil, mapError(err)
		}
		return nil, &SearchRecordsResponse{Results: results}, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "list_records",
		Description: "Browse cheaply: list RecordRefs by parent/state/type (use limit/offset for pagination).",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input ListRecordsParams) (*sdkmcp.CallToolResult, *ListRecordsResponse, error) {
		tenantID := getTenantID(ctx)
		proj, err := getProjectOrDefault(ctx, svc.Projects, tenantID, input.ProjectID)
		if err != nil {
			return nil, nil, mapError(err)
		}
		results, err := svc.Records.List(ctx, tenantID, record.ListRecordsOptions{
			ProjectID: proj.ID,
			ParentID:  input.ParentID,
			States:    input.States,
			Types:     input.Types,
			Limit:     input.Limit,
			Offset:    input.Offset,
		})
		if err != nil {
			return nil, nil, mapError(err)
		}
		return nil, &ListRecordsResponse{Records: results}, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "get_record_ref",
		Description: "Get a lightweight RecordRef (summary view) by record id (no body).",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input GetRecordRefParams) (*sdkmcp.CallToolResult, record.RecordRef, error) {
		tenantID := getTenantID(ctx)
		ref, err := svc.Records.GetRef(ctx, tenantID, input.ID)
		return nil, ref, mapError(err)
	})
}

// Activation tools
func registerActivationTools(server *sdkmcp.Server, svc Services) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "activate",
		Description: "Enter reasoning mode: create/continue a session and load a minimal ContextBundle for a record.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input ActivateParams) (*sdkmcp.CallToolResult, *ActivateResponse, error) {
		tenantID := getTenantID(ctx)
		sessionID := getSessionID(ctx)

		result, err := svc.Sessions.Activate(ctx, tenantID, session.ActivateRequest{
			SessionID: sessionID,
			RecordID:  input.ID,
		})
		if err != nil {
			return nil, nil, mapError(err)
		}

		return nil, &ActivateResponse{
			SessionID: result.SessionID,
			Context:   result.Context,
			Warnings:  result.Warnings,
		}, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "sync_session",
		Description: "Refresh a session’s staleness (tick gap). Use when resuming work or before significant edits.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input SyncSessionParams) (*sdkmcp.CallToolResult, *SyncSessionResponse, error) {
		tenantID := getTenantID(ctx)
		sessionID := getSessionID(ctx)

		currentSessionID := sessionID
		if currentSessionID == "" {
			currentSessionID = input.SessionID
		}

		result, err := svc.Sessions.SyncSession(ctx, tenantID, currentSessionID)
		if err != nil {
			return nil, nil, mapError(err)
		}

		warning := ""
		if result.TickGap > 0 {
			warning = fmt.Sprintf("%d writes occurred since last sync", result.TickGap)
		}

		return nil, &SyncSessionResponse{
			SessionID:     result.SessionID,
			Staleness:     result.TickGap,
			SessionStatus: result.Status,
			Warning:       warning,
		}, nil
	})
}

// Mutation tools
func registerMutationTools(server *sdkmcp.Server, svc Services) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "create_record",
		Description: "Create a record (optionally under parent_id). Use when the user asks to persist; write it to stand alone; see `trellis://docs/record-writing`. If a session is active, the record is auto-activated.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input CreateRecordParams) (*sdkmcp.CallToolResult, *CreateRecordResponse, error) {
		tenantID := getTenantID(ctx)
		sessionID := getSessionID(ctx)

		proj, err := getProjectOrDefault(ctx, svc.Projects, tenantID, "")
		if err != nil {
			return nil, nil, mapError(err)
		}

		rec, err := svc.Records.Create(ctx, tenantID, record.CreateRequest{
			SessionID: sessionID,
			ProjectID: proj.ID,
			ParentID:  input.ParentID,
			Type:      input.Type,
			Title:     input.Title,
			Summary:   input.Summary,
			Body:      input.Body,
			State:     input.State,
			Related:   input.Related,
		})
		if err != nil {
			return nil, nil, mapError(err)
		}

		return nil, &CreateRecordResponse{
			Record:        *rec,
			AutoActivated: sessionID != "",
		}, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "update_record",
		Description: "Update an activated record when the user asks to persist changes. Keep it self-explaining; see `trellis://docs/record-writing`. May return a conflict unless force=true; requires a session id context.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input UpdateRecordParams) (*sdkmcp.CallToolResult, *UpdateRecordResponse, error) {
		tenantID := getTenantID(ctx)
		sessionID := getSessionID(ctx)
		if input.SessionID != "" {
			sessionID = input.SessionID
		}

		rec, conflict, err := svc.Records.Update(ctx, tenantID, record.UpdateRequest{
			SessionID: sessionID,
			ID:        input.ID,
			Title:     input.Title,
			Summary:   input.Summary,
			Body:      input.Body,
			Related:   input.Related,
			Force:     input.Force,
		})
		if err != nil {
			return nil, nil, mapError(err)
		}

		resp := &UpdateRecordResponse{Record: rec}
		if conflict != nil && conflict.RemoteVersion != nil {
			resp.Record = nil
			resp.Conflict = &RecordConflictResult{
				Message:      conflict.Message,
				OtherVersion: *conflict.RemoteVersion,
			}
		}
		return nil, resp, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "transition",
		Description: "Transition an activated record to a new workflow state (OPEN/LATER/RESOLVED/DISCARDED).",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input TransitionParams) (*sdkmcp.CallToolResult, *record.Record, error) {
		tenantID := getTenantID(ctx)
		sessionID := getSessionID(ctx)

		rec, err := svc.Records.Transition(ctx, tenantID, record.TransitionRequest{
			SessionID:  sessionID,
			ID:         input.ID,
			ToState:    input.ToState,
			Reason:     input.Reason,
			ResolvedBy: input.ResolvedBy,
		})
		return nil, rec, mapError(err)
	})
}

// Session lifecycle tools
func registerSessionTools(server *sdkmcp.Server, svc Services) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "save_session",
		Description: "Persist a session checkpoint (updates last_sync_tick) when the user asks to save/checkpoint. Uses current session or session_id argument.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input SaveSessionParams) (*sdkmcp.CallToolResult, map[string]string, error) {
		tenantID := getTenantID(ctx)
		sessionID := getSessionID(ctx)

		currentSessionID := sessionID
		if currentSessionID == "" {
			currentSessionID = input.SessionID
		}

		if err := svc.Sessions.SaveSession(ctx, tenantID, currentSessionID); err != nil {
			return nil, nil, mapError(err)
		}
		return nil, map[string]string{"status": "ok"}, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "close_session",
		Description: "Close a session when a thread of work is done (closing does not imply saving). Uses current session or session_id argument.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input CloseSessionParams) (*sdkmcp.CallToolResult, map[string]string, error) {
		tenantID := getTenantID(ctx)
		sessionID := getSessionID(ctx)

		currentSessionID := sessionID
		if currentSessionID == "" {
			currentSessionID = input.SessionID
		}

		if err := svc.Sessions.CloseSession(ctx, tenantID, currentSessionID); err != nil {
			return nil, nil, mapError(err)
		}
		return nil, map[string]string{"status": "closed"}, nil
	})
}

// History and conflict resolution tools
func registerHistoryTools(server *sdkmcp.Server, svc Services) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "get_record_history",
		Description: "Get recent change history entries for a record (lightweight, derived from activity log).",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input GetRecordHistoryParams) (*sdkmcp.CallToolResult, *GetRecordHistoryResponse, error) {
		tenantID := getTenantID(ctx)

		entries, err := svc.Activity.GetRecentActivity(ctx, tenantID, activity.ListActivityOptions{
			RecordID: &input.ID,
			Limit:    input.Limit,
		})
		if err != nil {
			return nil, nil, mapError(err)
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
		return nil, &GetRecordHistoryResponse{History: resp}, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "get_record_diff",
		Description: "Placeholder: returns the current record for both versions; no computed diff yet.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input GetRecordDiffParams) (*sdkmcp.CallToolResult, *RecordDiffResponse, error) {
		tenantID := getTenantID(ctx)

		current, err := svc.Records.Get(ctx, tenantID, input.ID)
		if err != nil {
			return nil, nil, mapError(err)
		}

		// Simple implementation - just return current version (no actual diff yet)
		return nil, &RecordDiffResponse{
			FromVersion: *current,
			ToVersion:   *current,
			Diff:        RecordDiff{},
		}, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "get_active_sessions",
		Description: "Get active sessions currently associated with a record (useful for concurrency awareness).",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input GetActiveSessionsParams) (*sdkmcp.CallToolResult, *GetActiveSessionsResponse, error) {
		tenantID := getTenantID(ctx)
		sessionID := getSessionID(ctx)

		sessions, err := svc.Sessions.GetActiveSessionsForRecord(ctx, tenantID, input.RecordID)
		if err != nil {
			return nil, nil, mapError(err)
		}

		resp := make([]ActiveSessionStatus, 0, len(sessions))
		for _, sess := range sessions {
			resp = append(resp, ActiveSessionStatus{
				SessionID:    sess.SessionID,
				LastActivity: sess.LastActivity,
				IsCurrent:    sess.SessionID == sessionID,
			})
		}
		return nil, &GetActiveSessionsResponse{Sessions: resp}, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "get_recent_activity",
		Description: "Get recent activity for a project or record without activating record bodies.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, input GetRecentActivityParams) (*sdkmcp.CallToolResult, *GetRecentActivityResponse, error) {
		tenantID := getTenantID(ctx)

		opts := activity.ListActivityOptions{
			ProjectID: input.ProjectID,
			RecordID:  input.RecordID,
			Limit:     input.Limit,
		}

		entries, err := svc.Activity.GetRecentActivity(ctx, tenantID, opts)
		if err != nil {
			return nil, nil, mapError(err)
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
		return nil, &GetRecentActivityResponse{Activity: resp}, nil
	})
}

// Utility tools (minimal implementations)
func registerUtilityTools(server *sdkmcp.Server, svc Services) {
	// ping - simple health check
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "ping",
		Description: "Health check; returns pong.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, _ struct{}) (*sdkmcp.CallToolResult, map[string]string, error) {
		return nil, map[string]string{"status": "pong"}, nil
	})

	// Note: Other utility tools (health, get_server_info, validate_ref, format_record,
	// get_schema, get_capabilities) can be added as needed. The SDK handles most
	// protocol-level operations automatically.
}
