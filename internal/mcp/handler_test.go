package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ganot/threds-mcp/internal/domain/activity"
	"github.com/ganot/threds-mcp/internal/domain/project"
	"github.com/ganot/threds-mcp/internal/domain/record"
	"github.com/ganot/threds-mcp/internal/domain/session"
	"github.com/stretchr/testify/require"
)

type projectStub struct {
	createFn  func(context.Context, string, project.CreateRequest) (*project.Project, error)
	listFn    func(context.Context, string) ([]project.ProjectSummary, error)
	getFn     func(context.Context, string, string) (*project.Project, error)
	defaultFn func(context.Context, string) (*project.Project, error)
}

func (p projectStub) Create(ctx context.Context, tenantID string, req project.CreateRequest) (*project.Project, error) {
	return p.createFn(ctx, tenantID, req)
}
func (p projectStub) List(ctx context.Context, tenantID string) ([]project.ProjectSummary, error) {
	return p.listFn(ctx, tenantID)
}
func (p projectStub) Get(ctx context.Context, tenantID, id string) (*project.Project, error) {
	return p.getFn(ctx, tenantID, id)
}
func (p projectStub) GetDefault(ctx context.Context, tenantID string) (*project.Project, error) {
	return p.defaultFn(ctx, tenantID)
}

type recordStub struct {
	createFn     func(context.Context, string, record.CreateRequest) (*record.Record, error)
	updateFn     func(context.Context, string, record.UpdateRequest) (*record.Record, *record.ConflictInfo, error)
	transitionFn func(context.Context, string, record.TransitionRequest) (*record.Record, error)
	getFn        func(context.Context, string, string) (*record.Record, error)
	getRefFn     func(context.Context, string, string) (record.RecordRef, error)
	listFn       func(context.Context, string, record.ListRecordsOptions) ([]record.RecordRef, error)
	searchFn     func(context.Context, string, string, string, record.SearchOptions) ([]record.SearchResult, error)
}

func (r recordStub) Create(ctx context.Context, tenantID string, req record.CreateRequest) (*record.Record, error) {
	return r.createFn(ctx, tenantID, req)
}
func (r recordStub) Update(ctx context.Context, tenantID string, req record.UpdateRequest) (*record.Record, *record.ConflictInfo, error) {
	return r.updateFn(ctx, tenantID, req)
}
func (r recordStub) Transition(ctx context.Context, tenantID string, req record.TransitionRequest) (*record.Record, error) {
	return r.transitionFn(ctx, tenantID, req)
}
func (r recordStub) Get(ctx context.Context, tenantID, id string) (*record.Record, error) {
	return r.getFn(ctx, tenantID, id)
}
func (r recordStub) GetRef(ctx context.Context, tenantID, id string) (record.RecordRef, error) {
	return r.getRefFn(ctx, tenantID, id)
}
func (r recordStub) List(ctx context.Context, tenantID string, opts record.ListRecordsOptions) ([]record.RecordRef, error) {
	return r.listFn(ctx, tenantID, opts)
}
func (r recordStub) Search(ctx context.Context, tenantID, projectID, query string, opts record.SearchOptions) ([]record.SearchResult, error) {
	return r.searchFn(ctx, tenantID, projectID, query, opts)
}

type sessionStub struct {
	activateFn        func(context.Context, string, session.ActivateRequest) (*session.ActivateResult, error)
	syncFn            func(context.Context, string, string) (*session.SyncResult, error)
	saveFn            func(context.Context, string, string) error
	closeFn           func(context.Context, string, string) error
	branchFn          func(context.Context, string, string, string) (*session.Session, error)
	activeForRecordFn func(context.Context, string, string) ([]session.SessionInfo, error)
	listActiveFn      func(context.Context, string, string) ([]session.SessionInfo, error)
}

