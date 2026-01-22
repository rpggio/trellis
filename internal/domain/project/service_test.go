package project_test

import (
	"context"
	"testing"

	"github.com/rpggio/trellis/internal/domain/project"
	"github.com/rpggio/trellis/internal/repository"
	"github.com/rpggio/trellis/internal/repository/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestProjectService_GetDefaultCreates(t *testing.T) {
	ctx := context.Background()
	tenantID := "tenant1"

	repo := &mocks.ProjectRepository{}
	repo.On("GetDefault", ctx, tenantID).Return((*project.Project)(nil), repository.ErrNotFound)
	repo.On("Create", ctx, tenantID, mock.Anything).Return(nil)

	svc := project.NewService(repo, nil)
	proj, err := svc.GetDefault(ctx, tenantID)
	require.NoError(t, err)
	require.NotEmpty(t, proj.ID)
	require.Equal(t, "Default Project", proj.Name)
}

func TestProjectService_CreateValidation(t *testing.T) {
	ctx := context.Background()
	tenantID := "tenant1"

	repo := &mocks.ProjectRepository{}
	svc := project.NewService(repo, nil)
	_, err := svc.Create(ctx, tenantID, project.CreateRequest{Name: ""})
	require.ErrorIs(t, err, project.ErrInvalidInput)
}
