package activity

// ListActivityOptions provides filtering options for listing activity.
type ListActivityOptions struct {
	ProjectID    string
	RecordID     *string
	SessionID    *string
	ActivityType *ActivityType
	Limit        int
	Offset       int
}
