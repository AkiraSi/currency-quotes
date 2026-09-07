package rate

import (
	"bufio"
	"bytes"
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

func newJob(client *cbr.Client) *job {
	return &job{client: client}
}

func (j *job) processMessage(data []byte) error {
	var msg messages.RateMsg
	if err := msg.Deserialize(bufio.NewReader(bytes.NewReader(data))); err != nil {
		return err
	}

	j.msg = msg

	return nil
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
