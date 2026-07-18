package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("PORT", "9090")
	t.Setenv("POSTGRES_HOST", "test-db")
	t.Setenv("POSTGRES_PORT", "5433")
	t.Setenv("POSTGRES_USER", "testuser")
	t.Setenv("POSTGRES_PASSWORD", "testpass")
	t.Setenv("POSTGRES_DB", "testdb")
	t.Setenv("POSTGRES_MAX_CONNS", "15")
	t.Setenv("REDIS_HOST", "test-redis")
	t.Setenv("REDIS_PORT", "6380")
	t.Setenv("REDIS_PASSWORD", "redispass")
	t.Setenv("REDIS_DB", "1")
	t.Setenv("REDIS_POOL_SIZE", "5")
	t.Setenv("RABBITMQ_HOST", "test-rabbit")
	t.Setenv("RABBITMQ_PORT", "5673")
	t.Setenv("RABBITMQ_USER", "testrabbit")
	t.Setenv("RABBITMQ_PASSWORD", "rabbitpass")
	t.Setenv("JWT_SECRET", "test-secret-key")
	t.Setenv("JWT_EXPIRY", "12h")
	t.Setenv("QUEUE_EXCHANGE", "test.exchange")
	t.Setenv("QUEUE_NAME", "test.queue")
	t.Setenv("QUEUE_ROUTING_KEY", "test.route")
	t.Setenv("WORKER_COUNT", "8")
	t.Setenv("RATE_LIMIT", "50")

	cfg := Load()

	assert.Equal(t, "test", cfg.App.Env)
	assert.Equal(t, "9090", cfg.Server.Port)
	assert.Equal(t, "test-db", cfg.Database.Host)
	assert.Equal(t, "5433", cfg.Database.Port)
	assert.Equal(t, "testuser", cfg.Database.User)
	assert.Equal(t, "testpass", cfg.Database.Password)
	assert.Equal(t, "testdb", cfg.Database.Name)
	assert.Equal(t, 15, cfg.Database.MaxConns)
	assert.Equal(t, "test-redis", cfg.Redis.Host)
	assert.Equal(t, "6380", cfg.Redis.Port)
	assert.Equal(t, "redispass", cfg.Redis.Password)
	assert.Equal(t, 1, cfg.Redis.DB)
	assert.Equal(t, 5, cfg.Redis.PoolSize)
	assert.Equal(t, "test-rabbit", cfg.RabbitMQ.Host)
	assert.Equal(t, "5673", cfg.RabbitMQ.Port)
	assert.Equal(t, "testrabbit", cfg.RabbitMQ.User)
	assert.Equal(t, "rabbitpass", cfg.RabbitMQ.Password)
	assert.Equal(t, "test-secret-key", cfg.JWT.Secret)
	assert.Equal(t, 12*time.Hour, cfg.JWT.Expiry)
	assert.Equal(t, "test.exchange", cfg.Queue.Exchange)
	assert.Equal(t, "test.queue", cfg.Queue.Queue)
	assert.Equal(t, "test.route", cfg.Queue.RoutingKey)
	assert.Equal(t, 8, cfg.Worker.WorkerCount)
	assert.Equal(t, 50, cfg.RateLimit)
}