func (s sessionStub) Activate(ctx context.Context, tenantID string, req session.ActivateRequest) (*session.ActivateResult, error) {
	return s.activateFn(ctx, tenantID, req)
}
func (s sessionStub) SyncSession(ctx context.Context, tenantID, sessionID string) (*session.SyncResult, error) {
	return s.syncFn(ctx, tenantID, sessionID)
}
func (s sessionStub) SaveSession(ctx context.Context, tenantID, sessionID string) error {
	return s.saveFn(ctx, tenantID, sessionID)
}
func (s sessionStub) CloseSession(ctx context.Context, tenantID, sessionID string) error {
	return s.closeFn(ctx, tenantID, sessionID)
}
func (s sessionStub) BranchSession(ctx context.Context, tenantID, sourceSessionID, focusRecordID string) (*session.Session, error) {
	return s.branchFn(ctx, tenantID, sourceSessionID, focusRecordID)
}
func (s sessionStub) GetActiveSessionsForRecord(ctx context.Context, tenantID, recordID string) ([]session.SessionInfo, error) {
	return s.activeForRecordFn(ctx, tenantID, recordID)
}
func (s sessionStub) ListActiveSessions(ctx context.Context, tenantID, projectID string) ([]session.SessionInfo, error) {
	return s.listActiveFn(ctx, tenantID, projectID)
}

type activityStub struct {
	listFn func(context.Context, string, activity.ListActivityOptions) ([]activity.ActivityEntry, error)
}

func (a activityStub) GetRecentActivity(ctx context.Context, tenantID string, opts activity.ListActivityOptions) ([]activity.ActivityEntry, error) {
	return a.listFn(ctx, tenantID, opts)
}

func TestHandler_ProjectCommands(t *testing.T) {
	ctx := context.Background()
	tenantID := "tenant1"

	handler := NewHandler(
		projectStub{
			createFn: func(_ context.Context, _ string, req project.CreateRequest) (*project.Project, error) {
				return &project.Project{ID: req.ID, Name: req.Name}, nil
			},
			listFn: func(_ context.Context, _ string) ([]project.ProjectSummary, error) {
				return []project.ProjectSummary{{ID: "p1", Name: "Proj", Tick: 1, OpenRecords: 2, ActiveSessions: 1}}, nil
			},
			getFn: func(_ context.Context, _ string, id string) (*project.Project, error) {
				return &project.Project{ID: id, Name: "Proj"}, nil
			},
			defaultFn: func(_ context.Context, _ string) (*project.Project, error) {
				return &project.Project{ID: "default", Name: "Default"}, nil
			},
		},
		recordStub{
			listFn: func(_ context.Context, _ string, _ record.ListRecordsOptions) ([]record.RecordRef, error) {
				return []record.RecordRef{}, nil
			},
			searchFn: func(_ context.Context, _ string, _ string, _ string, _ record.SearchOptions) ([]record.SearchResult, error) {
				return []record.SearchResult{}, nil
			},
			getRefFn: func(_ context.Context, _ string, _ string) (record.RecordRef, error) {
				return record.RecordRef{ID: "r1"}, nil
			},
		},
		sessionStub{
			listActiveFn: func(_ context.Context, _ string, _ string) ([]session.SessionInfo, error) {
				return []session.SessionInfo{{SessionID: "s1", LastSyncTick: 1, LastActivity: time.Now()}}, nil
			},
		},
		activityStub{listFn: func(_ context.Context, _ string, _ activity.ListActivityOptions) ([]activity.ActivityEntry, error) {
			return []activity.ActivityEntry{}, nil
		}},
	)

	_, err := callTool(t, handler, ctx, tenantID, "", "create_project", CreateProjectParams{Name: "Proj"})
	require.NoError(t, err)

	_, err = callTool(t, handler, ctx, tenantID, "", "list_projects", nil)
	require.NoError(t, err)

	_, err = callTool(t, handler, ctx, tenantID, "", "get_project", GetProjectParams{ID: "p1"})
	require.NoError(t, err)

	_, err = callTool(t, handler, ctx, tenantID, "", "get_project_overview", GetProjectOverviewParams{ProjectID: "p1"})
	require.NoError(t, err)
}

