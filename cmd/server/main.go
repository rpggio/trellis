package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ganot/threds-mcp/internal/config"
	"github.com/ganot/threds-mcp/internal/domain/activity"
	"github.com/ganot/threds-mcp/internal/domain/project"
	"github.com/ganot/threds-mcp/internal/domain/record"
	"github.com/ganot/threds-mcp/internal/domain/session"
	"github.com/ganot/threds-mcp/internal/mcp"
	"github.com/ganot/threds-mcp/internal/sqlite"
	"github.com/ganot/threds-mcp/internal/transport"
	"github.com/ganot/threds-mcp/migrations"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.Log.Level),
	}))

	if err := ensureDBDir(cfg.DB.Path); err != nil {
		logger.Error("failed to prepare database path", "error", err)
		os.Exit(1)
	}

	db, err := sqlite.New(cfg.DB.Path)
	if err != nil {
		logger.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := runEmbeddedMigrations(db); err != nil {
		logger.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	projectRepo := sqlite.NewProjectRepository(db)
	recordRepo := sqlite.NewRecordRepository(db)
	sessionRepo := sqlite.NewSessionRepository(db)
	activityRepo := sqlite.NewActivityRepository(db)
	searchRepo := sqlite.NewSearchRepository(db)

	projectSvc := project.NewService(projectRepo, logger)
	activitySvc := activity.NewService(activityRepo, logger)
	recordSvc := record.NewService(recordRepo, sessionRepo, projectRepo, activityRepo, searchRepo, logger)
	sessionSvc := session.NewService(recordRepo, sessionRepo, projectRepo, logger)

	handler := mcp.NewHandler(projectSvc, recordSvc, sessionSvc, activitySvc)
	resolver := &apiKeyResolver{db: db}

	router := transport.NewServer(handler, transport.AuthMiddleware(resolver))

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		logger.Info("server listening", "addr", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
		}
	}()

	waitForShutdown(logger, httpServer)
}

func runEmbeddedMigrations(db *sqlite.DB) error {
	data, err := migrations.FS.ReadFile("001_initial_schema.up.sql")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	if _, err := db.Exec(string(data)); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

func ensureDBDir(path string) error {
	if path == ":memory:" || path == "" {
		return nil
	}
	dir := filepath.Dir(path)
	if dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func waitForShutdown(logger *slog.Logger, server *http.Server) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger.Info("shutting down")
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", "error", err)
	}
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
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
