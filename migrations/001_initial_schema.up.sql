-- Projects table
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    tick INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_tenant_projects ON projects(tenant_id);

-- Records table
CREATE TABLE IF NOT EXISTS records (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    summary TEXT NOT NULL,
    body TEXT NOT NULL,
    state TEXT NOT NULL CHECK(state IN ('OPEN', 'LATER', 'RESOLVED', 'DISCARDED')),
    parent_id TEXT,
    resolved_by TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    tick INTEGER NOT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id),
    FOREIGN KEY (parent_id) REFERENCES records(id),
    FOREIGN KEY (resolved_by) REFERENCES records(id)
);
CREATE INDEX IF NOT EXISTS idx_tenant_records ON records(tenant_id);
CREATE INDEX IF NOT EXISTS idx_project_records ON records(project_id);
CREATE INDEX IF NOT EXISTS idx_parent_children ON records(parent_id);
CREATE INDEX IF NOT EXISTS idx_state ON records(state);
CREATE INDEX IF NOT EXISTS idx_type ON records(type);

-- Record relations (non-hierarchical)
CREATE TABLE IF NOT EXISTS record_relations (
    from_record_id TEXT NOT NULL,
    to_record_id TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (from_record_id, to_record_id),
    FOREIGN KEY (from_record_id) REFERENCES records(id),
    FOREIGN KEY (to_record_id) REFERENCES records(id)
);

-- Sessions table
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    status TEXT NOT NULL CHECK(status IN ('active', 'stale', 'closed')),
    focus_record TEXT,
    parent_session TEXT,
    last_sync_tick INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_activity TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    closed_at TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id),
    FOREIGN KEY (focus_record) REFERENCES records(id),
    FOREIGN KEY (parent_session) REFERENCES sessions(id)
);
CREATE INDEX IF NOT EXISTS idx_tenant_sessions ON sessions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_project_sessions ON sessions(project_id);
CREATE INDEX IF NOT EXISTS idx_status ON sessions(status);

-- Session activated records (many-to-many)
CREATE TABLE IF NOT EXISTS session_activations (
    session_id TEXT NOT NULL,
    record_id TEXT NOT NULL,
    activated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    activation_tick INTEGER NOT NULL,
    PRIMARY KEY (session_id, record_id),
    FOREIGN KEY (session_id) REFERENCES sessions(id),
    FOREIGN KEY (record_id) REFERENCES records(id)
);

-- Activity log
CREATE TABLE IF NOT EXISTS activity_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    session_id TEXT,
    record_id TEXT,
    activity_type TEXT NOT NULL,
    summary TEXT NOT NULL,
    details TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    tick INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_tenant_activity ON activity_log(tenant_id);
CREATE INDEX IF NOT EXISTS idx_project_activity ON activity_log(project_id);
CREATE INDEX IF NOT EXISTS idx_record_activity ON activity_log(record_id);
CREATE INDEX IF NOT EXISTS idx_created_at ON activity_log(created_at);

-- Full-text search (SQLite FTS5)
CREATE VIRTUAL TABLE IF NOT EXISTS records_fts USING fts5(
    title,
    summary,
    body,
    content='records',
    content_rowid='rowid'
);

-- Triggers to keep FTS index synchronized
CREATE TRIGGER IF NOT EXISTS records_ai AFTER INSERT ON records BEGIN
    INSERT INTO records_fts(rowid, title, summary, body)
    VALUES (new.rowid, new.title, new.summary, new.body);
END;

CREATE TRIGGER IF NOT EXISTS records_ad AFTER DELETE ON records BEGIN
    DELETE FROM records_fts WHERE rowid = old.rowid;
END;

CREATE TRIGGER IF NOT EXISTS records_au AFTER UPDATE ON records BEGIN
    INSERT INTO records_fts(records_fts, rowid, title, summary, body)
    VALUES('delete', old.rowid, old.title, old.summary, old.body);
    INSERT INTO records_fts(rowid, title, summary, body)
    VALUES (new.rowid, new.title, new.summary, new.body);
END;

-- API keys for authentication
CREATE TABLE IF NOT EXISTS api_keys (
    key_hash TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_used TIMESTAMP,
    description TEXT
);
CREATE INDEX IF NOT EXISTS idx_tenant_keys ON api_keys(tenant_id);
