package record

import "time"

// RecordState represents the workflow state of a record
type RecordState string

const (
	StateOpen      RecordState = "OPEN"
	StateLater     RecordState = "LATER"
	StateResolved  RecordState = "RESOLVED"
	StateDiscarded RecordState = "DISCARDED"
)

// Record represents a unit of design reasoning
type Record struct {
	ID         string      `json:"id"`
	TenantID   string      `json:"tenant_id"`
	ProjectID  string      `json:"project_id"`
	Type       string      `json:"type"`
	Title      string      `json:"title"`
	Summary    string      `json:"summary"`
	Body       string      `json:"body"`
	State      RecordState `json:"state"`
	ParentID   *string     `json:"parent_id,omitempty"`
	ResolvedBy *string     `json:"resolved_by,omitempty"`
	CreatedAt  time.Time   `json:"created_at"`
	ModifiedAt time.Time   `json:"modified_at"`
	Tick       int64       `json:"tick"`
	Related    []string    `json:"related,omitempty"`
}

// RecordRef is a lightweight reference to a record
type RecordRef struct {
	ID                string      `json:"id"`
	Type              string      `json:"type"`
	Title             string      `json:"title"`
	Summary           string      `json:"summary"`
	State             RecordState `json:"state"`
	ParentID          *string     `json:"parent_id,omitempty"`
	ChildrenCount     int         `json:"children_count"`
	OpenChildrenCount int         `json:"open_children_count"`
}

// SearchResult represents a search hit with relevance
type SearchResult struct {
	Record  RecordRef `json:"record"`
	Rank    float64   `json:"rank"`
	Snippet string    `json:"snippet,omitempty"`
}
