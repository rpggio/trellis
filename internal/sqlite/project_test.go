package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/rpggio/trellis/internal/domain/project"
	"github.com/rpggio/trellis/internal/repository"
	"github.com/stretchr/testify/require"
)

func TestProjectRepository_Create(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProjectRepository(db)
	ctx := context.Background()

	proj := &project.Project{
		ID:          "p1",
		TenantID:    "tenant1",
		Name:        "Test Project",
		Description: "A test project",
		Tick:        0,
		CreatedAt:   time.Now(),
	}

	err := repo.Create(ctx, "tenant1", proj)
	require.NoError(t, err)

	// Verify it was created
	retrieved, err := repo.Get(ctx, "tenant1", "p1")
	require.NoError(t, err)
	require.Equal(t, proj.ID, retrieved.ID)
	require.Equal(t, proj.Name, retrieved.Name)
	require.Equal(t, proj.Description, retrieved.Description)
}

func TestProjectRepository_Get(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProjectRepository(db)
	ctx := context.Background()

	// Create a project
	proj := &project.Project{
		ID:          "p1",
		TenantID:    "tenant1",
		Name:        "Test Project",
		Description: "A test project",
		Tick:        0,
		CreatedAt:   time.Now(),
	}
	err := repo.Create(ctx, "tenant1", proj)
	require.NoError(t, err)

	// Get it
	retrieved, err := repo.Get(ctx, "tenant1", "p1")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	require.Equal(t, "p1", retrieved.ID)

	// Try to get non-existent project
	_, err = repo.Get(ctx, "tenant1", "nonexistent")
	require.Equal(t, repository.ErrNotFound, err)
}

func TestProjectRepository_TenantIsolation(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProjectRepository(db)
	ctx := context.Background()

	// Create project for tenant1
	proj := &project.Project{
		ID:          "p1",
		TenantID:    "tenant1",
		Name:        "Tenant 1 Project",
		Description: "Project for tenant 1",
		Tick:        0,
		CreatedAt:   time.Now(),
	}
	err := repo.Create(ctx, "tenant1", proj)
	require.NoError(t, err)

	// Tenant 2 should not be able to see tenant 1's project
	_, err = repo.Get(ctx, "tenant2", "p1")
	require.Equal(t, repository.ErrNotFound, err)
}

func TestProjectRepository_GetDefault(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProjectRepository(db)
	ctx := context.Background()

	// No projects yet
	_, err := repo.GetDefault(ctx, "tenant1")
	require.Equal(t, repository.ErrNotFound, err)

	// Create two projects
	proj1 := &project.Project{
		ID:          "p1",
		TenantID:    "tenant1",
		Name:        "First Project",
		Description: "Created first",
		Tick:        0,
		CreatedAt:   time.Now(),
	}
	err = repo.Create(ctx, "tenant1", proj1)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond) // Ensure different timestamps

	proj2 := &project.Project{
		ID:          "p2",
		TenantID:    "tenant1",
		Name:        "Second Project",
		Description: "Created second",
		Tick:        0,
		CreatedAt:   time.Now(),
	}
	err = repo.Create(ctx, "tenant1", proj2)
	require.NoError(t, err)

	// Get default should return the first created project
	defaultProj, err := repo.GetDefault(ctx, "tenant1")
	require.NoError(t, err)
	require.Equal(t, "p1", defaultProj.ID)
}

func TestProjectRepository_List(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProjectRepository(db)
	ctx := context.Background()

	// Create two projects
	proj1 := &project.Project{
		ID:          "p1",
		TenantID:    "tenant1",
		Name:        "Project 1",
		Description: "First project",
		Tick:        0,
		CreatedAt:   time.Now(),
	}
	err := repo.Create(ctx, "tenant1", proj1)
	require.NoError(t, err)

	proj2 := &project.Project{
		ID:          "p2",
		TenantID:    "tenant1",
		Name:        "Project 2",
		Description: "Second project",
		Tick:        0,
		CreatedAt:   time.Now(),
	}
	err = repo.Create(ctx, "tenant1", proj2)
	require.NoError(t, err)

	// List projects
	summaries, err := repo.List(ctx, "tenant1")
	require.NoError(t, err)
	require.Len(t, summaries, 2)

	// Should be ordered by created_at DESC (newest first)
	require.Equal(t, "p2", summaries[0].ID)
	require.Equal(t, "p1", summaries[1].ID)

	// Check summary fields
	require.Equal(t, "Project 2", summaries[0].Name)
	require.Equal(t, 0, summaries[0].RecordCount)
	require.Equal(t, 0, summaries[0].OpenRecords)
	require.Equal(t, 0, summaries[0].ActiveSessions)
}

func TestProjectRepository_IncrementTick(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProjectRepository(db)
	ctx := context.Background()

	// Create a project
	proj := &project.Project{
		ID:          "p1",
		TenantID:    "tenant1",
		Name:        "Test Project",
		Description: "A test project",
		Tick:        0,
		CreatedAt:   time.Now(),
	}
	err := repo.Create(ctx, "tenant1", proj)
	require.NoError(t, err)

	// Increment tick
	newTick, err := repo.IncrementTick(ctx, "tenant1", "p1")
	require.NoError(t, err)
	require.Equal(t, int64(1), newTick)

	// Increment again
	newTick, err = repo.IncrementTick(ctx, "tenant1", "p1")
	require.NoError(t, err)
	require.Equal(t, int64(2), newTick)

	// Verify the project has the updated tick
	retrieved, err := repo.Get(ctx, "tenant1", "p1")
	require.NoError(t, err)
	require.Equal(t, int64(2), retrieved.Tick)

	// Try to increment non-existent project
	_, err = repo.IncrementTick(ctx, "tenant1", "nonexistent")
	require.Equal(t, repository.ErrNotFound, err)
}

func TestProjectRepository_IncrementTick_Multiple(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProjectRepository(db)
	ctx := context.Background()

	// Create a project
	proj := &project.Project{
		ID:          "p1",
		TenantID:    "tenant1",
		Name:        "Test Project",
		Description: "A test project",
		Tick:        0,
		CreatedAt:   time.Now(),
	}
	err := repo.Create(ctx, "tenant1", proj)
	require.NoError(t, err)

	// Increment tick multiple times sequentially
	numIncrements := 10
	for i := 1; i <= numIncrements; i++ {
		newTick, err := repo.IncrementTick(ctx, "tenant1", "p1")
		require.NoError(t, err)
		require.Equal(t, int64(i), newTick)
	}

	// Final tick should be numIncrements
	retrieved, err := repo.Get(ctx, "tenant1", "p1")
	require.NoError(t, err)
	require.Equal(t, int64(numIncrements), retrieved.Tick)
}
