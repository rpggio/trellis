package functional_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

// stdioSession wraps an MCP client session for stdio transport testing
type stdioSession struct {
	session *sdkmcp.ClientSession
	cancel  context.CancelFunc
}

func newStdioSession(t *testing.T) *stdioSession {
	t.Helper()
	return newStdioSessionWithEnv(t, nil)
}

func newStdioSessionWithEnv(t *testing.T, extraEnv []string) *stdioSession {
	t.Helper()

	// Find the binary
	binaryPath := "./bin/trellis"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		binaryPath = "../../bin/trellis"
		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			t.Skip("Server binary not found. Run 'make build' first.")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	cmd := exec.CommandContext(ctx, binaryPath)
	cmd.Env = append(os.Environ(),
		"TRELLIS_TRANSPORT=stdio",
		"TRELLIS_DB_PATH=:memory:",
		"TRELLIS_AUTH_ENABLED=false",
	)
	if len(extraEnv) > 0 {
		cmd.Env = append(cmd.Env, extraEnv...)
	}

	transport := &sdkmcp.CommandTransport{Command: cmd}

	client := sdkmcp.NewClient(&sdkmcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		cancel()
		t.Fatalf("Failed to connect: %v", err)
	}

	t.Cleanup(func() {
		session.Close()
		cancel()
	})

	return &stdioSession{session: session, cancel: cancel}
}

func (s *stdioSession) callTool(t *testing.T, name string, args map[string]any) json.RawMessage {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := s.session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	require.NoError(t, err, "CallTool %s failed", name)
	require.False(t, result.IsError, "Tool %s returned error", name)
	require.NotEmpty(t, result.Content, "Tool %s returned no content", name)

	// Extract text content
	for _, content := range result.Content {
		if textContent, ok := content.(*sdkmcp.TextContent); ok {
			return json.RawMessage(textContent.Text)
		}
	}
	t.Fatalf("Tool %s returned no text content", name)
	return nil
}

func TestStdioFunctional_ProjectAndOverview(t *testing.T) {
	s := newStdioSession(t)

	create := s.callTool(t, "create_project", map[string]any{"name": "Project"})
	require.NotEmpty(t, create)

	list := s.callTool(t, "list_projects", nil)
	require.NotEmpty(t, list)

	get := s.callTool(t, "get_project", map[string]any{})
	require.NotEmpty(t, get)

	overview := s.callTool(t, "get_project_overview", map[string]any{})
	require.NotEmpty(t, overview)
}

func TestStdioFunctional_ActivationWorkflow(t *testing.T) {
	s := newStdioSession(t)

	projectResp := s.callTool(t, "create_project", map[string]any{"name": "Project"})
	var project struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(projectResp, &project))

	rootResp := s.callTool(t, "create_record", map[string]any{
		"type":    "question",
		"title":   "Root",
		"summary": "Root summary",
		"body":    "Root body",
	})
	var root struct {
		Record struct {
			ID string `json:"id"`
		} `json:"record"`
	}
	require.NoError(t, json.Unmarshal(rootResp, &root))

	// Test activation returns session info
	activateResp := s.callTool(t, "activate", map[string]any{"id": root.Record.ID})
	var activation struct {
		SessionID string `json:"session_id"`
		Context   struct {
			Target struct {
				ID string `json:"id"`
			} `json:"target"`
		} `json:"context"`
	}
	require.NoError(t, json.Unmarshal(activateResp, &activation))
	require.NotEmpty(t, activation.SessionID)
	require.Equal(t, root.Record.ID, activation.Context.Target.ID)

	overviewResp := s.callTool(t, "get_project_overview", map[string]any{})
	var overview struct {
		OpenSessions []struct {
			ID            string   `json:"id"`
			ActiveRecords []string `json:"active_records"`
		} `json:"open_sessions"`
	}
	require.NoError(t, json.Unmarshal(overviewResp, &overview))
	require.NotEmpty(t, overview.OpenSessions)

	foundSession := false
	for _, sess := range overview.OpenSessions {
		if sess.ID == activation.SessionID {
			foundSession = true
			require.Contains(t, sess.ActiveRecords, root.Record.ID)
		}
	}
	require.True(t, foundSession)

	// Test session management tools that accept session_id as argument
	saveResp := s.callTool(t, "save_session", map[string]any{
		"session_id": activation.SessionID,
	})
	require.NotEmpty(t, saveResp)

	closeResp := s.callTool(t, "close_session", map[string]any{
		"session_id": activation.SessionID,
	})
	require.NotEmpty(t, closeResp)

	// Note: Full session workflow with record updates is tested via HTTP transport
	// where session_id can be passed via Mcp-Session-Id header.
	// The stdio SDK client doesn't support setting _meta for session context.
}

