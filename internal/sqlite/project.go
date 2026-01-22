package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rpggio/trellis/internal/domain/project"
	"github.com/rpggio/trellis/internal/repository"
)

// ProjectRepository implements repository.ProjectRepository for SQLite
type ProjectRepository struct {
	db *DB
}

// NewProjectRepository creates a new ProjectRepository
func NewProjectRepository(db *DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// Create creates a new project
func (r *ProjectRepository) Create(ctx context.Context, tenantID string, proj *project.Project) error {
	query := `
		INSERT INTO projects (id, tenant_id, name, description, tick, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		proj.ID,
		tenantID,
		proj.Name,
		proj.Description,
		proj.Tick,
		proj.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	return nil
}

// Get retrieves a project by ID
func (r *ProjectRepository) Get(ctx context.Context, tenantID, id string) (*project.Project, error) {
	query := `
		SELECT id, tenant_id, name, description, tick, created_at
		FROM projects
		WHERE id = ? AND tenant_id = ?
	`

	var proj project.Project
	err := r.db.QueryRowContext(ctx, query, id, tenantID).Scan(
		&proj.ID,
		&proj.TenantID,
		&proj.Name,
		&proj.Description,
		&proj.Tick,
		&proj.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return &proj, nil
}

// GetDefault retrieves the default project for a tenant (the first created project)
func (r *ProjectRepository) GetDefault(ctx context.Context, tenantID string) (*project.Project, error) {
	query := `
		SELECT id, tenant_id, name, description, tick, created_at
		FROM projects
		WHERE tenant_id = ?
		ORDER BY created_at ASC
		LIMIT 1
	`

	var proj project.Project
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&proj.ID,
		&proj.TenantID,
		&proj.Name,
		&proj.Description,
		&proj.Tick,
		&proj.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get default project: %w", err)
	}

	return &proj, nil
}

// List returns all projects for a tenant with summary information
func (r *ProjectRepository) List(ctx context.Context, tenantID string) ([]project.ProjectSummary, error) {
	query := `
		SELECT
			p.id,
			p.name,
			p.description,
			p.tick,
			p.created_at,
			COUNT(DISTINCT r.id) as record_count,
			COUNT(DISTINCT CASE WHEN r.state = 'OPEN' THEN r.id END) as open_records,
			COUNT(DISTINCT s.id) as active_sessions
		FROM projects p
		LEFT JOIN records r ON r.project_id = p.id AND r.tenant_id = p.tenant_id
		LEFT JOIN sessions s ON s.project_id = p.id AND s.tenant_id = p.tenant_id AND s.status = 'active'
		WHERE p.tenant_id = ?
		GROUP BY p.id, p.name, p.description, p.tick, p.created_at
		ORDER BY p.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	var summaries []project.ProjectSummary
	for rows.Next() {
		var summary project.ProjectSummary
		err := rows.Scan(
			&summary.ID,
			&summary.Name,
			&summary.Description,
			&summary.Tick,
			&summary.CreatedAt,
			&summary.RecordCount,
			&summary.OpenRecords,
			&summary.ActiveSessions,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project summary: %w", err)
		}
		summaries = append(summaries, summary)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating project rows: %w", err)
	}

	return summaries, nil
}

// IncrementTick atomically increments the project tick and returns the new value
func (r *ProjectRepository) IncrementTick(ctx context.Context, tenantID, projectID string) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update the tick
	updateQuery := `
		UPDATE projects
		SET tick = tick + 1
		WHERE id = ? AND tenant_id = ?
	`

	result, err := tx.ExecContext(ctx, updateQuery, projectID, tenantID)
	if err != nil {
		return 0, fmt.Errorf("failed to increment tick: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return 0, repository.ErrNotFound
	}

	// Get the new tick value
	selectQuery := `
		SELECT tick
		FROM projects
		WHERE id = ? AND tenant_id = ?
	`

	var newTick int64
	err = tx.QueryRowContext(ctx, selectQuery, projectID, tenantID).Scan(&newTick)
	if err != nil {
		return 0, fmt.Errorf("failed to get new tick: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return newTick, nil
}
