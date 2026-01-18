package sqlite

// TODO: Review and test. Ran out of tokens for prior coding run.

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/ganot/threds-mcp/internal/domain/record"
	"github.com/ganot/threds-mcp/internal/repository"
)

// RecordRepository implements repository.RecordRepository for SQLite
type RecordRepository struct {
	db *DB
}

// NewRecordRepository creates a new RecordRepository
func NewRecordRepository(db *DB) *RecordRepository {
	return &RecordRepository{db: db}
}

// Create creates a new record
func (r *RecordRepository) Create(ctx context.Context, tenantID string, rec *record.Record) error {
	query := `
		INSERT INTO records (
			id, tenant_id, project_id, type, title, summary, body,
			state, parent_id, resolved_by, created_at, modified_at, tick
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		rec.ID,
		tenantID,
		rec.ProjectID,
		rec.Type,
		rec.Title,
		rec.Summary,
		rec.Body,
		rec.State,
		rec.ParentID,
		rec.ResolvedBy,
		rec.CreatedAt,
		rec.ModifiedAt,
		rec.Tick,
	)

	if err != nil {
		if isForeignKeyViolation(err) {
			return repository.ErrForeignKeyViolation
		}
		return fmt.Errorf("failed to create record: %w", err)
	}

	// Add relations if any
	if len(rec.Related) > 0 {
		for _, relatedID := range rec.Related {
			if err := r.AddRelation(ctx, rec.ID, relatedID); err != nil {
				return fmt.Errorf("failed to add relation: %w", err)
			}
		}
	}

	return nil
}

// Get retrieves a record by ID
func (r *RecordRepository) Get(ctx context.Context, tenantID, id string) (*record.Record, error) {
	query := `
		SELECT
			id, tenant_id, project_id, type, title, summary, body,
			state, parent_id, resolved_by, created_at, modified_at, tick
		FROM records
		WHERE id = ? AND tenant_id = ?
	`

	var rec record.Record
	err := r.db.QueryRowContext(ctx, query, id, tenantID).Scan(
		&rec.ID,
		&rec.TenantID,
		&rec.ProjectID,
		&rec.Type,
		&rec.Title,
		&rec.Summary,
		&rec.Body,
		&rec.State,
		&rec.ParentID,
		&rec.ResolvedBy,
		&rec.CreatedAt,
		&rec.ModifiedAt,
		&rec.Tick,
	)

	if err == sql.ErrNoRows {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get record: %w", err)
	}

	// Load related records
	related, err := r.GetRelated(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get related records: %w", err)
	}
	rec.Related = related

	return &rec, nil
}

// Update updates a record with optimistic concurrency control
func (r *RecordRepository) Update(ctx context.Context, tenantID string, rec *record.Record, expectedTick int64) error {
	query := `
		UPDATE records
		SET type = ?, title = ?, summary = ?, body = ?,
		    state = ?, resolved_by = ?, modified_at = ?, tick = ?
		WHERE id = ? AND tenant_id = ? AND tick = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		rec.Type,
		rec.Title,
		rec.Summary,
		rec.Body,
		rec.State,
		rec.ResolvedBy,
		rec.ModifiedAt,
		rec.Tick,
		rec.ID,
		tenantID,
		expectedTick,
	)

	if err != nil {
		return fmt.Errorf("failed to update record: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		// Check if record exists
		var exists bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM records WHERE id = ? AND tenant_id = ?)`
		err = r.db.QueryRowContext(ctx, checkQuery, rec.ID, tenantID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check record existence: %w", err)
		}

		if !exists {
			return repository.ErrNotFound
		}

		// Record exists but tick doesn't match - conflict
		return repository.ErrConflict
	}

	return nil
}

// Delete deletes a record
func (r *RecordRepository) Delete(ctx context.Context, tenantID, id string) error {
	query := `DELETE FROM records WHERE id = ? AND tenant_id = ?`

	result, err := r.db.ExecContext(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
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

// List returns records matching the given options as lightweight references
func (r *RecordRepository) List(ctx context.Context, tenantID string, opts repository.ListRecordsOptions) ([]record.RecordRef, error) {
	query := `
		SELECT
			r.id, r.type, r.title, r.summary, r.state, r.parent_id,
			COUNT(DISTINCT c.id) as children_count,
			COUNT(DISTINCT CASE WHEN c.state = 'OPEN' THEN c.id END) as open_children_count
		FROM records r
		LEFT JOIN records c ON c.parent_id = r.id AND c.tenant_id = r.tenant_id
		WHERE r.tenant_id = ?
	`

	args := []interface{}{tenantID}
	conditions := []string{}

	if opts.ProjectID != "" {
		conditions = append(conditions, "r.project_id = ?")
		args = append(args, opts.ProjectID)
	}

	if opts.ParentID != nil {
		if *opts.ParentID == "" {
			conditions = append(conditions, "r.parent_id IS NULL")
		} else {
			conditions = append(conditions, "r.parent_id = ?")
			args = append(args, *opts.ParentID)
		}
	}

	if len(opts.States) > 0 {
		placeholders := make([]string, len(opts.States))
		for i, state := range opts.States {
			placeholders[i] = "?"
			args = append(args, state)
		}
		conditions = append(conditions, fmt.Sprintf("r.state IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(opts.Types) > 0 {
		placeholders := make([]string, len(opts.Types))
		for i, typ := range opts.Types {
			placeholders[i] = "?"
			args = append(args, typ)
		}
		conditions = append(conditions, fmt.Sprintf("r.type IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	query += " GROUP BY r.id, r.type, r.title, r.summary, r.state, r.parent_id"
	query += " ORDER BY r.created_at DESC"

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
		return nil, fmt.Errorf("failed to list records: %w", err)
	}
	defer rows.Close()

	var refs []record.RecordRef
	for rows.Next() {
		var ref record.RecordRef
		err := rows.Scan(
			&ref.ID,
			&ref.Type,
			&ref.Title,
			&ref.Summary,
			&ref.State,
			&ref.ParentID,
			&ref.ChildrenCount,
			&ref.OpenChildrenCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan record ref: %w", err)
		}
		refs = append(refs, ref)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating record rows: %w", err)
	}

	return refs, nil
}

// GetChildren returns all child records (full content)
func (r *RecordRepository) GetChildren(ctx context.Context, tenantID, parentID string) ([]record.Record, error) {
	query := `
		SELECT
			id, tenant_id, project_id, type, title, summary, body,
			state, parent_id, resolved_by, created_at, modified_at, tick
		FROM records
		WHERE parent_id = ? AND tenant_id = ?
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, parentID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get children: %w", err)
	}
	defer rows.Close()

	var children []record.Record
	for rows.Next() {
		var rec record.Record
		err := rows.Scan(
			&rec.ID,
			&rec.TenantID,
			&rec.ProjectID,
			&rec.Type,
			&rec.Title,
			&rec.Summary,
			&rec.Body,
			&rec.State,
			&rec.ParentID,
			&rec.ResolvedBy,
			&rec.CreatedAt,
			&rec.ModifiedAt,
			&rec.Tick,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan child record: %w", err)
		}

		// Load related records for each child
		related, err := r.GetRelated(ctx, tenantID, rec.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get related records: %w", err)
		}
		rec.Related = related

		children = append(children, rec)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating children rows: %w", err)
	}

	return children, nil
}

