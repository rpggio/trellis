package mcp

// buildToolCatalog returns all available MCP tools
func buildToolCatalog() []ToolDefinition {
	return []ToolDefinition{
		// Projects
		{
			Name:        "create_project",
			Description: "Create a new project to organize records",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{
						"type":        "string",
						"description": "Unique project identifier (optional, will be generated if not provided)",
					},
					"name": map[string]any{
						"type":        "string",
						"description": "Project display name",
					},
					"description": map[string]any{
						"type":        "string",
						"description": "Project description",
					},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "list_projects",
			Description: "List all projects for the current tenant",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "get_project",
			Description: "Get details for a specific project or the default project",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{
						"type":        "string",
						"description": "Project ID (omit to get default project)",
					},
				},
			},
		},

		// Orientation
		{
			Name:        "get_project_overview",
			Description: "Get a comprehensive overview of a project including open sessions and root records",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{
						"type":        "string",
						"description": "Project ID (omit to use default project)",
					},
				},
			},
		},
		{
			Name:        "search_records",
			Description: "Search for records by query text, optionally filtered by state and type",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{
						"type":        "string",
						"description": "Project ID (omit to use default project)",
					},
					"query": map[string]any{
						"type":        "string",
						"description": "Search query text",
					},
					"states": map[string]any{
						"type":        "array",
						"description": "Filter by record states (open, in_progress, resolved, closed)",
						"items": map[string]any{
							"type": "string",
							"enum": []string{"open", "in_progress", "resolved", "closed"},
						},
					},
					"types": map[string]any{
						"type":        "array",
						"description": "Filter by record types",
						"items":       map[string]any{"type": "string"},
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of results",
					},
					"offset": map[string]any{
						"type":        "integer",
						"description": "Offset for pagination",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "list_records",
			Description: "List records in a project, optionally filtered by parent, state, and type",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{
						"type":        "string",
						"description": "Project ID (omit to use default project)",
					},
					"parent_id": map[string]any{
						"type":        "string",
						"description": "Parent record ID (null/empty for root records)",
					},
					"states": map[string]any{
						"type":        "array",
						"description": "Filter by record states",
						"items": map[string]any{
							"type": "string",
							"enum": []string{"open", "in_progress", "resolved", "closed"},
						},
					},
					"types": map[string]any{
						"type":        "array",
						"description": "Filter by record types",
						"items":       map[string]any{"type": "string"},
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of results",
					},
					"offset": map[string]any{
						"type":        "integer",
						"description": "Offset for pagination",
					},
				},
			},
		},
		{
			Name:        "get_record_ref",
			Description: "Get a record reference (summary view) by ID",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{
						"type":        "string",
						"description": "Record ID",
					},
				},
				"required": []string{"id"},
			},
		},

		// Activation
		{
			Name:        "activate",
			Description: "Activate a record for editing in the current session, retrieving its full context",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{
						"type":        "string",
						"description": "Record ID to activate",
					},
				},
				"required": []string{"id"},
			},
		},
		{
			Name:        "sync_session",
			Description: "Synchronize the current session to detect changes made by other sessions",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]any{
						"type":        "string",
						"description": "Session ID (omit to use current session from header)",
					},
				},
			},
		},

		// Mutations
		{
			Name:        "create_record",
			Description: "Create a new record in the project",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"parent_id": map[string]any{
						"type":        "string",
						"description": "Parent record ID (null for root-level records)",
					},
					"type": map[string]any{
						"type":        "string",
						"description": "Record type (e.g., 'question', 'thread', 'conclusion')",
					},
					"title": map[string]any{
						"type":        "string",
						"description": "Record title",
					},
					"summary": map[string]any{
						"type":        "string",
						"description": "Brief summary of the record",
					},
					"body": map[string]any{
						"type":        "string",
						"description": "Full record content",
					},
					"state": map[string]any{
						"type":        "string",
						"description": "Initial state",
						"enum":        []string{"open", "in_progress", "resolved", "closed"},
					},
					"related": map[string]any{
						"type":        "array",
						"description": "Related record IDs",
						"items":       map[string]any{"type": "string"},
					},
				},
				"required": []string{"type", "title", "summary", "body"},
			},
		},
		{
			Name:        "update_record",
			Description: "Update an existing record's content",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{
						"type":        "string",
						"description": "Record ID to update",
					},
					"title": map[string]any{
						"type":        "string",
						"description": "New title",
					},
					"summary": map[string]any{
						"type":        "string",
						"description": "New summary",
					},
					"body": map[string]any{
						"type":        "string",
						"description": "New body content",
					},
					"related": map[string]any{
						"type":        "array",
						"description": "New related record IDs",
						"items":       map[string]any{"type": "string"},
					},
					"force": map[string]any{
						"type":        "boolean",
						"description": "Force update even if conflicts detected",
					},
				},
				"required": []string{"id"},
			},
		},
		{
			Name:        "transition",
			Description: "Transition a record to a new state (e.g., open → in_progress → resolved)",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{
						"type":        "string",
						"description": "Record ID",
					},
					"to_state": map[string]any{
						"type":        "string",
						"description": "Target state",
						"enum":        []string{"open", "in_progress", "resolved", "closed"},
					},
					"reason": map[string]any{
						"type":        "string",
						"description": "Reason for transition",
					},
					"resolved_by": map[string]any{
						"type":        "string",
						"description": "Record ID that resolves this one (for resolved state)",
					},
				},
				"required": []string{"id", "to_state"},
			},
		},

		// Session Lifecycle
		{
			Name:        "save_session",
			Description: "Save the current session state",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]any{
						"type":        "string",
						"description": "Session ID (omit to use current session from header)",
					},
				},
			},
		},
		{
			Name:        "close_session",
			Description: "Close the current session and release all locks",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]any{
						"type":        "string",
						"description": "Session ID (omit to use current session from header)",
					},
				},
			},
		},
		{
			Name:        "branch_session",
			Description: "Create a new session branched from an existing one",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]any{
						"type":        "string",
						"description": "Source session ID to branch from",
					},
					"focus_record": map[string]any{
						"type":        "string",
						"description": "Record ID to focus on in the new session",
					},
				},
				"required": []string{"session_id"},
			},
		},

		// History/Conflict
		{
			Name:        "get_record_history",
			Description: "Get the change history for a record",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{
						"type":        "string",
						"description": "Record ID",
					},
					"since": map[string]any{
						"type":        "string",
						"description": "Timestamp to fetch history since (ISO 8601)",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of history entries",
					},
				},
				"required": []string{"id"},
			},
		},
		{
			Name:        "get_record_diff",
			Description: "Get the diff between two versions of a record",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{
						"type":        "string",
						"description": "Record ID",
					},
					"from": map[string]any{
						"type":        "string",
						"description": "Source version timestamp",
					},
					"to": map[string]any{
						"type":        "string",
						"description": "Target version timestamp (omit for current)",
					},
				},
				"required": []string{"id", "from"},
			},
		},
		{
			Name:        "get_active_sessions",
			Description: "Get all active sessions working on a specific record",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"record_id": map[string]any{
						"type":        "string",
						"description": "Record ID",
					},
				},
				"required": []string{"record_id"},
			},
		},
		{
			Name:        "get_recent_activity",
			Description: "Get recent activity entries for a project or specific record",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{
						"type":        "string",
						"description": "Project ID to filter by",
					},
					"record_id": map[string]any{
						"type":        "string",
						"description": "Record ID to filter by",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of activity entries",
					},
					"since": map[string]any{
						"type":        "string",
						"description": "Timestamp to fetch activity since (ISO 8601)",
					},
					"types": map[string]any{
						"type":        "array",
						"description": "Filter by activity types",
						"items":       map[string]any{"type": "string"},
					},
				},
			},
		},
	}
}
