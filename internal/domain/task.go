package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Task struct {
	ID          uuid.UUID       `json:"id,omitempty"`
	Type        string          `json:"type"`
	Status      string          `json:"status,omitempty"` // pending, running, completed, failed, cancelled
	Payload     json.RawMessage `json:"payload"`
	Result      json.RawMessage `json:"result,omitempty"`
	Error       string          `json:"error,omitempty"`
	Priority    int             `json:"priority"` // 1-5
	CreatedAt   time.Time       `json:"created_at,omitempty"`
	StartedAt   time.Time       `json:"started_at,omitempty"`
	CompletedAt time.Time       `json:"completed_at,omitempty"`
	RetryCount  int             `json:"retry_count,omitempty"`
	MaxRetries  int             `json:"max_retries,omitempty"` // max 3
}

func NewTask(Type, status string, payload json.RawMessage) Task {
	return Task{Type: Type, Status: status, Payload: payload, CreatedAt: time.Now(), MaxRetries: 3}
}

type TaskLog struct {
	ID        uuid.UUID `json:"id"`
	TaskID    uuid.UUID `json:"task_id"`
	Event     string    `json:"event"` //  -- created, started, completed, failed, retried
	Metadata  []byte    `json:"metadata"`
	CreatedAt time.Time `json:"created_at"`
}

type TaskStatusResult struct {
	ID     uuid.UUID       `json:"id"`
	Status string          `json:"status"`
	Result json.RawMessage `json:"result"`
}
