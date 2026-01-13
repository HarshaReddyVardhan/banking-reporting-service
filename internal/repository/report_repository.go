package repository

import (
	"context"

	"github.com/banking/reporting-service/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportRepository interface {
	Create(ctx context.Context, report *domain.Report) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Report, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ReportStatus, path string, errMsg *string) error
}

type postgresReportRepo struct {
	db *pgxpool.Pool
}

func NewReportRepository(db *pgxpool.Pool) ReportRepository {
	return &postgresReportRepo{db: db}
}

func (r *postgresReportRepo) Create(ctx context.Context, report *domain.Report) error {
	query := `
		INSERT INTO reports (id, type, status, format, generated_by, filters, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(ctx, query,
		report.ID, report.Type, report.Status, report.Format,
		report.GeneratedBy, report.Filters, report.CreatedAt,
	)
	return err
}

func (r *postgresReportRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Report, error) {
	query := `SELECT id, type, status, format, generated_by, filters, storage_path, created_at, completed_at, error FROM reports WHERE id = $1`

	var report domain.Report
	err := r.db.QueryRow(ctx, query, id).Scan(
		&report.ID, &report.Type, &report.Status, &report.Format,
		&report.GeneratedBy, &report.Filters, &report.StoragePath,
		&report.CreatedAt, &report.CompletedAt, &report.Error,
	)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrReportNotFound
	}
	if err != nil {
		return nil, err
	}
	return &report, nil
}

func (r *postgresReportRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ReportStatus, path string, errMsg *string) error {
	query := `
		UPDATE reports 
		SET status = $1, storage_path = $2, error = $3, completed_at = CASE WHEN $1 = 'COMPLETED' THEN NOW() ELSE NULL END
		WHERE id = $4
	`
	_, err := r.db.Exec(ctx, query, status, path, errMsg, id)
	return err
}
