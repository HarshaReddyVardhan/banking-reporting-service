package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/banking/reporting-service/internal/api"
	"github.com/banking/reporting-service/internal/audit"
	"github.com/banking/reporting-service/internal/config"
	"github.com/banking/reporting-service/internal/event"
	"github.com/banking/reporting-service/internal/repository"
	"github.com/banking/reporting-service/internal/resilience"
	"github.com/banking/reporting-service/internal/security"
	"github.com/banking/reporting-service/internal/service"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

func main() {
	// 1. Initialize Logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	// 2. Load Config
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// 3. Initialize Infrastructure (Postgres)
	dbPool, err := repository.NewPostgresDB(cfg.Postgres)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer dbPool.Close()

	// 4. Initialize Repositories
	reportRepo := repository.NewReportRepository(dbPool)
	metricRepo := repository.NewMetricRepository(dbPool)

	// 5. Initialize Security Components
	rbac := security.NewRBACManager()
	auditLogger := audit.NewAuditLogger(dbPool)
	masker := security.NewDataMasker()

	// Initialize Encryption (key from env, base64 encoded)
	encKeyB64 := os.Getenv("REPORTING_SERVICE_ENCRYPTION_KEY")
	if encKeyB64 == "" {
		logger.Fatal("REPORTING_SERVICE_ENCRYPTION_KEY must be set")
	}
	encKey, err := base64.StdEncoding.DecodeString(encKeyB64)
	if err != nil {
		logger.Fatal("Failed to decode encryption key", zap.Error(err))
	}
	_, err = security.NewEncryptor(encKey)
	if err != nil {
		logger.Fatal("Failed to initialize encryptor", zap.Error(err))
	}

	// 6. Initialize Resilience
	resilienceManager := resilience.NewResilience(logger)
	resilienceManager.AddCircuitBreaker(resilience.CircuitBreakerConfig{
		Name:         "database",
		MaxRequests:  5,
		Interval:     10 * time.Second,
		Timeout:      30 * time.Second,
		FailureRatio: 0.5,
		MinRequests:  3,
	})
	resilienceManager.AddCircuitBreaker(resilience.CircuitBreakerConfig{
		Name:         "kafka",
		MaxRequests:  5,
		Interval:     10 * time.Second,
		Timeout:      60 * time.Second,
		FailureRatio: 0.5,
		MinRequests:  3,
	})

	// 7. Initialize Services
	reportService := service.NewReportService(reportRepo, metricRepo)

	// 8. Initialize Kafka Consumer with Idempotency
	idempotencyStore := event.NewIdempotencyStore(dbPool)
	kafkaConsumer := event.NewConsumer(metricRepo, idempotencyStore, resilienceManager, logger)
	go func() {
		logger.Info("Starting Kafka consumer...")
		if err := kafkaConsumer.Start(cfg.Kafka); err != nil {
			logger.Fatal("Kafka consumer failed", zap.Error(err))
		}
	}()

	// 9. Initialize API with Middleware
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.Secure()) // Security headers

	// Rate limiting
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))

	// Custom middleware
	authMiddleware := api.AuthMiddleware(cfg.Server.GatewaySecret)
	rbacMiddleware := api.RBACMiddleware(rbac, auditLogger)
	auditMiddleware := api.AuditMiddleware(auditLogger)

	handler := api.NewHandler(reportService, rbac, auditLogger, masker)
	handler.RegisterRoutes(e, authMiddleware, rbacMiddleware, auditMiddleware)

	// Health check (unauthenticated)
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "healthy"})
	})

	// 10. Start Server
	go func() {
		addr := fmt.Sprintf(":%d", cfg.Server.Port)
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	logger.Info("Server started", zap.Int("port", cfg.Server.Port))

	// 11. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		logger.Fatal("Server shutdown failed", zap.Error(err))
	}
	logger.Info("Server exited gracefully")
}
