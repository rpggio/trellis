package record

import "strings"

// ValidateCreateInput validates fields required to create a record.
func ValidateCreateInput(req CreateRequest) error {
	if strings.TrimSpace(req.ProjectID) == "" {
		return ErrInvalidInput
	}
	if strings.TrimSpace(req.Type) == "" {
		return ErrInvalidInput
	}
	if strings.TrimSpace(req.Title) == "" {
		return ErrInvalidInput
	}
	if strings.TrimSpace(req.Summary) == "" {
		return ErrInvalidInput
	}
	if strings.TrimSpace(req.Body) == "" {
		return ErrInvalidInput
	}
	return nil
}

// ValidateTransition validates a requested state transition.
func ValidateTransition(fromState, toState RecordState, reason, resolvedBy *string) error {
	valid := false
	switch fromState {
	case StateOpen:
		switch toState {
		case StateLater, StateResolved, StateDiscarded:
			valid = true
		}
	case StateLater:
		if toState == StateOpen || toState == StateDiscarded {
			valid = true
		}
	case StateResolved:
		if toState == StateOpen {
			valid = true
		}
	case StateDiscarded:
		if toState == StateOpen {
			valid = true
		}
	}

	if !valid {
		return ErrInvalidTransition
	}

	if toState == StateLater || toState == StateDiscarded {
		if reason == nil || strings.TrimSpace(*reason) == "" {
			return ErrMissingReason
		}
	}
	if toState == StateResolved {
		if resolvedBy == nil || strings.TrimSpace(*resolvedBy) == "" {
			return ErrMissingResolvedBy
		}
	}

	return nil
}
