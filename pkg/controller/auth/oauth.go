package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"

	oauthBridge "github.com/OpsMx/oauth-bridge-client"
	"github.com/opsmx/ai-guardian-api/pkg/auth/oauth"
	"github.com/opsmx/ai-guardian-api/pkg/auth/session"
	"github.com/opsmx/ai-guardian-api/pkg/repository"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

type OAuthController struct {
	googleOAuth *oauth.GoogleOAuth
	githubOAuth *oauth.GithubOAuth
	logger      *utils.ErrorLogger

	userRepo *repository.UserRepository
}

// NewOAuthController creates a new OAuth controller
func NewOAuthController() *OAuthController {
	return &OAuthController{
		googleOAuth: oauth.NewGoogleOAuth(),
		githubOAuth: oauth.NewGithubOAuth(),
		logger:      utils.NewErrorLogger("oauth_controller"),
		userRepo:    repository.NewUserRepository(),
	}
}

// GoogleLogin initiates Google OAuth flow
// @Summary Initiate Google OAuth login
// @Description Starts the OAuth 2.0 flow for Google authentication
// @Tags Authentication
// @Accept  json
// @Produce  json
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/google/login [get]
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
// @Summary Handle Google OAuth callback
// @Description Handles the OAuth 2.0 callback from Google
// @Tags Authentication
// @Accept  json
// @Produce  json
// @Param code query string true "OAuth authorization code"
// @Param state query string true "OAuth state parameter"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid request parameters"
// @Failure 401 {object} map[string]string "Authentication failed"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/google/callback [get]
func (ctrl *OAuthController) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	ctrl.googleOAuth.HandleCallback(w, r)
}

// Logout handles user logout
// @Summary Logout user
// @Description Invalidates the current user's session
// @Tags Authentication
// @Accept  json
// @Produce  json
// @Success 200 {object} map[string]string "Logout successful"
// @Router /auth/logout [post]
func (ctrl *OAuthController) Logout(w http.ResponseWriter, r *http.Request) {
	session.DeleteSession(w, r)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Logged out successfully",
	})
}

// GetProfile returns current user profile
// @Summary Get user profile
// @Description Returns the profile of the currently authenticated user
// @Tags Users
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /auth/profile [get]
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
	dbUser, err := ctrl.userRepo.GetByProviderUserID(context.TODO(), username)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"username":      username,
		"name":          dbUser.Name.String,
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

// GithubUserAuth returns GitHub OAuth URL for user authentication using bridge client
// @Summary GitHub User Auth
// @Description Returns OAuth URL for user authentication with email and profile scopes only
// @Tags Authentication
// @Accept  json
// @Produce  json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/github/oauth [get]
func (ctrl *OAuthController) GithubUserAuth(w http.ResponseWriter, r *http.Request) {
	// Initialize bridge client
	bridgeClient, err := oauthBridge.NewClient("ai-guardian")
	if err != nil {
		ctrl.logger.LogError(err, "Failed to initialize OAuth bridge client", nil)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Start OAuth installation with user scopes
	oauthResp, err := bridgeClient.StartOAuthInstallation("read:user", "user:email")
	if err != nil {
		ctrl.logger.LogError(err, "Failed to start OAuth installation", nil)
		ctrl.logger.LogWarning("This is expected if OAuth App credentials are not configured", nil)
		http.Error(w, "Failed to start OAuth installation", http.StatusInternalServerError)
		return
	}

	if !oauthResp.Success {
		ctrl.logger.LogError(errors.New(oauthResp.Error), "OAuth installation failed", nil)
		http.Error(w, "OAuth installation failed", http.StatusInternalServerError)
		return
	}

	// Return the OAuth URL
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]string{
			"url":   oauthResp.InstallURL,
			"state": oauthResp.State,
		},
	})
}

// GithubLogin verifies the email from github and redirects to UI
// @Summary Verifies github OAuth login
// @Description Starts the OAuth 2.0 flow for Github authentication
// @Tags Authentication
// @Accept  json
// @Produce  json
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /auth/github/login [post]
func (ctrl *OAuthController) GithubLogin(w http.ResponseWriter, r *http.Request) {
	ctrl.githubOAuth.HandleLogin(w, r)
}
