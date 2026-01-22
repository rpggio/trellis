package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rpggio/trellis/internal/domain/activity"
)

// ActivityRepository implements repository.ActivityRepository for SQLite
type ActivityRepository struct {
	db *DB
}

// NewActivityRepository creates a new ActivityRepository
func NewActivityRepository(db *DB) *ActivityRepository {
	return &ActivityRepository{db: db}
}

// Log inserts a new activity entry
func (r *ActivityRepository) Log(ctx context.Context, tenantID string, entry *activity.ActivityEntry) error {
	createdAt := entry.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	query := `
		INSERT INTO activity_log (
			tenant_id, project_id, session_id, record_id,
			activity_type, summary, details, created_at, tick
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		tenantID,
		entry.ProjectID,
		entry.SessionID,
		entry.RecordID,
		entry.ActivityType,
		entry.Summary,
		entry.Details,
		createdAt,
		entry.Tick,
	)
	if err != nil {
		return fmt.Errorf("failed to log activity: %w", err)
	}

	id, err := result.LastInsertId()
	if err == nil {
		entry.ID = id
	}

	entry.TenantID = tenantID
	entry.CreatedAt = createdAt

	return nil
}

// List returns activity entries matching the given filters
func (r *ActivityRepository) List(ctx context.Context, tenantID string, opts activity.ListActivityOptions) ([]activity.ActivityEntry, error) {
	query := `
		SELECT
			id, tenant_id, project_id, session_id, record_id,
			activity_type, summary, details, created_at, tick
		FROM activity_log
		WHERE tenant_id = ?
	`

	args := []interface{}{tenantID}
	conditions := []string{}

	if opts.ProjectID != "" {
		conditions = append(conditions, "project_id = ?")
		args = append(args, opts.ProjectID)
	}
	if opts.RecordID != nil {
		conditions = append(conditions, "record_id = ?")
		args = append(args, *opts.RecordID)
	}
	if opts.SessionID != nil {
		conditions = append(conditions, "session_id = ?")
		args = append(args, *opts.SessionID)
	}
	if opts.ActivityType != nil {
		conditions = append(conditions, "activity_type = ?")
		args = append(args, *opts.ActivityType)
	}

	if len(conditions) > 0 {
		query += " AND " + joinConditions(conditions)
	}

	query += " ORDER BY created_at DESC"

	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}
	if opts.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, opts.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list activity: %w", err)
	}
	defer rows.Close()

	var entries []activity.ActivityEntry
	for rows.Next() {
		var entry activity.ActivityEntry
		var sessionID sql.NullString
		var recordID sql.NullString
		if err := rows.Scan(
			&entry.ID,
			&entry.TenantID,
			&entry.ProjectID,
			&sessionID,
			&recordID,
			&entry.ActivityType,
			&entry.Summary,
			&entry.Details,
			&entry.CreatedAt,
			&entry.Tick,
		); err != nil {
			return nil, fmt.Errorf("failed to scan activity entry: %w", err)
		}
		if sessionID.Valid {
			entry.SessionID = &sessionID.String
		}
		if recordID.Valid {
			entry.RecordID = &recordID.String
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating activity rows: %w", err)
	}

	return entries, nil
}

func joinConditions(conditions []string) string {
	if len(conditions) == 0 {
		return ""
	}
	joined := conditions[0]
	for i := 1; i < len(conditions); i++ {
		joined += " AND " + conditions[i]
	}
	return joined
}
