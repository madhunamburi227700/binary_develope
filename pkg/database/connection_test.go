package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Test Helpers
// =============================================================================

type testDB struct {
	pg         pgxmock.PgxPoolIface
	redis      *redis.Client
	redisMock  redismock.ClientMock
	originalDB *DB
}

func setupTest(t *testing.T, withPG, withRedis bool) *testDB {
	t.Helper()
	td := &testDB{originalDB: db}

	if withPG {
		var err error
		td.pg, err = pgxmock.NewPool()
		assert.NoError(t, err)
	}

	if withRedis {
		td.redis, td.redisMock = redismock.NewClientMock()
	}

	db = &DB{
		Postgres: td.pg,
		Redis:    td.redis,
		logger:   utils.NewErrorLogger("database"),
	}

	return td
}

func (td *testDB) cleanup() {
	if td.pg != nil {
		td.pg.Close()
	}
	if td.redis != nil {
		td.redis.Close()
	}
	db = td.originalDB
}

// =============================================================================
// IsInitialized Tests
// =============================================================================

func TestIsInitialized(t *testing.T) {
	tests := []struct {
		name     string
		setupDB  func() *DB
		expected bool
	}{
		{
			name:     "DB is nil",
			setupDB:  func() *DB { return nil },
			expected: false,
		},
		{
			name: "Both connections nil",
			setupDB: func() *DB {
				return &DB{Postgres: nil, Redis: nil}
			},
			expected: false,
		},
		{
			name: "Only Postgres set",
			setupDB: func() *DB {
				mockPG, _ := pgxmock.NewPool()
				return &DB{Postgres: mockPG, Redis: nil}
			},
			expected: false,
		},
		{
			name: "Only Redis set",
			setupDB: func() *DB {
				mockRedis, _ := redismock.NewClientMock()
				return &DB{Postgres: nil, Redis: mockRedis}
			},
			expected: false,
		},
		{
			name: "Both connections set",
			setupDB: func() *DB {
				mockPG, _ := pgxmock.NewPool()
				mockRedis, _ := redismock.NewClientMock()
				return &DB{Postgres: mockPG, Redis: mockRedis}
			},
			expected: true,
		},
	}

	originalDB := db
	defer func() { db = originalDB }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db = tt.setupDB()
			assert.Equal(t, tt.expected, IsInitialized())

			// Cleanup
			if db != nil {
				if db.Postgres != nil {
					db.Postgres.Close()
				}
				if db.Redis != nil {
					db.Redis.Close()
				}
			}
		})
	}
}

// =============================================================================
// GetPostgres Tests
// =============================================================================

func TestGetPostgres(t *testing.T) {
	tests := []struct {
		name      string
		setupDB   func() *DB
		expectMsg string
	}{
		{
			name:      "DB is nil",
			setupDB:   func() *DB { return nil },
			expectMsg: "database not initialized",
		},
		{
			name:      "Postgres is nil",
			setupDB:   func() *DB { return &DB{Postgres: nil, Redis: nil} },
			expectMsg: "database not initialized",
		},
		{
			name: "Postgres is interface not pool",
			setupDB: func() *DB {
				mockPG, _ := pgxmock.NewPool()
				return &DB{Postgres: mockPG, Redis: nil}
			},
			expectMsg: "postgres connection is not a *pgxpool.Pool",
		},
	}

	originalDB := db
	defer func() { db = originalDB }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db = tt.setupDB()
			assert.PanicsWithValue(t, tt.expectMsg, func() {
				GetPostgres()
			})

			if db != nil && db.Postgres != nil {
				db.Postgres.Close()
			}
		})
	}
}

// =============================================================================
// GetRedis Tests
// =============================================================================

