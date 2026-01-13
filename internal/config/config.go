// Package config provides configuration management for the reporting service.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the reporting service
type Config struct {
	Server        ServerConfig
	Postgres      DatabaseConfig
	ClickHouse    ClickHouseConfig
	Elasticsearch SearchConfig
	Kafka         KafkaConfig
	Auth          AuthConfig
	Logging       LoggingConfig
	Tracing       TracingConfig
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	GatewaySecret   string        `mapstructure:"gateway_secret"`
}

// DatabaseConfig holds PostgreSQL connection settings (Read Replica)
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// DSN returns the PostgreSQL connection string
func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.SSLMode,
	)
}

// ClickHouseConfig holds ClickHouse connection settings
type ClickHouseConfig struct {
	Addr            []string      `mapstructure:"addr"`
	Database        string        `mapstructure:"database"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	MaxExecution    int           `mapstructure:"max_execution_time"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// SearchConfig holds Elasticsearch connection settings
type SearchConfig struct {
	Addresses []string `mapstructure:"addresses"`
	Username  string   `mapstructure:"username"`
	Password  string   `mapstructure:"password"`
}

// KafkaConfig holds Kafka connection settings
type KafkaConfig struct {
	Brokers         []string `mapstructure:"brokers"`
	ConsumerGroupID string   `mapstructure:"consumer_group_id"`

	// Topics
	TransferCompletedTopic string `mapstructure:"transfer_completed_topic"`
	FraudDetectedTopic     string `mapstructure:"fraud_detected_topic"`
	UserRegisteredTopic    string `mapstructure:"user_registered_topic"`
}

// AuthConfig holds authentication settings
type AuthConfig struct {
	JWTPublicKeyPath string        `mapstructure:"jwt_public_key_path"`
	TokenExpiry      time.Duration `mapstructure:"token_expiry"`
	Issuer           string        `mapstructure:"issuer"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level         string `mapstructure:"level"`
	Format        string `mapstructure:"format"`
	OutputPath    string `mapstructure:"output_path"`
	EnablePIIMask bool   `mapstructure:"enable_pii_mask"`
}

// TracingConfig holds distributed tracing settings
type TracingConfig struct {
	Enabled      bool    `mapstructure:"enabled"`
	ServiceName  string  `mapstructure:"service_name"`
	OTLPEndpoint string  `mapstructure:"otlp_endpoint"`
	SampleRate   float64 `mapstructure:"sample_rate"`
}

// Load reads configuration from environment variables and config files
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Read from environment variables
	v.SetEnvPrefix("REPORTING_SERVICE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Try to read config file
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./configs")
	v.AddConfigPath("/etc/reporting-service")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8088)
	v.SetDefault("server.read_timeout", "15s")
	v.SetDefault("server.write_timeout", "15s")
	v.SetDefault("server.idle_timeout", "60s")
	v.SetDefault("server.shutdown_timeout", "30s")

	// Postgres defaults
	v.SetDefault("postgres.host", "localhost")
	v.SetDefault("postgres.port", 5433) // Replica port
	v.SetDefault("postgres.user", "postgres")
	v.SetDefault("postgres.password", "postgres")
	v.SetDefault("postgres.database", "banking_reporting")
	v.SetDefault("postgres.ssl_mode", "disable")
	v.SetDefault("postgres.max_open_conns", 50)
	v.SetDefault("postgres.max_idle_conns", 5)
	v.SetDefault("postgres.conn_max_lifetime", "1h")

	// ClickHouse defaults
	v.SetDefault("clickhouse.addr", []string{"localhost:9000"})
	v.SetDefault("clickhouse.database", "default")
	v.SetDefault("clickhouse.user", "default")
	v.SetDefault("clickhouse.password", "")
	v.SetDefault("clickhouse.max_execution_time", 60)
	v.SetDefault("clickhouse.max_open_conns", 10)
	v.SetDefault("clickhouse.max_idle_conns", 5)

	// Elasticsearch defaults
	v.SetDefault("elasticsearch.addresses", []string{"http://localhost:9200"})

	// Kafka defaults
	v.SetDefault("kafka.brokers", []string{"localhost:9092"})
	v.SetDefault("kafka.consumer_group_id", "reporting-service")
	v.SetDefault("kafka.transfer_completed_topic", "banking.transfers.completed")
	v.SetDefault("kafka.fraud_detected_topic", "banking.fraud.detected")
	v.SetDefault("kafka.user_registered_topic", "banking.users.registered")

	// Auth defaults
	v.SetDefault("auth.jwt_public_key_path", "./keys/jwt_public.pem")
	v.SetDefault("auth.token_expiry", "1h")
	v.SetDefault("auth.issuer", "banking-auth-service")

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.output_path", "stdout")
	v.SetDefault("logging.enable_pii_mask", true)

	// Tracing defaults
	v.SetDefault("tracing.enabled", true)
	v.SetDefault("tracing.service_name", "reporting-service")
	v.SetDefault("tracing.otlp_endpoint", "localhost:4317")
	v.SetDefault("tracing.sample_rate", 0.1)
}
