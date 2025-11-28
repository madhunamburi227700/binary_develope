package repository

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opsmx/ai-guardian-api/pkg/database"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

// BaseRepository provides common database operations
type BaseRepository struct {
	db     *pgxpool.Pool
	logger *utils.ErrorLogger
	table  string
}

// NewBaseRepository creates a new base repository
func NewBaseRepository(table string) *BaseRepository {
	return &BaseRepository{
		db:     database.GetPostgres(),
		logger: utils.NewErrorLogger(fmt.Sprintf("repository_%s", table)),
		table:  table,
	}
}

// QueryOptions provides options for database queries
type QueryOptions struct {
	Limit    int
	Offset   int
	OrderBy  string
	OrderDir string // ASC or DESC
	Filters  map[string]interface{}
	Select   []string
	Joins    []string
	GroupBy  []string
	Having   map[string]interface{}
}

// PaginationResult provides pagination information
type PaginationResult struct {
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Pages    int   `json:"pages"`
}

// QueryResult provides a generic query result
type QueryResult[T any] struct {
	Data       []T               `json:"data"`
	Pagination *PaginationResult `json:"pagination,omitempty"`
}

// Repository defines the interface for repository operations
type Repository[T any] interface {
	Create(ctx context.Context, entity *T) error
	GetByID(ctx context.Context, id uuid.UUID) (*T, error)
	Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, options *QueryOptions) (*QueryResult[T], error)
	Count(ctx context.Context, filters map[string]interface{}) (int64, error)
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

// Create inserts a new record
func (r *BaseRepository) Create(ctx context.Context, table string, data map[string]interface{}) (string, error) {

	now := time.Now()
	data["created_at"] = now

	// Build query
	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))
	argIndex := 1

	for column, value := range data {
		columns = append(columns, column)
		placeholders = append(placeholders, fmt.Sprintf("$%d", argIndex))
		values = append(values, value)
		argIndex++
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (%s) 
		VALUES (%s) 
		RETURNING id`,
		table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	var id string
	err := r.db.QueryRow(ctx, query, values...).Scan(&id)
	if err != nil {
		r.logger.LogError(err, "Failed to create record", map[string]interface{}{
			"table": table,
			"data":  data,
		})
		return "", fmt.Errorf("failed to create record: %w", err)
	}

	r.logger.LogInfo("Record created successfully", map[string]interface{}{
		"table": table,
		"id":    id,
	})

	return id, nil
}

// GetByID retrieves a record by ID
func (r *BaseRepository) GetByID(ctx context.Context, table string, id string, dest interface{}) error {
	query := fmt.Sprintf("SELECT * FROM %s WHERE id = $1", table)

	row := r.db.QueryRow(ctx, query, id)
	err := r.scanRow(row, dest)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("record not found")
		}
		r.logger.LogError(err, "Failed to get record by ID", map[string]interface{}{
			"table": table,
			"id":    id,
		})
		return fmt.Errorf("failed to get record: %w", err)
	}

	return nil
}

// Update updates a record by ID
func (r *BaseRepository) Update(ctx context.Context, table string, id uuid.UUID, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return fmt.Errorf("no updates provided")
	}

	// Add updated_at timestamp
	updates["updated_at"] = time.Now()

	// Build query
	setParts := make([]string, 0, len(updates))
	values := make([]interface{}, 0, len(updates)+1)
	argIndex := 1

	for column, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", column, argIndex))
		values = append(values, value)
		argIndex++
	}

	values = append(values, id) // Add ID as last parameter

	query := fmt.Sprintf(`
		UPDATE %s 
		SET %s 
		WHERE id = $%d`,
		table,
		strings.Join(setParts, ", "),
		argIndex,
	)

	result, err := r.db.Exec(ctx, query, values...)
	if err != nil {
		r.logger.LogError(err, "Failed to update record", map[string]interface{}{
			"table":   table,
			"id":      id,
			"updates": updates,
		})
		return fmt.Errorf("failed to update record: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("record not found")
	}

	r.logger.LogInfo("Record updated successfully", map[string]interface{}{
		"table":         table,
		"id":            id,
		"rows_affected": rowsAffected,
	})

	return nil
}

// Delete deletes a record by ID
func (r *BaseRepository) Delete(ctx context.Context, table string, id uuid.UUID) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1", table)

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		r.logger.LogError(err, "Failed to delete record", map[string]interface{}{
			"table": table,
			"id":    id,
		})
		return fmt.Errorf("failed to delete record: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("record not found")
	}

	r.logger.LogInfo("Record deleted successfully", map[string]interface{}{
		"table":         table,
		"id":            id,
		"rows_affected": rowsAffected,
	})

	return nil
}

// List retrieves records with pagination and filtering
func (r *BaseRepository) List(ctx context.Context, table string, options *QueryOptions, dest interface{}) (*PaginationResult, error) {
	// Build WHERE clause
	whereClause, whereArgs := r.buildWhereClause(options.Filters)

	// Build SELECT clause
	selectClause := "*"
	if len(options.Select) > 0 {
		selectClause = strings.Join(options.Select, ", ")
	}

	// Build JOIN clause
	joinClause := ""
	if len(options.Joins) > 0 {
		joinClause = " " + strings.Join(options.Joins, " ")
	}

	// Build ORDER BY clause
	orderClause := ""
	if options.OrderBy != "" {
		orderDir := "ASC"
		if options.OrderDir == "DESC" {
			orderDir = "DESC"
		}
		orderClause = fmt.Sprintf(" ORDER BY %s %s", options.OrderBy, orderDir)
	}

	// Build LIMIT and OFFSET clause
	limitClause := ""
	args := whereArgs
	argIndex := len(whereArgs) + 1

	if options.Limit > 0 {
		limitClause = fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, options.Limit)
		argIndex++
	}

	if options.Offset > 0 {
		limitClause += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, options.Offset)
	}

	// Build main query
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s%s%s%s%s`,
		selectClause,
		table,
		joinClause,
		whereClause,
		orderClause,
		limitClause,
	)

	// Execute query
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		r.logger.LogError(err, "Failed to list records", map[string]interface{}{
			"table":   table,
			"options": options,
		})
		return nil, fmt.Errorf("failed to list records: %w", err)
	}
	defer rows.Close()

	// Scan results
	err = r.scanRows(rows, dest)
	if err != nil {
		return nil, fmt.Errorf("failed to scan results: %w", err)
	}

	// Get total count for pagination
	total, err := r.Count(ctx, table, options.Filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Calculate pagination
	pageSize := options.Limit
	if pageSize <= 0 {
		pageSize = 10 // Default page size
	}
	page := (options.Offset / pageSize) + 1
	pages := int((total + int64(pageSize) - 1) / int64(pageSize))

	pagination := &PaginationResult{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Pages:    pages,
	}

	return pagination, nil
}

