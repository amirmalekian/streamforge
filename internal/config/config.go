package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	App      AppConfig
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	RabbitMQ RabbitMQConfig
	JWT      JWTConfig
	Queue    QueueConfig
	Worker   WorkerConfig
	RateLimit int
}

type AppConfig struct {
	Env string
}

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	MaxConns int
}

func (d DatabaseConfig) DSN() string {
	return "postgres://" + d.User + ":" + d.Password + "@" + d.Host + ":" + d.Port + "/" + d.Name
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	PoolSize int
}

func (r RedisConfig) Addr() string {
	return r.Host + ":" + r.Port
}

type RabbitMQConfig struct {
	Host     string
	Port     string
	User     string
	Password string
}

func (r RabbitMQConfig) URI() string {
	return "amqp://" + r.User + ":" + r.Password + "@" + r.Host + ":" + r.Port + "/"
}

type JWTConfig struct {
	Secret string
	Expiry time.Duration
}

type QueueConfig struct {
	Exchange   string
	Queue      string
	RoutingKey string
}

type WorkerConfig struct {
	WorkerCount int
}

func Load() *Config {
	return &Config{
		App: AppConfig{
			Env: getEnv("APP_ENV", "development"),
		},
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnv("POSTGRES_PORT", "5432"),
			User:     getEnv("POSTGRES_USER", "streamforge"),
			Password: getEnv("POSTGRES_PASSWORD", "streamforge"),
			Name:     getEnv("POSTGRES_DB", "streamforge"),
			MaxConns: getEnvInt("POSTGRES_MAX_CONNS", 20),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
			PoolSize: getEnvInt("REDIS_POOL_SIZE", 10),
		},
		RabbitMQ: RabbitMQConfig{
			Host:     getEnv("RABBITMQ_HOST", "localhost"),
			Port:     getEnv("RABBITMQ_PORT", "5672"),
			User:     getEnv("RABBITMQ_USER", "streamforge"),
			Password: getEnv("RABBITMQ_PASSWORD", "streamforge"),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", "dev-secret-change-in-production"),
			Expiry: getEnvDuration("JWT_EXPIRY", 24*time.Hour),
		},
		Queue: QueueConfig{
			Exchange:   getEnv("QUEUE_EXCHANGE", "streamforge.exchange"),
			Queue:      getEnv("QUEUE_NAME", "media.processing.queue"),
			RoutingKey: getEnv("QUEUE_ROUTING_KEY", "media.process"),
		},
		Worker: WorkerConfig{
			WorkerCount: getEnvInt("WORKER_COUNT", 4),
		},
		RateLimit: getEnvInt("RATE_LIMIT", 100),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}