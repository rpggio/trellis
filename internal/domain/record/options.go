package record

// ListRecordsOptions provides filtering options for listing records.
type ListRecordsOptions struct {
	ProjectID string
	ParentID  *string
	States    []RecordState
	Types     []string
	Limit     int
	Offset    int
}

// SearchOptions provides filtering options for search.
type SearchOptions struct {
	States []RecordState
	Types  []string
	Limit  int
	Offset int
}
