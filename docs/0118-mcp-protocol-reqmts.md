## Authoritative Summary: HTTP MCP Requirements

### The Core Confusion: There Are Two HTTP Transports

| Transport | Spec Version | Status |
|-----------|--------------|--------|
| **HTTP+SSE** (legacy) | 2024-11-05 | **Deprecated** (March 2025) |
| **Streamable HTTP** | 2025-03-26 | **Current standard** |

**Key insight:** "Streamable HTTP replaces the HTTP+SSE transport from protocol version 2024-11-05."

---

### Is SSE Required?

**NO.** SSE is **optional** in Streamable HTTP.

From the official spec: "Server can optionally make use of Server-Sent Events (SSE) to stream multiple server messages. This permits basic MCP servers, as well as more feature-rich servers supporting streaming."

A minimal Streamable HTTP server:
- **MUST** expose a single endpoint supporting POST and GET
- **MAY** return `application/json` responses (no SSE at all)
- **MAY** return `text/event-stream` for streaming (optional)

---

### Client Support Matrix (as of Jan 2026)

| Client | Streamable HTTP | Legacy SSE | Auth Required? |
|--------|-----------------|------------|----------------|
| **Claude.ai / Desktop** | ✅ Yes | ✅ Yes (deprecating) | No (authless OK) |
| **Claude Code** | ✅ Yes (`--transport http`) | ✅ Yes (`--transport sse`) | No |
| **ChatGPT** | ✅ Yes | ✅ Yes | No (OAuth optional) |
| **OpenAI Codex** | ✅ Yes | ❌ Not mentioned | No |

From Anthropic's help center: "Claude supports both SSE- and Streamable HTTP-based remote servers, although support for SSE may be deprecated in the coming months. Claude supports both authless and OAuth-based remote servers."

From OpenAI docs: "The Responses API works with remote MCP servers that support either the Streamable HTTP or the HTTP/SSE transport protocols."

---

### Why Claude Code Might Be Asking for SSE

Possible reasons:
1. **You're using an older Claude Code version** - check your version
2. **Your URL format** - Claude Code uses `--transport http` for Streamable HTTP, `--transport sse` for legacy
3. **Server detection** - if your server returns 4xx on POST, Claude Code falls back to assuming legacy SSE

The GitHub issue you're probably hitting: Claude Code issue #1387 was about Streamable HTTP support being added (it's now closed/resolved).

---

### Minimal Requirements for a Universal HTTP MCP Server

```
1. Single endpoint (e.g., /mcp)
2. Support POST method
3. Accept: application/json, text/event-stream header handling
4. Return Content-Type: application/json for simple responses
5. Validate Origin header (security requirement)
6. JSON-RPC 2.0 message format
```

**Auth:** Optional. Both authless and OAuth work.

**SSE streaming:** Optional. Only needed if you want to:
- Send server-initiated notifications
- Stream long-running responses
- Support resumability

---

### Quick Reference: Adding to Clients

**Claude Code (Streamable HTTP):**
```bash
claude mcp add --transport http myserver https://example.com/mcp
```

**Claude Code (Legacy SSE):**
```bash
claude mcp add --transport sse myserver https://example.com/sse
```

**Claude Desktop config:**
```json
{
  "mcpServers": {
    "myserver": {
      "url": "https://example.com/mcp"
    }
  }
}
```

---

### Sources
1. Official MCP Spec (2025-03-26): https://modelcontextprotocol.io/specification/2025-03-26/basic/transports
2. Anthropic Help Center: https://support.claude.com/en/articles/11503834-building-custom-connectors-via-remote-mcp-servers
3. Claude Code MCP Docs: https://code.claude.com/docs/en/mcp
4. OpenAI Connectors/MCP: https://platform.openai.com/docs/guides/tools-connectors-mcp

---

**TL;DR:** SSE is optional in the current spec. Both Claude and ChatGPT support Streamable HTTP. A plain JSON POST/response server works fine. The confusion comes from the spec evolution and client version differences.