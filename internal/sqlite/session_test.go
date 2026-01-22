package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/ganot/threds-mcp/internal/domain/session"
	"github.com/ganot/threds-mcp/internal/repository"
	"github.com/stretchr/testify/require"
)

func TestSessionRepository_CreateGet(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()
	insertProject(t, db, "p1", "tenant1")
	insertRecord(t, db, "r1", "p1", "tenant1")

	repo := NewSessionRepository(db)
	now := time.Now()
	sess := &session.Session{
		ID:           "s1",
		ProjectID:    "p1",
		Status:       session.StatusActive,
		LastSyncTick: 0,
		CreatedAt:    now,
		LastActivity: now,
	}

	require.NoError(t, repo.Create(ctx, "tenant1", sess))
	require.NoError(t, repo.AddActivation(ctx, "s1", "r1", 1))

	loaded, err := repo.Get(ctx, "tenant1", "s1")
	require.NoError(t, err)
	require.Equal(t, "tenant1", loaded.TenantID)
	require.Equal(t, "p1", loaded.ProjectID)
	require.Len(t, loaded.ActiveRecords, 1)
	require.Equal(t, "r1", loaded.ActiveRecords[0])
}

func TestSessionRepository_UpdateClose(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()
	insertProject(t, db, "p1", "tenant1")

	repo := NewSessionRepository(db)
	now := time.Now()
	sess := &session.Session{
		ID:           "s1",
		ProjectID:    "p1",
		Status:       session.StatusActive,
		LastSyncTick: 0,
		CreatedAt:    now,
		LastActivity: now,
	}

	require.NoError(t, repo.Create(ctx, "tenant1", sess))

	sess.Status = session.StatusStale
	sess.LastSyncTick = 2
	sess.LastActivity = time.Now()
	require.NoError(t, repo.Update(ctx, "tenant1", sess))

	loaded, err := repo.Get(ctx, "tenant1", "s1")
	require.NoError(t, err)
	require.Equal(t, session.StatusStale, loaded.Status)

	require.NoError(t, repo.Close(ctx, "tenant1", "s1"))
	loaded, err = repo.Get(ctx, "tenant1", "s1")
	require.NoError(t, err)
	require.Equal(t, session.StatusClosed, loaded.Status)
	require.NotNil(t, loaded.ClosedAt)
}

func TestSessionRepository_ListActiveAndGetByRecord(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()
	insertProject(t, db, "p1", "tenant1")
	insertRecord(t, db, "r1", "p1", "tenant1")

	repo := NewSessionRepository(db)
	now := time.Now()
	active := &session.Session{
		ID:           "s1",
		ProjectID:    "p1",
		Status:       session.StatusActive,
		LastSyncTick: 0,
		CreatedAt:    now,
		LastActivity: now,
	}
	stale := &session.Session{
		ID:           "s2",
		ProjectID:    "p1",
		Status:       session.StatusStale,
		LastSyncTick: 0,
		CreatedAt:    now,
		LastActivity: now,
	}

	require.NoError(t, repo.Create(ctx, "tenant1", active))
	require.NoError(t, repo.Create(ctx, "tenant1", stale))
	require.NoError(t, repo.AddActivation(ctx, "s1", "r1", 1))
	require.NoError(t, repo.AddActivation(ctx, "s2", "r1", 1))

	activeSessions, err := repo.ListActive(ctx, "tenant1", "p1")
	require.NoError(t, err)
	require.Len(t, activeSessions, 1)
	require.Equal(t, "s1", activeSessions[0].SessionID)

	sessionsByRecord, err := repo.GetByRecordID(ctx, "tenant1", "r1")
	require.NoError(t, err)
	require.Len(t, sessionsByRecord, 2)
}

func TestSessionRepository_Activations(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()
	insertProject(t, db, "p1", "tenant1")
	insertRecord(t, db, "r1", "p1", "tenant1")

	repo := NewSessionRepository(db)
	now := time.Now()
	sess := &session.Session{
		ID:           "s1",
		ProjectID:    "p1",
		Status:       session.StatusActive,
		LastSyncTick: 0,
		CreatedAt:    now,
		LastActivity: now,
	}

	require.NoError(t, repo.Create(ctx, "tenant1", sess))
	require.NoError(t, repo.AddActivation(ctx, "s1", "r1", 1))

	records, err := repo.GetActivations(ctx, "s1")
	require.NoError(t, err)
	require.Equal(t, []string{"r1"}, records)

	tick, err := repo.GetActivationTick(ctx, "s1", "r1")
	require.NoError(t, err)
	require.Equal(t, int64(1), tick)

	require.NoError(t, repo.AddActivation(ctx, "s1", "r1", 2))
	tick, err = repo.GetActivationTick(ctx, "s1", "r1")
	require.NoError(t, err)
	require.Equal(t, int64(2), tick)

	_, err = repo.GetActivationTick(ctx, "s1", "missing")
	require.Equal(t, repository.ErrNotFound, err)
}

func TestSessionRepository_TenantIsolation(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()
	insertProject(t, db, "p1", "tenant1")

	repo := NewSessionRepository(db)
	now := time.Now()
	sess := &session.Session{
		ID:           "s1",
		ProjectID:    "p1",
		Status:       session.StatusActive,
		LastSyncTick: 0,
		CreatedAt:    now,
		LastActivity: now,
	}
	require.NoError(t, repo.Create(ctx, "tenant1", sess))

	_, err := repo.Get(ctx, "tenant2", "s1")
	require.Equal(t, repository.ErrNotFound, err)
}

func insertRecord(t *testing.T, db *DB, id, projectID, tenantID string) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO records (id, tenant_id, project_id, type, title, summary, body, state, tick)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, tenantID, projectID, "note", "Title", "Summary", "Body", "OPEN", 1,
	)
	require.NoError(t, err)
}
