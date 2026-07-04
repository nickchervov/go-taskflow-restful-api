package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

func SendEmail(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	time.Sleep(300 * time.Millisecond)
	result, err := json.Marshal(map[string]string{"result": "письмо отправлено"})
	if err != nil {
		return nil, fmt.Errorf("serializing result of worker: %w", err)
	}
	return result, nil
}
