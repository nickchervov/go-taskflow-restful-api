package dto

import (
	"restful-taskflow/internal/domain"

	"github.com/google/uuid"
)

type CreateTaskInput struct {
	domain.Task
}

type CreateTaskOutput struct {
	Id uuid.UUID `json:"id"`
}
