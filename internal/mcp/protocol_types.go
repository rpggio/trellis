package mcp

// Protocol types for MCP specification (2025-03-26)

// InitializeParams represents the initialize request parameters
type InitializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    ClientCapabilities     `json:"capabilities"`
	ClientInfo      ImplementationInfo     `json:"clientInfo"`
}

// InitializeResult represents the initialize response
type InitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    ServerCapabilities     `json:"capabilities"`
	ServerInfo      ImplementationInfo     `json:"serverInfo"`
	Instructions    string                 `json:"instructions,omitempty"`
}

// ImplementationInfo describes the client or server implementation
type ImplementationInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ClientCapabilities describes client capabilities
type ClientCapabilities struct {
	Roots        *RootsCapability    `json:"roots,omitempty"`
	Sampling     *SamplingCapability `json:"sampling,omitempty"`
	Experimental map[string]any      `json:"experimental,omitempty"`
}

// ServerCapabilities describes server capabilities
type ServerCapabilities struct {
	Prompts      *PromptsCapability      `json:"prompts,omitempty"`
	Resources    *ResourcesCapability    `json:"resources,omitempty"`
	Tools        *ToolsCapability        `json:"tools,omitempty"`
	Logging      *LoggingCapability      `json:"logging,omitempty"`
	Completions  *CompletionsCapability  `json:"completions,omitempty"`
	Experimental map[string]any          `json:"experimental,omitempty"`
}

// RootsCapability indicates client can provide filesystem roots
type RootsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// SamplingCapability indicates client supports LLM sampling
type SamplingCapability struct{}

// PromptsCapability indicates server offers prompt templates
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability indicates server provides readable resources
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// ToolsCapability indicates server exposes callable tools
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// LoggingCapability indicates server emits structured logs
type LoggingCapability struct{}

// CompletionsCapability indicates server supports autocompletion
type CompletionsCapability struct{}

// ToolsListParams represents the tools/list request parameters
type ToolsListParams struct {
	Cursor string `json:"cursor,omitempty"`
}

// ToolsListResult represents the tools/list response
type ToolsListResult struct {
	Tools      []ToolDefinition `json:"tools"`
	NextCursor string           `json:"nextCursor,omitempty"`
}

// ToolDefinition describes a callable tool
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	Annotations map[string]any `json:"annotations,omitempty"`
}

// ToolCallParams represents the tools/call request parameters
type ToolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// ToolCallResult represents the tools/call response
type ToolCallResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ContentItem represents a piece of content in a tool result
type ContentItem struct {
	Type     string                 `json:"type"`
	Text     string                 `json:"text,omitempty"`
	Data     string                 `json:"data,omitempty"`
	MimeType string                 `json:"mimeType,omitempty"`
	Resource *EmbeddedResource      `json:"resource,omitempty"`
	Extra    map[string]any         `json:"-"`
}

// EmbeddedResource represents a resource embedded in content
type EmbeddedResource struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"`
}
