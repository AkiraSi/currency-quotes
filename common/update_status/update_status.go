package update_status

type UpdateStatus byte

const (
	UpdateStatusAccepted UpdateStatus = iota
	UpdateStatusRunning
	UpdateStatusSucceeded
	UpdateStatusFailed
)

func (status UpdateStatus) String() string {
	switch status {
	case UpdateStatusAccepted:
		return "accepted"
	case UpdateStatusRunning:
		return "running"
	case UpdateStatusSucceeded:
		return "succeeded"
	case UpdateStatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}
