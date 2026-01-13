package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/banking/reporting-service/internal/audit"
	"github.com/banking/reporting-service/internal/domain"
	"github.com/banking/reporting-service/internal/security"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// ReportService defines the interface for report operations
type ReportServicer interface {
	RequestReport(ctx context.Context, req *domain.ReportRequest, userID uuid.UUID) (*domain.Report, error)
	GetReport(ctx context.Context, id uuid.UUID) (*domain.Report, error)
}

type Handler struct {
	reportService ReportServicer
	rbac          *security.RBACManager
	auditLogger   *audit.AuditLogger
	masker        *security.DataMasker
}

func NewHandler(
	reportService ReportServicer,
	rbac *security.RBACManager,
	auditLogger *audit.AuditLogger,
	masker *security.DataMasker,
) *Handler {
	return &Handler{
		reportService: reportService,
		rbac:          rbac,
		auditLogger:   auditLogger,
		masker:        masker,
	}
}

func (h *Handler) RegisterRoutes(e *echo.Echo, authMiddleware, rbacMiddleware, auditMiddleware echo.MiddlewareFunc) {
	v1 := e.Group("/api/v1")

	// Apply middleware chain: Auth -> RBAC -> Audit
	v1.Use(authMiddleware)
	v1.Use(rbacMiddleware)
	v1.Use(auditMiddleware)

	v1.POST("/reports", h.RequestReport)
	v1.GET("/reports/:id", h.GetReportStatus)
	v1.GET("/reports/:id/download", h.DownloadReport)
	v1.GET("/dashboard/system", h.GetSystemDashboard)
}

func (h *Handler) RequestReport(c echo.Context) error {
	userCtx, ok := GetUserContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "User context not found")
	}

	var req domain.ReportRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	// RBAC: Check if user can access this report type
	if !h.rbac.CanAccessReportType(userCtx.Role, req.Type) {
		reason := fmt.Sprintf("Role %s cannot access report type %s", userCtx.Role, req.Type)
		h.auditLogger.LogAccessDenied(c.Request().Context(), userCtx.UserID, string(userCtx.Role), string(req.Type), reason, userCtx.IP, userCtx.Agent)
		return echo.NewHTTPError(http.StatusForbidden, "Access denied: insufficient permissions for this report type")
	}

	report, err := h.reportService.RequestReport(c.Request().Context(), &req, userCtx.UserID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Internal server error")
	}

	return c.JSON(http.StatusAccepted, report)
}

func (h *Handler) GetReportStatus(c echo.Context) error {
	userCtx, ok := GetUserContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "User context not found")
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid report ID")
	}

	report, err := h.reportService.GetReport(c.Request().Context(), id)
	if errors.Is(err, domain.ErrReportNotFound) {
		return echo.NewHTTPError(http.StatusNotFound, "Report not found")
	}
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Internal server error")
	}

	// RBAC: Check if user can access this report type
	if !h.rbac.CanAccessReportType(userCtx.Role, report.Type) {
		reason := fmt.Sprintf("Role %s cannot access report type %s", userCtx.Role, report.Type)
		h.auditLogger.LogAccessDenied(c.Request().Context(), userCtx.UserID, string(userCtx.Role), idStr, reason, userCtx.IP, userCtx.Agent)
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	return c.JSON(http.StatusOK, report)
}

func (h *Handler) DownloadReport(c echo.Context) error {
	userCtx, ok := GetUserContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "User context not found")
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid report ID")
	}

	report, err := h.reportService.GetReport(c.Request().Context(), id)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Internal server error")
	}

	// RBAC: Check if user can download
	if !h.rbac.CanDownload(userCtx.Role) {
		reason := fmt.Sprintf("Role %s cannot download reports", userCtx.Role)
		h.auditLogger.LogAccessDenied(c.Request().Context(), userCtx.UserID, string(userCtx.Role), idStr, reason, userCtx.IP, userCtx.Agent)
		return echo.NewHTTPError(http.StatusForbidden, "Access denied: download not permitted")
	}

	// RBAC: Check report type access
	if !h.rbac.CanAccessReportType(userCtx.Role, report.Type) {
		reason := fmt.Sprintf("Role %s cannot access report type %s", userCtx.Role, report.Type)
		h.auditLogger.LogAccessDenied(c.Request().Context(), userCtx.UserID, string(userCtx.Role), idStr, reason, userCtx.IP, userCtx.Agent)
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	if report.Status != domain.ReportStatusCompleted {
		return echo.NewHTTPError(http.StatusBadRequest, "Report is not ready yet")
	}

	if _, err := os.Stat(report.StoragePath); os.IsNotExist(err) {
		return echo.NewHTTPError(http.StatusNotFound, "File not found on server")
	}

	// Audit: Log download
	h.auditLogger.LogReportDownload(c.Request().Context(), userCtx.UserID, string(userCtx.Role), idStr, userCtx.IP, userCtx.Agent)

	return c.Attachment(report.StoragePath, fmt.Sprintf("report_%s.%s", report.ID, report.Format))
}

func (h *Handler) GetSystemDashboard(c echo.Context) error {
	userCtx, ok := GetUserContext(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "User context not found")
	}

	// Only Operations and Admin can see system dashboard
	if userCtx.Role != security.RoleAdmin && userCtx.Role != security.RoleOperationsTeam {
		reason := fmt.Sprintf("Role %s cannot access system dashboard", userCtx.Role)
		h.auditLogger.LogAccessDenied(c.Request().Context(), userCtx.UserID, string(userCtx.Role), "system_dashboard", reason, userCtx.IP, userCtx.Agent)
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	// TODO: Implement real dashboard aggregation
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":             "ok",
		"transactions_today": 1234,
		"error_rate":         0.01,
		"avg_latency_ms":     45,
	})
}
