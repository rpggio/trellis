package functional_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/ganot/threds-mcp/internal/testserver"
	"github.com/stretchr/testify/require"
)

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
	ID      any             `json:"id,omitempty"`
}

type rpcError struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data,omitempty"`
}

func rpcCall(t *testing.T, ts *testserver.TestServer, sessionID, method string, params any) rpcResponse {
	t.Helper()

	payload := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"id":      1,
	}
	if params != nil {
		payload["params"] = params
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, ts.Server.URL+"/mcp", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+ts.Token)
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(bodyBytes))
	}

	var result rpcResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	return result
}

// initializeSession performs the MCP initialize handshake
func initializeSession(t *testing.T, ts *testserver.TestServer) {
	t.Helper()

	resp := rpcCall(t, ts, "", "initialize", map[string]any{
		"protocolVersion": "2025-11-25",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "test-client",
			"version": "1.0.0",
		},
	})
	require.Nil(t, resp.Error, "Initialize failed: %v", resp.Error)
}

// callTool makes a tools/call RPC call and unwraps the result
func callTool(t *testing.T, ts *testserver.TestServer, sessionID, toolName string, args any) json.RawMessage {
	t.Helper()

	params := map[string]any{
		"name": toolName,
	}
	if args != nil {
		params["arguments"] = args
	}

	resp := rpcCall(t, ts, sessionID, "tools/call", params)
	require.Nil(t, resp.Error, "RPC error: %v", resp.Error)

	// Parse the ToolCallResult
	var toolResult struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	require.NoError(t, json.Unmarshal(resp.Result, &toolResult))
	require.False(t, toolResult.IsError, "Tool error: %s", toolResult.Content[0].Text)
	require.NotEmpty(t, toolResult.Content)

	// Extract the JSON from the text content
	return json.RawMessage(toolResult.Content[0].Text)
}

