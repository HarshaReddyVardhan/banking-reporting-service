package domain

import (
	"time"

	"github.com/google/uuid"
)

type ReportType string

const (
	ReportTypeTransactionSummary ReportType = "TRANSACTION_SUMMARY"
	ReportTypeUserActivity       ReportType = "USER_ACTIVITY"
	ReportTypeFinancialStatement ReportType = "FINANCIAL_STATEMENT"
	ReportTypeFraudAnalysis      ReportType = "FRAUD_ANALYSIS"
)

type ReportStatus string

const (
	ReportStatusPending    ReportStatus = "PENDING"
	ReportStatusProcessing ReportStatus = "PROCESSING"
	ReportStatusCompleted  ReportStatus = "COMPLETED"
	ReportStatusFailed     ReportStatus = "FAILED"
)

type ReportFormat string

const (
	ReportFormatCSV   ReportFormat = "CSV"
	ReportFormatPDF   ReportFormat = "PDF"
	ReportFormatExcel ReportFormat = "EXCEL"
	ReportFormatJSON  ReportFormat = "JSON"
)

type Report struct {
	ID          uuid.UUID    `json:"id" db:"id"`
	Type        ReportType   `json:"type" db:"type"`
	Status      ReportStatus `json:"status" db:"status"`
	Format      ReportFormat `json:"format" db:"format"`
	GeneratedBy uuid.UUID    `json:"generated_by" db:"generated_by"` // User ID who requested it
	Filters     string       `json:"filters" db:"filters"`           // JSON string of filters
	StoragePath string       `json:"storage_path" db:"storage_path"` // Path in S3 or local
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
	CompletedAt *time.Time   `json:"completed_at,omitempty" db:"completed_at"`
	Error       *string      `json:"error,omitempty" db:"error"`
}

type ReportRequest struct {
	Type      ReportType        `json:"type"`
	Format    ReportFormat      `json:"format"`
	StartDate time.Time         `json:"start_date"`
	EndDate   time.Time         `json:"end_date"`
	Filters   map[string]string `json:"filters"`
}
