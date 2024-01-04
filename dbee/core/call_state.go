package core

type CallState int

const (
	CallStateUnknown CallState = iota
	CallStateExecuting
	CallStateExecutingFailed
	CallStateRetrieving
	CallStateRetrievingFailed
	CallStateArchived
	CallStateArchiveFailed
	CallStateCanceled
)

func CallStateFromString(s string) CallState {
	switch s {
	case CallStateUnknown.String():
		return CallStateUnknown

	case CallStateExecuting.String():
		return CallStateExecuting
	case CallStateExecutingFailed.String():
		return CallStateExecutingFailed

	case CallStateRetrieving.String():
		return CallStateRetrieving
	case CallStateRetrievingFailed.String():
		return CallStateRetrievingFailed

	case CallStateArchived.String():
		return CallStateArchived
	case CallStateArchiveFailed.String():
		return CallStateArchiveFailed

	case CallStateCanceled.String():
		return CallStateCanceled

	default:
		return CallStateUnknown
	}
}

func (s CallState) String() string {
	switch s {
	case CallStateUnknown:
		return "unknown"

	case CallStateExecuting:
		return "executing"
	case CallStateExecutingFailed:
		return "executing_failed"

	case CallStateRetrieving:
		return "retrieving"
	case CallStateRetrievingFailed:
		return "retrieving_failed"

	case CallStateArchived:
		return "archived"
	case CallStateArchiveFailed:
		return "archive_failed"

	case CallStateCanceled:
		return "canceled"

	default:
		return "unknown"
	}
}