func TestGetRedis(t *testing.T) {
	tests := []struct {
		name        string
		setupDB     func() *DB
		shouldPanic bool
		panicMsg    string
		shouldPass  bool
	}{
		{
			name:        "DB is nil",
			setupDB:     func() *DB { return nil },
			shouldPanic: true,
			panicMsg:    "database not initialized",
		},
		{
			name:        "Redis is nil",
			setupDB:     func() *DB { return &DB{Postgres: nil, Redis: nil} },
			shouldPanic: true,
			panicMsg:    "database not initialized",
		},
		{
			name: "Redis is set but Postgres is nil",
			setupDB: func() *DB {
				mockRedis, _ := redismock.NewClientMock()
				return &DB{Postgres: nil, Redis: mockRedis}
			},
			shouldPass: true,
		},
		{
			name: "Both connections set",
			setupDB: func() *DB {
				mockPG, _ := pgxmock.NewPool()
				mockRedis, _ := redismock.NewClientMock()
				return &DB{Postgres: mockPG, Redis: mockRedis}
			},
			shouldPass: true,
		},
	}

	originalDB := db
	defer func() { db = originalDB }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db = tt.setupDB()

			if tt.shouldPanic {
				assert.PanicsWithValue(t, tt.panicMsg, func() {
					GetRedis()
				})
			} else if tt.shouldPass {
				client := GetRedis()
				assert.NotNil(t, client)
			}

			if db != nil {
				if db.Postgres != nil {
					db.Postgres.Close()
				}
				if db.Redis != nil {
					db.Redis.Close()
				}
			}
		})
	}
}

// =============================================================================
// Close Tests
// =============================================================================

func TestClose(t *testing.T) {
	tests := []struct {
		name    string
		setupDB func() *DB
	}{
		{
			name:    "DB is nil",
			setupDB: func() *DB { return nil },
		},
		{
			name: "Both connections nil",
			setupDB: func() *DB {
				return &DB{Postgres: nil, Redis: nil, logger: utils.NewErrorLogger("database")}
			},
		},
		{
			name: "Only Postgres set",
			setupDB: func() *DB {
				mockPG, _ := pgxmock.NewPool()
				return &DB{Postgres: mockPG, Redis: nil, logger: utils.NewErrorLogger("database")}
			},
		},
		{
			name: "Only Redis set",
			setupDB: func() *DB {
				mockRedis, _ := redismock.NewClientMock()
				return &DB{Postgres: nil, Redis: mockRedis, logger: utils.NewErrorLogger("database")}
			},
		},
		{
			name: "Both connections set",
			setupDB: func() *DB {
				mockPG, _ := pgxmock.NewPool()
				mockRedis, _ := redismock.NewClientMock()
				return &DB{Postgres: mockPG, Redis: mockRedis, logger: utils.NewErrorLogger("database")}
			},
		},
	}

	originalDB := db
	defer func() { db = originalDB }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db = tt.setupDB()
			err := Close()
			assert.NoError(t, err)
		})
	}
}

func TestMultipleClose(t *testing.T) {
	originalDB := db
	defer func() { db = originalDB }()

	db = nil
	for i := 0; i < 3; i++ {
		err := Close()
		assert.NoError(t, err)
	}
}

// =============================================================================
// Health Tests
// =============================================================================

