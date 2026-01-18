package sqlite

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// NewTestDB creates a new in-memory SQLite database for testing
func NewTestDB(t *testing.T) *DB {
	t.Helper()

	db, err := New(":memory:")
	require.NoError(t, err, "failed to create test database")

	err = db.RunMigrations()
	require.NoError(t, err, "failed to run migrations")

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

// TestMigrations verifies that migrations run successfully
func TestMigrations(t *testing.T) {
	db := NewTestDB(t)

	// Verify all tables were created
	tables := []string{
		"projects",
		"records",
		"record_relations",
		"sessions",
		"session_activations",
		"activity_log",
		"records_fts",
		"api_keys",
	}

	for _, table := range tables {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		require.NoError(t, err, "failed to query table %s", table)
		require.Equal(t, 1, count, "table %s not found", table)
	}
}

// TestForeignKeys verifies that foreign key constraints are enabled
func TestForeignKeys(t *testing.T) {
	db := NewTestDB(t)

	var enabled int
	err := db.QueryRow("PRAGMA foreign_keys").Scan(&enabled)
	require.NoError(t, err)
	require.Equal(t, 1, enabled, "foreign keys not enabled")
}

// TestProjectsTable verifies the projects table structure
func TestProjectsTable(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()

	// Insert a project
	_, err := db.ExecContext(ctx,
		`INSERT INTO projects (id, tenant_id, name, description, tick) VALUES (?, ?, ?, ?, ?)`,
		"p1", "tenant1", "Test Project", "A test project", 0)
	require.NoError(t, err)

	// Query it back
	var id, tenantID, name, description string
	var tick int64
	err = db.QueryRowContext(ctx,
		`SELECT id, tenant_id, name, description, tick FROM projects WHERE id = ?`,
		"p1").Scan(&id, &tenantID, &name, &description, &tick)
	require.NoError(t, err)
	require.Equal(t, "p1", id)
	require.Equal(t, "tenant1", tenantID)
	require.Equal(t, "Test Project", name)
	require.Equal(t, "A test project", description)
	require.Equal(t, int64(0), tick)
}

// TestRecordsTable verifies the records table structure and constraints
func TestRecordsTable(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()

	// Create a project first
	_, err := db.ExecContext(ctx,
		`INSERT INTO projects (id, tenant_id, name, tick) VALUES (?, ?, ?, ?)`,
		"p1", "tenant1", "Test Project", 0)
	require.NoError(t, err)

	// Insert a root record
	_, err = db.ExecContext(ctx,
		`INSERT INTO records (id, tenant_id, project_id, type, title, summary, body, state, tick)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"r1", "tenant1", "p1", "question", "Test Question", "Summary", "Body", "OPEN", 1)
	require.NoError(t, err)

	// Insert a child record
	_, err = db.ExecContext(ctx,
		`INSERT INTO records (id, tenant_id, project_id, type, title, summary, body, state, parent_id, tick)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"r2", "tenant1", "p1", "note", "Child Note", "Summary", "Body", "OPEN", "r1", 2)
	require.NoError(t, err)

	// Test foreign key constraint - should fail with invalid project_id
	_, err = db.ExecContext(ctx,
		`INSERT INTO records (id, tenant_id, project_id, type, title, summary, body, state, tick)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"r3", "tenant1", "invalid", "question", "Test", "Summary", "Body", "OPEN", 3)
	require.Error(t, err, "should fail with invalid project_id")

	// Test state constraint - should fail with invalid state
	_, err = db.ExecContext(ctx,
		`INSERT INTO records (id, tenant_id, project_id, type, title, summary, body, state, tick)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"r4", "tenant1", "p1", "question", "Test", "Summary", "Body", "INVALID", 4)
	require.Error(t, err, "should fail with invalid state")
}

// TestSessionsTable verifies the sessions table structure
func TestSessionsTable(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()

	// Create a project
	_, err := db.ExecContext(ctx,
		`INSERT INTO projects (id, tenant_id, name, tick) VALUES (?, ?, ?, ?)`,
		"p1", "tenant1", "Test Project", 0)
	require.NoError(t, err)

	// Insert a session
	_, err = db.ExecContext(ctx,
		`INSERT INTO sessions (id, tenant_id, project_id, status, last_sync_tick)
		 VALUES (?, ?, ?, ?, ?)`,
		"s1", "tenant1", "p1", "active", 0)
	require.NoError(t, err)

	// Query it back
	var id, tenantID, projectID, status string
	var lastSyncTick int64
	err = db.QueryRowContext(ctx,
		`SELECT id, tenant_id, project_id, status, last_sync_tick FROM sessions WHERE id = ?`,
		"s1").Scan(&id, &tenantID, &projectID, &status, &lastSyncTick)
	require.NoError(t, err)
	require.Equal(t, "s1", id)
	require.Equal(t, "tenant1", tenantID)
	require.Equal(t, "p1", projectID)
	require.Equal(t, "active", status)
}

// TestSessionActivations verifies the session_activations join table
func TestSessionActivations(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()

	// Create project, record, and session
	_, err := db.ExecContext(ctx,
		`INSERT INTO projects (id, tenant_id, name, tick) VALUES (?, ?, ?, ?)`,
		"p1", "tenant1", "Test Project", 0)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx,
		`INSERT INTO records (id, tenant_id, project_id, type, title, summary, body, state, tick)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"r1", "tenant1", "p1", "question", "Test", "Summary", "Body", "OPEN", 1)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx,
		`INSERT INTO sessions (id, tenant_id, project_id, status, last_sync_tick)
		 VALUES (?, ?, ?, ?, ?)`,
		"s1", "tenant1", "p1", "active", 0)
	require.NoError(t, err)

	// Insert activation
	_, err = db.ExecContext(ctx,
		`INSERT INTO session_activations (session_id, record_id, activation_tick)
		 VALUES (?, ?, ?)`,
		"s1", "r1", 1)
	require.NoError(t, err)

	// Query it back
	var sessionID, recordID string
	var activationTick int64
	err = db.QueryRowContext(ctx,
		`SELECT session_id, record_id, activation_tick FROM session_activations
		 WHERE session_id = ? AND record_id = ?`,
		"s1", "r1").Scan(&sessionID, &recordID, &activationTick)
	require.NoError(t, err)
	require.Equal(t, "s1", sessionID)
	require.Equal(t, "r1", recordID)
	require.Equal(t, int64(1), activationTick)
}

// TestFTSIndex verifies the full-text search index is synchronized
func TestFTSIndex(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()

	// Create project and record
	_, err := db.ExecContext(ctx,
		`INSERT INTO projects (id, tenant_id, name, tick) VALUES (?, ?, ?, ?)`,
		"p1", "tenant1", "Test Project", 0)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx,
		`INSERT INTO records (id, tenant_id, project_id, type, title, summary, body, state, tick)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"r1", "tenant1", "p1", "question", "Unique Question Title", "This is a summary", "Full body text here", "OPEN", 1)
	require.NoError(t, err)

	// Search the FTS index - verify the trigger populated it
	var count int
	err = db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM records_fts WHERE records_fts MATCH ?`,
		"unique").Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count, "should find 1 record matching 'unique'")

	// Update record and verify FTS is updated
	_, err = db.ExecContext(ctx,
		`UPDATE records SET title = ? WHERE id = ?`,
		"Updated Title", "r1")
	require.NoError(t, err)

	err = db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM records_fts WHERE records_fts MATCH ?`,
		"updated").Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count, "should find 1 record matching 'updated' after update")

	// Verify old title is no longer in index
	err = db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM records_fts WHERE records_fts MATCH ?`,
		"unique").Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count, "should find 0 records matching 'unique' after update")
}
