package rate

import (
	"errors"

	"currency-quotes/common/messages"
	"currency-quotes/pkg/clients/cbr"
)

var errUpdateQueueFull = errors.New("update queue is full")

type job struct {
	msg    messages.RateMsg
	client *cbr.Client
}

type updateTask struct {
	msg    messages.RateMsg
	client *cbr.Client
}

func newJob(msg messages.RateMsg, client *cbr.Client) *job {
	return &job{
		msg:    msg,
		client: client,
	}
}

func (j *job) CreateTasks(queue chan<- updateTask) error {
	task := updateTask{
		msg:    j.msg,
		client: j.client,
	}

	select {
	case queue <- task:
		return nil
	default:
		return errUpdateQueueFull
	}
}
