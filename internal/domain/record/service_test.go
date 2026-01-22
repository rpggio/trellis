package record_test

import (
	"context"
	"testing"

	"github.com/rpggio/trellis/internal/domain/record"
	"github.com/rpggio/trellis/internal/repository/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRecordService_Create_WithParentActivation(t *testing.T) {
	ctx := context.Background()
	tenantID := "tenant1"
	parentID := "parent"

	recordsRepo := &mocks.RecordRepository{}
	sessionsRepo := &mocks.SessionRepository{}
	projectsRepo := &mocks.ProjectRepository{}
	activitiesRepo := &mocks.ActivityRepository{}

	sessionsRepo.On("GetActivations", ctx, "sess1").Return([]string{parentID}, nil)
	projectsRepo.On("IncrementTick", ctx, tenantID, "proj1").Return(int64(5), nil)
	recordsRepo.On("Create", ctx, tenantID, mock.Anything).Return(nil)
	sessionsRepo.On("AddActivation", ctx, "sess1", mock.Anything, int64(5)).Return(nil)
	activitiesRepo.On("Log", ctx, tenantID, mock.Anything).Return(nil)

	svc := record.NewService(recordsRepo, sessionsRepo, projectsRepo, activitiesRepo, nil, nil)
	rec, err := svc.Create(ctx, tenantID, record.CreateRequest{
		SessionID: "sess1",
		ProjectID: "proj1",
		ParentID:  &parentID,
		Type:      "question",
		Title:     "Title",
		Summary:   "Summary",
		Body:      "Body",
	})
	require.NoError(t, err)
	require.Equal(t, "proj1", rec.ProjectID)
	require.Equal(t, record.StateOpen, rec.State)
}

func TestRecordService_Create_ParentNotActivated(t *testing.T) {
	ctx := context.Background()
	tenantID := "tenant1"
	parentID := "parent"

	recordsRepo := &mocks.RecordRepository{}
	sessionsRepo := &mocks.SessionRepository{}
	projectsRepo := &mocks.ProjectRepository{}

	sessionsRepo.On("GetActivations", ctx, "sess1").Return([]string{}, nil)

	svc := record.NewService(recordsRepo, sessionsRepo, projectsRepo, nil, nil, nil)
	_, err := svc.Create(ctx, tenantID, record.CreateRequest{
		SessionID: "sess1",
		ProjectID: "proj1",
		ParentID:  &parentID,
		Type:      "question",
		Title:     "Title",
		Summary:   "Summary",
		Body:      "Body",
	})
	require.ErrorIs(t, err, record.ErrParentNotActivated)
}

func TestRecordService_Update_Conflict(t *testing.T) {
	ctx := context.Background()
	tenantID := "tenant1"
	recordID := "r1"

	recordsRepo := &mocks.RecordRepository{}
	sessionsRepo := &mocks.SessionRepository{}
	projectsRepo := &mocks.ProjectRepository{}

	sessionsRepo.On("GetActivations", ctx, "sess1").Return([]string{recordID}, nil)
	sessionsRepo.On("GetActivationTick", ctx, "sess1", recordID).Return(int64(1), nil)
	recordsRepo.On("Get", ctx, tenantID, recordID).Return(&record.Record{
		ID:        recordID,
		ProjectID: "proj1",
		Tick:      2,
	}, nil)

	svc := record.NewService(recordsRepo, sessionsRepo, projectsRepo, nil, nil, nil)
	updated, conflict, err := svc.Update(ctx, tenantID, record.UpdateRequest{
		SessionID: "sess1",
		ID:        recordID,
	})
	require.NoError(t, err)
	require.Nil(t, updated)
	require.NotNil(t, conflict)
}

func TestRecordService_Transition_Invalid(t *testing.T) {
	ctx := context.Background()
	tenantID := "tenant1"
	recordID := "r1"

	recordsRepo := &mocks.RecordRepository{}
	sessionsRepo := &mocks.SessionRepository{}
	projectsRepo := &mocks.ProjectRepository{}

	sessionsRepo.On("GetActivations", ctx, "sess1").Return([]string{recordID}, nil)
	recordsRepo.On("Get", ctx, tenantID, recordID).Return(&record.Record{
		ID:        recordID,
		ProjectID: "proj1",
		State:     record.StateResolved,
	}, nil)

	svc := record.NewService(recordsRepo, sessionsRepo, projectsRepo, nil, nil, nil)
	_, err := svc.Transition(ctx, tenantID, record.TransitionRequest{
		SessionID: "sess1",
		ID:        recordID,
		ToState:   record.StateLater,
	})
	require.ErrorIs(t, err, record.ErrInvalidTransition)
}
