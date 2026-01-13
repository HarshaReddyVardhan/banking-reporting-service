package event

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// IdempotencyStore handles exactly-once semantics for Kafka messages
type IdempotencyStore struct {
	db *pgxpool.Pool
}

func NewIdempotencyStore(db *pgxpool.Pool) *IdempotencyStore {
	return &IdempotencyStore{db: db}
}

// IsProcessed checks if a message has already been processed
func (s *IdempotencyStore) IsProcessed(ctx context.Context, topic string, partition int32, offset int64) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM processed_messages WHERE topic = $1 AND partition = $2 AND offset_id = $3)`
	var exists bool
	err := s.db.QueryRow(ctx, query, topic, partition, offset).Scan(&exists)
	return exists, err
}

// MarkProcessed marks a message as processed (with transaction support)
func (s *IdempotencyStore) MarkProcessed(ctx context.Context, topic string, partition int32, offset int64, messageID uuid.UUID) error {
	query := `
		INSERT INTO processed_messages (id, topic, partition, offset_id, processed_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (topic, partition, offset_id) DO NOTHING
	`
	_, err := s.db.Exec(ctx, query, messageID, topic, partition, offset, time.Now().UTC())
	return err
}

// ProcessWithIdempotency wraps message processing with exactly-once semantics
func (s *IdempotencyStore) ProcessWithIdempotency(ctx context.Context, topic string, partition int32, offset int64, processor func() error) error {
	// Check if already processed
	processed, err := s.IsProcessed(ctx, topic, partition, offset)
	if err != nil {
		return fmt.Errorf("idempotency check failed: %w", err)
	}
	if processed {
		return nil // Skip, already processed
	}

	// Process the message
	if err := processor(); err != nil {
		return err
	}

	// Mark as processed
	return s.MarkProcessed(ctx, topic, partition, offset, uuid.New())
}
