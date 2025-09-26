package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
	"github.com/redis/go-redis/v9"
)

// DB holds database connections
type DB struct {
	Postgres *pgxpool.Pool
	Redis    *redis.Client
	logger   *utils.ErrorLogger
}

// Global database instance
var db *DB

// InitDatabase initializes database connections
func InitDatabase(pgConnStr, rdUser, rdPassword, rdAddress string) error {
	logger := utils.NewErrorLogger("database")

	db = &DB{
		logger: logger,
	}

	// Initialize PostgreSQL connection
	if err := db.initPostgres(pgConnStr); err != nil {
		logger.LogError(err, "Failed to initialize PostgreSQL", nil)
		return fmt.Errorf("postgres initialization failed: %w", err)
	}

	// Initialize Redis connection
	if err := db.initRedis(rdUser, rdPassword, rdAddress); err != nil {
		logger.LogError(err, "Failed to initialize Redis", nil)
		return fmt.Errorf("redis initialization failed: %w", err)
	}

	logger.LogInfo("Database connections initialized successfully", nil)
	return nil
}

// initPostgres initializes PostgreSQL connection with pgx
func (db *DB) initPostgres(connStr string) error {
	// Configure connection pool
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return fmt.Errorf("failed to parse postgres config: %w", err)
	}

	// Set connection pool settings
	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = time.Minute * 30

	// Create connection pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create postgres pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("failed to ping postgres: %w", err)
	}

	db.Postgres = pool
	db.logger.LogInfo("PostgreSQL connection established", map[string]interface{}{
		"max_conns": config.MaxConns,
		"min_conns": config.MinConns,
	})

	return nil
}

// initRedis initializes Redis connection
func (db *DB) initRedis(user, password, address string) error {
	// Configure Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     address,
		Username: user,
		Password: password,
		DB:       0,
		PoolSize: 10,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close()
		return fmt.Errorf("failed to ping redis: %w", err)
	}

	db.Redis = rdb
	db.logger.LogInfo("Redis connection established", map[string]interface{}{
		"addr": address,
	})

	return nil
}

// GetPostgres returns the PostgreSQL connection pool
func GetPostgres() *pgxpool.Pool {
	if db == nil || db.Postgres == nil {
		panic("database not initialized")
	}
	return db.Postgres
}

// GetRedis returns the Redis client
func GetRedis() *redis.Client {
	if db == nil || db.Redis == nil {
		panic("database not initialized")
	}
	return db.Redis
}

// Close closes all database connections
func Close() error {
	if db == nil {
		return nil
	}

	var errs []error

	if db.Postgres != nil {
		db.Postgres.Close()
	}

	if db.Redis != nil {
		if err := db.Redis.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing database connections: %v", errs)
	}

	db.logger.LogInfo("Database connections closed", nil)
	return nil
}

// Health checks database connections
func Health() map[string]string {
	status := make(map[string]string)

	// Check PostgreSQL
	if db != nil && db.Postgres != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := db.Postgres.Ping(ctx); err != nil {
			status["postgres"] = "unhealthy: " + err.Error()
		} else {
			status["postgres"] = "healthy"
		}
	} else {
		status["postgres"] = "not initialized"
	}

	// Check Redis
	if db != nil && db.Redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := db.Redis.Ping(ctx).Err(); err != nil {
			status["redis"] = "unhealthy: " + err.Error()
		} else {
			status["redis"] = "healthy"
		}
	} else {
		status["redis"] = "not initialized"
	}

	return status
}

// IsInitialized checks if database is initialized
func IsInitialized() bool {
	return db != nil && db.Postgres != nil && db.Redis != nil
}
