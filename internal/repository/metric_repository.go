package repository

import (
	"context"
	"time"

	"github.com/banking/reporting-service/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MetricRepository interface {
	RecordTransaction(ctx context.Context, metric *domain.TransactionEvent, latency int64) error
	GetHourlyAggregates(ctx context.Context, start, end time.Time) ([]domain.MetricAggregate, error)
}

type postgresMetricRepo struct {
	db *pgxpool.Pool
}

func NewMetricRepository(db *pgxpool.Pool) MetricRepository {
	return &postgresMetricRepo{db: db}
}

func (r *postgresMetricRepo) RecordTransaction(ctx context.Context, tx *domain.TransactionEvent, latency int64) error {
	query := `
		INSERT INTO transaction_metrics (time, transaction_id, amount, currency, type, status, latency_ms, source_account, target_account)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.Exec(ctx, query,
		tx.Timestamp, tx.TransactionID, tx.Amount, tx.Currency,
		tx.Type, tx.Status, latency, tx.SourceAccount, tx.TargetAccount,
	)
	return err
}

func (r *postgresMetricRepo) GetHourlyAggregates(ctx context.Context, start, end time.Time) ([]domain.MetricAggregate, error) {
	query := `
		SELECT 
			time_bucket('1 hour', time) AS bucket,
			COUNT(*) AS count,
			SUM(amount) AS total_amount,
			AVG(latency_ms) AS avg_latency
		FROM transaction_metrics
		WHERE time BETWEEN $1 AND $2
		GROUP BY bucket
		ORDER BY bucket ASC
	`
	rows, err := r.db.Query(ctx, query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var aggs []domain.MetricAggregate
	for rows.Next() {
		var a domain.MetricAggregate
		if err := rows.Scan(&a.Bucket, &a.Count, &a.TotalAmount, &a.AvgLatency); err != nil {
			return nil, err
		}
		aggs = append(aggs, a)
	}
	return aggs, nil
}