func TestHealth(t *testing.T) {
	originalDB := db
	defer func() { db = originalDB }()

	tests := []struct {
		name           string
		setupDB        func() (*DB, pgxmock.PgxPoolIface, redismock.ClientMock)
		expectPG       string
		expectRedis    string
		expectContains bool
	}{
		{
			name: "DB is nil",
			setupDB: func() (*DB, pgxmock.PgxPoolIface, redismock.ClientMock) {
				return nil, nil, nil
			},
			expectPG:    "not initialized",
			expectRedis: "not initialized",
		},
		{
			name: "Connections are nil",
			setupDB: func() (*DB, pgxmock.PgxPoolIface, redismock.ClientMock) {
				return &DB{Postgres: nil, Redis: nil}, nil, nil
			},
			expectPG:    "not initialized",
			expectRedis: "not initialized",
		},
		{
			name: "Postgres healthy, Redis not initialized",
			setupDB: func() (*DB, pgxmock.PgxPoolIface, redismock.ClientMock) {
				mockPG, _ := pgxmock.NewPool()
				mockPG.ExpectPing()
				return &DB{Postgres: mockPG, Redis: nil}, mockPG, nil
			},
			expectPG:    "healthy",
			expectRedis: "not initialized",
		},
		{
			name: "Postgres not initialized, Redis healthy",
			setupDB: func() (*DB, pgxmock.PgxPoolIface, redismock.ClientMock) {
				mockRedis, redisMock := redismock.NewClientMock()
				redisMock.ExpectPing().SetVal("PONG")
				return &DB{Postgres: nil, Redis: mockRedis}, nil, redisMock
			},
			expectPG:    "not initialized",
			expectRedis: "healthy",
		},
		{
			name: "Both healthy",
			setupDB: func() (*DB, pgxmock.PgxPoolIface, redismock.ClientMock) {
				mockPG, _ := pgxmock.NewPool()
				mockRedis, redisMock := redismock.NewClientMock()
				mockPG.ExpectPing()
				redisMock.ExpectPing().SetVal("PONG")
				return &DB{Postgres: mockPG, Redis: mockRedis}, mockPG, redisMock
			},
			expectPG:    "healthy",
			expectRedis: "healthy",
		},
		{
			name: "Postgres unhealthy, Redis healthy",
			setupDB: func() (*DB, pgxmock.PgxPoolIface, redismock.ClientMock) {
				mockPG, _ := pgxmock.NewPool()
				mockRedis, redisMock := redismock.NewClientMock()
				mockPG.ExpectPing().WillReturnError(errors.New("postgres down"))
				redisMock.ExpectPing().SetVal("PONG")
				return &DB{Postgres: mockPG, Redis: mockRedis}, mockPG, redisMock
			},
			expectPG:       "unhealthy",
			expectRedis:    "healthy",
			expectContains: true,
		},
		{
			name: "Postgres healthy, Redis unhealthy",
			setupDB: func() (*DB, pgxmock.PgxPoolIface, redismock.ClientMock) {
				mockPG, _ := pgxmock.NewPool()
				mockRedis, redisMock := redismock.NewClientMock()
				mockPG.ExpectPing()
				redisMock.ExpectPing().SetErr(errors.New("redis down"))
				return &DB{Postgres: mockPG, Redis: mockRedis}, mockPG, redisMock
			},
			expectPG:       "healthy",
			expectRedis:    "unhealthy",
			expectContains: true,
		},
		{
			name: "Both unhealthy",
			setupDB: func() (*DB, pgxmock.PgxPoolIface, redismock.ClientMock) {
				mockPG, _ := pgxmock.NewPool()
				mockRedis, redisMock := redismock.NewClientMock()
				mockPG.ExpectPing().WillReturnError(errors.New("postgres down"))
				redisMock.ExpectPing().SetErr(errors.New("redis down"))
				return &DB{Postgres: mockPG, Redis: mockRedis}, mockPG, redisMock
			},
			expectPG:       "unhealthy",
			expectRedis:    "unhealthy",
			expectContains: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mockPG pgxmock.PgxPoolIface
			var redisMock redismock.ClientMock
			db, mockPG, redisMock = tt.setupDB()

			status := Health()

			if tt.expectContains {
				assert.Contains(t, status["postgres"], tt.expectPG)
				assert.Contains(t, status["redis"], tt.expectRedis)
			} else {
				assert.Equal(t, tt.expectPG, status["postgres"])
				assert.Equal(t, tt.expectRedis, status["redis"])
			}

			// Verify expectations
			if mockPG != nil {
				assert.NoError(t, mockPG.ExpectationsWereMet())
				mockPG.Close()
			}
			if redisMock != nil {
				assert.NoError(t, redisMock.ExpectationsWereMet())
			}
			if db != nil && db.Redis != nil {
				db.Redis.Close()
			}
		})
	}
}

// =============================================================================
// InitDatabase Tests
// =============================================================================

