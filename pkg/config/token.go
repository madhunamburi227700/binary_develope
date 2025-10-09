package config

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

// SessionManager handles automatic session generation and refresh
type SessionManager struct {
	baseURL    string
	username   string
	password   string
	sessionID  string
	orgID      string
	httpClient *http.Client
	logger     *utils.ErrorLogger
	mutex      sync.RWMutex
	stopChan   chan struct{}
}

// SessionInfo holds session information
type SessionInfo struct {
	SessionID string    `json:"session_id"`
	OrgID     string    `json:"org_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		baseURL:    GetSSDBaseURL(),
		username:   GetUserOrgName(),
		password:   GetUserOrgPassword(),
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     utils.NewErrorLogger("session_manager"),
		stopChan:   make(chan struct{}),
	}
}

// Start starts the session manager with automatic refresh every 10 minutes
func (sm *SessionManager) Start(ctx context.Context) error {
	// Initial login
	if err := sm.login(); err != nil {
		return fmt.Errorf("initial login failed: %w", err)
	}

	// Start refresh goroutine
	go sm.refreshLoop(ctx)

	sm.logger.LogInfo("SSD's session manager started", nil)

	return nil
}

// Stop stops the session manager
func (sm *SessionManager) Stop() {
	close(sm.stopChan)
	sm.logger.LogInfo("SSD's session manager stopped", nil)
}

// GetSessionInfo returns current session information
func (sm *SessionManager) GetSessionInfo() SessionInfo {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return SessionInfo{
		SessionID: sm.sessionID,
		OrgID:     sm.orgID,
		ExpiresAt: time.Now().Add(10 * time.Minute), // Approximate expiration
	}
}

// GetSessionID returns the current session ID
func (sm *SessionManager) GetSessionID() string {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.sessionID
}

// GetOrgID returns the current organization ID
func (sm *SessionManager) GetOrgID() string {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.orgID
}

// login performs the login request to get a session
func (sm *SessionManager) login() error {
	loginURL := sm.baseURL + "/login?to="

	sm.logger.LogInfo("Attempting login", nil)

	// Prepare form data
	data := url.Values{}
	data.Set("username", sm.username)
	data.Set("password", sm.password)

	// Create request
	req, err := http.NewRequest("POST", loginURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	// Set headers
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", sm.baseURL)
	req.Header.Set("Referer", sm.baseURL+"/login?redir=")
	req.Header.Set("Sec-Ch-Ua", `"Chromium";v="134", "Not:A-Brand";v="24", "Opera";v="119"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Linux"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36 OPR/119.0.0.0")

	// Create HTTP client that doesn't follow redirects automatically
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	// Log response details for debugging
	sm.logger.LogInfo("Login response received", nil)

	// Extract session ID from the initial login response cookies first
	sessionID := sm.extractSessionFromCookies(resp.Cookies())
	if sessionID != "" {
		sm.logger.LogInfo("Session ID found in initial response", nil)

		// Get organization ID
		orgID, err := sm.getOrganizationID(sessionID)
		if err != nil {
			sm.logger.LogError(err, "Failed to get organization ID", nil)
			orgID = ""
		}

		// Update session info
		sm.mutex.Lock()
		sm.sessionID = sessionID
		sm.orgID = orgID
		sm.mutex.Unlock()

		sm.logger.LogInfo("Login successful", nil)

		return nil
	}

	// If no session in initial response, check if we got a redirect
	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusSeeOther || resp.StatusCode == 303 {
		location := resp.Header.Get("Location")
		sm.logger.LogInfo("Login redirect received", nil)

		// Handle relative URLs
		var redirectURL string
		if strings.HasPrefix(location, "http") {
			redirectURL = location
		} else {
			redirectURL = sm.baseURL + location
		}

		sm.logger.LogInfo("Following redirect", nil)

		// Follow the redirect to get the session cookie
		if location != "" {
			redirectReq, err := http.NewRequest("GET", redirectURL, nil)
			if err != nil {
				return fmt.Errorf("failed to create redirect request: %w", err)
			}

			// Copy cookies from the login response
			for _, cookie := range resp.Cookies() {
				redirectReq.AddCookie(cookie)
			}

			// Set headers for redirect request
			redirectReq.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
			redirectReq.Header.Set("Accept-Language", "en-US,en;q=0.9")
			redirectReq.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36 OPR/119.0.0.0")

			redirectResp, err := client.Do(redirectReq)
			if err != nil {
				return fmt.Errorf("redirect request failed: %w", err)
			}
			defer redirectResp.Body.Close()

			sm.logger.LogInfo("Redirect response received", nil)

			// Extract session ID from redirect response cookies
			sessionID := sm.extractSessionFromCookies(redirectResp.Cookies())
			if sessionID == "" {
				return fmt.Errorf("no session ID found in redirect response cookies")
			}

			// Get organization ID
			orgID, err := sm.getOrganizationID(sessionID)
			if err != nil {
				sm.logger.LogError(err, "Failed to get organization ID", nil)
				orgID = ""
			}

			// Update session info
			sm.mutex.Lock()
			sm.sessionID = sessionID
			sm.orgID = orgID
			sm.mutex.Unlock()

			sm.logger.LogInfo("Login successful via redirect", nil)

			return nil
		}
	}

	return fmt.Errorf("login failed with status: %d", resp.StatusCode)
}

// extractSessionFromCookies extracts session ID from response cookies
func (sm *SessionManager) extractSessionFromCookies(cookies []*http.Cookie) string {
	for _, cookie := range cookies {
		if cookie.Name == "SESSION" {
			return cookie.Value
		}
	}
	return ""
}

// getOrganizationID retrieves the organization ID using the session
func (sm *SessionManager) getOrganizationID(sessionID string) (string, error) {
	orgid := GetUserOrgID()
	if orgid == "" {
		return "983ebedf-99e7-4e4d-96c6-7a55a50d335e", nil
	}
	return orgid, nil
}

// refreshLoop runs the refresh loop every 10 minutes
func (sm *SessionManager) refreshLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := sm.login(); err != nil {
				sm.logger.LogError(err, "Session refresh failed", nil)
				// Continue trying - don't exit the loop
			} else {
				sm.logger.LogInfo("Session refreshed successfully", nil)
			}
		case <-sm.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// Global session manager instance
var sessionManager *SessionManager
var sessionManagerOnce sync.Once

// GetSessionManager returns the global session manager instance
func GetSessionManager() *SessionManager {
	sessionManagerOnce.Do(func() {
		sessionManager = NewSessionManager()
	})
	return sessionManager
}

// StartSessionManager starts the global session manager
func StartSessionManager(ctx context.Context) error {
	return GetSessionManager().Start(ctx)
}

// StopSessionManager stops the global session manager
func StopSessionManager() {
	if sessionManager != nil {
		sessionManager.Stop()
	}
}

// GetCurrentSessionID returns the current session ID
func GetCurrentSessionID() string {
	return GetSessionManager().GetSessionID()
}

// GetCurrentOrgID returns the current organization ID
func GetCurrentOrgID() string {
	return GetSessionManager().GetOrgID()
}

// GetCurrentSessionInfo returns current session information
func GetCurrentSessionInfo() SessionInfo {
	return GetSessionManager().GetSessionInfo()
}
