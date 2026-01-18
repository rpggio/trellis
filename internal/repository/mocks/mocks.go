package mocks

import (
	"context"

	"github.com/ganot/threds-mcp/internal/domain/activity"
	"github.com/ganot/threds-mcp/internal/domain/project"
	"github.com/ganot/threds-mcp/internal/domain/record"
	"github.com/ganot/threds-mcp/internal/domain/session"
	"github.com/stretchr/testify/mock"
)

// ProjectRepository is a mock for repository.ProjectRepository.
type ProjectRepository struct {
	mock.Mock
}

func (m *ProjectRepository) Create(ctx context.Context, tenantID string, proj *project.Project) error {
	args := m.Called(ctx, tenantID, proj)
	return args.Error(0)
}

func (m *ProjectRepository) Get(ctx context.Context, tenantID, id string) (*project.Project, error) {
	args := m.Called(ctx, tenantID, id)
	if proj, ok := args.Get(0).(*project.Project); ok {
		return proj, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *ProjectRepository) GetDefault(ctx context.Context, tenantID string) (*project.Project, error) {
	args := m.Called(ctx, tenantID)
	if proj, ok := args.Get(0).(*project.Project); ok {
		return proj, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *ProjectRepository) List(ctx context.Context, tenantID string) ([]project.ProjectSummary, error) {
	args := m.Called(ctx, tenantID)
	if list, ok := args.Get(0).([]project.ProjectSummary); ok {
		return list, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *ProjectRepository) IncrementTick(ctx context.Context, tenantID, projectID string) (int64, error) {
	args := m.Called(ctx, tenantID, projectID)
	return args.Get(0).(int64), args.Error(1)
}

// RecordRepository is a mock for repository.RecordRepository.
type RecordRepository struct {
	mock.Mock
}

func (m *RecordRepository) Create(ctx context.Context, tenantID string, rec *record.Record) error {
	args := m.Called(ctx, tenantID, rec)
	return args.Error(0)
}

func (m *RecordRepository) Get(ctx context.Context, tenantID, id string) (*record.Record, error) {
	args := m.Called(ctx, tenantID, id)
	if rec, ok := args.Get(0).(*record.Record); ok {
		return rec, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *RecordRepository) Update(ctx context.Context, tenantID string, rec *record.Record, expectedTick int64) error {
	args := m.Called(ctx, tenantID, rec, expectedTick)
	return args.Error(0)
}

func (m *RecordRepository) Delete(ctx context.Context, tenantID, id string) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *RecordRepository) List(ctx context.Context, tenantID string, opts record.ListRecordsOptions) ([]record.RecordRef, error) {
	args := m.Called(ctx, tenantID, opts)
	if list, ok := args.Get(0).([]record.RecordRef); ok {
		return list, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *RecordRepository) GetChildren(ctx context.Context, tenantID, parentID string) ([]record.Record, error) {
	args := m.Called(ctx, tenantID, parentID)
	if list, ok := args.Get(0).([]record.Record); ok {
		return list, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *RecordRepository) GetChildrenRefs(ctx context.Context, tenantID, parentID string) ([]record.RecordRef, error) {
	args := m.Called(ctx, tenantID, parentID)
	if list, ok := args.Get(0).([]record.RecordRef); ok {
		return list, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *RecordRepository) GetRelated(ctx context.Context, tenantID, recordID string) ([]string, error) {
	args := m.Called(ctx, tenantID, recordID)
	if list, ok := args.Get(0).([]string); ok {
		return list, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *RecordRepository) AddRelation(ctx context.Context, fromRecordID, toRecordID string) error {
	args := m.Called(ctx, fromRecordID, toRecordID)
	return args.Error(0)
}

// SessionRepository is a mock for repository.SessionRepository.
type SessionRepository struct {
	mock.Mock
}

func (m *SessionRepository) Create(ctx context.Context, tenantID string, sess *session.Session) error {
	args := m.Called(ctx, tenantID, sess)
	return args.Error(0)
}

func (m *SessionRepository) Get(ctx context.Context, tenantID, id string) (*session.Session, error) {
	args := m.Called(ctx, tenantID, id)
	if sess, ok := args.Get(0).(*session.Session); ok {
		return sess, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *SessionRepository) Update(ctx context.Context, tenantID string, sess *session.Session) error {
	args := m.Called(ctx, tenantID, sess)
	return args.Error(0)
}

func (m *SessionRepository) Close(ctx context.Context, tenantID, id string) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *SessionRepository) ListActive(ctx context.Context, tenantID, projectID string) ([]session.SessionInfo, error) {
	args := m.Called(ctx, tenantID, projectID)
	if list, ok := args.Get(0).([]session.SessionInfo); ok {
		return list, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *SessionRepository) GetByRecordID(ctx context.Context, tenantID, recordID string) ([]session.SessionInfo, error) {
	args := m.Called(ctx, tenantID, recordID)
	if list, ok := args.Get(0).([]session.SessionInfo); ok {
		return list, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *SessionRepository) AddActivation(ctx context.Context, sessionID, recordID string, tick int64) error {
	args := m.Called(ctx, sessionID, recordID, tick)
	return args.Error(0)
}

func (m *SessionRepository) GetActivations(ctx context.Context, sessionID string) ([]string, error) {
	args := m.Called(ctx, sessionID)
	if list, ok := args.Get(0).([]string); ok {
		return list, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *SessionRepository) GetActivationTick(ctx context.Context, sessionID, recordID string) (int64, error) {
	args := m.Called(ctx, sessionID, recordID)
	return args.Get(0).(int64), args.Error(1)
}

// ActivityRepository is a mock for repository.ActivityRepository.
type ActivityRepository struct {
	mock.Mock
}

func (m *ActivityRepository) Log(ctx context.Context, tenantID string, entry *activity.ActivityEntry) error {
	args := m.Called(ctx, tenantID, entry)
	return args.Error(0)
}

func (m *ActivityRepository) List(ctx context.Context, tenantID string, opts activity.ListActivityOptions) ([]activity.ActivityEntry, error) {
	args := m.Called(ctx, tenantID, opts)
	if list, ok := args.Get(0).([]activity.ActivityEntry); ok {
		return list, args.Error(1)
	}
	return nil, args.Error(1)
}

// SearchRepository is a mock for repository.SearchRepository.
type SearchRepository struct {
	mock.Mock
}

func (m *SearchRepository) Search(ctx context.Context, tenantID, projectID, query string, opts record.SearchOptions) ([]record.SearchResult, error) {
	args := m.Called(ctx, tenantID, projectID, query, opts)
	if list, ok := args.Get(0).([]record.SearchResult); ok {
		return list, args.Error(1)
	}
	return nil, args.Error(1)
}
