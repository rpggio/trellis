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

	_, err := handler.Handle(ctx, tenantID, "", "create_project", mustJSON(t, CreateProjectParams{Name: "Proj"}))
	require.NoError(t, err)

	_, err = handler.Handle(ctx, tenantID, "", "list_projects", nil)
	require.NoError(t, err)

	_, err = handler.Handle(ctx, tenantID, "", "get_project", mustJSON(t, GetProjectParams{ID: "p1"}))
	require.NoError(t, err)

	_, err = handler.Handle(ctx, tenantID, "", "get_project_overview", mustJSON(t, GetProjectOverviewParams{ProjectID: "p1"}))
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

	_, err := handler.Handle(ctx, tenantID, sessionID, "activate", mustJSON(t, ActivateParams{ID: "r1"}))
	require.NoError(t, err)
	_, err = handler.Handle(ctx, tenantID, sessionID, "sync_session", nil)
	require.NoError(t, err)
	_, err = handler.Handle(ctx, tenantID, sessionID, "create_record", mustJSON(t, CreateRecordParams{Type: "note", Title: "t", Summary: "s", Body: "b"}))
	require.NoError(t, err)
	_, err = handler.Handle(ctx, tenantID, sessionID, "update_record", mustJSON(t, UpdateRecordParams{ID: "r1"}))
	require.NoError(t, err)
	_, err = handler.Handle(ctx, tenantID, sessionID, "transition", mustJSON(t, TransitionParams{ID: "r1", ToState: record.StateOpen}))
	require.NoError(t, err)
	_, err = handler.Handle(ctx, tenantID, sessionID, "save_session", nil)
	require.NoError(t, err)
	_, err = handler.Handle(ctx, tenantID, sessionID, "close_session", nil)
	require.NoError(t, err)
	_, err = handler.Handle(ctx, tenantID, sessionID, "branch_session", mustJSON(t, BranchSessionParams{SessionID: sessionID}))
	require.NoError(t, err)
	_, err = handler.Handle(ctx, tenantID, sessionID, "get_record_history", mustJSON(t, GetRecordHistoryParams{ID: "r1"}))
	require.NoError(t, err)
	_, err = handler.Handle(ctx, tenantID, sessionID, "get_record_diff", mustJSON(t, GetRecordDiffParams{ID: "r1", From: "last_save"}))
	require.NoError(t, err)
	_, err = handler.Handle(ctx, tenantID, sessionID, "get_active_sessions", mustJSON(t, GetActiveSessionsParams{RecordID: "r1"}))
	require.NoError(t, err)
	_, err = handler.Handle(ctx, tenantID, sessionID, "get_recent_activity", mustJSON(t, GetRecentActivityParams{}))
	require.NoError(t, err)
	_, err = handler.Handle(ctx, tenantID, sessionID, "search_records", mustJSON(t, SearchRecordsParams{Query: "q"}))
	require.NoError(t, err)
	_, err = handler.Handle(ctx, tenantID, sessionID, "list_records", mustJSON(t, ListRecordsParams{}))
	require.NoError(t, err)
	_, err = handler.Handle(ctx, tenantID, sessionID, "get_record_ref", mustJSON(t, GetRecordRefParams{ID: "r1"}))
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

	_, err := handler.Handle(ctx, tenantID, "", "update_record", mustJSON(t, UpdateRecordParams{ID: "r1"}))
	require.Error(t, err)
	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	require.Equal(t, "NOT_ACTIVATED", apiErr.Code)
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return data
}