func TestHandler_RecordAndSessionCommands(t *testing.T) {
	ctx := context.Background()
	tenantID := "tenant1"
	sessionID := "sess1"

	handler := NewHandler(
		projectStub{
			getFn: func(_ context.Context, _ string, id string) (*project.Project, error) {
				return &project.Project{ID: id, Tick: 3}, nil
			},
			defaultFn: func(_ context.Context, _ string) (*project.Project, error) {
				return &project.Project{ID: "proj1", Tick: 3}, nil
			},
		},
		recordStub{
			createFn: func(_ context.Context, _ string, _ record.CreateRequest) (*record.Record, error) {
				return &record.Record{ID: "r1", ProjectID: "proj1"}, nil
			},
			updateFn: func(_ context.Context, _ string, _ record.UpdateRequest) (*record.Record, *record.ConflictInfo, error) {
				return &record.Record{ID: "r1"}, nil, nil
			},
			transitionFn: func(_ context.Context, _ string, _ record.TransitionRequest) (*record.Record, error) {
				return &record.Record{ID: "r1"}, nil
			},
			getFn: func(_ context.Context, _ string, _ string) (*record.Record, error) {
				return &record.Record{ID: "r1"}, nil
			},
			getRefFn: func(_ context.Context, _ string, _ string) (record.RecordRef, error) {
				return record.RecordRef{ID: "r1"}, nil
			},
			listFn: func(_ context.Context, _ string, _ record.ListRecordsOptions) ([]record.RecordRef, error) {
				return []record.RecordRef{}, nil
			},
			searchFn: func(_ context.Context, _ string, _ string, _ string, _ record.SearchOptions) ([]record.SearchResult, error) {
				return []record.SearchResult{}, nil
			},
		},
		sessionStub{
			activateFn: func(_ context.Context, _ string, _ session.ActivateRequest) (*session.ActivateResult, error) {
				return &session.ActivateResult{SessionID: sessionID}, nil
			},
			syncFn: func(_ context.Context, _ string, _ string) (*session.SyncResult, error) {
				return &session.SyncResult{SessionID: sessionID, TickGap: 2, Status: session.StatusActive}, nil
			},
			saveFn:  func(_ context.Context, _ string, _ string) error { return nil },
			closeFn: func(_ context.Context, _ string, _ string) error { return nil },
			branchFn: func(_ context.Context, _ string, _ string, _ string) (*session.Session, error) {
				return &session.Session{ID: "branch"}, nil
			},
			activeForRecordFn: func(_ context.Context, _ string, _ string) ([]session.SessionInfo, error) {
				return []session.SessionInfo{{SessionID: sessionID, LastActivity: time.Now()}}, nil
			},
		},
		activityStub{listFn: func(_ context.Context, _ string, _ activity.ListActivityOptions) ([]activity.ActivityEntry, error) {
			return []activity.ActivityEntry{{ActivityType: activity.TypeRecordUpdated}}, nil
		}},
	)

	_, err := callTool(t, handler, ctx, tenantID, sessionID, "activate", ActivateParams{ID: "r1"})
	require.NoError(t, err)
	_, err = callTool(t, handler, ctx, tenantID, sessionID, "sync_session", nil)
	require.NoError(t, err)
	_, err = callTool(t, handler, ctx, tenantID, sessionID, "create_record", CreateRecordParams{Type: "note", Title: "t", Summary: "s", Body: "b"})
	require.NoError(t, err)
	_, err = callTool(t, handler, ctx, tenantID, sessionID, "update_record", UpdateRecordParams{ID: "r1"})
	require.NoError(t, err)
	_, err = callTool(t, handler, ctx, tenantID, sessionID, "transition", TransitionParams{ID: "r1", ToState: record.StateOpen})
	require.NoError(t, err)
	_, err = callTool(t, handler, ctx, tenantID, sessionID, "save_session", nil)
	require.NoError(t, err)
	_, err = callTool(t, handler, ctx, tenantID, sessionID, "close_session", nil)
	require.NoError(t, err)
	_, err = callTool(t, handler, ctx, tenantID, sessionID, "branch_session", BranchSessionParams{SessionID: sessionID})
	require.NoError(t, err)
	_, err = callTool(t, handler, ctx, tenantID, sessionID, "get_record_history", GetRecordHistoryParams{ID: "r1"})
	require.NoError(t, err)
	_, err = callTool(t, handler, ctx, tenantID, sessionID, "get_record_diff", GetRecordDiffParams{ID: "r1", From: "last_save"})
	require.NoError(t, err)
	_, err = callTool(t, handler, ctx, tenantID, sessionID, "get_active_sessions", GetActiveSessionsParams{RecordID: "r1"})
	require.NoError(t, err)
	_, err = callTool(t, handler, ctx, tenantID, sessionID, "get_recent_activity", GetRecentActivityParams{})
	require.NoError(t, err)
	_, err = callTool(t, handler, ctx, tenantID, sessionID, "search_records", SearchRecordsParams{Query: "q"})
	require.NoError(t, err)
	_, err = callTool(t, handler, ctx, tenantID, sessionID, "list_records", ListRecordsParams{})
	require.NoError(t, err)
	_, err = callTool(t, handler, ctx, tenantID, sessionID, "get_record_ref", GetRecordRefParams{ID: "r1"})
	require.NoError(t, err)
}