func TestInitDatabase(t *testing.T) {
	originalDB := db
	defer func() { db = originalDB }()

	tests := []struct {
		name        string
		pgConn      string
		redisAddr   string
		expectError string
	}{
		{
			name:        "Invalid Postgres connection string",
			pgConn:      "invalid-connection-string",
			redisAddr:   "localhost:6379",
			expectError: "postgres initialization failed",
		},
		{
			name:        "Postgres connection refused",
			pgConn:      "postgres://user:pass@localhost:59999/nonexistent?sslmode=disable",
			redisAddr:   "localhost:6379",
			expectError: "postgres initialization failed",
		},
		{
			name:        "Empty connection strings",
			pgConn:      "",
			redisAddr:   "",
			expectError: "postgres initialization failed",
		},
		{
			name:        "Both invalid connections",
			pgConn:      "invalid-postgres",
			redisAddr:   "invalid-redis:99999",
			expectError: "postgres initialization failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := InitDatabase(tt.pgConn, "", "", tt.redisAddr)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectError)
		})
	}
}

func TestInitDatabase_WithMocks(t *testing.T) {
	originalDB := db
	defer func() { db = originalDB }()

	mockPG, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mockPG.Close()

	mockRedis, _ := redismock.NewClientMock()
	defer mockRedis.Close()

	db = &DB{
		Postgres: mockPG,
		Redis:    mockRedis,
		logger:   utils.NewErrorLogger("database"),
	}

	assert.NotNil(t, db.Postgres)
	assert.NotNil(t, db.Redis)
	assert.True(t, IsInitialized())
}

// =============================================================================
// InitRedis Tests
// =============================================================================

func TestInitRedis(t *testing.T) {
	originalDB := db
	defer func() { db = originalDB }()

	tests := []struct {
		name     string
		username string
		password string
		address  string
	}{
		{
			name:     "Invalid address",
			username: "",
			password: "",
			address:  "invalid-address:99999",
		},
		{
			name:     "Connection refused",
			username: "",
			password: "",
			address:  "localhost:99999",
		},
		{
			name:     "With credentials - localhost refused",
			username: "testuser",
			password: "testpass",
			address:  "127.0.0.1:59999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDB := &DB{logger: utils.NewErrorLogger("database")}
			err := testDB.initRedis(tt.username, tt.password, tt.address)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to ping redis")
		})
	}
}

func TestInitRedis_Success(t *testing.T) {
	mockRedis, redisMock := redismock.NewClientMock()
	defer mockRedis.Close()

	redisMock.ExpectPing().SetVal("PONG")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := mockRedis.Ping(ctx).Err()
	assert.NoError(t, err)
	assert.NoError(t, redisMock.ExpectationsWereMet())
}

func TestInitRedis_Timeout(t *testing.T) {
	mockRedis, redisMock := redismock.NewClientMock()
	defer mockRedis.Close()

	redisMock.ExpectPing().SetErr(context.DeadlineExceeded)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := mockRedis.Ping(ctx).Err()
	assert.Error(t, err)
	assert.NoError(t, redisMock.ExpectationsWereMet())
}

// =============================================================================
// InitPostgres Tests
// =============================================================================

func TestInitPostgres(t *testing.T) {
	originalDB := db
	defer func() { db = originalDB }()

	tests := []struct {
		name        string
		connString  string
		expectError string
	}{
		{
			name:        "Invalid connection string",
			connString:  "not-a-valid-connection-string",
			expectError: "failed to parse postgres config",
		},
		{
			name:        "Connection refused",
			connString:  "postgres://user:pass@localhost:59999/testdb?sslmode=disable",
			expectError: "failed to ping postgres",
		},
		{
			name:        "Completely invalid string",
			connString:  "completely-invalid-string-not-a-url",
			expectError: "failed to parse postgres config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDB := &DB{logger: utils.NewErrorLogger("test")}
			err := testDB.initPostgres(tt.connString)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectError)
		})
	}
}

// =============================================================================
// Redis Interface Tests
// =============================================================================

