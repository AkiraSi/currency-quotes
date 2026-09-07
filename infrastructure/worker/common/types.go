package common

type WorkerType byte

const (
	WorkerUnknown WorkerType = 0
	WorkerRate    WorkerType = 1
)

func (workerType WorkerType) String() string {
	switch workerType {
	case WorkerRate:
		return "rate"
	default:
		return "unknown"
	}
}
