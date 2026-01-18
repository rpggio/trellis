package functional_test

import (
	"bytes"
	"encoding/json"
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
	req.Header.Set("Authorization", "Bearer "+ts.Token)
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result rpcResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	return result
}

func TestFunctional_Authentication(t *testing.T) {
	ts := testserver.New(t, "token", "tenant1")

	req, err := http.NewRequest(http.MethodPost, ts.Server.URL+"/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","method":"list_projects","id":1}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestFunctional_ProjectAndOverview(t *testing.T) {
	ts := testserver.New(t, "token", "tenant1")

	create := rpcCall(t, ts, "", "create_project", map[string]any{
		"name": "Project",
	})
	require.Nil(t, create.Error)

	list := rpcCall(t, ts, "", "list_projects", nil)
	require.Nil(t, list.Error)

	get := rpcCall(t, ts, "", "get_project", map[string]any{})
	require.Nil(t, get.Error)

	overview := rpcCall(t, ts, "", "get_project_overview", map[string]any{})
	require.Nil(t, overview.Error)
}

func TestFunctional_ActivationWorkflow(t *testing.T) {
	ts := testserver.New(t, "token", "tenant1")

	projectResp := rpcCall(t, ts, "", "create_project", map[string]any{"name": "Project"})
	require.Nil(t, projectResp.Error)

	var project struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(projectResp.Result, &project))

	rootResp := rpcCall(t, ts, "", "create_record", map[string]any{
		"type":    "question",
		"title":   "Root",
		"summary": "Root summary",
		"body":    "Root body",
	})
	require.Nil(t, rootResp.Error)

	var root struct {
		Record struct {
			ID string `json:"id"`
		} `json:"record"`
	}
	require.NoError(t, json.Unmarshal(rootResp.Result, &root))

	activateResp := rpcCall(t, ts, "", "activate", map[string]any{"id": root.Record.ID})
	require.Nil(t, activateResp.Error)

	var activation struct {
		SessionID string `json:"session_id"`
	}
	require.NoError(t, json.Unmarshal(activateResp.Result, &activation))

	childResp := rpcCall(t, ts, activation.SessionID, "create_record", map[string]any{
		"parent_id": root.Record.ID,
		"type":      "note",
		"title":     "Child",
		"summary":   "Child summary",
		"body":      "Child body",
	})
	require.Nil(t, childResp.Error)

	newTitle := "Updated"
	updateResp := rpcCall(t, ts, activation.SessionID, "update_record", map[string]any{
		"id":    root.Record.ID,
		"title": newTitle,
	})
	require.Nil(t, updateResp.Error)

	transitionResp := rpcCall(t, ts, activation.SessionID, "transition", map[string]any{
		"id":       root.Record.ID,
		"to_state": "LATER",
		"reason":   "wait",
	})
	require.Nil(t, transitionResp.Error)

	saveResp := rpcCall(t, ts, activation.SessionID, "save_session", nil)
	require.Nil(t, saveResp.Error)

	closeResp := rpcCall(t, ts, activation.SessionID, "close_session", nil)
	require.Nil(t, closeResp.Error)

	_ = project.ID
}

func TestFunctional_ConflictAndActiveSessions(t *testing.T) {
	ts := testserver.New(t, "token", "tenant1")

	rootResp := rpcCall(t, ts, "", "create_record", map[string]any{
		"type":    "question",
		"title":   "Root",
		"summary": "Root summary",
		"body":    "Root body",
	})
	require.Nil(t, rootResp.Error)

	var root struct {
		Record struct {
			ID string `json:"id"`
		} `json:"record"`
	}
	require.NoError(t, json.Unmarshal(rootResp.Result, &root))

	s1 := rpcCall(t, ts, "", "activate", map[string]any{"id": root.Record.ID})
	require.Nil(t, s1.Error)
	var sess1 struct {
		SessionID string `json:"session_id"`
	}
	require.NoError(t, json.Unmarshal(s1.Result, &sess1))

	s2 := rpcCall(t, ts, "", "activate", map[string]any{"id": root.Record.ID})
	require.Nil(t, s2.Error)
	var sess2 struct {
		SessionID string `json:"session_id"`
	}
	require.NoError(t, json.Unmarshal(s2.Result, &sess2))

	update1 := rpcCall(t, ts, sess1.SessionID, "update_record", map[string]any{"id": root.Record.ID, "title": "A"})
	require.Nil(t, update1.Error)

	update2 := rpcCall(t, ts, sess2.SessionID, "update_record", map[string]any{"id": root.Record.ID, "title": "B"})
	require.Nil(t, update2.Error)
	require.Contains(t, string(update2.Result), "conflict")

	activeSessions := rpcCall(t, ts, sess1.SessionID, "get_active_sessions", map[string]any{"record_id": root.Record.ID})
	require.Nil(t, activeSessions.Error)
}

