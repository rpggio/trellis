package session_test

import (
	"context"
	"testing"
	"time"

	"github.com/ganot/threds-mcp/internal/domain/project"
	"github.com/ganot/threds-mcp/internal/domain/record"
	"github.com/ganot/threds-mcp/internal/domain/session"
	"github.com/ganot/threds-mcp/internal/repository/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSessionService_Activate_ContextLoading(t *testing.T) {
	ctx := context.Background()
	tenantID := "tenant1"
	recordID := "r1"
	parentID := "p1"

	recordsRepo := &mocks.RecordRepository{}
	sessionsRepo := &mocks.SessionRepository{}
	projectsRepo := &mocks.ProjectRepository{}

	target := &record.Record{
		ID:        recordID,
		ProjectID: "proj1",
		ParentID:  &parentID,
		State:     record.StateOpen,
	}
	parent := &record.Record{ID: parentID, ProjectID: "proj1"}

	children := []record.Record{
		{ID: "c1", State: record.StateOpen},
		{ID: "c2", State: record.StateResolved},
	}
	childRefs := []record.RecordRef{
		{ID: "c1", State: record.StateOpen},
		{ID: "c2", State: record.StateResolved},
	}

	recordsRepo.On("Get", ctx, tenantID, recordID).Return(target, nil)
	recordsRepo.On("Get", ctx, tenantID, parentID).Return(parent, nil)
	recordsRepo.On("GetChildren", ctx, tenantID, recordID).Return(children, nil)
	recordsRepo.On("GetChildrenRefs", ctx, tenantID, recordID).Return(childRefs, nil)
	recordsRepo.On("GetChildrenRefs", ctx, tenantID, "c1").Return([]record.RecordRef{}, nil)
	recordsRepo.On("GetChildrenRefs", ctx, tenantID, "c2").Return([]record.RecordRef{}, nil)

	projectsRepo.On("Get", ctx, tenantID, "proj1").Return(&project.Project{
		ID:   "proj1",
		Tick: 7,
	}, nil)

	sessionsRepo.On("Create", ctx, tenantID, mock.Anything).Return(nil)
	sessionsRepo.On("AddActivation", ctx, mock.Anything, recordID, int64(7)).Return(nil)
	sessionsRepo.On("GetByRecordID", ctx, tenantID, recordID).Return([]session.SessionInfo{}, nil)

	svc := session.NewService(recordsRepo, sessionsRepo, projectsRepo, nil)
	result, err := svc.Activate(ctx, tenantID, session.ActivateRequest{RecordID: recordID})
	require.NoError(t, err)
	require.NotEmpty(t, result.SessionID)
	require.NotNil(t, result.Context.Parent)
	require.Len(t, result.Context.OpenChildren, 1)
	require.Len(t, result.Context.OtherChildren, 1)
}

func TestSessionService_Activate_Warnings(t *testing.T) {
	ctx := context.Background()
	tenantID := "tenant1"
	recordID := "r1"

	recordsRepo := &mocks.RecordRepository{}
	sessionsRepo := &mocks.SessionRepository{}
	projectsRepo := &mocks.ProjectRepository{}

	recordsRepo.On("Get", ctx, tenantID, recordID).Return(&record.Record{
		ID:        recordID,
		ProjectID: "proj1",
	}, nil)
	recordsRepo.On("GetChildren", ctx, tenantID, recordID).Return([]record.Record{}, nil)
	recordsRepo.On("GetChildrenRefs", ctx, tenantID, recordID).Return([]record.RecordRef{}, nil)

	projectsRepo.On("Get", ctx, tenantID, "proj1").Return(&project.Project{
		ID:   "proj1",
		Tick: 2,
	}, nil)

	sessionsRepo.On("Create", ctx, tenantID, mock.Anything).Return(nil)
	sessionsRepo.On("AddActivation", ctx, mock.Anything, recordID, int64(2)).Return(nil)
	sessionsRepo.On("GetByRecordID", ctx, tenantID, recordID).Return([]session.SessionInfo{
		{SessionID: "other"},
	}, nil)

	svc := session.NewService(recordsRepo, sessionsRepo, projectsRepo, nil)
	result, err := svc.Activate(ctx, tenantID, session.ActivateRequest{RecordID: recordID})
	require.NoError(t, err)
	require.Len(t, result.Warnings, 1)
}

