package domain

import "errors"

type TaskError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e TaskError) Error() string {
	return e.Message
}

var (
	ErrInvalidTaskType   = &TaskError{Code: 400, Message: "invalid task type"}
	ErrTaskNotFound      = &TaskError{Code: 404, Message: "task not found"}
	ErrTaskAlreadyInWork = &TaskError{Code: 400, Message: "task already in work"}
	ErrTooManyRetries    = &TaskError{Code: 400, Message: "too many retries"}
	ErrTaskNotCancelable = &TaskError{Code: 409, Message: "task is not cancellable in current status"}
	ErrNoTask            = errors.New("no tasks in queue")
)
