package integration_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ganot/threds-mcp/internal/domain/activity"
	"github.com/ganot/threds-mcp/internal/domain/project"
	"github.com/ganot/threds-mcp/internal/domain/record"
	"github.com/ganot/threds-mcp/internal/domain/session"
	"github.com/ganot/threds-mcp/internal/sqlite"
	"github.com/stretchr/testify/require"
)

type testEnv struct {
	db           *sqlite.DB
	projectRepo  *sqlite.ProjectRepository
	recordRepo   *sqlite.RecordRepository
	sessionRepo  *sqlite.SessionRepository
	activityRepo *sqlite.ActivityRepository
	searchRepo   *sqlite.SearchRepository

	projectSvc  *project.Service
	recordSvc   *record.Service
	sessionSvc  *session.Service
	activitySvc *activity.Service
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := sqlite.New(dsn)
	require.NoError(t, err)
	require.NoError(t, db.RunMigrations())
	t.Cleanup(func() { _ = db.Close() })

	projectRepo := sqlite.NewProjectRepository(db)
	recordRepo := sqlite.NewRecordRepository(db)
	sessionRepo := sqlite.NewSessionRepository(db)
	activityRepo := sqlite.NewActivityRepository(db)
	searchRepo := sqlite.NewSearchRepository(db)

	projectSvc := project.NewService(projectRepo, nil)
	activitySvc := activity.NewService(activityRepo, nil)
	recordSvc := record.NewService(recordRepo, sessionRepo, projectRepo, activityRepo, searchRepo, nil)
	sessionSvc := session.NewService(recordRepo, sessionRepo, projectRepo, nil)

	return &testEnv{
		db:           db,
		projectRepo:  projectRepo,
		recordRepo:   recordRepo,
		sessionRepo:  sessionRepo,
		activityRepo: activityRepo,
		searchRepo:   searchRepo,
		projectSvc:   projectSvc,
		recordSvc:    recordSvc,
		sessionSvc:   sessionSvc,
		activitySvc:  activitySvc,
	}
}

func TestIntegration_ColdStartWorkflow(t *testing.T) {
	ctx := context.Background()
	env := newTestEnv(t)
	tenantID := "tenant1"

	proj, err := env.projectSvc.Create(ctx, tenantID, project.CreateRequest{Name: "Demo"})
	require.NoError(t, err)

	root, err := env.recordSvc.Create(ctx, tenantID, record.CreateRequest{
		ProjectID: proj.ID,
		Type:      "question",
		Title:     "Root",
		Summary:   "Root summary",
		Body:      "Root body",
	})
	require.NoError(t, err)

	activation, err := env.sessionSvc.Activate(ctx, tenantID, session.ActivateRequest{RecordID: root.ID})
	require.NoError(t, err)

	child, err := env.recordSvc.Create(ctx, tenantID, record.CreateRequest{
		SessionID: activation.SessionID,
		ProjectID: proj.ID,
		ParentID:  &root.ID,
		Type:      "note",
		Title:     "Child",
		Summary:   "Child summary",
		Body:      "Child body",
	})
	require.NoError(t, err)
	require.Equal(t, root.ID, *child.ParentID)
}

func TestIntegration_ConflictDetection(t *testing.T) {
	ctx := context.Background()
	env := newTestEnv(t)
	tenantID := "tenant1"

	proj, err := env.projectSvc.Create(ctx, tenantID, project.CreateRequest{Name: "Demo"})
	require.NoError(t, err)

	root, err := env.recordSvc.Create(ctx, tenantID, record.CreateRequest{
		ProjectID: proj.ID,
		Type:      "question",
		Title:     "Root",
		Summary:   "Root summary",
		Body:      "Root body",
	})
	require.NoError(t, err)

	sess1, err := env.sessionSvc.Activate(ctx, tenantID, session.ActivateRequest{RecordID: root.ID})
	require.NoError(t, err)
	sess2, err := env.sessionSvc.Activate(ctx, tenantID, session.ActivateRequest{RecordID: root.ID})
	require.NoError(t, err)

	newTitle := "Updated by session1"
	updated, conflict, err := env.recordSvc.Update(ctx, tenantID, record.UpdateRequest{
		SessionID: sess1.SessionID,
		ID:        root.ID,
		Title:     &newTitle,
	})
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Nil(t, conflict)

	otherTitle := "Updated by session2"
	updated, conflict, err = env.recordSvc.Update(ctx, tenantID, record.UpdateRequest{
		SessionID: sess2.SessionID,
		ID:        root.ID,
		Title:     &otherTitle,
	})
	require.NoError(t, err)
	require.Nil(t, updated)
	require.NotNil(t, conflict)
}

