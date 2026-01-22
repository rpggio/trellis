package record

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/rpggio/trellis/internal/domain/activity"
	"github.com/rpggio/trellis/internal/repository"
	"github.com/google/uuid"
)

// Service handles record business logic.
type Service struct {
	records    RecordRepository
	sessions   SessionRepository
	projects   ProjectRepository
	activities ActivityRepository
	search     SearchRepository
	logger     *slog.Logger
}

// NewService creates a new record service.
func NewService(
	records RecordRepository,
	sessions SessionRepository,
	projects ProjectRepository,
	activities ActivityRepository,
	search SearchRepository,
	logger *slog.Logger,
) *Service {
	return &Service{
		records:    records,
		sessions:   sessions,
		projects:   projects,
		activities: activities,
		search:     search,
		logger:     logger,
	}
}

// CreateRequest describes a record creation request.
type CreateRequest struct {
	SessionID string
	ProjectID string
	ParentID  *string
	Type      string
	Title     string
	Summary   string
	Body      string
	State     RecordState
	Related   []string
}

// UpdateRequest describes a record update request.
type UpdateRequest struct {
	SessionID string
	ID        string
	Title     *string
	Summary   *string
	Body      *string
	Related   []string
	Force     bool
}

// TransitionRequest describes a state transition request.
type TransitionRequest struct {
	SessionID  string
	ID         string
	ToState    RecordState
	Reason     *string
	ResolvedBy *string
}

// Create creates a new record with validation and tick increment.
func (s *Service) Create(ctx context.Context, tenantID string, req CreateRequest) (*Record, error) {
	if err := ValidateCreateInput(req); err != nil {
		return nil, err
	}

	if req.ParentID != nil {
		if req.SessionID == "" {
			return nil, ErrParentNotActivated
		}
		if err := s.ensureActivated(ctx, tenantID, req.SessionID, *req.ParentID, ErrParentNotActivated); err != nil {
			return nil, err
		}
	}

	state := req.State
	if state == "" {
		state = StateOpen
	}

	now := time.Now()
	newTick, err := s.projects.IncrementTick(ctx, tenantID, req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("incrementing tick: %w", err)
	}

	rec := &Record{
		ID:         uuid.NewString(),
		TenantID:   tenantID,
		ProjectID:  req.ProjectID,
		Type:       req.Type,
		Title:      req.Title,
		Summary:    req.Summary,
		Body:       req.Body,
		State:      state,
		ParentID:   req.ParentID,
		CreatedAt:  now,
		ModifiedAt: now,
		Tick:       newTick,
		Related:    req.Related,
	}

	if err := s.records.Create(ctx, tenantID, rec); err != nil {
		return nil, fmt.Errorf("creating record: %w", err)
	}

	if req.SessionID != "" {
		if err := s.sessions.AddActivation(ctx, req.SessionID, rec.ID, rec.Tick); err != nil {
			return nil, fmt.Errorf("adding activation: %w", err)
		}
	}

	if s.activities != nil {
		_ = s.activities.Log(ctx, tenantID, &activity.ActivityEntry{
			ProjectID:    rec.ProjectID,
			RecordID:     &rec.ID,
			ActivityType: activity.TypeRecordCreated,
			Summary:      fmt.Sprintf("created record %s", rec.ID),
			Tick:         rec.Tick,
		})
	}

	return rec, nil
}

// Update modifies an active record and performs conflict detection.
func (s *Service) Update(ctx context.Context, tenantID string, req UpdateRequest) (*Record, *ConflictInfo, error) {
	if req.SessionID == "" || req.ID == "" {
		return nil, nil, ErrInvalidInput
	}

	if err := s.ensureActivated(ctx, tenantID, req.SessionID, req.ID, ErrNotActivated); err != nil {
		return nil, nil, err
	}

	current, err := s.records.Get(ctx, tenantID, req.ID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil, ErrRecordNotFound
		}
		return nil, nil, fmt.Errorf("loading record: %w", err)
	}

	activationTick, err := s.sessions.GetActivationTick(ctx, req.SessionID, req.ID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil, ErrNotActivated
		}
		return nil, nil, fmt.Errorf("loading activation tick: %w", err)
	}

	if activationTick != current.Tick && !req.Force {
		return nil, &ConflictInfo{
			ConflictType:  "update",
			LocalVersion:  nil,
			RemoteVersion: current,
			Message:       "record modified since activation",
		}, nil
	}

	updated := *current
	if req.Title != nil {
		updated.Title = *req.Title
	}
	if req.Summary != nil {
		updated.Summary = *req.Summary
	}
	if req.Body != nil {
		updated.Body = *req.Body
	}
	if req.Related != nil {
		updated.Related = req.Related
	}
	updated.ModifiedAt = time.Now()

	newTick, err := s.projects.IncrementTick(ctx, tenantID, current.ProjectID)
	if err != nil {
		return nil, nil, fmt.Errorf("incrementing tick: %w", err)
	}
	updated.Tick = newTick

	if err := s.records.Update(ctx, tenantID, &updated, current.Tick); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, nil, ErrConflict
		}
		return nil, nil, fmt.Errorf("updating record: %w", err)
	}

	if s.activities != nil {
		_ = s.activities.Log(ctx, tenantID, &activity.ActivityEntry{
			ProjectID:    updated.ProjectID,
			RecordID:     &updated.ID,
			ActivityType: activity.TypeRecordUpdated,
			Summary:      fmt.Sprintf("updated record %s", updated.ID),
			Tick:         updated.Tick,
		})
	}

	return &updated, nil, nil
}

