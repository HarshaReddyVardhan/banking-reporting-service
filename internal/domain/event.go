package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Inbound Events

type TransactionEvent struct {
	TransactionID uuid.UUID       `json:"transaction_id"`
	SourceAccount uuid.UUID       `json:"source_account"`
	TargetAccount uuid.UUID       `json:"target_account"`
	Amount        decimal.Decimal `json:"amount"`
	Currency      string          `json:"currency"`
	Type          string          `json:"type"` // TRANSFER, DEPOSIT, etc.
	Status        string          `json:"status"`
	Timestamp     time.Time       `json:"timestamp"`
}

type FraudAlertEvent struct {
	AlertID       uuid.UUID `json:"alert_id"`
	TransactionID uuid.UUID `json:"transaction_id"`
	Score         int       `json:"score"`
	Reason        string    `json:"reason"`
	Timestamp     time.Time `json:"timestamp"`
}

// Aggregates for Reporting

type MetricAggregate struct {
	Bucket      time.Time       `json:"bucket" db:"bucket"`
	Count       int64           `json:"count" db:"count"`
	TotalAmount decimal.Decimal `json:"total_amount" db:"total_amount"`
	AvgLatency  float64         `json:"avg_latency" db:"avg_latency"` // In ms
}
