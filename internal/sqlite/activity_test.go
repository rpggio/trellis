package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/ganot/threds-mcp/internal/domain/activity"
	"github.com/stretchr/testify/require"
)

func TestActivityRepository_LogList(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()
	insertProject(t, db, "p1", "tenant1")

	repo := NewActivityRepository(db)
	entry1 := &activity.ActivityEntry{
		ProjectID:    "p1",
		ActivityType: activity.TypeRecordCreated,
		Summary:      "Created record",
		Details:      `{"id":"r1"}`,
		Tick:         1,
	}
	entry2 := &activity.ActivityEntry{
		ProjectID:    "p1",
		ActivityType: activity.TypeSessionStarted,
		Summary:      "Started session",
		Details:      `{"id":"s1"}`,
		Tick:         2,
	}

	require.NoError(t, repo.Log(ctx, "tenant1", entry1))
	time.Sleep(10 * time.Millisecond)
	require.NoError(t, repo.Log(ctx, "tenant1", entry2))

	entries, err := repo.List(ctx, "tenant1", activity.ListActivityOptions{ProjectID: "p1"})
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.Equal(t, entry2.ActivityType, entries[0].ActivityType)
	require.Equal(t, entry1.ActivityType, entries[1].ActivityType)
}

func TestActivityRepository_FiltersAndTenantIsolation(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()
	insertProject(t, db, "p1", "tenant1")
	insertProject(t, db, "p2", "tenant2")

	repo := NewActivityRepository(db)
	sessionID := "s1"
	recordID := "r1"
	entry := &activity.ActivityEntry{
		ProjectID:    "p1",
		SessionID:    &sessionID,
		RecordID:     &recordID,
		ActivityType: activity.TypeRecordUpdated,
		Summary:      "Updated record",
		Details:      "{}",
		Tick:         3,
	}
	require.NoError(t, repo.Log(ctx, "tenant1", entry))

	activityType := activity.TypeRecordUpdated
	opts := activity.ListActivityOptions{
		ProjectID:    "p1",
		SessionID:    &sessionID,
		RecordID:     &recordID,
		ActivityType: &activityType,
	}
	entries, err := repo.List(ctx, "tenant1", opts)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	entries, err = repo.List(ctx, "tenant2", activity.ListActivityOptions{ProjectID: "p2"})
	require.NoError(t, err)
	require.Len(t, entries, 0)
}
