package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/ganot/threds-mcp/internal/domain/record"
	"github.com/ganot/threds-mcp/internal/repository"
	"github.com/stretchr/testify/require"
)

func TestRecordRepository_CreateGet(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()
	insertProject(t, db, "p1", "tenant1")

	repo := NewRecordRepository(db)
	now := time.Now()
	rec := &record.Record{
		ID:         "r1",
		ProjectID:  "p1",
		Type:       "question",
		Title:      "Title",
		Summary:    "Summary",
		Body:       "Body",
		State:      record.StateOpen,
		CreatedAt:  now,
		ModifiedAt: now,
		Tick:       1,
	}

	err := repo.Create(ctx, "tenant1", rec)
	require.NoError(t, err)

	loaded, err := repo.Get(ctx, "tenant1", "r1")
	require.NoError(t, err)
	require.Equal(t, "tenant1", loaded.TenantID)
	require.Equal(t, rec.Title, loaded.Title)
	require.Equal(t, rec.State, loaded.State)
}

func TestRecordRepository_TenantIsolation(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()
	insertProject(t, db, "p1", "tenant1")

	repo := NewRecordRepository(db)
	now := time.Now()
	rec := &record.Record{
		ID:         "r1",
		ProjectID:  "p1",
		Type:       "note",
		Title:      "Title",
		Summary:    "Summary",
		Body:       "Body",
		State:      record.StateOpen,
		CreatedAt:  now,
		ModifiedAt: now,
		Tick:       1,
	}

	err := repo.Create(ctx, "tenant1", rec)
	require.NoError(t, err)

	_, err = repo.Get(ctx, "tenant2", "r1")
	require.Equal(t, repository.ErrNotFound, err)
}

func TestRecordRepository_UpdateAndConflict(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()
	insertProject(t, db, "p1", "tenant1")

	repo := NewRecordRepository(db)
	now := time.Now()
	rec := &record.Record{
		ID:         "r1",
		ProjectID:  "p1",
		Type:       "question",
		Title:      "Title",
		Summary:    "Summary",
		Body:       "Body",
		State:      record.StateOpen,
		CreatedAt:  now,
		ModifiedAt: now,
		Tick:       1,
	}
	require.NoError(t, repo.Create(ctx, "tenant1", rec))

	rec.Title = "Updated"
	rec.ModifiedAt = time.Now()
	rec.Tick = 2
	require.NoError(t, repo.Update(ctx, "tenant1", rec, 1))

	rec.Title = "Conflict"
	rec.Tick = 3
	err := repo.Update(ctx, "tenant1", rec, 1)
	require.Equal(t, repository.ErrConflict, err)

	rec.ID = "missing"
	err = repo.Update(ctx, "tenant1", rec, 1)
	require.Equal(t, repository.ErrNotFound, err)
}

func TestRecordRepository_ListFilters(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()
	insertProject(t, db, "p1", "tenant1")

	repo := NewRecordRepository(db)
	now := time.Now()
	records := []*record.Record{
		{ID: "r1", ProjectID: "p1", Type: "question", Title: "Q1", Summary: "S1", Body: "B1", State: record.StateOpen, CreatedAt: now, ModifiedAt: now, Tick: 1},
		{ID: "r2", ProjectID: "p1", Type: "note", Title: "N1", Summary: "S2", Body: "B2", State: record.StateLater, CreatedAt: now, ModifiedAt: now, Tick: 2},
		{ID: "r3", ProjectID: "p1", Type: "question", Title: "Q2", Summary: "S3", Body: "B3", State: record.StateOpen, CreatedAt: now, ModifiedAt: now, Tick: 3},
	}
	for _, rec := range records {
		require.NoError(t, repo.Create(ctx, "tenant1", rec))
	}

	opts := repository.ListRecordsOptions{
		ProjectID: "p1",
		States:    []record.RecordState{record.StateOpen},
		Types:     []string{"question"},
	}
	refs, err := repo.List(ctx, "tenant1", opts)
	require.NoError(t, err)
	require.Len(t, refs, 2)
}

func TestRecordRepository_ChildrenAndRelations(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()
	insertProject(t, db, "p1", "tenant1")

	repo := NewRecordRepository(db)
	now := time.Now()
	parent := &record.Record{
		ID:         "parent",
		ProjectID:  "p1",
		Type:       "question",
		Title:      "Parent",
		Summary:    "Summary",
		Body:       "Body",
		State:      record.StateOpen,
		CreatedAt:  now,
		ModifiedAt: now,
		Tick:       1,
	}
	child := &record.Record{
		ID:         "child",
		ProjectID:  "p1",
		Type:       "note",
		Title:      "Child",
		Summary:    "Summary",
		Body:       "Body",
		State:      record.StateOpen,
		ParentID:   stringPtr("parent"),
		CreatedAt:  now,
		ModifiedAt: now,
		Tick:       2,
	}
	related := &record.Record{
		ID:         "related",
		ProjectID:  "p1",
		Type:       "note",
		Title:      "Related",
		Summary:    "Summary",
		Body:       "Body",
		State:      record.StateOpen,
		CreatedAt:  now,
		ModifiedAt: now,
		Tick:       3,
	}

	require.NoError(t, repo.Create(ctx, "tenant1", parent))
	require.NoError(t, repo.Create(ctx, "tenant1", child))
	require.NoError(t, repo.Create(ctx, "tenant1", related))

	require.NoError(t, repo.AddRelation(ctx, "parent", "related"))

	children, err := repo.GetChildren(ctx, "tenant1", "parent")
	require.NoError(t, err)
	require.Len(t, children, 1)
	require.Equal(t, "child", children[0].ID)

	childRefs, err := repo.GetChildrenRefs(ctx, "tenant1", "parent")
	require.NoError(t, err)
	require.Len(t, childRefs, 1)
	require.Equal(t, "child", childRefs[0].ID)

	relatedIDs, err := repo.GetRelated(ctx, "tenant1", "parent")
	require.NoError(t, err)
	require.Equal(t, []string{"related"}, relatedIDs)

	err = repo.AddRelation(ctx, "parent", "missing")
	require.Equal(t, repository.ErrForeignKeyViolation, err)
}

func TestRecordRepository_Delete(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()
	insertProject(t, db, "p1", "tenant1")

	repo := NewRecordRepository(db)
	now := time.Now()
	rec := &record.Record{
		ID:         "r1",
		ProjectID:  "p1",
		Type:       "question",
		Title:      "Title",
		Summary:    "Summary",
		Body:       "Body",
		State:      record.StateOpen,
		CreatedAt:  now,
		ModifiedAt: now,
		Tick:       1,
	}
	require.NoError(t, repo.Create(ctx, "tenant1", rec))

	require.NoError(t, repo.Delete(ctx, "tenant1", "r1"))

	_, err := repo.Get(ctx, "tenant1", "r1")
	require.Equal(t, repository.ErrNotFound, err)

	err = repo.Delete(ctx, "tenant1", "r1")
	require.Equal(t, repository.ErrNotFound, err)
}

func insertProject(t *testing.T, db *DB, id, tenantID string) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO projects (id, tenant_id, name, tick) VALUES (?, ?, ?, ?)`,
		id, tenantID, "Project", 0,
	)
	require.NoError(t, err)
}

func stringPtr(val string) *string {
	return &val
}