// GetChildrenRefs returns all child records as lightweight references
func (r *RecordRepository) GetChildrenRefs(ctx context.Context, tenantID, parentID string) ([]record.RecordRef, error) {
	query := `
		SELECT
			r.id, r.type, r.title, r.summary, r.state, r.parent_id,
			COUNT(DISTINCT c.id) as children_count,
			COUNT(DISTINCT CASE WHEN c.state = 'OPEN' THEN c.id END) as open_children_count
		FROM records r
		LEFT JOIN records c ON c.parent_id = r.id AND c.tenant_id = r.tenant_id
		WHERE r.parent_id = ? AND r.tenant_id = ?
		GROUP BY r.id, r.type, r.title, r.summary, r.state, r.parent_id
		ORDER BY r.created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, parentID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get children refs: %w", err)
	}
	defer rows.Close()

	var refs []record.RecordRef
	for rows.Next() {
		var ref record.RecordRef
		err := rows.Scan(
			&ref.ID,
			&ref.Type,
			&ref.Title,
			&ref.Summary,
			&ref.State,
			&ref.ParentID,
			&ref.ChildrenCount,
			&ref.OpenChildrenCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan child ref: %w", err)
		}
		refs = append(refs, ref)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating children ref rows: %w", err)
	}

	return refs, nil
}

// GetRelated returns IDs of related records (non-hierarchical relationships)
func (r *RecordRepository) GetRelated(ctx context.Context, tenantID, recordID string) ([]string, error) {
	query := `
		SELECT rr.to_record_id
		FROM record_relations rr
		JOIN records r ON r.id = rr.from_record_id
		WHERE rr.from_record_id = ? AND r.tenant_id = ?
	`

	rows, err := r.db.QueryContext(ctx, query, recordID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get related records: %w", err)
	}
	defer rows.Close()

	var related []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan related record ID: %w", err)
		}
		related = append(related, id)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating related rows: %w", err)
	}

	return related, nil
}

// AddRelation adds a non-hierarchical relation between two records
func (r *RecordRepository) AddRelation(ctx context.Context, fromRecordID, toRecordID string) error {
	query := `
		INSERT INTO record_relations (from_record_id, to_record_id)
		VALUES (?, ?)
	`

	_, err := r.db.ExecContext(ctx, query, fromRecordID, toRecordID)
	if err != nil {
		if isForeignKeyViolation(err) {
			return repository.ErrForeignKeyViolation
		}
		return fmt.Errorf("failed to add relation: %w", err)
	}

	return nil
}
