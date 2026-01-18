package project

import "time"

// Project represents a container for records with a monotonic tick counter
type Project struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Tick        int64     `json:"tick"`
	CreatedAt   time.Time `json:"created_at"`
}

// ProjectSummary is a lightweight representation for listing
type ProjectSummary struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	Tick           int64     `json:"tick"`
	RecordCount    int       `json:"record_count"`
	OpenRecords    int       `json:"open_records"`
	ActiveSessions int       `json:"active_sessions"`
	CreatedAt      time.Time `json:"created_at"`
}