func TestIntegration_SessionBranching(t *testing.T) {
	ctx := context.Background()
	env := newTestEnv(t)
	tenantID := "tenant1"

	proj, err := env.projectSvc.Create(ctx, tenantID, project.CreateRequest{Name: "Demo"})
	require.NoError(t, err)

	root, err := env.recordSvc.Create(ctx, tenantID, record.CreateRequest{
		ProjectID: proj.ID,
		Type:      "question",
		Title:     "Root",
		Summary:   "Root summary",
		Body:      "Root body",
	})
	require.NoError(t, err)

	activation, err := env.sessionSvc.Activate(ctx, tenantID, session.ActivateRequest{RecordID: root.ID})
	require.NoError(t, err)

	branched, err := env.sessionSvc.BranchSession(ctx, tenantID, activation.SessionID, "")
	require.NoError(t, err)
	require.NotEqual(t, activation.SessionID, branched.ID)
	require.Equal(t, root.ID, *branched.FocusRecord)
	require.NotNil(t, branched.ParentSession)
}

func TestIntegration_StateTransitions(t *testing.T) {
	ctx := context.Background()
	env := newTestEnv(t)
	tenantID := "tenant1"

	proj, err := env.projectSvc.Create(ctx, tenantID, project.CreateRequest{Name: "Demo"})
	require.NoError(t, err)

	root, err := env.recordSvc.Create(ctx, tenantID, record.CreateRequest{
		ProjectID: proj.ID,
		Type:      "question",
		Title:     "Root",
		Summary:   "Root summary",
		Body:      "Root body",
	})
	require.NoError(t, err)

	resolver, err := env.recordSvc.Create(ctx, tenantID, record.CreateRequest{
		ProjectID: proj.ID,
		Type:      "note",
		Title:     "Resolver",
		Summary:   "Resolver summary",
		Body:      "Resolver body",
	})
	require.NoError(t, err)

	activation, err := env.sessionSvc.Activate(ctx, tenantID, session.ActivateRequest{RecordID: root.ID})
	require.NoError(t, err)

	reason := "needs more info"
	updated, err := env.recordSvc.Transition(ctx, tenantID, record.TransitionRequest{
		SessionID: activation.SessionID,
		ID:        root.ID,
		ToState:   record.StateLater,
		Reason:    &reason,
	})
	require.NoError(t, err)
	require.Equal(t, record.StateLater, updated.State)

	updated, err = env.recordSvc.Transition(ctx, tenantID, record.TransitionRequest{
		SessionID: activation.SessionID,
		ID:        root.ID,
		ToState:   record.StateOpen,
	})
	require.NoError(t, err)
	require.Equal(t, record.StateOpen, updated.State)

	resolvedBy := resolver.ID
	updated, err = env.recordSvc.Transition(ctx, tenantID, record.TransitionRequest{
		SessionID:  activation.SessionID,
		ID:         root.ID,
		ToState:    record.StateResolved,
		ResolvedBy: &resolvedBy,
	})
	require.NoError(t, err)
	require.Equal(t, record.StateResolved, updated.State)

	_, err = env.recordSvc.Transition(ctx, tenantID, record.TransitionRequest{
		SessionID: activation.SessionID,
		ID:        root.ID,
		ToState:   record.StateLater,
	})
	require.ErrorIs(t, err, record.ErrInvalidTransition)
}