func TestFunctional_Authentication(t *testing.T) {
	ts := testserver.New(t, "token", "tenant1")

	// Test without authorization header
	req, err := http.NewRequest(http.MethodPost, ts.Server.URL+"/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","method":"tools/call","params":{"name":"list_projects"},"id":1}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestFunctional_ProjectAndOverview(t *testing.T) {
	ts := testserver.New(t, "token", "tenant1")
	initializeSession(t, ts)

	create := callTool(t, ts, "", "create_project", map[string]any{
		"name": "Project",
	})
	require.NotEmpty(t, create)

	list := callTool(t, ts, "", "list_projects", nil)
	require.NotEmpty(t, list)

	get := callTool(t, ts, "", "get_project", map[string]any{})
	require.NotEmpty(t, get)

	overview := callTool(t, ts, "", "get_project_overview", map[string]any{})
	require.NotEmpty(t, overview)
}

func TestFunctional_ActivationWorkflow(t *testing.T) {
	ts := testserver.New(t, "token", "tenant1")
	initializeSession(t, ts)

	projectResp := callTool(t, ts, "", "create_project", map[string]any{"name": "Project"})
	var project struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(projectResp, &project))

	rootResp := callTool(t, ts, "", "create_record", map[string]any{
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

	activateResp := callTool(t, ts, "", "activate", map[string]any{"id": root.Record.ID})
	var activation struct {
		SessionID string `json:"session_id"`
	}
	require.NoError(t, json.Unmarshal(activateResp, &activation))

	childResp := callTool(t, ts, activation.SessionID, "create_record", map[string]any{
		"parent_id": root.Record.ID,
		"type":      "note",
		"title":     "Child",
		"summary":   "Child summary",
		"body":      "Child body",
	})
	require.NotEmpty(t, childResp)

	newTitle := "Updated"
	updateResp := callTool(t, ts, activation.SessionID, "update_record", map[string]any{
		"id":    root.Record.ID,
		"title": newTitle,
	})
	require.NotEmpty(t, updateResp)

	transitionResp := callTool(t, ts, activation.SessionID, "transition", map[string]any{
		"id":       root.Record.ID,
		"to_state": "LATER",
		"reason":   "wait",
	})
	require.NotEmpty(t, transitionResp)

	saveResp := callTool(t, ts, activation.SessionID, "save_session", nil)
	require.NotEmpty(t, saveResp)

	closeResp := callTool(t, ts, activation.SessionID, "close_session", nil)
	require.NotEmpty(t, closeResp)

	_ = project.ID
}

func TestFunctional_ConflictAndActiveSessions(t *testing.T) {
	ts := testserver.New(t, "token", "tenant1")
	initializeSession(t, ts)

	rootResp := callTool(t, ts, "", "create_record", map[string]any{
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

	s1 := callTool(t, ts, "", "activate", map[string]any{"id": root.Record.ID})
	var sess1 struct {
		SessionID string `json:"session_id"`
	}
	require.NoError(t, json.Unmarshal(s1, &sess1))

	s2 := callTool(t, ts, "", "activate", map[string]any{"id": root.Record.ID})
	var sess2 struct {
		SessionID string `json:"session_id"`
	}
	require.NoError(t, json.Unmarshal(s2, &sess2))

	update1 := callTool(t, ts, sess1.SessionID, "update_record", map[string]any{"id": root.Record.ID, "title": "A"})
	require.NotEmpty(t, update1)

	update2 := callTool(t, ts, sess2.SessionID, "update_record", map[string]any{"id": root.Record.ID, "title": "B"})
	require.Contains(t, string(update2), "conflict")

	activeSessions := callTool(t, ts, sess1.SessionID, "get_active_sessions", map[string]any{"record_id": root.Record.ID})
	require.NotEmpty(t, activeSessions)
}

func TestFunctional_ContextLoadingAndSearch(t *testing.T) {
	ts := testserver.New(t, "token", "tenant1")
	initializeSession(t, ts)

	parentResp := callTool(t, ts, "", "create_record", map[string]any{
		"type":    "question",
		"title":   "Parent",
		"summary": "Parent summary",
		"body":    "Parent body",
	})
	var parent struct {
		Record struct {
			ID string `json:"id"`
		} `json:"record"`
	}
	require.NoError(t, json.Unmarshal(parentResp, &parent))

	parentActivate := callTool(t, ts, "", "activate", map[string]any{"id": parent.Record.ID})
	var parentSess struct {
		SessionID string `json:"session_id"`
	}
	require.NoError(t, json.Unmarshal(parentActivate, &parentSess))

	targetResp := callTool(t, ts, parentSess.SessionID, "create_record", map[string]any{
		"parent_id": parent.Record.ID,
		"type":      "question",
		"title":     "Target",
		"summary":   "Target summary",
		"body":      "Target body",
	})
	var target struct {
		Record struct {
			ID string `json:"id"`
		} `json:"record"`
	}
	require.NoError(t, json.Unmarshal(targetResp, &target))

	callTool(t, ts, parentSess.SessionID, "create_record", map[string]any{
		"parent_id": target.Record.ID,
		"type":      "note",
		"title":     "Open child",
		"summary":   "Open summary",
		"body":      "Open body",
	})

	callTool(t, ts, parentSess.SessionID, "create_record", map[string]any{
		"parent_id": target.Record.ID,
		"type":      "note",
		"title":     "Resolved child",
		"summary":   "Resolved summary",
		"body":      "Resolved body",
		"state":     "RESOLVED",
	})

	activateTarget := callTool(t, ts, parentSess.SessionID, "activate", map[string]any{"id": target.Record.ID})
	require.Contains(t, string(activateTarget), "open_children")

	search := callTool(t, ts, "", "search_records", map[string]any{"query": "Target"})
	require.NotEmpty(t, search)
}

func TestFunctional_HistoryDiffAndActivity(t *testing.T) {
	ts := testserver.New(t, "token", "tenant1")
	initializeSession(t, ts)

	rootResp := callTool(t, ts, "", "create_record", map[string]any{
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

	activation := callTool(t, ts, "", "activate", map[string]any{"id": root.Record.ID})
	var sess struct {
		SessionID string `json:"session_id"`
	}
	require.NoError(t, json.Unmarshal(activation, &sess))

	_ = callTool(t, ts, sess.SessionID, "update_record", map[string]any{"id": root.Record.ID, "title": "New"})

	history := callTool(t, ts, "", "get_record_history", map[string]any{"id": root.Record.ID})
	require.NotEmpty(t, history)

	diff := callTool(t, ts, "", "get_record_diff", map[string]any{"id": root.Record.ID, "from": "last_save"})
	require.NotEmpty(t, diff)

	activityResp := callTool(t, ts, "", "get_recent_activity", map[string]any{})
	require.NotEmpty(t, activityResp)
}

func TestFunctional_TenantIsolation(t *testing.T) {
	ts := testserver.New(t, "token", "tenant1")
	require.NoError(t, ts.AddAPIKey("token2", "tenant2"))
	initializeSession(t, ts)

	_ = callTool(t, ts, "", "create_project", map[string]any{"name": "Tenant1"})

	// Use tools/call for the second tenant
	req, err := http.NewRequest(http.MethodPost, ts.Server.URL+"/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","method":"tools/call","params":{"name":"list_projects"},"id":1}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token2")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result rpcResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	require.Nil(t, result.Error)
}

func TestFunctional_MCPProtocolCompliance(t *testing.T) {
	ts := testserver.New(t, "token", "tenant1")

	// Test initialize handshake
	initResp := rpcCall(t, ts, "", "initialize", map[string]any{
		"protocolVersion": "2025-03-26",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "test-client",
			"version": "1.0.0",
		},
	})
	require.Nil(t, initResp.Error)

	var initResult struct {
		ProtocolVersion string `json:"protocolVersion"`
		Capabilities    struct {
			Tools struct {
				ListChanged bool `json:"listChanged"`
			} `json:"tools"`
		} `json:"capabilities"`
		ServerInfo struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"serverInfo"`
	}
	require.NoError(t, json.Unmarshal(initResp.Result, &initResult))
	require.Equal(t, "2025-03-26", initResult.ProtocolVersion)
	require.True(t, initResult.Capabilities.Tools.ListChanged)
	require.Equal(t, "threds-mcp", initResult.ServerInfo.Name)

	// Test tools/list discovery
	toolsResp := rpcCall(t, ts, "", "tools/list", map[string]any{})
	require.Nil(t, toolsResp.Error)

	var toolsResult struct {
		Tools []struct {
			Name        string         `json:"name"`
			Description string         `json:"description"`
			InputSchema map[string]any `json:"inputSchema"`
		} `json:"tools"`
	}
	require.NoError(t, json.Unmarshal(toolsResp.Result, &toolsResult))
	require.Greater(t, len(toolsResult.Tools), 15, "should have at least 16 tools")

	// Verify some expected tools exist
	toolNames := make(map[string]bool)
	for _, tool := range toolsResult.Tools {
		toolNames[tool.Name] = true
		require.NotEmpty(t, tool.Description, "tool %s should have description", tool.Name)
		require.NotNil(t, tool.InputSchema, "tool %s should have inputSchema", tool.Name)
	}
	require.True(t, toolNames["create_project"], "should have create_project tool")
	require.True(t, toolNames["activate"], "should have activate tool")
	require.True(t, toolNames["create_record"], "should have create_record tool")

	// Test tools/call execution
	createProjectResp := rpcCall(t, ts, "", "tools/call", map[string]any{
		"name": "create_project",
		"arguments": map[string]any{
			"name": "Test Project",
		},
	})
	require.Nil(t, createProjectResp.Error)

	var toolCallResult struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	require.NoError(t, json.Unmarshal(createProjectResp.Result, &toolCallResult))
	require.False(t, toolCallResult.IsError)
	require.NotEmpty(t, toolCallResult.Content)
	require.Equal(t, "text", toolCallResult.Content[0].Type)
	require.Contains(t, toolCallResult.Content[0].Text, "Test Project")
}
