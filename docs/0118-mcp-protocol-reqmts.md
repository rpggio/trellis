### Authoritative Summary: HTTP MCP Requirements (Jan 2026)

#### 1. The Core Transport Duality

There remain two ways for clients to interact with HTTP-based MCP servers. The industry has standardized on the unified "Streamable HTTP" model.

| Transport | Spec Version | Status | Primary Characteristic |
| --- | --- | --- | --- |
| **HTTP+SSE** (legacy) | 2024-11-05 | **Deprecated** | Required two separate endpoints (`/sse` and `/messages`). |
| **Streamable HTTP** | 2025-03-26 | **Current standard** | Uses a **single endpoint** for both POST and GET. |

**Key insight:** SSE is now **optional**. A minimal server can return standard `application/json` for simple tool calls. SSE is only triggered via `text/event-stream` when the server needs to stream a long response or send notifications.

---

#### 2. Claude Desktop (Mac) Support & Registration

Claude Desktop for Mac (v1.x+, Jan 2026) supports both **Local (stdio)** and **Remote (HTTP)** MCP servers, but they use different registration paths.

**A. Remote HTTP Servers (The "Direct" Way)**

* **Plan Requirement:** Limited to Pro, Max, Team, and Enterprise plans.
* **Registration:** **MUST** be added through the UI. Navigate to **Settings > Extensions** (or **Connectors**) and click "Add Custom Connector".
* **Constraint:** Claude Desktop will **not** connect to remote HTTP servers if they are configured directly in the `claude_desktop_config.json` file.

**B. Local/Bridged Servers (The "JSON" Way)**

* **Plan Requirement:** Available for all tiers, including Free (as it appears as a local process).
* **Config Path:** `~/Library/Application Support/Claude/claude_desktop_config.json`.
* **Method:** To connect to a remote HTTP server via JSON, you must use a **bridge** (like `npx mcp-remote` or `mcp-proxy`) that converts HTTP traffic into stdio for Claude.
* **Example JSON:**
```json
{
  "mcpServers": {
    "bridged-server": {
      "command": "npx",
      "args": ["-y", "mcp-remote", "https://api.example.com/mcp"]
    }
  }
}

```



---

#### 3. Minimal Server Requirements

To be a "Universal" Streamable HTTP server, your endpoint must:

* **Single Endpoint:** Support both POST (for messages) and GET (for SSE initiation) on one URL.
* **Protocol Logic:** Validate the `Origin` header to prevent CSRF and handle the `Mcp-Session-Id` if supporting persistence.
* **JSON-RPC 2.0:** All communication must follow the JSON-RPC 2.0 structure.
* **Content Negotiation:** Accept `application/json` for standard calls and `text/event-stream` if the client requests a stream.

---

#### 4. Implementation Gotchas (User-Reported)

* **Response Codes:** Some users report that servers returning **HTTP 204 (No Content)** can cause connection timeouts in Claude Desktop; **HTTP 202 (Accepted)** is preferred for non-streaming notifications.
* **Localhost Restrictions:** Claude Desktop may block `localhost` connections due to CORS. Using a tunnel (e.g., ngrok) is recommended for development.
* **Discovery:** Servers must implement `list_tools` and `list_resources` discovery methods, or Claude will not show them in the chat interface.
