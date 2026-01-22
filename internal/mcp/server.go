package mcp

import (
	"context"
	"log/slog"

	"github.com/rpggio/trellis/internal/domain/activity"
	"github.com/rpggio/trellis/internal/domain/project"
	"github.com/rpggio/trellis/internal/domain/record"
	"github.com/rpggio/trellis/internal/domain/session"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ProjectService defines project operations needed by MCP.
type ProjectService interface {
	Create(ctx context.Context, tenantID string, req project.CreateRequest) (*project.Project, error)
	List(ctx context.Context, tenantID string) ([]project.ProjectSummary, error)
	Get(ctx context.Context, tenantID, id string) (*project.Project, error)
	GetDefault(ctx context.Context, tenantID string) (*project.Project, error)
}

// RecordService defines record operations needed by MCP.
type RecordService interface {
	Create(ctx context.Context, tenantID string, req record.CreateRequest) (*record.Record, error)
	Update(ctx context.Context, tenantID string, req record.UpdateRequest) (*record.Record, *record.ConflictInfo, error)
	Transition(ctx context.Context, tenantID string, req record.TransitionRequest) (*record.Record, error)
	Get(ctx context.Context, tenantID, id string) (*record.Record, error)
	GetRef(ctx context.Context, tenantID, id string) (record.RecordRef, error)
	List(ctx context.Context, tenantID string, opts record.ListRecordsOptions) ([]record.RecordRef, error)
	Search(ctx context.Context, tenantID, projectID, query string, opts record.SearchOptions) ([]record.SearchResult, error)
}

// SessionService defines session operations needed by MCP.
type SessionService interface {
	Activate(ctx context.Context, tenantID string, req session.ActivateRequest) (*session.ActivateResult, error)
	SyncSession(ctx context.Context, tenantID, sessionID string) (*session.SyncResult, error)
	SaveSession(ctx context.Context, tenantID, sessionID string) error
	CloseSession(ctx context.Context, tenantID, sessionID string) error
	GetActiveSessionsForRecord(ctx context.Context, tenantID, recordID string) ([]session.SessionInfo, error)
	ListActiveSessions(ctx context.Context, tenantID, projectID string) ([]session.SessionInfo, error)
}

// ActivityService defines activity operations needed by MCP.
type ActivityService interface {
	GetRecentActivity(ctx context.Context, tenantID string, opts activity.ListActivityOptions) ([]activity.ActivityEntry, error)
}

// Services contains all domain services needed by MCP.
type Services struct {
	Projects ProjectService
	Records  RecordService
	Sessions SessionService
	Activity ActivityService
}

// Config contains server configuration.
type Config struct {
	Services      Services
	Resolver      TenantResolver
	AuthEnabled   bool
	TransportMode string // "stdio" or "http"
	Logger        *slog.Logger
}

// NewServer creates and configures an MCP server with all tools and middleware.
func NewServer(cfg Config) *sdkmcp.Server {
	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    "trellis",
		Version: "0.1.0",
	}, &sdkmcp.ServerOptions{
		Instructions: serverInstructions,
		Logger:       cfg.Logger,
	})

	registerDocResources(server)

	// Add middleware (auth + session extraction)
	// Stdio mode: always disable auth (local dev only)
	if cfg.TransportMode == "stdio" {
		server.AddReceivingMiddleware(noAuthMiddleware("default"))
	} else {
		// HTTP mode: auth based on config
		if cfg.AuthEnabled {
			server.AddReceivingMiddleware(authMiddleware(cfg.Resolver))
		} else {
			server.AddReceivingMiddleware(noAuthMiddleware("default"))
		}
	}
	server.AddReceivingMiddleware(sessionMiddleware())
	server.AddReceivingMiddleware(trafficLoggingMiddleware(cfg.Logger, "inbound"))
	server.AddSendingMiddleware(trafficLoggingMiddleware(cfg.Logger, "outbound"))

	// Register all tools
	registerTools(server, cfg.Services)

	return server
}