func TestFunctional_ContextLoadingAndSearch(t *testing.T) {
	ts := testserver.New(t, "token", "tenant1")

	parentResp := rpcCall(t, ts, "", "create_record", map[string]any{
		"type":    "question",
		"title":   "Parent",
		"summary": "Parent summary",
		"body":    "Parent body",
	})
	require.Nil(t, parentResp.Error)

	var parent struct {
		Record struct {
			ID string `json:"id"`
		} `json:"record"`
	}
	require.NoError(t, json.Unmarshal(parentResp.Result, &parent))

	parentActivate := rpcCall(t, ts, "", "activate", map[string]any{"id": parent.Record.ID})
	require.Nil(t, parentActivate.Error)

	var parentSess struct {
		SessionID string `json:"session_id"`
	}
	require.NoError(t, json.Unmarshal(parentActivate.Result, &parentSess))

	targetResp := rpcCall(t, ts, parentSess.SessionID, "create_record", map[string]any{
		"parent_id": parent.Record.ID,
		"type":      "question",
		"title":     "Target",
		"summary":   "Target summary",
		"body":      "Target body",
	})
	require.Nil(t, targetResp.Error)

	var target struct {
		Record struct {
			ID string `json:"id"`
		} `json:"record"`
	}
	require.NoError(t, json.Unmarshal(targetResp.Result, &target))

	rpcCall(t, ts, parentSess.SessionID, "create_record", map[string]any{
		"parent_id": target.Record.ID,
		"type":      "note",
		"title":     "Open child",
		"summary":   "Open summary",
		"body":      "Open body",
	})

	rpcCall(t, ts, parentSess.SessionID, "create_record", map[string]any{
		"parent_id": target.Record.ID,
		"type":      "note",
		"title":     "Resolved child",
		"summary":   "Resolved summary",
		"body":      "Resolved body",
		"state":     "RESOLVED",
	})

	activateTarget := rpcCall(t, ts, parentSess.SessionID, "activate", map[string]any{"id": target.Record.ID})
	require.Nil(t, activateTarget.Error)
	require.Contains(t, string(activateTarget.Result), "open_children")

	search := rpcCall(t, ts, "", "search_records", map[string]any{"query": "Target"})
	require.Nil(t, search.Error)
}

func TestFunctional_HistoryDiffAndActivity(t *testing.T) {
	ts := testserver.New(t, "token", "tenant1")

	rootResp := rpcCall(t, ts, "", "create_record", map[string]any{
		"type":    "question",
		"title":   "Root",
		"summary": "Root summary",
		"body":    "Root body",
	})
	require.Nil(t, rootResp.Error)

	var root struct {
		Record struct {
			ID string `json:"id"`
		} `json:"record"`
	}
	require.NoError(t, json.Unmarshal(rootResp.Result, &root))

	activation := rpcCall(t, ts, "", "activate", map[string]any{"id": root.Record.ID})
	require.Nil(t, activation.Error)
	var sess struct {
		SessionID string `json:"session_id"`
	}
	require.NoError(t, json.Unmarshal(activation.Result, &sess))

	_ = rpcCall(t, ts, sess.SessionID, "update_record", map[string]any{"id": root.Record.ID, "title": "New"})

	history := rpcCall(t, ts, "", "get_record_history", map[string]any{"id": root.Record.ID})
	require.Nil(t, history.Error)

	diff := rpcCall(t, ts, "", "get_record_diff", map[string]any{"id": root.Record.ID, "from": "last_save"})
	require.Nil(t, diff.Error)

	activityResp := rpcCall(t, ts, "", "get_recent_activity", map[string]any{})
	require.Nil(t, activityResp.Error)
}

func TestFunctional_TenantIsolation(t *testing.T) {
	ts := testserver.New(t, "token", "tenant1")
	require.NoError(t, ts.AddAPIKey("token2", "tenant2"))

	_ = rpcCall(t, ts, "", "create_project", map[string]any{"name": "Tenant1"})

	req, err := http.NewRequest(http.MethodPost, ts.Server.URL+"/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","method":"list_projects","id":1}`))
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
