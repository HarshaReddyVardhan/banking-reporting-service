package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditAction defines the type of audit action
type AuditAction string

const (
	ActionReportRequested  AuditAction = "REPORT_REQUESTED"
	ActionReportGenerated  AuditAction = "REPORT_GENERATED"
	ActionReportDownloaded AuditAction = "REPORT_DOWNLOADED"
	ActionReportAccessed   AuditAction = "REPORT_ACCESSED"
	ActionReportDenied     AuditAction = "REPORT_ACCESS_DENIED"
	ActionDashboardAccess  AuditAction = "DASHBOARD_ACCESSED"
	ActionDataExported     AuditAction = "DATA_EXPORTED"
)

// AuditEntry represents an immutable audit log entry
type AuditEntry struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	Timestamp    time.Time       `json:"timestamp" db:"timestamp"`
	UserID       uuid.UUID       `json:"user_id" db:"user_id"`
	UserRole     string          `json:"user_role" db:"user_role"`
	Action       AuditAction     `json:"action" db:"action"`
	ResourceType string          `json:"resource_type" db:"resource_type"`
	ResourceID   string          `json:"resource_id" db:"resource_id"`
	IPAddress    string          `json:"ip_address" db:"ip_address"`
	UserAgent    string          `json:"user_agent" db:"user_agent"`
	Details      json.RawMessage `json:"details" db:"details"`
	Success      bool            `json:"success" db:"success"`
	ErrorMessage *string         `json:"error_message,omitempty" db:"error_message"`
}

// AuditLogger handles SOX-compliant audit logging
type AuditLogger struct {
	db *pgxpool.Pool
}

func NewAuditLogger(db *pgxpool.Pool) *AuditLogger {
	return &AuditLogger{db: db}
}

// Log creates an immutable audit entry
func (a *AuditLogger) Log(ctx context.Context, entry *AuditEntry) error {
	entry.ID = uuid.New()
	entry.Timestamp = time.Now().UTC()

	query := `
		INSERT INTO audit_logs (id, timestamp, user_id, user_role, action, resource_type, resource_id, ip_address, user_agent, details, success, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := a.db.Exec(ctx, query,
		entry.ID, entry.Timestamp, entry.UserID, entry.UserRole,
		entry.Action, entry.ResourceType, entry.ResourceID,
		entry.IPAddress, entry.UserAgent, entry.Details,
		entry.Success, entry.ErrorMessage,
	)
	return err
}

// LogReportAccess logs a report access event
func (a *AuditLogger) LogReportAccess(ctx context.Context, userID uuid.UUID, role, reportID, ip, userAgent string, success bool, errMsg *string) error {
	details, _ := json.Marshal(map[string]string{"report_id": reportID})
	return a.Log(ctx, &AuditEntry{
		UserID:       userID,
		UserRole:     role,
		Action:       ActionReportAccessed,
		ResourceType: "report",
		ResourceID:   reportID,
		IPAddress:    ip,
		UserAgent:    userAgent,
		Details:      details,
		Success:      success,
		ErrorMessage: errMsg,
	})
}

// LogReportDownload logs a report download event
func (a *AuditLogger) LogReportDownload(ctx context.Context, userID uuid.UUID, role, reportID, ip, userAgent string) error {
	details, _ := json.Marshal(map[string]string{"report_id": reportID, "action": "download"})
	return a.Log(ctx, &AuditEntry{
		UserID:       userID,
		UserRole:     role,
		Action:       ActionReportDownloaded,
		ResourceType: "report",
		ResourceID:   reportID,
		IPAddress:    ip,
		UserAgent:    userAgent,
		Details:      details,
		Success:      true,
	})
}

// LogAccessDenied logs an access denied event (security monitoring)
func (a *AuditLogger) LogAccessDenied(ctx context.Context, userID uuid.UUID, role, resource, reason, ip, userAgent string) error {
	details, _ := json.Marshal(map[string]string{"resource": resource, "reason": reason})
	return a.Log(ctx, &AuditEntry{
		UserID:       userID,
		UserRole:     role,
		Action:       ActionReportDenied,
		ResourceType: "report",
		ResourceID:   resource,
		IPAddress:    ip,
		UserAgent:    userAgent,
		Details:      details,
		Success:      false,
		ErrorMessage: &reason,
	})
}
