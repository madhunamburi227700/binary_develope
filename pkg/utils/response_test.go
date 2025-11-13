package utils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test SendErrorResponse function
func TestSendErrorResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		message    string
	}{
		{"Bad Request", http.StatusBadRequest, "Invalid input"},
		{"Unauthorized", http.StatusUnauthorized, "Authentication required"},
		{"Forbidden", http.StatusForbidden, "Access denied"},
		{"Not Found", http.StatusNotFound, "Resource not found"},
		{"Internal Server Error", http.StatusInternalServerError, "Something went wrong"},
		{"Empty message", http.StatusBadRequest, ""},
		{"Long message", http.StatusBadRequest, "This is a very long error message that contains detailed information about what went wrong in the application"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a ResponseRecorder to record the response
			rr := httptest.NewRecorder()

			// Call the function
			SendErrorResponse(rr, tt.statusCode, tt.message)

			// Check status code
			if rr.Code != tt.statusCode {
				t.Errorf("SendErrorResponse() status = %d, want %d", rr.Code, tt.statusCode)
			}

			// Check Content-Type header
			contentType := rr.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("SendErrorResponse() Content-Type = %s, want application/json", contentType)
			}

			// Parse response body
			var response map[string]interface{}
			err := json.Unmarshal(rr.Body.Bytes(), &response)
			if err != nil {
				t.Fatalf("Failed to parse response JSON: %v", err)
			}

			// Check success field
			if success, ok := response["success"].(bool); !ok || success {
				t.Errorf("SendErrorResponse() success = %v, want false", response["success"])
			}

			// Check message field
			if message, ok := response["message"].(string); !ok || message != tt.message {
				t.Errorf("SendErrorResponse() message = %v, want %s", response["message"], tt.message)
			}
		})
	}
}

// Test SendSuccessResponse function
func TestSendSuccessResponse(t *testing.T) {
	tests := []struct {
		name    string
		data    interface{}
		message string
	}{
		{
			name:    "String data",
			data:    "test data",
			message: "Operation successful",
		},
		{
			name:    "Map data",
			data:    map[string]string{"key": "value", "foo": "bar"},
			message: "Data retrieved",
		},
		{
			name:    "Array data",
			data:    []int{1, 2, 3, 4, 5},
			message: "List retrieved",
		},
		{
			name:    "Nil data",
			data:    nil,
			message: "Success",
		},
		{
			name:    "Empty message",
			data:    "data",
			message: "",
		},
		{
			name: "Struct data",
			data: struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			}{ID: 1, Name: "Test"},
			message: "User created",
		},
		{
			name:    "Boolean data",
			data:    true,
			message: "Flag set",
		},
		{
			name:    "Number data",
			data:    42,
			message: "Count retrieved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a ResponseRecorder
			rr := httptest.NewRecorder()

			// Call the function
			SendSuccessResponse(rr, tt.data, tt.message)

			// Check status code
			if rr.Code != http.StatusOK {
				t.Errorf("SendSuccessResponse() status = %d, want %d", rr.Code, http.StatusOK)
			}

			// Check Content-Type header
			contentType := rr.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("SendSuccessResponse() Content-Type = %s, want application/json", contentType)
			}

			// Parse response body
			var response map[string]interface{}
			err := json.Unmarshal(rr.Body.Bytes(), &response)
			if err != nil {
				t.Fatalf("Failed to parse response JSON: %v", err)
			}

			// Check success field
			if success, ok := response["success"].(bool); !ok || !success {
				t.Errorf("SendSuccessResponse() success = %v, want true", response["success"])
			}

			// Check message field
			if message, ok := response["message"].(string); !ok || message != tt.message {
				t.Errorf("SendSuccessResponse() message = %v, want %s", response["message"], tt.message)
			}

			// Check data field exists
			if _, ok := response["data"]; !ok {
				t.Errorf("SendSuccessResponse() missing 'data' field")
			}
		})
	}
}

// Test SendSuccessResponseWithNoData function
func TestSendSuccessResponseWithNoData(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{"Standard message", "Operation completed"},
		{"Empty message", ""},
		{"Long message", "This is a very long success message with lots of details about the operation"},
		{"Special characters", "Success! ✓ Done."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a ResponseRecorder
			rr := httptest.NewRecorder()

			// Call the function
			SendSuccessResponseWithNoData(rr, tt.message)

			// Check status code
			if rr.Code != http.StatusOK {
				t.Errorf("SendSuccessResponseWithNoData() status = %d, want %d", rr.Code, http.StatusOK)
			}

			// Check Content-Type header
			contentType := rr.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("SendSuccessResponseWithNoData() Content-Type = %s, want application/json", contentType)
			}

			// Parse response body
			var response map[string]interface{}
			err := json.Unmarshal(rr.Body.Bytes(), &response)
			if err != nil {
				t.Fatalf("Failed to parse response JSON: %v", err)
			}

			// Check success field
			if success, ok := response["success"].(bool); !ok || !success {
				t.Errorf("SendSuccessResponseWithNoData() success = %v, want true", response["success"])
			}

			// Check message field
			if message, ok := response["message"].(string); !ok || message != tt.message {
				t.Errorf("SendSuccessResponseWithNoData() message = %v, want %s", response["message"], tt.message)
			}

			// Check that data field does NOT exist
			if _, ok := response["data"]; ok {
				t.Errorf("SendSuccessResponseWithNoData() should not have 'data' field")
			}
		})
	}
}

// Test all three functions to ensure they set headers correctly
func TestResponseHeaders(t *testing.T) {
	tests := []struct {
		name string
		fn   func(http.ResponseWriter)
	}{
		{
			name: "SendErrorResponse sets headers",
			fn: func(w http.ResponseWriter) {
				SendErrorResponse(w, http.StatusBadRequest, "error")
			},
		},
		{
			name: "SendSuccessResponse sets headers",
			fn: func(w http.ResponseWriter) {
				SendSuccessResponse(w, nil, "success")
			},
		},
		{
			name: "SendSuccessResponseWithNoData sets headers",
			fn: func(w http.ResponseWriter) {
				SendSuccessResponseWithNoData(w, "success")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			tt.fn(rr)

			// Verify Content-Type is set
			if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %s, want application/json", ct)
			}

			// Verify response is valid JSON
			var result map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
				t.Errorf("Response is not valid JSON: %v", err)
			}
		})
	}
}

// Test concurrent usage
func TestConcurrentResponses(t *testing.T) {
	iterations := 100
	done := make(chan bool, iterations)

	for i := 0; i < iterations; i++ {
		go func(idx int) {
			rr := httptest.NewRecorder()
			if idx%3 == 0 {
				SendErrorResponse(rr, http.StatusBadRequest, "error")
			} else if idx%3 == 1 {
				SendSuccessResponse(rr, map[string]int{"id": idx}, "success")
			} else {
				SendSuccessResponseWithNoData(rr, "success")
			}

			// Verify response is valid
			var result map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
				t.Errorf("Concurrent test failed: invalid JSON")
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < iterations; i++ {
		<-done
	}
}

// Benchmark tests
func BenchmarkSendErrorResponse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		SendErrorResponse(rr, http.StatusBadRequest, "error message")
	}
}

func BenchmarkSendSuccessResponse(b *testing.B) {
	data := map[string]string{"key": "value"}
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		SendSuccessResponse(rr, data, "success message")
	}
}

func BenchmarkSendSuccessResponseWithNoData(b *testing.B) {
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		SendSuccessResponseWithNoData(rr, "success message")
	}
}
