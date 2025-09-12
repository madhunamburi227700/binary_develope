package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/opsmx/ai-gyardian-api/pkg/auth/oauth"
	"github.com/opsmx/ai-gyardian-api/pkg/auth/session"
	"github.com/opsmx/ai-gyardian-api/pkg/utils"
)

type OAuthController struct {
	googleOAuth *oauth.GoogleOAuth
	logger      *utils.ErrorLogger
}

// NewOAuthController creates a new OAuth controller
func NewOAuthController() *OAuthController {
	return &OAuthController{
		googleOAuth: oauth.NewGoogleOAuth(),
		logger:      utils.NewErrorLogger("oauth_controller"),
	}
}

// GoogleLogin initiates Google OAuth flow
func (ctrl *OAuthController) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	// Generate state parameter for CSRF protection
	state, err := generateState()
	if err != nil {
		ctrl.logger.LogError(err, "Failed to generate state", nil)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Store state in session for validation
	// TODO: Store state in Redis for validation

	// Get OAuth URL
	authURL := ctrl.googleOAuth.GetAuthURL(state)
	if authURL == "" {
		ctrl.logger.LogError(nil, "Failed to generate auth URL", nil)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return auth URL
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"auth_url": authURL,
		"state":    state,
	})
}

// GoogleCallback handles Google OAuth callback
func (ctrl *OAuthController) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	ctrl.googleOAuth.HandleCallback(w, r)
}

// Logout handles user logout
func (ctrl *OAuthController) Logout(w http.ResponseWriter, r *http.Request) {
	session.DeleteSession(w, r)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Logged out successfully",
	})
}

// GetProfile returns current user profile
func (ctrl *OAuthController) GetProfile(w http.ResponseWriter, r *http.Request) {
	username, err := session.GetSession(r)
	if err != nil {
		ctrl.logger.LogWarning("User not authenticated", map[string]interface{}{
			"request_ip": r.RemoteAddr,
		})
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Get user details from database
	// TODO: Implement user details retrieval

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"username":      username,
		"authenticated": true,
	})
}

// generateState generates a random state parameter
func generateState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
