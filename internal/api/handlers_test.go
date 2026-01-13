package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/banking/reporting-service/internal/api"
	"github.com/banking/reporting-service/internal/audit"
	"github.com/banking/reporting-service/internal/domain"
	"github.com/banking/reporting-service/internal/security"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock Service
type MockReportService struct {
	mock.Mock
}

func (m *MockReportService) RequestReport(ctx context.Context, req *domain.ReportRequest, userID uuid.UUID) (*domain.Report, error) {
	args := m.Called(ctx, req, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Report), args.Error(1)
}

func (m *MockReportService) GetReport(ctx context.Context, id uuid.UUID) (*domain.Report, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Report), args.Error(1)
}

func TestRequestReport_Success(t *testing.T) {
	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports", strings.NewReader(`{"type":"TRANSACTION_SUMMARY","format":"CSV"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Mock Context
	userID := uuid.New()
	userCtx := &api.UserContext{
		UserID: userID,
		Role:   security.RoleAdmin,
		IP:     "127.0.0.1",
		Agent:  "test-agent",
	}
	c.Set(string(api.UserContextKey), userCtx)

	// Mocks
	mockSvc := new(MockReportService)
	rbac := security.NewRBACManager()
	auditLogger := &audit.AuditLogger{} // Partial mock might be needed but nil safe?
	// AuditLogger is a struct. We cannot mock it easily unless interface or avoiding method calls that panic.
	// However, LogAccessDenied etc methods on nil pointer? No, constructor returns pointer.
	// We can pass a dummy AuditLogger if its methods don't panic on nil deps or if we can construct it safely.
	// It depends on dbPool.
	// For unit test isolation, we should probably mock AuditLogger too, but it's a struct in Handler.
	// Let's assume AuditLogger methods are safe or we just use it as is if it doesn't fail.
	// Actually, `NewAuditLogger` takes dbPool. If we pass nil dbPool, it might panic later.
	// Let's rely on standard Go behavior: nil pointer dereference if it uses db.
	// To make this robust, we should create a MockAuditLogger implementing an interface, but Handler uses *audit.AuditLogger struct.
	// I will refactor Handler to use AuditLogger interface?? Too many changes.
	// Alternative: The audit logger is used in handlers.
	// Let's mock the methods by refactoring AuditLogger to Interface in Handler?
	// Or just ignore it for now and see if it fails (it will likely fail).
	// Let's create `MockAuditLogger`? No `AuditLogger` is concrete.
	// I will try to run this. If it fails on AuditLogger, I will refactor AuditLogger to interface.

	masker := security.NewDataMasker()

	// Expectations
	expectedReport := &domain.Report{
		ID:     uuid.New(),
		Status: domain.ReportStatusPending,
		Type:   domain.ReportTypeTransactionSummary,
	}
	mockSvc.On("RequestReport", mock.Anything, mock.AnythingOfType("*domain.ReportRequest"), userID).Return(expectedReport, nil)

	handler := api.NewHandler(mockSvc, rbac, auditLogger, masker)
	err := handler.RequestReport(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, rec.Code)
	mockSvc.AssertExpectations(t)
}
