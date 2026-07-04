package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func InitDb(ctx context.Context, dbURL string) (*pgxpool.Pool, error) {
	dbConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("creating db config: %w", err)
	}
	dbPool, err := pgxpool.NewWithConfig(ctx, dbConfig)
	if err != nil {
		return nil, fmt.Errorf("connecting postgres db: %w", err)
	}

	err = createTables(ctx, dbPool)
	if err != nil {
		return nil, err
	}
	return dbPool, nil
}

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
