package testserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
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
	"github.com/stretchr/testify/require"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type TransportMode string

const (
	TransportHTTP  TransportMode = "http"
	TransportStdio TransportMode = "stdio"
)

type TestServer struct {
	Server    *httptest.Server // For HTTP transport only
	DB        *sqlite.DB
	Token     string
	TenantID  string
	Mode      TransportMode
	mcpServer *sdkmcp.Server // For stdio testing
}

func New(t *testing.T, token, tenantID string) *TestServer {
	return NewWithTransport(t, token, tenantID, TransportHTTP)
}

func NewWithTransport(t *testing.T, token, tenantID string, mode TransportMode) *TestServer {
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

	// Create MCP server with SDK
	resolver := &apiKeyResolver{db: db}
	authEnabled := mode == TransportHTTP // Auth only for HTTP
	mcpServer := mcp.NewServer(mcp.Config{
		Services: mcp.Services{
			Projects: projectSvc,
			Records:  recordSvc,
			Sessions: sessionSvc,
			Activity: activitySvc,
		},
		Resolver:      resolver,
		AuthEnabled:   authEnabled,
		TransportMode: string(mode),
		Logger:        nil,
	})

	ts := &TestServer{
		DB:        db,
		Token:     token,
		TenantID:  tenantID,
		Mode:      mode,
		mcpServer: mcpServer,
	}

	if mode == TransportHTTP {
		// HTTP mode: create httptest.Server
		mcpHandler := sdkmcp.NewStreamableHTTPHandler(
			func(r *http.Request) *sdkmcp.Server { return mcpServer },
			&sdkmcp.StreamableHTTPOptions{
				Stateless:    true, // Stateless mode for simpler JSON-RPC interactions
				JSONResponse: true,  // Use JSON instead of streaming
			},
		)

		server := httptest.NewServer(mcpHandler)
		ts.Server = server

		require.NoError(t, ts.AddAPIKey(token, tenantID))

		t.Cleanup(func() {
			server.Close()
			_ = db.Close()
		})
	} else {
		// Stdio mode: no HTTP server needed
		t.Cleanup(func() { _ = db.Close() })
	}

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
		return "", fmt.Errorf("unauthorized: invalid token")
	}
	return tenantID, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
