package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Transaction represents the source data from the read replica
type Transaction struct {
	ID               uuid.UUID       `json:"id" db:"id"`
	UserID           uuid.UUID       `json:"user_id" db:"user_id"`
	FromAccountID    uuid.UUID       `json:"from_account_id" db:"from_account_id"`
	ToAccountID      uuid.UUID       `json:"to_account_id" db:"to_account_id"`
	Amount           decimal.Decimal `json:"amount" db:"amount"`
	Currency         string          `json:"currency" db:"currency"`
	Status           string          `json:"status" db:"status"`
	TransferType     string          `json:"transfer_type" db:"transfer_type"`
	InitiationMethod string          `json:"initiation_method" db:"initiation_method"`
	Reference        string          `json:"reference" db:"reference"`
	FraudScore       *float64        `json:"fraud_score,omitempty" db:"fraud_score"`
	FraudDecision    *string         `json:"fraud_decision,omitempty" db:"fraud_decision"`
	SourceIP         string          `json:"source_ip,omitempty" db:"source_ip"`
	DeviceID         string          `json:"device_id,omitempty" db:"device_id"`
	InitiatedAt      time.Time       `json:"initiated_at" db:"initiated_at"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
}

// TransactionAggregate represents aggregated data for ClickHouse
type TransactionAggregate struct {
	Timestamp    time.Time       `json:"timestamp"`
	TotalVolume  decimal.Decimal `json:"total_volume"`
	TotalCount   uint64          `json:"total_count"`
	AvgAmount    decimal.Decimal `json:"avg_amount"`
	FraudCount   uint64          `json:"fraud_count"`
	Currency     string          `json:"currency"`
	GeoCountry   string          `json:"geo_country"` // Derived from IP
	TransferType string          `json:"transfer_type"`
}

// DashboardData represents the executive summary data
type DashboardData struct {
	TotalVolume       decimal.Decimal `json:"total_volume"`
	TotalTransactions int64           `json:"total_transactions"`
	FraudRate         float64         `json:"fraud_rate"`
	ActiveUsers       int64           `json:"active_users"`
	VolumeTrend       float64         `json:"volume_trend"` // Percentage change
}

// FraudStats represents fraud monitoring statistics
type FraudStats struct {
	TotalAnalyzed     int64           `json:"total_analyzed"`
	FlaggedSuspicious int64           `json:"flagged_suspicious"`
	ConfirmedFraud    int64           `json:"confirmed_fraud"`
	PreventedLoss     decimal.Decimal `json:"prevented_loss"`
	TopFraudCountries []CountryStat   `json:"top_fraud_countries"`
}

type CountryStat struct {
	Country string `json:"country"`
	Count   int64  `json:"count"`
}
