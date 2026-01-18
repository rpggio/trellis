package record

// ConflictInfo describes a conflict detected during an update.
type ConflictInfo struct {
	ConflictType  string  `json:"conflict_type"`
	LocalVersion  *Record `json:"local_version,omitempty"`
	RemoteVersion *Record `json:"remote_version,omitempty"`
	Message       string  `json:"message"`
}
