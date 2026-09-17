package dto

import (
	"restful-taskflow/internal/domain"

	"github.com/google/uuid"
)

type GetStatusResultInput struct {
	Id uuid.UUID
}

type GetStatusResultOutput struct {
	domain.TaskStatusResult
}
