package sqlite

import (
	"context"
	"fmt"
	"strings"

	"github.com/rpggio/trellis/internal/domain/record"
)

// SearchRepository implements repository.SearchRepository for SQLite
type SearchRepository struct {
	db *DB
}

// NewSearchRepository creates a new SearchRepository
func NewSearchRepository(db *DB) *SearchRepository {
	return &SearchRepository{db: db}
}

// Search performs a full-text search over records
func (r *SearchRepository) Search(ctx context.Context, tenantID, projectID, query string, opts record.SearchOptions) ([]record.SearchResult, error) {
	baseQuery := `
		SELECT
			r.id, r.type, r.title, r.summary, r.state, r.parent_id,
			(SELECT COUNT(*) FROM records c WHERE c.parent_id = r.id AND c.tenant_id = r.tenant_id) as children_count,
			(SELECT COUNT(*) FROM records c WHERE c.parent_id = r.id AND c.tenant_id = r.tenant_id AND c.state = 'OPEN') as open_children_count,
			0.0 as rank,
			'' as snippet
		FROM records_fts
		JOIN records r ON r.rowid = records_fts.rowid
		WHERE r.tenant_id = ? AND r.project_id = ? AND records_fts MATCH ?
	`

	args := []interface{}{tenantID, projectID, query}
	conditions := []string{}

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
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	baseQuery += " GROUP BY r.id, r.type, r.title, r.summary, r.state, r.parent_id"
	baseQuery += " ORDER BY rank"

	if opts.Limit > 0 {
		baseQuery += " LIMIT ?"
		args = append(args, opts.Limit)
	}
	if opts.Offset > 0 {
		baseQuery += " OFFSET ?"
		args = append(args, opts.Offset)
	}

	rows, err := r.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search records: %w", err)
	}
	defer rows.Close()

	var results []record.SearchResult
	for rows.Next() {
		var result record.SearchResult
		err := rows.Scan(
			&result.Record.ID,
			&result.Record.Type,
			&result.Record.Title,
			&result.Record.Summary,
			&result.Record.State,
			&result.Record.ParentID,
			&result.Record.ChildrenCount,
			&result.Record.OpenChildrenCount,
			&result.Rank,
			&result.Snippet,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search results: %w", err)
	}

	return results, nil
}
