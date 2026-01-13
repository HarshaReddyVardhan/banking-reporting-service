package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/banking/reporting-service/internal/domain"
	"github.com/banking/reporting-service/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock Repositories
type MockReportRepository struct {
	mock.Mock
}

func (m *MockReportRepository) Create(ctx context.Context, report *domain.Report) error {
	args := m.Called(ctx, report)
	return args.Error(0)
}

func (m *MockReportRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ReportStatus, path string, err *string) error {
	args := m.Called(ctx, id, status, path, err)
	return args.Error(0)
}

func (m *MockReportRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Report, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Report), args.Error(1)
}

type MockMetricRepository struct {
	mock.Mock
}

func (m *MockMetricRepository) RecordTransaction(ctx context.Context, metric *domain.TransactionEvent, latency int64) error {
	args := m.Called(ctx, metric, latency)
	return args.Error(0)
}

func (m *MockMetricRepository) GetHourlyAggregates(ctx context.Context, start, end time.Time) ([]domain.MetricAggregate, error) {
	args := m.Called(ctx, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.MetricAggregate), args.Error(1)
}

func TestRequestReport(t *testing.T) {
	mockRepo := new(MockReportRepository)
	mockMetricRepo := new(MockMetricRepository)
	svc := service.NewReportService(mockRepo, mockMetricRepo)

	ctx := context.Background()
	userID := uuid.New()
	req := &domain.ReportRequest{
		Type:      domain.ReportTypeTransactionSummary,
		Format:    domain.ReportFormatCSV,
		StartDate: time.Now().Add(-24 * time.Hour),
		EndDate:   time.Now(),
	}

	mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.Report")).Return(nil)

	report, err := svc.RequestReport(ctx, req, userID)

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, domain.ReportStatusPending, report.Status)
	mockRepo.AssertExpectations(t)
}