func TestHandler_ErrorMapping(t *testing.T) {
	ctx := context.Background()
	tenantID := "tenant1"

	handler := NewHandler(
		projectStub{
			defaultFn: func(_ context.Context, _ string) (*project.Project, error) {
				return &project.Project{ID: "p1"}, nil
			},
			getFn: func(_ context.Context, _ string, _ string) (*project.Project, error) {
				return &project.Project{ID: "p1"}, nil
			},
		},
		recordStub{
			updateFn: func(_ context.Context, _ string, _ record.UpdateRequest) (*record.Record, *record.ConflictInfo, error) {
				return nil, nil, record.ErrNotActivated
			},
			getFn: func(_ context.Context, _ string, _ string) (*record.Record, error) {
				return &record.Record{ID: "r1"}, nil
			},
			getRefFn: func(_ context.Context, _ string, _ string) (record.RecordRef, error) {
				return record.RecordRef{}, nil
			},
			listFn: func(_ context.Context, _ string, _ record.ListRecordsOptions) ([]record.RecordRef, error) {
				return []record.RecordRef{}, nil
			},
			searchFn: func(_ context.Context, _ string, _ string, _ string, _ record.SearchOptions) ([]record.SearchResult, error) {
				return []record.SearchResult{}, nil
			},
			createFn: func(_ context.Context, _ string, _ record.CreateRequest) (*record.Record, error) {
				return &record.Record{ID: "r1"}, nil
			},
			transitionFn: func(_ context.Context, _ string, _ record.TransitionRequest) (*record.Record, error) {
				return &record.Record{ID: "r1"}, nil
			},
		},
		sessionStub{
			activateFn: func(_ context.Context, _ string, _ session.ActivateRequest) (*session.ActivateResult, error) {
				return &session.ActivateResult{SessionID: "s1"}, nil
			},
			syncFn: func(_ context.Context, _ string, _ string) (*session.SyncResult, error) {
				return &session.SyncResult{SessionID: "s1"}, nil
			},
			saveFn:  func(_ context.Context, _ string, _ string) error { return nil },
			closeFn: func(_ context.Context, _ string, _ string) error { return nil },
			branchFn: func(_ context.Context, _ string, _ string, _ string) (*session.Session, error) {
				return &session.Session{ID: "s1"}, nil
			},
			activeForRecordFn: func(_ context.Context, _ string, _ string) ([]session.SessionInfo, error) {
				return []session.SessionInfo{}, nil
			},
			listActiveFn: func(_ context.Context, _ string, _ string) ([]session.SessionInfo, error) {
				return []session.SessionInfo{}, nil
			},
		},
		activityStub{listFn: func(_ context.Context, _ string, _ activity.ListActivityOptions) ([]activity.ActivityEntry, error) {
			return []activity.ActivityEntry{}, nil
		}},
	)

	result, err := callTool(t, handler, ctx, tenantID, "", "update_record", UpdateRecordParams{ID: "r1"})
	require.NoError(t, err) // tools/call doesn't return errors for domain errors

	// Check that the result is a ToolCallResult with isError=true
	toolResult, ok := result.(ToolCallResult)
	require.True(t, ok, "result should be a ToolCallResult")
	require.True(t, toolResult.IsError, "result should have IsError=true")
	require.NotEmpty(t, toolResult.Content)
	require.Contains(t, toolResult.Content[0].Text, "NOT_ACTIVATED")
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return data
}

// callTool is a helper to call tools using the MCP tools/call protocol
func callTool(t *testing.T, handler *Handler, ctx context.Context, tenantID, sessionID, toolName string, args any) (any, error) {
	t.Helper()
	argsMap := make(map[string]any)
	if args != nil {
		// Convert args to map
		argsJSON, err := json.Marshal(args)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(argsJSON, &argsMap))
	}
	params := ToolCallParams{
		Name:      toolName,
		Arguments: argsMap,
	}
	return handler.Handle(ctx, tenantID, sessionID, "tools/call", mustJSON(t, params))
}

