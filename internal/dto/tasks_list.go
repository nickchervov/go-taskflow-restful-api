package dto

import "restful-taskflow/internal/domain"

type TasksListsOutput struct {
	Tasks []domain.Task `json:"tasks"`
}
