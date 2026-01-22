package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rpggio/trellis/internal/domain/session"
	"github.com/rpggio/trellis/internal/repository"
)

// SessionRepository implements repository.SessionRepository for SQLite
type SessionRepository struct {
	db *DB
}

// NewSessionRepository creates a new SessionRepository
func NewSessionRepository(db *DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create creates a new session
func (r *SessionRepository) Create(ctx context.Context, tenantID string, sess *session.Session) error {
	query := `
		INSERT INTO sessions (
			id, tenant_id, project_id, status, parent_session,
			last_sync_tick, created_at, last_activity, closed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		sess.ID,
		tenantID,
		sess.ProjectID,
		sess.Status,
		sess.ParentSession,
		sess.LastSyncTick,
		sess.CreatedAt,
		sess.LastActivity,
		sess.ClosedAt,
	)
	if err != nil {
		if isForeignKeyViolation(err) {
			return repository.ErrForeignKeyViolation
		}
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// Get retrieves a session by ID
func (r *SessionRepository) Get(ctx context.Context, tenantID, id string) (*session.Session, error) {
	query := `
		SELECT
			id, tenant_id, project_id, status, parent_session,
			last_sync_tick, created_at, last_activity, closed_at
		FROM sessions
		WHERE id = ? AND tenant_id = ?
	`

	var sess session.Session
	var parentSession sql.NullString
	var closedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, query, id, tenantID).Scan(
		&sess.ID,
		&sess.TenantID,
		&sess.ProjectID,
		&sess.Status,
		&parentSession,
		&sess.LastSyncTick,
		&sess.CreatedAt,
		&sess.LastActivity,
		&closedAt,
	)
	if err == sql.ErrNoRows {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if parentSession.Valid {
		sess.ParentSession = &parentSession.String
	}
	if closedAt.Valid {
		sess.ClosedAt = &closedAt.Time
	}

	activations, err := r.getActivationsForTenant(ctx, tenantID, sess.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load activations: %w", err)
	}
	sess.ActiveRecords = activations

	return &sess, nil
}

// Update updates a session
func (r *SessionRepository) Update(ctx context.Context, tenantID string, sess *session.Session) error {
	query := `
		UPDATE sessions
		SET status = ?, parent_session = ?, last_sync_tick = ?,
		    last_activity = ?, closed_at = ?
		WHERE id = ? AND tenant_id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		sess.Status,
		sess.ParentSession,
		sess.LastSyncTick,
		sess.LastActivity,
		sess.ClosedAt,
		sess.ID,
		tenantID,
	)
	if err != nil {
		if isForeignKeyViolation(err) {
			return repository.ErrForeignKeyViolation
		}
		return fmt.Errorf("failed to update session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// Close marks a session as closed
func (r *SessionRepository) Close(ctx context.Context, tenantID, id string) error {
	now := time.Now()
	query := `
		UPDATE sessions
		SET status = ?, closed_at = ?, last_activity = ?
		WHERE id = ? AND tenant_id = ?
	`

	result, err := r.db.ExecContext(ctx, query, session.StatusClosed, now, now, id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to close session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// ListActive returns active sessions for a project
func (r *SessionRepository) ListActive(ctx context.Context, tenantID, projectID string) ([]session.SessionInfo, error) {
	query := `
		SELECT id, created_at, last_activity, last_sync_tick
		FROM sessions
		WHERE tenant_id = ? AND project_id = ? AND status = 'active'
		ORDER BY last_activity DESC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []session.SessionInfo
	for rows.Next() {
		var info session.SessionInfo
		if err := rows.Scan(&info.SessionID, &info.CreatedAt, &info.LastActivity, &info.LastSyncTick); err != nil {
			return nil, fmt.Errorf("failed to scan session info: %w", err)
		}
		sessions = append(sessions, info)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sessions: %w", err)
	}

	return sessions, nil
}

// GetByRecordID returns active or stale sessions where a record is activated
func (r *SessionRepository) GetByRecordID(ctx context.Context, tenantID, recordID string) ([]session.SessionInfo, error) {
	query := `
		SELECT s.id, s.created_at, s.last_activity, s.last_sync_tick
		FROM sessions s
		JOIN session_activations sa ON sa.session_id = s.id
		WHERE s.tenant_id = ? AND sa.record_id = ? AND s.status IN ('active', 'stale')
		ORDER BY s.last_activity DESC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, recordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions by record: %w", err)
	}
	defer rows.Close()

	var sessions []session.SessionInfo
	for rows.Next() {
		var info session.SessionInfo
		if err := rows.Scan(&info.SessionID, &info.CreatedAt, &info.LastActivity, &info.LastSyncTick); err != nil {
			return nil, fmt.Errorf("failed to scan session info: %w", err)
		}
		sessions = append(sessions, info)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sessions: %w", err)
	}

	return sessions, nil
}

// AddActivation tracks a record activation for a session
func (r *SessionRepository) AddActivation(ctx context.Context, sessionID, recordID string, tick int64) error {
	query := `
		INSERT INTO session_activations (session_id, record_id, activation_tick, activated_at)
		VALUES (?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query, sessionID, recordID, tick, time.Now())
	if err != nil {
		if isForeignKeyViolation(err) {
			return repository.ErrForeignKeyViolation
		}
		if isUniqueViolation(err) {
			updateQuery := `
				UPDATE session_activations
				SET activation_tick = ?, activated_at = ?
				WHERE session_id = ? AND record_id = ?
			`
			if _, updateErr := r.db.ExecContext(ctx, updateQuery, tick, time.Now(), sessionID, recordID); updateErr != nil {
				return fmt.Errorf("failed to refresh activation: %w", updateErr)
			}
			return nil
		}
		return fmt.Errorf("failed to add activation: %w", err)
	}

	return nil
}

// GetActivations returns all record IDs activated in a session
func (r *SessionRepository) GetActivations(ctx context.Context, sessionID string) ([]string, error) {
	query := `
		SELECT record_id
		FROM session_activations
		WHERE session_id = ?
		ORDER BY activated_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get activations: %w", err)
	}
	defer rows.Close()

	var records []string
	for rows.Next() {
		var recordID string
		if err := rows.Scan(&recordID); err != nil {
			return nil, fmt.Errorf("failed to scan activation: %w", err)
		}
		records = append(records, recordID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating activations: %w", err)
	}

	return records, nil
}

// GetActivationTick returns the activation tick for a record in a session
func (r *SessionRepository) GetActivationTick(ctx context.Context, sessionID, recordID string) (int64, error) {
	query := `
		SELECT activation_tick
		FROM session_activations
		WHERE session_id = ? AND record_id = ?
	`

	var tick int64
	err := r.db.QueryRowContext(ctx, query, sessionID, recordID).Scan(&tick)
	if err == sql.ErrNoRows {
		return 0, repository.ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get activation tick: %w", err)
	}

	return tick, nil
}

func (r *SessionRepository) getActivationsForTenant(ctx context.Context, tenantID, sessionID string) ([]string, error) {
	query := `
		SELECT sa.record_id
		FROM session_activations sa
		JOIN sessions s ON s.id = sa.session_id
		WHERE sa.session_id = ? AND s.tenant_id = ?
		ORDER BY sa.activated_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, sessionID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get activations: %w", err)
	}
	defer rows.Close()

	var records []string
	for rows.Next() {
		var recordID string
		if err := rows.Scan(&recordID); err != nil {
			return nil, fmt.Errorf("failed to scan activation: %w", err)
		}
		records = append(records, recordID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating activations: %w", err)
	}

	return records, nil
}
