package testserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ganot/threds-mcp/internal/domain/activity"
	"github.com/ganot/threds-mcp/internal/domain/project"
	"github.com/ganot/threds-mcp/internal/domain/record"
	"github.com/ganot/threds-mcp/internal/domain/session"
	"github.com/ganot/threds-mcp/internal/mcp"
	"github.com/ganot/threds-mcp/internal/sqlite"
	"github.com/ganot/threds-mcp/internal/transport"
	"github.com/stretchr/testify/require"
)

type TestServer struct {
	Server   *httptest.Server
	DB       *sqlite.DB
	Token    string
	TenantID string
}

func New(t *testing.T, token, tenantID string) *TestServer {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := sqlite.New(dsn)
	require.NoError(t, err)
	require.NoError(t, db.RunMigrations())

	projectRepo := sqlite.NewProjectRepository(db)
	recordRepo := sqlite.NewRecordRepository(db)
	sessionRepo := sqlite.NewSessionRepository(db)
	activityRepo := sqlite.NewActivityRepository(db)
	searchRepo := sqlite.NewSearchRepository(db)

	projectSvc := project.NewService(projectRepo, nil)
	activitySvc := activity.NewService(activityRepo, nil)
	recordSvc := record.NewService(recordRepo, sessionRepo, projectRepo, activityRepo, searchRepo, nil)
	sessionSvc := session.NewService(recordRepo, sessionRepo, projectRepo, nil)

	handler := mcp.NewHandler(projectSvc, recordSvc, sessionSvc, activitySvc)

	resolver := &apiKeyResolver{db: db}
	server := httptest.NewServer(transport.NewServer(handler, transport.AuthMiddleware(resolver)))

	ts := &TestServer{
		Server:   server,
		DB:       db,
		Token:    token,
		TenantID: tenantID,
	}

	require.NoError(t, ts.AddAPIKey(token, tenantID))

	t.Cleanup(func() {
		server.Close()
		_ = db.Close()
	})

	return ts
}

func (ts *TestServer) AddAPIKey(token, tenantID string) error {
	hash := hashToken(token)
	_, err := ts.DB.Exec(
		`INSERT INTO api_keys (key_hash, tenant_id, created_at) VALUES (?, ?, ?)`,
		hash, tenantID, time.Now(),
	)
	return err
}

type apiKeyResolver struct {
	db *sqlite.DB
}

func (r *apiKeyResolver) ResolveTenant(ctx context.Context, token string) (string, error) {
	hash := hashToken(token)
	var tenantID string
	err := r.db.QueryRowContext(ctx, `SELECT tenant_id FROM api_keys WHERE key_hash = ?`, hash).Scan(&tenantID)
	if err != nil || tenantID == "" {
		return "", transport.ErrUnauthorized
	}
	return tenantID, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
