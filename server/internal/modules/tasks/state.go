package tasks

type Status string

const (
	StatusPending   Status = "pending"
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusCanceling Status = "canceling"
	StatusSuccess   Status = "success"
	StatusFailed    Status = "failed"
	StatusTimeout   Status = "timeout"
	StatusCanceled  Status = "canceled"
)

func CanTransition(from, to Status) bool {
	switch from {
	case StatusPending:
		return to == StatusQueued || to == StatusCanceled
	case StatusQueued:
		return to == StatusRunning || to == StatusCanceled
	case StatusRunning:
		return to == StatusSuccess || to == StatusFailed || to == StatusTimeout || to == StatusCanceling || to == StatusCanceled
	case StatusCanceling:
		return to == StatusCanceled
	default:
		return false
	}
}

func IsTerminal(status Status) bool {
	switch status {
	case StatusSuccess, StatusFailed, StatusTimeout, StatusCanceled:
		return true
	default:
		return false
	}
}
