package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// TransactionRepository defines read-only access to the source of truth (Postgres)
type TransactionRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Transaction, error)
	GetByDateRange(ctx context.Context, start, end time.Time) ([]*Transaction, error)
	GetForCompliance(ctx context.Context, userID uuid.UUID) ([]*Transaction, error)
}

// AnalyticsRepository defines access to the aggregation engine (ClickHouse)
type AnalyticsRepository interface {
	SaveAggregate(ctx context.Context, agg *TransactionAggregate) error
	GetHourlyAggregates(ctx context.Context, start, end time.Time) ([]*TransactionAggregate, error)
	GetFraudStats(ctx context.Context, start, end time.Time) (*FraudStats, error)
}

// SearchRepository defines access to the search engine (Elasticsearch)
type SearchRepository interface {
	IndexTransaction(ctx context.Context, tx *Transaction) error
	Search(ctx context.Context, query string) ([]*Transaction, error)
}

// DashboardService defines the methods for retrieving dashboard data
type DashboardService interface {
	GetExecutiveSummary(ctx context.Context) (*DashboardData, error)
	GetFraudMonitoring(ctx context.Context) (*FraudStats, error)
}

// ReportService defines the methods for generating reports
// ReportService defines the methods for generating reports
type ReportService interface {
	RequestReport(ctx context.Context, req *ReportRequest, userID uuid.UUID) (*Report, error)
	GetReport(ctx context.Context, id uuid.UUID) (*Report, error)
}