func TestLoad_Defaults(t *testing.T) {
	unsetEnv(t, "APP_ENV")
	unsetEnv(t, "PORT")
	unsetEnv(t, "POSTGRES_HOST")
	unsetEnv(t, "POSTGRES_PORT")
	unsetEnv(t, "POSTGRES_USER")
	unsetEnv(t, "POSTGRES_PASSWORD")
	unsetEnv(t, "POSTGRES_DB")
	unsetEnv(t, "POSTGRES_MAX_CONNS")
	unsetEnv(t, "REDIS_HOST")
	unsetEnv(t, "REDIS_PORT")
	unsetEnv(t, "REDIS_PASSWORD")
	unsetEnv(t, "REDIS_DB")
	unsetEnv(t, "REDIS_POOL_SIZE")
	unsetEnv(t, "RABBITMQ_HOST")
	unsetEnv(t, "RABBITMQ_PORT")
	unsetEnv(t, "RABBITMQ_USER")
	unsetEnv(t, "RABBITMQ_PASSWORD")
	unsetEnv(t, "JWT_SECRET")
	unsetEnv(t, "JWT_EXPIRY")
	unsetEnv(t, "QUEUE_EXCHANGE")
	unsetEnv(t, "QUEUE_NAME")
	unsetEnv(t, "QUEUE_ROUTING_KEY")
	unsetEnv(t, "WORKER_COUNT")
	unsetEnv(t, "RATE_LIMIT")

	cfg := Load()

	assert.Equal(t, "development", cfg.App.Env)
	assert.Equal(t, "8080", cfg.Server.Port)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, "5432", cfg.Database.Port)
	assert.Equal(t, "streamforge", cfg.Database.User)
	assert.Equal(t, "streamforge", cfg.Database.Password)
	assert.Equal(t, "streamforge", cfg.Database.Name)
	assert.Equal(t, 20, cfg.Database.MaxConns)
	assert.Equal(t, "localhost", cfg.Redis.Host)
	assert.Equal(t, "6379", cfg.Redis.Port)
	assert.Equal(t, "", cfg.Redis.Password)
	assert.Equal(t, 0, cfg.Redis.DB)
	assert.Equal(t, 10, cfg.Redis.PoolSize)
	assert.Equal(t, "localhost", cfg.RabbitMQ.Host)
	assert.Equal(t, "5672", cfg.RabbitMQ.Port)
	assert.Equal(t, "streamforge", cfg.RabbitMQ.User)
	assert.Equal(t, "streamforge", cfg.RabbitMQ.Password)
	assert.Equal(t, "dev-secret-change-in-production", cfg.JWT.Secret)
	assert.Equal(t, 24*time.Hour, cfg.JWT.Expiry)
	assert.Equal(t, "streamforge.exchange", cfg.Queue.Exchange)
	assert.Equal(t, "media.processing.queue", cfg.Queue.Queue)
	assert.Equal(t, "media.process", cfg.Queue.RoutingKey)
	assert.Equal(t, 4, cfg.Worker.WorkerCount)
	assert.Equal(t, 100, cfg.RateLimit)
}

func TestDatabaseConfig_DSN(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     "5432",
		User:     "user",
		Password: "pass",
		Name:     "dbname",
	}

	expected := "postgres://user:pass@localhost:5432/dbname"
	assert.Equal(t, expected, cfg.DSN())
}

func TestRedisConfig_Addr(t *testing.T) {
	cfg := RedisConfig{
		Host: "localhost",
		Port: "6379",
	}

	assert.Equal(t, "localhost:6379", cfg.Addr())
}

func TestRabbitMQConfig_URI(t *testing.T) {
	cfg := RabbitMQConfig{
		Host:     "localhost",
		Port:     "5672",
		User:     "user",
		Password: "pass",
	}

	assert.Equal(t, "amqp://user:pass@localhost:5672/", cfg.URI())
}

func TestGetEnvInt(t *testing.T) {
	t.Setenv("TEST_INT", "42")

	val := getEnvInt("TEST_INT", 0)
	assert.Equal(t, 42, val)

	val = getEnvInt("NON_EXISTENT", 100)
	assert.Equal(t, 100, val)

	t.Setenv("TEST_INT_INVALID", "not-a-number")
	val = getEnvInt("TEST_INT_INVALID", 999)
	assert.Equal(t, 999, val)
}

func TestGetEnvDuration(t *testing.T) {
	t.Setenv("TEST_DURATION", "5m")

	val := getEnvDuration("TEST_DURATION", 0)
	assert.Equal(t, 5*time.Minute, val)

	val = getEnvDuration("NON_EXISTENT", 10*time.Second)
	assert.Equal(t, 10*time.Second, val)

	t.Setenv("TEST_DURATION_INVALID", "invalid")
	val = getEnvDuration("TEST_DURATION_INVALID", 7*time.Second)
	assert.Equal(t, 7*time.Second, val)
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	require.NoError(t, os.Unsetenv(key))
}