func TestHandler_Initialize(t *testing.T) {
	ctx := context.Background()
	handler := NewHandler(
		projectStub{},
		recordStub{},
		sessionStub{},
		activityStub{},
	)

	// Test successful initialize with newer version
	result, err := handler.Handle(ctx, "", "", "initialize", mustJSON(t, InitializeParams{
		ProtocolVersion: "2025-11-25",
		Capabilities: ClientCapabilities{},
		ClientInfo: ImplementationInfo{
			Name:    "test-client",
			Version: "1.0.0",
		},
	}))
	require.NoError(t, err)

	initResult, ok := result.(InitializeResult)
	require.True(t, ok)
	require.Equal(t, "2025-11-25", initResult.ProtocolVersion)
	require.NotNil(t, initResult.Capabilities.Tools)
	require.True(t, initResult.Capabilities.Tools.ListChanged)
	require.Equal(t, "threds-mcp", initResult.ServerInfo.Name)

	// Test successful initialize with older version
	result, err = handler.Handle(ctx, "", "", "initialize", mustJSON(t, InitializeParams{
		ProtocolVersion: "2025-03-26",
		Capabilities: ClientCapabilities{},
		ClientInfo: ImplementationInfo{
			Name:    "test-client",
			Version: "1.0.0",
		},
	}))
	require.NoError(t, err)

	initResult, ok = result.(InitializeResult)
	require.True(t, ok)
	require.Equal(t, "2025-03-26", initResult.ProtocolVersion)

	// Test unsupported protocol version
	_, err = handler.Handle(ctx, "", "", "initialize", mustJSON(t, InitializeParams{
		ProtocolVersion: "1.0.0",
		Capabilities: ClientCapabilities{},
		ClientInfo: ImplementationInfo{
			Name:    "test-client",
			Version: "1.0.0",
		},
	}))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported protocol version")
}

func TestHandler_ToolsList(t *testing.T) {
	ctx := context.Background()
	handler := NewHandler(
		projectStub{},
		recordStub{},
		sessionStub{},
		activityStub{},
	)

	result, err := handler.Handle(ctx, "", "", "tools/list", mustJSON(t, ToolsListParams{}))
	require.NoError(t, err)

	listResult, ok := result.(ToolsListResult)
	require.True(t, ok)
	require.Greater(t, len(listResult.Tools), 15, "should have at least 16 tools")

	// Check that some expected tools exist
	toolNames := make(map[string]bool)
	for _, tool := range listResult.Tools {
		toolNames[tool.Name] = true
		require.NotEmpty(t, tool.Description)
		require.NotNil(t, tool.InputSchema)
	}

	require.True(t, toolNames["create_project"])
	require.True(t, toolNames["list_projects"])
	require.True(t, toolNames["activate"])
	require.True(t, toolNames["create_record"])
	require.True(t, toolNames["update_record"])
}

func TestHandler_UnknownMethod(t *testing.T) {
	ctx := context.Background()
	handler := NewHandler(
		projectStub{},
		recordStub{},
		sessionStub{},
		activityStub{},
	)

	// Test unknown protocol method
	_, err := handler.Handle(ctx, "tenant1", "", "unknown_method", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown method")
	require.Contains(t, err.Error(), "use 'tools/call'")

	// Test legacy direct method call (should fail with helpful message)
	_, err = handler.Handle(ctx, "tenant1", "", "create_project", mustJSON(t, CreateProjectParams{Name: "Test"}))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown method")
	require.Contains(t, err.Error(), "use 'tools/call'")
}

func TestHandler_UnknownTool(t *testing.T) {
	ctx := context.Background()
	handler := NewHandler(
		projectStub{},
		recordStub{},
		sessionStub{},
		activityStub{},
	)

	result, err := callTool(t, handler, ctx, "tenant1", "", "nonexistent_tool", nil)
	require.NoError(t, err) // tools/call wraps errors

	toolResult, ok := result.(ToolCallResult)
	require.True(t, ok)
	require.True(t, toolResult.IsError)
	require.Contains(t, toolResult.Content[0].Text, "unknown tool")
}
