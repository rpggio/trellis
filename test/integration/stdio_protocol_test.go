package integration_test

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

// TestStdioProtocolCompliance verifies the server works correctly over stdio transport
// using the official MCP SDK client. This catches protocol issues that shell-based
// tests might miss.
func TestStdioProtocolCompliance(t *testing.T) {
	// Build the server first if binary doesn't exist
	binaryPath := "./bin/threds-mcp"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		// Try relative to test directory
		binaryPath = "../../bin/threds-mcp"
		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			t.Skip("Server binary not found. Run 'make build' first.")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create command with environment
	cmd := exec.CommandContext(ctx, binaryPath)
	cmd.Env = append(os.Environ(),
		"THREDS_TRANSPORT=stdio",
		"THREDS_DB_PATH=:memory:",
	)

	// Spawn server as subprocess using SDK's CommandTransport
	transport := &sdkmcp.CommandTransport{
		Command: cmd,
	}

	// Create client
	client := sdkmcp.NewClient(&sdkmcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	// Connect and initialize
	session, err := client.Connect(ctx, transport, nil)
	require.NoError(t, err, "Failed to connect to server")
	defer session.Close()

	// Verify initialize response
	t.Run("ServerInfo", func(t *testing.T) {
		initResult := session.InitializeResult()
		require.NotNil(t, initResult)
		require.NotNil(t, initResult.ServerInfo)
		require.Equal(t, "threds-mcp", initResult.ServerInfo.Name)
		require.Equal(t, "0.1.0", initResult.ServerInfo.Version)
	})

	// Test tools/list
	t.Run("ListTools", func(t *testing.T) {
		tools, err := session.ListTools(ctx, nil)
		require.NoError(t, err, "tools/list failed")
		require.Greater(t, len(tools.Tools), 10, "Expected at least 10 tools")

		// Verify expected core tools exist
		toolNames := make(map[string]bool)
		for _, tool := range tools.Tools {
			toolNames[tool.Name] = true
		}

		expectedTools := []string{
			"ping",
			"create_project",
			"list_projects",
			"create_record",
			"get_record_ref",
			"list_records",
			"activate",
		}
		for _, name := range expectedTools {
			require.True(t, toolNames[name], "Missing expected tool: %s", name)
		}
	})

	// Test calling ping tool
	t.Run("CallPingTool", func(t *testing.T) {
		result, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
			Name: "ping",
		})
		require.NoError(t, err, "tools/call ping failed")
		require.False(t, result.IsError, "ping returned error: %v", result)
		require.NotEmpty(t, result.Content, "ping returned no content")

		// Verify response contains expected text
		hasText := false
		for _, content := range result.Content {
			if textContent, ok := content.(*sdkmcp.TextContent); ok {
				hasText = true
				require.Contains(t, textContent.Text, "pong", "ping response should contain 'pong'")
			}
		}
		require.True(t, hasText, "ping should return text content")
	})

	// Test calling create_project tool
	t.Run("CallCreateProject", func(t *testing.T) {
		result, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
			Name: "create_project",
			Arguments: map[string]any{
				"name": "Test Project",
			},
		})
		require.NoError(t, err, "tools/call create_project failed")
		require.False(t, result.IsError, "create_project returned error: %v", result)
	})

	// Test list_projects after creation
	t.Run("CallListProjects", func(t *testing.T) {
		result, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
			Name: "list_projects",
		})
		require.NoError(t, err, "tools/call list_projects failed")
		require.False(t, result.IsError, "list_projects returned error: %v", result)
		require.NotEmpty(t, result.Content, "list_projects returned no content")
	})
}

// TestStdioProtocol_StdoutHygiene verifies that the server doesn't write
// anything to stdout except valid JSON-RPC messages.
func TestStdioProtocol_StdoutHygiene(t *testing.T) {
	// Build the server first if binary doesn't exist
	binaryPath := "./bin/threds-mcp"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		binaryPath = "../../bin/threds-mcp"
		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			t.Skip("Server binary not found. Run 'make build' first.")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Run server with a simple initialize request and capture stdout/stderr
	cmd := exec.CommandContext(ctx, binaryPath)
	cmd.Env = append(os.Environ(),
		"THREDS_TRANSPORT=stdio",
		"THREDS_DB_PATH=:memory:",
	)

	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)

	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)

	stderr, err := cmd.StderrPipe()
	require.NoError(t, err)

	err = cmd.Start()
	require.NoError(t, err)

	// Send initialize request and keep stdin open for a bit
	initReq := `{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}},"id":1}`
	_, err = stdin.Write([]byte(initReq + "\n"))
	require.NoError(t, err)

	// Read output with timeout (don't close stdin yet)
	done := make(chan struct{})
	var stdoutBytes, stderrBytes []byte

	go func() {
		stdoutBytes, _ = readWithTimeout(stdout, 2*time.Second)
		stderrBytes, _ = readWithTimeout(stderr, 2*time.Second)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		cmd.Process.Kill()
		t.Fatal("Timeout waiting for server response")
	}

	// Now close stdin
	stdin.Close()
	cmd.Process.Kill()
	cmd.Wait()

	// Verify stdout starts with valid JSON
	require.NotEmpty(t, stdoutBytes, "Server produced no stdout output")
	require.True(t, stdoutBytes[0] == '{', "First character of stdout should be '{', got: %q", string(stdoutBytes[:min(50, len(stdoutBytes))]))

	// Logs should be on stderr (if any)
	t.Logf("Stderr output (logs): %s", string(stderrBytes))
}

func readWithTimeout(r interface{ Read([]byte) (int, error) }, timeout time.Duration) ([]byte, error) {
	result := make([]byte, 0, 4096)
	buf := make([]byte, 1024)

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		// Try to read
		done := make(chan struct{})
		var n int
		var err error
		go func() {
			n, err = r.Read(buf)
			close(done)
		}()

		select {
		case <-done:
			if n > 0 {
				result = append(result, buf[:n]...)
			}
			if err != nil {
				return result, err
			}
		case <-time.After(100 * time.Millisecond):
			// No data available, check if we have enough
			if len(result) > 0 {
				return result, nil
			}
		}
	}
	return result, nil
}
