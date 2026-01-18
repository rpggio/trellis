package activity_test

import (
	"context"
	"testing"

	"github.com/ganot/threds-mcp/internal/domain/activity"
	"github.com/ganot/threds-mcp/internal/repository/mocks"
	"github.com/stretchr/testify/require"
)

func TestActivityService_LogAndList(t *testing.T) {
	ctx := context.Background()
	tenantID := "tenant1"

	repo := &mocks.ActivityRepository{}
	entry := &activity.ActivityEntry{
		ProjectID:    "proj1",
		ActivityType: activity.TypeRecordCreated,
		Summary:      "created",
		Tick:         1,
	}

	repo.On("Log", ctx, tenantID, entry).Return(nil)
	repo.On("List", ctx, tenantID, activity.ListActivityOptions{ProjectID: "proj1"}).Return([]activity.ActivityEntry{}, nil)

	svc := activity.NewService(repo, nil)
	require.NoError(t, svc.LogActivity(ctx, tenantID, entry))
	_, err := svc.GetRecentActivity(ctx, tenantID, activity.ListActivityOptions{ProjectID: "proj1"})
	require.NoError(t, err)
}