// list all records
func (r *BaseRepository) ListAll(ctx context.Context, table string, options *QueryOptions, dest interface{}) error {
	whereClause, whereArgs := r.buildWhereClause(options.Filters)

	query := fmt.Sprintf("SELECT * FROM %s %s", table, whereClause)

	rows, err := r.db.Query(ctx, query, whereArgs...)
	if err != nil {
		return fmt.Errorf("failed to list all records: %w", err)
	}

	defer rows.Close()

	err = r.scanRows(rows, dest)
	if err != nil {
		return fmt.Errorf("failed to scan results: %w", err)
	}

	return nil
}

// Count counts records with filters
func (r *BaseRepository) Count(ctx context.Context, table string, filters map[string]interface{}) (int64, error) {
	whereClause, whereArgs := r.buildWhereClause(filters)

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", table, whereClause)

	var count int64
	err := r.db.QueryRow(ctx, query, whereArgs...).Scan(&count)
	if err != nil {
		r.logger.LogError(err, "Failed to count records", map[string]interface{}{
			"table":   table,
			"filters": filters,
		})
		return 0, fmt.Errorf("failed to count records: %w", err)
	}

	return count, nil
}

// Exists checks if a record exists
func (r *BaseRepository) Exists(ctx context.Context, table string, id uuid.UUID) (bool, error) {
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE id = $1)", table)

	var exists bool
	err := r.db.QueryRow(ctx, query, id).Scan(&exists)
	if err != nil {
		r.logger.LogError(err, "Failed to check if record exists", map[string]interface{}{
			"table": table,
			"id":    id,
		})
		return false, fmt.Errorf("failed to check if record exists: %w", err)
	}

	return exists, nil
}

