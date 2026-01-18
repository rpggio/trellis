package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ganot/threds-mcp/internal/domain/record"
	"github.com/ganot/threds-mcp/internal/repository"
	"github.com/google/uuid"
)

// Service handles session operations.
type Service struct {
	records  RecordRepository
	sessions SessionRepository
	projects ProjectRepository
	logger   *slog.Logger
}

// NewService creates a new session service.
func NewService(
	records RecordRepository,
	sessions SessionRepository,
	projects ProjectRepository,
	logger *slog.Logger,
) *Service {
	return &Service{
		records:  records,
		sessions: sessions,
		projects: projects,
		logger:   logger,
	}
}

// ActivateRequest describes a session activation request.
type ActivateRequest struct {
	SessionID string
	RecordID  string
}

// ActivateResult holds activation response data.
type ActivateResult struct {
	SessionID string
	Context   ContextBundle
	Warnings  []string
}

// SyncResult describes a sync response.
type SyncResult struct {
	SessionID string
	TickGap   int64
	Status    SessionStatus
}

// Activate activates a record and returns a context bundle.
func (s *Service) Activate(ctx context.Context, tenantID string, req ActivateRequest) (*ActivateResult, error) {
	if req.RecordID == "" {
		return nil, ErrInvalidInput
	}

	target, err := s.records.Get(ctx, tenantID, req.RecordID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf("loading record: %w", err)
	}

	proj, err := s.projects.Get(ctx, tenantID, target.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("loading project: %w", err)
	}

	sessionID, err := s.ensureSession(ctx, tenantID, req.SessionID, target, proj.Tick)
	if err != nil {
		return nil, err
	}

	if err := s.sessions.AddActivation(ctx, sessionID, target.ID, proj.Tick); err != nil {
		return nil, fmt.Errorf("adding activation: %w", err)
	}

	context, err := s.loadContext(ctx, tenantID, target)
	if err != nil {
		return nil, err
	}

	warnings, err := s.activationWarnings(ctx, tenantID, sessionID, target.ID)
	if err != nil {
		return nil, err
	}

	return &ActivateResult{
		SessionID: sessionID,
		Context:   context,
		Warnings:  warnings,
	}, nil
}

// SyncSession updates last sync tick and returns staleness info.
func (s *Service) SyncSession(ctx context.Context, tenantID, sessionID string) (*SyncResult, error) {
	if sessionID == "" {
		return nil, ErrInvalidInput
	}

	sess, err := s.sessions.Get(ctx, tenantID, sessionID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("loading session: %w", err)
	}

	proj, err := s.projects.Get(ctx, tenantID, sess.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("loading project: %w", err)
	}

	tickGap := proj.Tick - sess.LastSyncTick
	sess.LastSyncTick = proj.Tick
	sess.LastActivity = time.Now()
	sess.Status = StatusActive

	if err := s.sessions.Update(ctx, tenantID, sess); err != nil {
		return nil, fmt.Errorf("updating session: %w", err)
	}

	return &SyncResult{
		SessionID: sessionID,
		TickGap:   tickGap,
		Status:    sess.Status,
	}, nil
}

// SaveSession records a sync point.
func (s *Service) SaveSession(ctx context.Context, tenantID, sessionID string) error {
	if sessionID == "" {
		return ErrInvalidInput
	}

	sess, err := s.sessions.Get(ctx, tenantID, sessionID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrSessionNotFound
		}
		return fmt.Errorf("loading session: %w", err)
	}

	proj, err := s.projects.Get(ctx, tenantID, sess.ProjectID)
	if err != nil {
		return fmt.Errorf("loading project: %w", err)
	}

	sess.LastSyncTick = proj.Tick
	sess.LastActivity = time.Now()
	sess.Status = StatusActive

	if err := s.sessions.Update(ctx, tenantID, sess); err != nil {
		return fmt.Errorf("updating session: %w", err)
	}

	return nil
}

// CloseSession closes a session.
func (s *Service) CloseSession(ctx context.Context, tenantID, sessionID string) error {
	if sessionID == "" {
		return ErrInvalidInput
	}

	if err := s.sessions.Close(ctx, tenantID, sessionID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrSessionNotFound
		}
		return fmt.Errorf("closing session: %w", err)
	}
	return nil
}