// Transition updates a record state with validation.
func (s *Service) Transition(ctx context.Context, tenantID string, req TransitionRequest) (*Record, error) {
	if req.SessionID == "" || req.ID == "" {
		return nil, ErrInvalidInput
	}

	if err := s.ensureActivated(ctx, tenantID, req.SessionID, req.ID, ErrNotActivated); err != nil {
		return nil, err
	}

	current, err := s.records.Get(ctx, tenantID, req.ID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf("loading record: %w", err)
	}

	if err := ValidateTransition(current.State, req.ToState, req.Reason, req.ResolvedBy); err != nil {
		return nil, err
	}

	updated := *current
	updated.State = req.ToState
	updated.ModifiedAt = time.Now()
	updated.ResolvedBy = req.ResolvedBy

	newTick, err := s.projects.IncrementTick(ctx, tenantID, current.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("incrementing tick: %w", err)
	}
	updated.Tick = newTick

	if err := s.records.Update(ctx, tenantID, &updated, current.Tick); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("transitioning record: %w", err)
	}

	if s.activities != nil {
		_ = s.activities.Log(ctx, tenantID, &activity.ActivityEntry{
			ProjectID:    updated.ProjectID,
			RecordID:     &updated.ID,
			ActivityType: activity.TypeStateTransition,
			Summary:      fmt.Sprintf("transitioned record %s", updated.ID),
			Tick:         updated.Tick,
		})
	}

	return &updated, nil
}

// Get returns a record by ID.
func (s *Service) Get(ctx context.Context, tenantID, id string) (*Record, error) {
	rec, err := s.records.Get(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf("getting record: %w", err)
	}
	return rec, nil
}

// GetRef returns a lightweight record reference with child counts.
func (s *Service) GetRef(ctx context.Context, tenantID, id string) (RecordRef, error) {
	rec, err := s.records.Get(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return RecordRef{}, ErrRecordNotFound
		}
		return RecordRef{}, fmt.Errorf("getting record: %w", err)
	}

	childRefs, err := s.records.GetChildrenRefs(ctx, tenantID, id)
	if err != nil {
		return RecordRef{}, fmt.Errorf("getting children: %w", err)
	}

	openCount := 0
	for _, ref := range childRefs {
		if ref.State == StateOpen {
			openCount++
		}
	}

	return RecordRef{
		ID:                rec.ID,
		Type:              rec.Type,
		Title:             rec.Title,
		Summary:           rec.Summary,
		State:             rec.State,
		ParentID:          rec.ParentID,
		ChildrenCount:     len(childRefs),
		OpenChildrenCount: openCount,
	}, nil
}

// List returns record references based on options.
func (s *Service) List(ctx context.Context, tenantID string, opts ListRecordsOptions) ([]RecordRef, error) {
	return s.records.List(ctx, tenantID, opts)
}

// Search runs full-text search.
func (s *Service) Search(ctx context.Context, tenantID, projectID, query string, opts SearchOptions) ([]SearchResult, error) {
	if s.search == nil {
		return nil, fmt.Errorf("search repository not configured")
	}
	return s.search.Search(ctx, tenantID, projectID, query, opts)
}

func (s *Service) ensureActivated(ctx context.Context, tenantID, sessionID, recordID string, errIfMissing error) error {
	activations, err := s.sessions.GetActivations(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("loading activations: %w", err)
	}

	for _, active := range activations {
		if active == recordID {
			return nil
		}
	}

	return errIfMissing
}
