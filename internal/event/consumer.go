package event

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/IBM/sarama"
	"github.com/banking/reporting-service/internal/config"
	"github.com/banking/reporting-service/internal/domain"
	"github.com/banking/reporting-service/internal/repository"
	"github.com/banking/reporting-service/internal/resilience"
	"go.uber.org/zap"
)

type Consumer struct {
	ready            chan bool
	metricRepo       repository.MetricRepository
	idempotencyStore *IdempotencyStore
	resilience       *resilience.Resilience
	logger           *zap.Logger
}

func NewConsumer(
	repo repository.MetricRepository,
	idempotencyStore *IdempotencyStore,
	resilienceManager *resilience.Resilience,
	logger *zap.Logger,
) *Consumer {
	return &Consumer{
		ready:            make(chan bool),
		metricRepo:       repo,
		idempotencyStore: idempotencyStore,
		resilience:       resilienceManager,
		logger:           logger,
	}
}

func (c *Consumer) Start(cfg config.KafkaConfig) error {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategySticky()}
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	// Enable manual offset commit for exactly-once semantics
	saramaConfig.Consumer.Offsets.AutoCommit.Enable = false

	client, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.ConsumerGroupID, saramaConfig)
	if err != nil {
		return fmt.Errorf("error creating consumer group: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			if err := client.Consume(ctx, []string{"banking.transfers.completed", "banking.fraud.detected"}, c); err != nil {
				c.logger.Error("Error from consumer", zap.Error(err))
			}
			if ctx.Err() != nil {
				return
			}
			c.ready = make(chan bool)
		}
	}()

	<-c.ready
	c.logger.Info("Kafka consumer started successfully")

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-ctx.Done():
		c.logger.Info("Terminating: context cancelled")
	case <-sigterm:
		c.logger.Info("Terminating: via signal")
	}
	cancel()
	wg.Wait()
	if err = client.Close(); err != nil {
		return fmt.Errorf("error closing client: %w", err)
	}
	return nil
}

func (c *Consumer) Setup(sarama.ConsumerGroupSession) error {
	close(c.ready)
	return nil
}

func (c *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim processes messages with exactly-once semantics and error handling
func (c *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		c.logger.Debug("Message received",
			zap.String("topic", message.Topic),
			zap.Int32("partition", message.Partition),
			zap.Int64("offset", message.Offset),
		)

		// Process with idempotency check
		err := c.idempotencyStore.ProcessWithIdempotency(
			session.Context(),
			message.Topic,
			message.Partition,
			message.Offset,
			func() error {
				return c.handleMessageWithRetry(session.Context(), message)
			},
		)

		if err != nil {
			c.logger.Error("Failed to process message after retries",
				zap.String("topic", message.Topic),
				zap.Int64("offset", message.Offset),
				zap.Error(err),
			)
			// In production: send to DLQ
			continue
		}

		// Commit offset only after successful processing
		session.MarkMessage(message, "")
		session.Commit()
	}
	return nil
}

func (c *Consumer) handleMessageWithRetry(ctx context.Context, msg *sarama.ConsumerMessage) error {
	cfg := resilience.DefaultRetryConfig()
	return resilience.RetryWithBackoff(ctx, cfg, func() error {
		return c.handleMessage(ctx, msg)
	}, c.logger)
}

func (c *Consumer) handleMessage(ctx context.Context, msg *sarama.ConsumerMessage) error {
	switch msg.Topic {
	case "banking.transfers.completed":
		var event domain.TransactionEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			c.logger.Error("Failed to unmarshal transaction event", zap.Error(err))
			return err
		}
		// Use circuit breaker for DB operations
		_, err := c.resilience.Execute("database", func() (interface{}, error) {
			return nil, c.metricRepo.RecordTransaction(ctx, &event, 0)
		})
		return err

	case "banking.fraud.detected":
		c.logger.Info("Fraud event received (processing not yet implemented)")
		return nil

	default:
		c.logger.Warn("Unknown topic", zap.String("topic", msg.Topic))
		return nil
	}
}
