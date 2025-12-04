package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/uzzalhcse/crawlify/microservices/shared/config"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

// DB wraps the PostgreSQL connection pool
type DB struct {
	Pool *pgxpool.Pool
}

// NewDB creates a new database connection with optimized pooling
func NewDB(cfg *config.DatabaseConfig) (*DB, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	// Connection pool settings - optimized for microservices
	poolConfig.MaxConns = int32(cfg.MaxConnections)
	poolConfig.MinConns = 1
	poolConfig.MaxConnLifetime = time.Duration(cfg.ConnMaxLifetime) * time.Second
	poolConfig.MaxConnIdleTime = 60 * time.Second
	poolConfig.HealthCheckPeriod = 30 * time.Second
	poolConfig.ConnConfig.ConnectTimeout = 10 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	logger.Info("Database connection established",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.Database),
		zap.Int("max_connections", cfg.MaxConnections),
	)

	return &DB{Pool: pool}, nil
}

// Close closes the database connection pool
func (db *DB) Close() {
	db.Pool.Close()
	logger.Info("Database connection closed")
}

// Health checks database connectivity
func (db *DB) Health(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// Acquire gets a connection from the pool
func (db *DB) Acquire(ctx context.Context) (*pgxpool.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return db.Pool.Acquire(ctx)
}
