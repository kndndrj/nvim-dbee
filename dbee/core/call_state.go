package core

type CallState int

const (
	CallStateUnknown CallState = iota
	CallStateExecuting
	CallStateRetrieving
	CallStateArchived
	CallStateFailed
	CallStateCanceled
)

func CallStateFromString(s string) CallState {
	switch s {
	case "unknown":
		return CallStateUnknown
	case "executing":
		return CallStateExecuting
	case "retrieving":
		return CallStateRetrieving
	case "archived":
		return CallStateArchived
	case "failed":
		return CallStateFailed
	case "canceled":
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
	case CallStateRetrieving:
		return "retrieving"
	case CallStateArchived:
		return "archived"
	case CallStateFailed:
		return "failed"
	case CallStateCanceled:
		return "canceled"
	default:
		return "unknown"
	}
}
