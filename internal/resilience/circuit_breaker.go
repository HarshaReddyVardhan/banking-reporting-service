package resilience

import (
	"context"
	"errors"
	"time"

	"github.com/sony/gobreaker"
	"go.uber.org/zap"
)

// CircuitBreakerConfig holds configuration for circuit breaker
type CircuitBreakerConfig struct {
	Name         string
	MaxRequests  uint32        // Max requests allowed in half-open state
	Interval     time.Duration // Interval to clear counts
	Timeout      time.Duration // Duration in open state before half-open
	FailureRatio float64       // Ratio of failures to trip
	MinRequests  uint32        // Min requests before calculating ratio
}

// Resilience provides circuit breaker and retry patterns
type Resilience struct {
	breakers map[string]*gobreaker.CircuitBreaker
	logger   *zap.Logger
}

func NewResilience(logger *zap.Logger) *Resilience {
	return &Resilience{
		breakers: make(map[string]*gobreaker.CircuitBreaker),
		logger:   logger,
	}
}

// AddCircuitBreaker adds a circuit breaker for a service
func (r *Resilience) AddCircuitBreaker(cfg CircuitBreakerConfig) {
	settings := gobreaker.Settings{
		Name:        cfg.Name,
		MaxRequests: cfg.MaxRequests,
		Interval:    cfg.Interval,
		Timeout:     cfg.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			if counts.Requests < cfg.MinRequests {
				return false
			}
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return failureRatio >= cfg.FailureRatio
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			r.logger.Warn("Circuit breaker state change",
				zap.String("name", name),
				zap.String("from", from.String()),
				zap.String("to", to.String()),
			)
		},
	}
	r.breakers[cfg.Name] = gobreaker.NewCircuitBreaker(settings)
}

// Execute runs a function with circuit breaker protection
func (r *Resilience) Execute(name string, fn func() (interface{}, error)) (interface{}, error) {
	breaker, ok := r.breakers[name]
	if !ok {
		return nil, errors.New("circuit breaker not found: " + name)
	}
	return breaker.Execute(fn)
}

// RetryConfig holds configuration for retry logic
type RetryConfig struct {
	MaxAttempts     int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	Multiplier      float64
	RetryableErrors []error
}

// DefaultRetryConfig returns sensible defaults
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
	}
}

// RetryWithBackoff retries a function with exponential backoff
func RetryWithBackoff(ctx context.Context, cfg RetryConfig, fn func() error, logger *zap.Logger) error {
	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		if err := fn(); err != nil {
			lastErr = err
			logger.Warn("Operation failed, retrying",
				zap.Int("attempt", attempt),
				zap.Int("max_attempts", cfg.MaxAttempts),
				zap.Duration("next_delay", delay),
				zap.Error(err),
			)

			if attempt == cfg.MaxAttempts {
				break
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}

			delay = time.Duration(float64(delay) * cfg.Multiplier)
			if delay > cfg.MaxDelay {
				delay = cfg.MaxDelay
			}
		} else {
			return nil
		}
	}

	return lastErr
}

// Fallback executes a function with a fallback on error
func Fallback[T any](primary func() (T, error), fallback func() (T, error)) (T, error) {
	result, err := primary()
	if err != nil {
		return fallback()
	}
	return result, nil
}
