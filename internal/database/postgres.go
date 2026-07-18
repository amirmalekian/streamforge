package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	MaxConns int
}

func (c Config) DSN() string {
	return "postgres://" + c.User + ":" + c.Password + "@" + c.Host + ":" + c.Port + "/" + c.Name
}

type DB struct {
	Pool *pgxpool.Pool
}

func Connect(cfg Config) *DB {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		panic(fmt.Sprintf("Failed to parse database config: %v", err))
	}

	poolConfig.MaxConns = int32(cfg.MaxConns)
	poolConfig.MinConns = int32(cfg.MaxConns / 4)
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to database: %v", err))
	}

	if err := pool.Ping(ctx); err != nil {
		panic(fmt.Sprintf("Failed to ping database: %v", err))
	}

	return &DB{Pool: pool}
}

func (db *DB) Close() {
	db.Pool.Close()
}
