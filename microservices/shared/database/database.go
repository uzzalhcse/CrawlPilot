package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/uzzalhcse/crawlify/microservices/shared/config"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

// DB wraps the PostgreSQL connection pool
type DB struct {
	Pool *pgxpool.Pool
	cfg  *config.DatabaseConfig
}

// NewDB creates a new database connection with optimized pooling
// Supports both direct PostgreSQL and PgBouncer connections
func NewDB(cfg *config.DatabaseConfig) (*DB, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	// Connection pool settings - optimized for microservices
	poolConfig.MaxConns = int32(cfg.MaxConnections)
	poolConfig.MinConns = int32(cfg.MaxIdleConns) // Use idle conns as min
	poolConfig.MaxConnLifetime = time.Duration(cfg.ConnMaxLifetime) * time.Second
	poolConfig.MaxConnIdleTime = 30 * time.Second // Shorter for PgBouncer
	poolConfig.HealthCheckPeriod = 15 * time.Second
	poolConfig.ConnConfig.ConnectTimeout = 10 * time.Second

	// PgBouncer compatibility: Disable prepared statements
	// Required when using PgBouncer in "transaction" pool mode
	// Prepared statements don't work across pooled connections
	poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	// Create pool with retry logic
	var pool *pgxpool.Pool
	maxRetries := 5
	retryDelay := time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		pool, err = pgxpool.NewWithConfig(ctx, poolConfig)
		cancel()

		if err == nil {
			// Test connection
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			pingErr := pool.Ping(ctx)
			cancel()

			if pingErr == nil {
				break // Success
			}
			pool.Close()
			err = pingErr
		}

		if attempt < maxRetries {
			logger.Warn("Database connection failed, retrying...",
				zap.Int("attempt", attempt),
				zap.Int("max_retries", maxRetries),
				zap.Duration("retry_delay", retryDelay),
				zap.Error(err),
			)
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff
		}
	}

	if err != nil {
		return nil, fmt.Errorf("unable to connect to database after %d attempts: %w", maxRetries, err)
	}

	logger.Info("Database connection established",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.Database),
		zap.Int("max_connections", cfg.MaxConnections),
		zap.Int("min_connections", cfg.MaxIdleConns),
		zap.Bool("pgbouncer_mode", cfg.Port == 6432), // PgBouncer typically on 6432
	)

	return &DB{Pool: pool, cfg: cfg}, nil
}

// Close closes the database connection pool
func (db *DB) Close() {
	db.Pool.Close()
	logger.Info("Database connection closed")
}

// Health checks database connectivity
func (db *DB) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return db.Pool.Ping(ctx)
}

// Acquire gets a connection from the pool
func (db *DB) Acquire(ctx context.Context) (*pgxpool.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return db.Pool.Acquire(ctx)
}

// Stats returns connection pool statistics for monitoring
func (db *DB) Stats() PoolStats {
	stat := db.Pool.Stat()
	return PoolStats{
		TotalConns:      stat.TotalConns(),
		IdleConns:       stat.IdleConns(),
		AcquiredConns:   stat.AcquiredConns(),
		MaxConns:        stat.MaxConns(),
		AcquireCount:    stat.AcquireCount(),
		AcquireDuration: stat.AcquireDuration(),
		EmptyAcquires:   stat.EmptyAcquireCount(),
	}
}

// PoolStats holds connection pool metrics
type PoolStats struct {
	TotalConns      int32
	IdleConns       int32
	AcquiredConns   int32
	MaxConns        int32
	AcquireCount    int64
	AcquireDuration time.Duration
	EmptyAcquires   int64 // Times pool was empty (indicates need more connections)
}

// IsHealthy returns true if connection pool is healthy
func (s PoolStats) IsHealthy() bool {
	// Unhealthy if >50% connections acquired and empty acquires happening
	utilizationPct := float64(s.AcquiredConns) / float64(s.MaxConns) * 100
	return s.EmptyAcquires == 0 || utilizationPct < 80
}