func TestIntegration_ContextLoading(t *testing.T) {
	ctx := context.Background()
	env := newTestEnv(t)
	tenantID := "tenant1"

	proj, err := env.projectSvc.Create(ctx, tenantID, project.CreateRequest{Name: "Demo"})
	require.NoError(t, err)

	parent, err := env.recordSvc.Create(ctx, tenantID, record.CreateRequest{
		ProjectID: proj.ID,
		Type:      "question",
		Title:     "Parent",
		Summary:   "Parent summary",
		Body:      "Parent body",
	})
	require.NoError(t, err)

	parentActivation, err := env.sessionSvc.Activate(ctx, tenantID, session.ActivateRequest{RecordID: parent.ID})
	require.NoError(t, err)

	target, err := env.recordSvc.Create(ctx, tenantID, record.CreateRequest{
		SessionID: parentActivation.SessionID,
		ProjectID: proj.ID,
		ParentID:  &parent.ID,
		Type:      "question",
		Title:     "Target",
		Summary:   "Target summary",
		Body:      "Target body",
	})
	require.NoError(t, err)

	openChild, err := env.recordSvc.Create(ctx, tenantID, record.CreateRequest{
		SessionID: parentActivation.SessionID,
		ProjectID: proj.ID,
		ParentID:  &target.ID,
		Type:      "note",
		Title:     "Open child",
		Summary:   "Open summary",
		Body:      "Open body",
		State:     record.StateOpen,
	})
	require.NoError(t, err)

	resolvedChild, err := env.recordSvc.Create(ctx, tenantID, record.CreateRequest{
		SessionID: parentActivation.SessionID,
		ProjectID: proj.ID,
		ParentID:  &target.ID,
		Type:      "note",
		Title:     "Resolved child",
		Summary:   "Resolved summary",
		Body:      "Resolved body",
		State:     record.StateResolved,
	})
	require.NoError(t, err)

	_, err = env.recordSvc.Create(ctx, tenantID, record.CreateRequest{
		SessionID: parentActivation.SessionID,
		ProjectID: proj.ID,
		ParentID:  &openChild.ID,
		Type:      "note",
		Title:     "Grandchild",
		Summary:   "Grand summary",
		Body:      "Grand body",
	})
	require.NoError(t, err)

	activation, err := env.sessionSvc.Activate(ctx, tenantID, session.ActivateRequest{
		SessionID: parentActivation.SessionID,
		RecordID:  target.ID,
	})
	require.NoError(t, err)

	require.NotNil(t, activation.Context.Parent)
	require.Equal(t, parent.ID, activation.Context.Parent.ID)
	require.Len(t, activation.Context.OpenChildren, 1)
	require.Equal(t, openChild.ID, activation.Context.OpenChildren[0].ID)
	require.Len(t, activation.Context.OtherChildren, 1)
	require.Equal(t, resolvedChild.ID, activation.Context.OtherChildren[0].ID)
	require.Len(t, activation.Context.Grandchildren, 1)
}

func TestIntegration_SessionStaleness(t *testing.T) {
	ctx := context.Background()
	env := newTestEnv(t)
	tenantID := "tenant1"

	proj, err := env.projectSvc.Create(ctx, tenantID, project.CreateRequest{Name: "Demo"})
	require.NoError(t, err)

	root, err := env.recordSvc.Create(ctx, tenantID, record.CreateRequest{
		ProjectID: proj.ID,
		Type:      "question",
		Title:     "Root",
		Summary:   "Root summary",
		Body:      "Root body",
	})
	require.NoError(t, err)

	activation, err := env.sessionSvc.Activate(ctx, tenantID, session.ActivateRequest{RecordID: root.ID})
	require.NoError(t, err)

	_, err = env.recordSvc.Create(ctx, tenantID, record.CreateRequest{
		ProjectID: proj.ID,
		Type:      "note",
		Title:     "Another",
		Summary:   "Another summary",
		Body:      "Another body",
	})
	require.NoError(t, err)

	sync, err := env.sessionSvc.SyncSession(ctx, tenantID, activation.SessionID)
	require.NoError(t, err)
	require.Greater(t, sync.TickGap, int64(0))
}
