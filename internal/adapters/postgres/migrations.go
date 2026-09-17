package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func createTables(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, `
	create table if not exists tasks (
	id UUID primary key,
	type VARCHAR(50) DEFAULT '',
	status VARCHAR(20) DEFAULT '',
	payload JSONB DEFAULT NULL,
	result JSONB DEFAULT NULL,
	error TEXT DEFAULT '',
	priority INT DEFAULT 0,
	created_at TIMESTAMP,
	started_at TIMESTAMP,
	completed_at TIMESTAMP,
	retry_count INT default 0 NOT NULL,
	max_retries INT default 3 NOT NULL
	);`)
	if err != nil {
		return fmt.Errorf("creating tasks table: %w", err)
	}
	return nil
}