// buildWhereClause builds WHERE clause from filters
func (r *BaseRepository) buildWhereClause(filters map[string]interface{}) (string, []interface{}) {
	if len(filters) == 0 {
		return "", []interface{}{}
	}

	conditions := make([]string, 0, len(filters))
	args := make([]interface{}, 0, len(filters))
	argIndex := 1

	for column, value := range filters {
		conditions = append(conditions, fmt.Sprintf("%s = $%d", column, argIndex))
		args = append(args, value)
		argIndex++
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

// scanRow scans a single row into dest
func (r *BaseRepository) scanRow(row pgx.Row, dest interface{}) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return fmt.Errorf("dest must be a pointer")
	}

	destElem := destValue.Elem()
	destType := destElem.Type()

	// Get struct fields and their db tags
	fields := make([]reflect.StructField, 0)
	for i := 0; i < destType.NumField(); i++ {
		field := destType.Field(i)
		if dbTag := field.Tag.Get("db"); dbTag != "" {
			fields = append(fields, field)
		}
	}

	// Create slice of pointers for scanning
	values := make([]interface{}, len(fields))
	for i, field := range fields {
		values[i] = destElem.FieldByIndex(field.Index).Addr().Interface()
	}

	return row.Scan(values...)
}

// scanRows scans multiple rows into dest
func (r *BaseRepository) scanRows(rows pgx.Rows, dest interface{}) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return fmt.Errorf("dest must be a pointer")
	}

	destElem := destValue.Elem()
	if destElem.Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to a slice")
	}

	destType := destElem.Type().Elem()
	slice := reflect.MakeSlice(destElem.Type(), 0, 0)

	// Get column names
	fieldDescriptions := rows.FieldDescriptions()
	columns := make([]string, len(fieldDescriptions))
	for i, desc := range fieldDescriptions {
		columns[i] = string(desc.Name)
	}

	for rows.Next() {
		// Create new instance
		item := reflect.New(destType).Elem()

		// Create slice of pointers for scanning
		values := make([]interface{}, len(columns))
		for i, column := range columns {
			field := r.findFieldByTag(destType, column)
			if field != nil {
				values[i] = item.FieldByIndex(field.Index).Addr().Interface()
			} else {
				values[i] = new(interface{})
			}
		}

		err := rows.Scan(values...)
		if err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		slice = reflect.Append(slice, item)
	}

	destElem.Set(slice)
	return nil
}

// check if a record exists with the given filters
func (r *BaseRepository) ExistsWithFilters(ctx context.Context, table string, filters map[string]interface{}) (bool, error) {
	var exists bool
	whereClause, whereArgs := r.buildWhereClause(filters)
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s%s)", table, whereClause)
	err := r.db.QueryRow(ctx, query, whereArgs...).Scan(&exists)
	if err != nil {
		r.logger.LogError(err, "Failed to check if record exists", map[string]interface{}{
			"table":   table,
			"filters": filters,
		})
		return false, fmt.Errorf("failed to check if record exists: %w", err)
	}

	return exists, nil
}

// findFieldByTag finds a struct field by its db tag
func (r *BaseRepository) findFieldByTag(structType reflect.Type, tagValue string) *reflect.StructField {
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if dbTag := field.Tag.Get("db"); dbTag == tagValue {
			return &field
		}
	}
	return nil
}

// Transaction provides database transaction support
type Transaction struct {
	tx pgx.Tx
}

// NewTransaction creates a new transaction
func (r *BaseRepository) NewTransaction(ctx context.Context) (*Transaction, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &Transaction{tx: tx}, nil
}

// Commit commits the transaction
func (t *Transaction) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

// Rollback rolls back the transaction
func (t *Transaction) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

// Exec executes a query within the transaction
func (t *Transaction) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	return t.tx.Exec(ctx, query, args...)
}

// Query executes a query within the transaction
func (t *Transaction) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	return t.tx.Query(ctx, query, args...)
}

// QueryRow executes a query that returns a single row within the transaction
func (t *Transaction) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	return t.tx.QueryRow(ctx, query, args...)
}
