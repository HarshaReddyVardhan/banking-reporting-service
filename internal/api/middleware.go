package api

import (
	"net/http"

	"github.com/banking/reporting-service/internal/audit"
	"github.com/banking/reporting-service/internal/security"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// UserContext holds authenticated user information
type UserContext struct {
	UserID uuid.UUID
	Role   security.Role
	IP     string
	Agent  string
}

// ContextKey for user context
type contextKey string

const UserContextKey contextKey = "user_context"

// AuthMiddleware validates Gateway Secret and extracts user context
func AuthMiddleware(gatewaySecret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 1. Verify Gateway Secret
			if gatewaySecret != "" {
				secret := c.Request().Header.Get("X-Gateway-Secret")
				if secret != gatewaySecret {
					return echo.NewHTTPError(http.StatusForbidden, "Invalid Gateway Secret")
				}
			}

			// 2. Extract User Context (Assumes Gateway has already validated JWT and injected headers)
			userIDStr := c.Request().Header.Get("X-User-ID")
			roleStr := c.Request().Header.Get("X-User-Role")

			if userIDStr == "" || roleStr == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing authentication headers")
			}

			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid user ID")
			}

			ctx := &UserContext{
				UserID: userID,
				Role:   security.Role(roleStr),
				IP:     c.RealIP(),
				Agent:  c.Request().UserAgent(),
			}

			c.Set(string(UserContextKey), ctx)
			return next(c)
		}
	}
}

// RBACMiddleware enforces role-based access control
func RBACMiddleware(rbac *security.RBACManager, auditLogger *audit.AuditLogger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userCtx, ok := c.Get(string(UserContextKey)).(*UserContext)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "User context not found")
			}

			// Check if role exists
			_, exists := rbac.GetPermission(userCtx.Role)
			if !exists {
				reason := "Unknown role: " + string(userCtx.Role)
				auditLogger.LogAccessDenied(c.Request().Context(), userCtx.UserID, string(userCtx.Role), c.Path(), reason, userCtx.IP, userCtx.Agent)
				return echo.NewHTTPError(http.StatusForbidden, "Access denied: unknown role")
			}

			return next(c)
		}
	}
}

// AuditMiddleware logs all API access
func AuditMiddleware(auditLogger *audit.AuditLogger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)

			userCtx, ok := c.Get(string(UserContextKey)).(*UserContext)
			if ok {
				success := err == nil && c.Response().Status < 400
				var errMsg *string
				if err != nil {
					msg := err.Error()
					errMsg = &msg
				}
				auditLogger.LogReportAccess(
					c.Request().Context(),
					userCtx.UserID,
					string(userCtx.Role),
					c.Path(),
					userCtx.IP,
					userCtx.Agent,
					success,
					errMsg,
				)
			}

			return err
		}
	}
}

// GetUserContext extracts user context from echo context
func GetUserContext(c echo.Context) (*UserContext, bool) {
	ctx, ok := c.Get(string(UserContextKey)).(*UserContext)
	return ctx, ok
}