func TestStdioFunctional_SearchAndList(t *testing.T) {
	s := newStdioSession(t)

	_ = s.callTool(t, "create_project", map[string]any{"name": "Search Project"})

	rootResp := s.callTool(t, "create_record", map[string]any{
		"type":    "question",
		"title":   "Searchable Record",
		"summary": "This is a searchable summary",
		"body":    "Body with unique content for searching",
	})
	var root struct {
		Record struct {
			ID string `json:"id"`
		} `json:"record"`
	}
	require.NoError(t, json.Unmarshal(rootResp, &root))

	listResp := s.callTool(t, "list_records", map[string]any{})
	require.NotEmpty(t, listResp)

	searchResp := s.callTool(t, "search_records", map[string]any{"query": "Searchable"})
	require.NotEmpty(t, searchResp)
	require.Contains(t, string(searchResp), root.Record.ID)
}

func TestStdioFunctional_MCPProtocolCompliance(t *testing.T) {
	s := newStdioSession(t)

	// Verify server info from initialization
	initResult := s.session.InitializeResult()
	require.NotNil(t, initResult)
	require.NotNil(t, initResult.ServerInfo)
	require.Equal(t, "trellis", initResult.ServerInfo.Name)
	require.Equal(t, "0.1.0", initResult.ServerInfo.Version)

	// Test tools/list
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tools, err := s.session.ListTools(ctx, nil)
	require.NoError(t, err)
	require.Greater(t, len(tools.Tools), 15, "should have at least 16 tools")

	// Verify expected tools exist with proper metadata
	toolMap := make(map[string]*sdkmcp.Tool)
	for _, tool := range tools.Tools {
		toolMap[tool.Name] = tool
	}

	require.Contains(t, toolMap, "create_project")
	require.Contains(t, toolMap, "activate")
	require.Contains(t, toolMap, "create_record")
	require.NotEmpty(t, toolMap["create_project"].Description)
}

func TestStdioFunctional_LogFile(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "trellis.log")
	s := newStdioSessionWithEnv(t, []string{
		"TRELLIS_LOG_PATH=" + logPath,
		"TRELLIS_LOG_LEVEL=debug",
	})

	_ = s.callTool(t, "list_projects", nil)

	require.Eventually(t, func() bool {
		data, err := os.ReadFile(logPath)
		if err != nil {
			return false
		}
		text := string(data)
		return strings.Contains(text, `msg="mcp traffic"`) &&
			strings.Contains(text, "stage=request") &&
			strings.Contains(text, "stage=response")
	}, 5*time.Second, 100*time.Millisecond)
}

func TestStdioFunctional_HistoryAndActivity(t *testing.T) {
	s := newStdioSession(t)

	_ = s.callTool(t, "create_project", map[string]any{"name": "History Project"})

	rootResp := s.callTool(t, "create_record", map[string]any{
		"type":    "question",
		"title":   "Root",
		"summary": "Root summary",
		"body":    "Root body",
	})
	var root struct {
		Record struct {
			ID string `json:"id"`
		} `json:"record"`
	}
	require.NoError(t, json.Unmarshal(rootResp, &root))

	activateResp := s.callTool(t, "activate", map[string]any{"id": root.Record.ID})
	var activation struct {
		SessionID string `json:"session_id"`
	}
	require.NoError(t, json.Unmarshal(activateResp, &activation))

	_ = s.callTool(t, "update_record", map[string]any{
		"id":         root.Record.ID,
		"title":      "Updated Title",
		"session_id": activation.SessionID,
	})

	history := s.callTool(t, "get_record_history", map[string]any{"id": root.Record.ID})
	require.NotEmpty(t, history)

	diff := s.callTool(t, "get_record_diff", map[string]any{
		"id":   root.Record.ID,
		"from": "last_save",
	})
	require.NotEmpty(t, diff)

	activity := s.callTool(t, "get_recent_activity", map[string]any{})
	require.NotEmpty(t, activity)
}

func TestStdioFunctional_DocumentationResources(t *testing.T) {
	s := newStdioSession(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resources, err := s.session.ListResources(ctx, nil)
	require.NoError(t, err)
	require.NotEmpty(t, resources.Resources)

	uris := make(map[string]*sdkmcp.Resource, len(resources.Resources))
	for _, r := range resources.Resources {
		uris[r.URI] = r
	}

	expected := []string{
		"trellis://docs/index",
		"trellis://docs/concepts",
		"trellis://docs/workflows/cold-start",
		"trellis://docs/workflows/activation-and-writing",
		"trellis://docs/workflows/conflicts",
		"trellis://docs/record-writing",
	}
	for _, uri := range expected {
		r, ok := uris[uri]
		require.True(t, ok, "missing expected doc resource: %s", uri)
		require.NotEmpty(t, r.Name)
		require.Equal(t, "text/markdown", r.MIMEType)
		require.Greater(t, r.Size, int64(0))
	}

	read, err := s.session.ReadResource(ctx, &sdkmcp.ReadResourceParams{URI: "trellis://docs/index"})
	require.NoError(t, err)
	require.NotEmpty(t, read.Contents)
	require.Equal(t, "trellis://docs/index", read.Contents[0].URI)
	require.Equal(t, "text/markdown", read.Contents[0].MIMEType)
	require.Contains(t, read.Contents[0].Text, "Agent Docs Index")
}