// BranchSession creates a new session from an existing one.
func (s *Service) BranchSession(ctx context.Context, tenantID, sourceSessionID, focusRecordID string) (*Session, error) {
	if sourceSessionID == "" {
		return nil, ErrInvalidInput
	}

	source, err := s.sessions.Get(ctx, tenantID, sourceSessionID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("loading session: %w", err)
	}

	proj, err := s.projects.Get(ctx, tenantID, source.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("loading project: %w", err)
	}

	if focusRecordID == "" && source.FocusRecord != nil {
		focusRecordID = *source.FocusRecord
	}

	now := time.Now()
	sess := &Session{
		ID:            uuid.NewString(),
		TenantID:      tenantID,
		ProjectID:     source.ProjectID,
		Status:        StatusActive,
		FocusRecord:   stringPtr(focusRecordID),
		ParentSession: &sourceSessionID,
		LastSyncTick:  proj.Tick,
		CreatedAt:     now,
		LastActivity:  now,
	}

	if err := s.sessions.Create(ctx, tenantID, sess); err != nil {
		return nil, fmt.Errorf("creating session: %w", err)
	}

	if focusRecordID != "" {
		if err := s.sessions.AddActivation(ctx, sess.ID, focusRecordID, proj.Tick); err != nil {
			return nil, fmt.Errorf("adding activation: %w", err)
		}
	}

	return sess, nil
}

// GetActiveSessionsForRecord returns active sessions for a record.
func (s *Service) GetActiveSessionsForRecord(ctx context.Context, tenantID, recordID string) ([]SessionInfo, error) {
	return s.sessions.GetByRecordID(ctx, tenantID, recordID)
}

func (s *Service) ensureSession(ctx context.Context, tenantID, sessionID string, target *record.Record, projectTick int64) (string, error) {
	now := time.Now()
	if sessionID == "" {
		newID := uuid.NewString()
		sess := &Session{
			ID:           newID,
			TenantID:     tenantID,
			ProjectID:    target.ProjectID,
			Status:       StatusActive,
			FocusRecord:  &target.ID,
			LastSyncTick: projectTick,
			CreatedAt:    now,
			LastActivity: now,
		}
		if err := s.sessions.Create(ctx, tenantID, sess); err != nil {
			return "", fmt.Errorf("creating session: %w", err)
		}
		return newID, nil
	}

	sess, err := s.sessions.Get(ctx, tenantID, sessionID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			sess = &Session{
				ID:           sessionID,
				TenantID:     tenantID,
				ProjectID:    target.ProjectID,
				Status:       StatusActive,
				FocusRecord:  &target.ID,
				LastSyncTick: projectTick,
				CreatedAt:    now,
				LastActivity: now,
			}
			if err := s.sessions.Create(ctx, tenantID, sess); err != nil {
				return "", fmt.Errorf("creating session: %w", err)
			}
			return sessionID, nil
		}
		return "", fmt.Errorf("loading session: %w", err)
	}

	sess.LastSyncTick = projectTick
	sess.LastActivity = now
	sess.Status = StatusActive
	if sess.FocusRecord == nil {
		sess.FocusRecord = &target.ID
	}
	if err := s.sessions.Update(ctx, tenantID, sess); err != nil {
		return "", fmt.Errorf("updating session: %w", err)
	}

	return sessionID, nil
}

func (s *Service) loadContext(ctx context.Context, tenantID string, target *record.Record) (ContextBundle, error) {
	var parent *record.Record
	if target.ParentID != nil {
		p, err := s.records.Get(ctx, tenantID, *target.ParentID)
		if err != nil {
			return ContextBundle{}, fmt.Errorf("loading parent: %w", err)
		}
		parent = p
	}

	children, err := s.records.GetChildren(ctx, tenantID, target.ID)
	if err != nil {
		return ContextBundle{}, fmt.Errorf("loading children: %w", err)
	}
	childRefs, err := s.records.GetChildrenRefs(ctx, tenantID, target.ID)
	if err != nil {
		return ContextBundle{}, fmt.Errorf("loading child refs: %w", err)
	}

	childrenByID := make(map[string]record.Record, len(children))
	for _, child := range children {
		childrenByID[child.ID] = child
	}

	var openChildren []record.Record
	var otherChildren []record.RecordRef
	for _, ref := range childRefs {
		if ref.State == record.StateOpen {
			if full, ok := childrenByID[ref.ID]; ok {
				openChildren = append(openChildren, full)
			}
		} else {
			otherChildren = append(otherChildren, ref)
		}
	}

	var grandchildren []record.RecordRef
	for _, ref := range childRefs {
		refs, err := s.records.GetChildrenRefs(ctx, tenantID, ref.ID)
		if err != nil {
			return ContextBundle{}, fmt.Errorf("loading grandchildren: %w", err)
		}
		grandchildren = append(grandchildren, refs...)
	}

	return ContextBundle{
		Target:        *target,
		Parent:        parent,
		OpenChildren:  openChildren,
		OtherChildren: otherChildren,
		Grandchildren: grandchildren,
	}, nil
}

func (s *Service) activationWarnings(ctx context.Context, tenantID, sessionID, recordID string) ([]string, error) {
	sessions, err := s.sessions.GetByRecordID(ctx, tenantID, recordID)
	if err != nil {
		return nil, fmt.Errorf("loading active sessions: %w", err)
	}

	var warnings []string
	for _, info := range sessions {
		if info.SessionID != sessionID {
			warnings = append(warnings, fmt.Sprintf("record active in session %s", info.SessionID))
		}
	}
	return warnings, nil
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
