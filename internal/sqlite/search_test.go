package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/ganot/threds-mcp/internal/domain/record"
	"github.com/ganot/threds-mcp/internal/repository"
	"github.com/stretchr/testify/require"
)

func TestSearchRepository_Search(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()
	insertProject(t, db, "p1", "tenant1")

	repo := NewRecordRepository(db)
	now := time.Now()
	rec := &record.Record{
		ID:         "r1",
		ProjectID:  "p1",
		Type:       "note",
		Title:      "Unique Search Title",
		Summary:    "Summary text",
		Body:       "Body text",
		State:      record.StateOpen,
		CreatedAt:  now,
		ModifiedAt: now,
		Tick:       1,
	}
	require.NoError(t, repo.Create(ctx, "tenant1", rec))

	searchRepo := NewSearchRepository(db)
	results, err := searchRepo.Search(ctx, "tenant1", "p1", "unique", repository.SearchOptions{})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "r1", results[0].Record.ID)
}

func TestSearchRepository_TenantIsolation(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()
	insertProject(t, db, "p1", "tenant1")
	insertProject(t, db, "p2", "tenant2")

	repo := NewRecordRepository(db)
	now := time.Now()
	rec := &record.Record{
		ID:         "r1",
		ProjectID:  "p1",
		Type:       "note",
		Title:      "Shared Term",
		Summary:    "Summary",
		Body:       "Body",
		State:      record.StateOpen,
		CreatedAt:  now,
		ModifiedAt: now,
		Tick:       1,
	}
	require.NoError(t, repo.Create(ctx, "tenant1", rec))

	rec2 := &record.Record{
		ID:         "r2",
		ProjectID:  "p2",
		Type:       "note",
		Title:      "Shared Term",
		Summary:    "Summary",
		Body:       "Body",
		State:      record.StateOpen,
		CreatedAt:  now,
		ModifiedAt: now,
		Tick:       1,
	}
	require.NoError(t, repo.Create(ctx, "tenant2", rec2))

	searchRepo := NewSearchRepository(db)
	results, err := searchRepo.Search(ctx, "tenant1", "p1", "shared", repository.SearchOptions{})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "r1", results[0].Record.ID)
}