func TestSessionService_SyncSession(t *testing.T) {
	ctx := context.Background()
	tenantID := "tenant1"
	sessionID := "sess1"

	recordsRepo := &mocks.RecordRepository{}
	sessionsRepo := &mocks.SessionRepository{}
	projectsRepo := &mocks.ProjectRepository{}

	sessionsRepo.On("Get", ctx, tenantID, sessionID).Return(&session.Session{
		ID:           sessionID,
		ProjectID:    "proj1",
		LastSyncTick: 1,
	}, nil)
	projectsRepo.On("Get", ctx, tenantID, "proj1").Return(&project.Project{
		ID:   "proj1",
		Tick: 5,
	}, nil)
	sessionsRepo.On("Update", ctx, tenantID, mock.Anything).Return(nil)

	svc := session.NewService(recordsRepo, sessionsRepo, projectsRepo, nil)
	result, err := svc.SyncSession(ctx, tenantID, sessionID)
	require.NoError(t, err)
	require.Equal(t, int64(4), result.TickGap)
}

func TestSessionService_SaveClose(t *testing.T) {
	ctx := context.Background()
	tenantID := "tenant1"
	sessionID := "sess1"

	recordsRepo := &mocks.RecordRepository{}
	sessionsRepo := &mocks.SessionRepository{}
	projectsRepo := &mocks.ProjectRepository{}

	sessionsRepo.On("Get", ctx, tenantID, sessionID).Return(&session.Session{
		ID:        sessionID,
		ProjectID: "proj1",
	}, nil)
	projectsRepo.On("Get", ctx, tenantID, "proj1").Return(&project.Project{
		ID:   "proj1",
		Tick: 2,
	}, nil)
	sessionsRepo.On("Update", ctx, tenantID, mock.Anything).Return(nil)
	sessionsRepo.On("Close", ctx, tenantID, sessionID).Return(nil)

	svc := session.NewService(recordsRepo, sessionsRepo, projectsRepo, nil)
	require.NoError(t, svc.SaveSession(ctx, tenantID, sessionID))
	require.NoError(t, svc.CloseSession(ctx, tenantID, sessionID))
}

func TestSessionService_Activate_WithExistingSession(t *testing.T) {
	ctx := context.Background()
	tenantID := "tenant1"
	recordID := "r1"
	sessionID := "sess1"
	now := time.Now()

	recordsRepo := &mocks.RecordRepository{}
	sessionsRepo := &mocks.SessionRepository{}
	projectsRepo := &mocks.ProjectRepository{}

	recordsRepo.On("Get", ctx, tenantID, recordID).Return(&record.Record{
		ID:        recordID,
		ProjectID: "proj1",
	}, nil)
	recordsRepo.On("GetChildren", ctx, tenantID, recordID).Return([]record.Record{}, nil)
	recordsRepo.On("GetChildrenRefs", ctx, tenantID, recordID).Return([]record.RecordRef{}, nil)

	projectsRepo.On("Get", ctx, tenantID, "proj1").Return(&project.Project{
		ID:   "proj1",
		Tick: 9,
	}, nil)

	sessionsRepo.On("Get", ctx, tenantID, sessionID).Return(&session.Session{
		ID:           sessionID,
		ProjectID:    "proj1",
		LastSyncTick: 2,
		LastActivity: now,
	}, nil)
	sessionsRepo.On("Update", ctx, tenantID, mock.Anything).Return(nil)
	sessionsRepo.On("AddActivation", ctx, sessionID, recordID, int64(9)).Return(nil)
	sessionsRepo.On("GetByRecordID", ctx, tenantID, recordID).Return([]session.SessionInfo{}, nil)

	svc := session.NewService(recordsRepo, sessionsRepo, projectsRepo, nil)
	_, err := svc.Activate(ctx, tenantID, session.ActivateRequest{
		SessionID: sessionID,
		RecordID:  recordID,
	})
	require.NoError(t, err)
}