func TestRedisInterface_Operations(t *testing.T) {
	td := setupTest(t, false, true)
	defer td.cleanup()

	ctx := context.Background()

	// Setup expectations
	td.redisMock.ExpectGet("test-key").SetVal("test-value")
	td.redisMock.ExpectSet("new-key", "new-value", 0).SetVal("OK")
	td.redisMock.ExpectDel("old-key").SetVal(1)

	// Test Get
	val, err := db.Redis.Get(ctx, "test-key").Result()
	assert.NoError(t, err)
	assert.Equal(t, "test-value", val)

	// Test Set
	err = db.Redis.Set(ctx, "new-key", "new-value", 0).Err()
	assert.NoError(t, err)

	// Test Del
	count, err := db.Redis.Del(ctx, "old-key").Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)

	assert.NoError(t, td.redisMock.ExpectationsWereMet())
}

func TestRedisInterface_Errors(t *testing.T) {
	td := setupTest(t, false, true)
	defer td.cleanup()

	ctx := context.Background()

	tests := []struct {
		name      string
		setup     func()
		operation func() error
		expectErr string
	}{
		{
			name: "GET key not found",
			setup: func() {
				td.redisMock.ExpectGet("missing-key").RedisNil()
			},
			operation: func() error {
				_, err := db.Redis.Get(ctx, "missing-key").Result()
				return err
			},
			expectErr: "redis: nil",
		},
		{
			name: "SET error",
			setup: func() {
				td.redisMock.ExpectSet("key", "value", 0).SetErr(errors.New("write error"))
			},
			operation: func() error {
				return db.Redis.Set(ctx, "key", "value", 0).Err()
			},
			expectErr: "write error",
		},
		{
			name: "DEL error",
			setup: func() {
				td.redisMock.ExpectDel("key").SetErr(errors.New("delete error"))
			},
			operation: func() error {
				_, err := db.Redis.Del(ctx, "key").Result()
				return err
			},
			expectErr: "delete error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := tt.operation()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectErr)
		})
	}

	assert.NoError(t, td.redisMock.ExpectationsWereMet())
}

func TestRedisInterface_FlushDB(t *testing.T) {
	td := setupTest(t, false, true)
	defer td.cleanup()

	td.redisMock.ExpectFlushDB().SetVal("OK")

	ctx := context.Background()
	err := db.Redis.FlushDB(ctx).Err()
	assert.NoError(t, err)
	assert.NoError(t, td.redisMock.ExpectationsWereMet())
}

// =============================================================================
// Postgres Interface Tests
// =============================================================================

func TestPostgresInterface_Operations(t *testing.T) {
	td := setupTest(t, true, false)
	defer td.cleanup()

	ctx := context.Background()

	t.Run("Stat", func(t *testing.T) {
		stat := db.Postgres.Stat()
		assert.NotNil(t, stat)
	})

	t.Run("QueryRow", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{"id"}).AddRow(123)
		td.pg.ExpectQuery("SELECT id FROM test").WillReturnRows(rows)

		var id int
		err := db.Postgres.QueryRow(ctx, "SELECT id FROM test").Scan(&id)
		assert.NoError(t, err)
		assert.Equal(t, 123, id)
	})

	t.Run("Exec", func(t *testing.T) {
		td.pg.ExpectExec("INSERT INTO users").
			WithArgs("test-user", "test@example.com").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		tag, err := db.Postgres.Exec(ctx, "INSERT INTO users (name, email) VALUES ($1, $2)", "test-user", "test@example.com")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), tag.RowsAffected())
	})

	t.Run("Transaction", func(t *testing.T) {
		td.pg.ExpectBegin()
		td.pg.ExpectExec("UPDATE users SET name").
			WithArgs("updated-name", 1).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		td.pg.ExpectCommit()

		tx, err := db.Postgres.Begin(ctx)
		assert.NoError(t, err)

		_, err = tx.Exec(ctx, "UPDATE users SET name = $1 WHERE id = $2", "updated-name", 1)
		assert.NoError(t, err)

		err = tx.Commit(ctx)
		assert.NoError(t, err)
	})

	assert.NoError(t, td.pg.ExpectationsWereMet())
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkIsInitialized(b *testing.B) {
	originalDB := db
	defer func() { db = originalDB }()

	db = nil
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsInitialized()
	}
}

func BenchmarkHealth(b *testing.B) {
	originalDB := db
	defer func() { db = originalDB }()

	db = nil
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Health()
	}
}
