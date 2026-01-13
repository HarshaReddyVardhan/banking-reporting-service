package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/banking/reporting-service/internal/domain"
	"github.com/banking/reporting-service/internal/report"
	"github.com/banking/reporting-service/internal/repository"
	"github.com/google/uuid"
)

type ReportService struct {
	reportRepo repository.ReportRepository
	metricRepo repository.MetricRepository
	factory    *report.GeneratorFactory
}

func NewReportService(rRepo repository.ReportRepository, mRepo repository.MetricRepository) *ReportService {
	return &ReportService{
		reportRepo: rRepo,
		metricRepo: mRepo,
		factory:    report.NewGeneratorFactory(),
	}
}

func (s *ReportService) RequestReport(ctx context.Context, req *domain.ReportRequest, userID uuid.UUID) (*domain.Report, error) {
	filtersJSON, err := json.Marshal(req.Filters)
	if err != nil {
		return nil, err
	}

	report := &domain.Report{
		ID:          uuid.New(),
		Type:        req.Type,
		Status:      domain.ReportStatusPending,
		Format:      req.Format,
		GeneratedBy: userID,
		Filters:     string(filtersJSON),
		CreatedAt:   time.Now(),
	}

	if err := s.reportRepo.Create(ctx, report); err != nil {
		return nil, err
	}

	// Trigger async generation (in a real system, publish to queue)
	go func() {
		// Create a background context
		bgCtx := context.Background()
		s.GenerateReport(bgCtx, report.ID, req)
	}()

	return report, nil
}

func (s *ReportService) GenerateReport(ctx context.Context, reportID uuid.UUID, req *domain.ReportRequest) {
	// 1. Update status to PROCESSING
	s.reportRepo.UpdateStatus(ctx, reportID, domain.ReportStatusProcessing, "", nil)

	var err error
	defer func() {
		if err != nil {
			errMsg := err.Error()
			s.reportRepo.UpdateStatus(ctx, reportID, domain.ReportStatusFailed, "", &errMsg)
		}
	}()

	// 2. Fetch Data
	var data interface{}
	switch req.Type {
	case domain.ReportTypeTransactionSummary:
		data, err = s.metricRepo.GetHourlyAggregates(ctx, req.StartDate, req.EndDate)
	default:
		err = fmt.Errorf("unsupported report type: %s", req.Type)
	}

	if err != nil {
		return
	}

	// 3. Generate Report File
	gen := s.factory.GetGenerator(req.Format)
	fileName := fmt.Sprintf("report_%s_%s.%s", req.Type, reportID.String(), s.getExtension(req.Format))
	filePath := filepath.Join("generated_reports", fileName) // Ensure this dir exists

	// Create directory if not exists
	if err = os.MkdirAll("generated_reports", 0755); err != nil {
		return
	}

	var file *os.File
	file, err = os.Create(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	var r *domain.Report
	r, err = s.reportRepo.GetByID(ctx, reportID)
	if err != nil {
		return
	}

	if err = gen.Generate(ctx, r, data, file); err != nil {
		return
	}

	// 4. Update status to COMPLETED
	err = s.reportRepo.UpdateStatus(ctx, reportID, domain.ReportStatusCompleted, filePath, nil)
}

func (s *ReportService) getExtension(format domain.ReportFormat) string {
	switch format {
	case domain.ReportFormatCSV:
		return "csv"
	case domain.ReportFormatExcel:
		return "xlsx"
	case domain.ReportFormatPDF:
		return "pdf"
	case domain.ReportFormatJSON:
		return "json"
	default:
		return "txt"
	}
}

func (s *ReportService) GetReport(ctx context.Context, id uuid.UUID) (*domain.Report, error) {
	return s.reportRepo.GetByID(ctx, id)
}
