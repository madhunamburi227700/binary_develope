package middleware

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/opsmx/ai-guardian-api/pkg/models"
	"github.com/opsmx/ai-guardian-api/pkg/repository"
)

const serviceName = "ai-guardian-api"

var auditRepo *repository.AuditRepository

// getAuditRepository lazily initializes and returns the audit repository
func getAuditRepository() *repository.AuditRepository {
	if auditRepo == nil {
		auditRepo = repository.NewAuditRepository()
	}
	return auditRepo
}

// responseWriterWithBody wraps http.ResponseWriter to capture status code and response body
type responseWriterWithBody struct {
	http.ResponseWriter
	statusCode   int
	responseBody *bytes.Buffer
}

func (w *responseWriterWithBody) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriterWithBody) Write(b []byte) (int, error) {
	w.responseBody.Write(b)
	return w.ResponseWriter.Write(b)
}

// shouldAudit checks if the request should be audited based on entity
func shouldAudit(path string) bool {
	// Only audit specific entities: scans, projects, remediation, auth
	return strings.Contains(path, "/scans") ||
		strings.Contains(path, "/rescan") ||
		strings.Contains(path, "/projects") ||
		strings.Contains(path, "/remediation") ||
		strings.Contains(path, "/login") ||
		strings.Contains(path, "/logout")
}

// AuditLog middleware logs user requests to database
func AuditLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := r.Method
		path := r.URL.Path

		// Only process POST, PUT, PATCH, DELETE requests and auditable entities
		if (method != "POST" && method != "PUT" && method != "PATCH" && method != "DELETE") || !shouldAudit(path) {
			next.ServeHTTP(w, r)
			return
		}

		// Start timing after checks
		start := time.Now()

		endpoint := path
		if r.URL.RawQuery != "" {
			endpoint = path + "?" + r.URL.RawQuery
		}

		// Read and store request body
		var requestBodyStr string
		bodyBytes, err := io.ReadAll(r.Body)
		if err == nil && len(bodyBytes) > 0 {
			// Restore the body for the actual handler
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			requestBodyStr = string(bodyBytes)
		}

		// Extract entity info
		entityName, entityID := extractEntity(path)

		// Wrap the response writer
		wrapped := &responseWriterWithBody{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			responseBody:   &bytes.Buffer{},
		}

		// Process the request
		next.ServeHTTP(wrapped, r)

		// Get user from header
		// x-user header comes after request auth process
		username := r.Header.Get(HeaderXUser)
		if username == "" {
			username = "anonymous"
		}

		// Calculate duration
		durationMs := time.Since(start).Milliseconds()

		// Parse response body
		var responseBodyStr string
		if wrapped.responseBody.Len() > 0 {
			responseBodyStr = wrapped.responseBody.String()
		}

		// Determine action (pass query params for remediation)
		action := determineAction(r)

		// Save to database asynchronously
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Create audit log entry
			auditLog := &models.AuditLog{
				UserID:         username,
				HTTPMethod:     sql.NullString{String: method, Valid: true},
				Action:         sql.NullString{String: action, Valid: true},
				Endpoint:       sql.NullString{String: endpoint, Valid: true},
				EntityName:     sql.NullString{String: entityName, Valid: entityName != ""},
				EntityID:       sql.NullString{String: entityID, Valid: entityID != ""},
				RequestBody:    sql.NullString{String: requestBodyStr, Valid: requestBodyStr != ""},
				ResponseStatus: sql.NullInt16{Int16: int16(wrapped.statusCode), Valid: true},
				ResponseBody:   sql.NullString{String: responseBodyStr, Valid: responseBodyStr != ""},
				DurationMs:     sql.NullInt32{Int32: int32(durationMs), Valid: true},
				ServiceName:    sql.NullString{String: serviceName, Valid: true},
				CreatedAt:      start,
			}

			// Save to database
			repo := getAuditRepository()
			if repo != nil {
				if err := repo.CreateAuditLog(ctx, auditLog); err != nil {
					log.Printf("Failed to save audit log to database: %v", err)
				}
			}
		}()
	})
}

// extractEntity extracts entity name and ID from the endpoint
func extractEntity(path string) (entityName, entityID string) {
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// Extract entity from path
	for i, part := range parts {
		if part == "scans" || part == "projects" || part == "remediation" {
			entityName = part
			// Try to get ID from next part if exists
			if i+1 < len(parts) && parts[i+1] != "" && parts[i+1] != "rescan" {
				entityID = parts[i+1]
			}
			break
		}
	}

	return
}

// determineAction determines the action based on HTTP method, path, and query params
func determineAction(r *http.Request) string {
	method := r.Method
	path := r.URL.Path
	isPost := method == "POST"

	switch {
	// --- Authentication ---
	case strings.Contains(path, "/login"):
		return models.ActionLogin
	case strings.Contains(path, "/logout"):
		return models.ActionLogout

	// --- Scans ---
	case strings.Contains(path, "/rescan") && isPost:
		return models.ActionRescanInit
	case strings.Contains(path, "/scans") && isPost:
		return models.ActionScanInit

	// --- Projects ---
	case strings.Contains(path, "/projects"):
		switch method {
		case "POST":
			return models.ActionProjectCreate
		case "PUT", "PATCH":
			return models.ActionProjectUpdate
		case "DELETE":
			return models.ActionProjectDelete
		}

	// --- Remediation ---
	case strings.Contains(path, "/remediation") && isPost:
		mode := r.URL.Query().Get("mode")
		action := r.URL.Query().Get("action")

		if mode == "apply" || action == "approve" {
			return models.ActionRemediationApprove
		}
		if mode == "generate" || action == "generate" {
			return models.ActionRemediationAttempt
		}

		// Default if no query param
		return models.ActionRemediationAttempt
	}

	// --- Fallback ---
	return method + "_" + strings.ToUpper(strings.Trim(path, "/"))
}
