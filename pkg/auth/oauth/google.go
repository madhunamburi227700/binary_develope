package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/opsmx/ai-guardian-api/pkg/auth/session"
	"github.com/opsmx/ai-guardian-api/pkg/config"
	"github.com/opsmx/ai-guardian-api/pkg/models"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleOAuth struct {
	config *oauth2.Config
	logger *utils.ErrorLogger
	// Store PKCE configs temporarily (in production, use Redis)
	pkceStore map[string]*PKCEConfig
}

type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

type PKCEConfig struct {
	CodeVerifier        string
	CodeChallenge       string
	CodeChallengeMethod string
}

func NewGoogleOAuth() *GoogleOAuth {
	conf := &oauth2.Config{
		ClientID:     config.GetGoogleOIDCClientID(),
		ClientSecret: config.GetGoogleOIDCClientSecret(),
		RedirectURL:  config.GetGoogleOIDCRedirectURL(),
		Scopes:       config.GetGoogleOIDCScopes(),
		Endpoint:     google.Endpoint,
	}

	return &GoogleOAuth{
		config:    conf,
		logger:    utils.NewErrorLogger("google_oauth"),
		pkceStore: make(map[string]*PKCEConfig),
	}
}

func (g *GoogleOAuth) GetAuthURL(state string) string {
	if config.GetGoogleOIDCPKCE() {
		pkce, err := g.generatePKCE()
		if err != nil {
			g.logger.LogError(err, "Failed to generate PKCE", map[string]interface{}{
				"state": state,
			})
			return ""
		}

		// Store PKCE config with state for later validation
		g.pkceStore[state] = pkce
		g.logger.LogInfo("PKCE config stored", map[string]interface{}{
			"state": state,
		})

		return g.config.AuthCodeURL(state,
			oauth2.AccessTypeOffline,
			oauth2.SetAuthURLParam("code_challenge", pkce.CodeChallenge),
			oauth2.SetAuthURLParam("code_challenge_method", pkce.CodeChallengeMethod))
	}
	return g.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (g *GoogleOAuth) HandleCallback(w http.ResponseWriter, r *http.Request) {
	// Verify state parameter
	state := r.URL.Query().Get("state")
	if state == "" {
		g.logger.LogWarning("Missing state parameter", map[string]interface{}{
			"request_ip": r.RemoteAddr,
		})
		http.Error(w, "Missing state parameter", http.StatusBadRequest)
		return
	}

	// Get authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		g.logger.LogWarning("Missing authorization code", map[string]interface{}{
			"state": state,
		})
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	// Exchange authorization code for token
	ctx := context.Background()
	var token *oauth2.Token
	var err error

	if config.GetGoogleOIDCPKCE() {
		// Retrieve PKCE config for this state
		pkce, exists := g.pkceStore[state]
		if !exists {
			g.logger.LogError(nil, "PKCE config not found for state", map[string]interface{}{
				"state": state,
			})
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			return
		}

		// Exchange with PKCE code verifier
		token, err = g.config.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", pkce.CodeVerifier))

		// Clean up PKCE config after successful exchange
		if err == nil {
			delete(g.pkceStore, state)
		}
	} else {
		// Standard OAuth flow without PKCE
		token, err = g.config.Exchange(ctx, code)
	}

	if err != nil {
		g.logger.LogError(err, "Failed to exchange token", map[string]interface{}{
			"state": state,
		})
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	// Get user info from Google
	userInfo, err := g.getUserInfo(ctx, token)
	if err != nil {
		g.logger.LogError(err, "Failed to get user info", map[string]interface{}{
			"state": state,
		})
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}

	// Validate email is verified
	if !userInfo.VerifiedEmail {
		g.logger.LogWarning("Email not verified", map[string]interface{}{
			"email": userInfo.Email,
		})
		http.Error(w, "Email not verified", http.StatusBadRequest)
		return
	}

	// Create or get user from database
	user, err := g.createOrGetUser(userInfo)
	if err != nil {
		g.logger.LogError(err, "Failed to create or get user", map[string]interface{}{
			"email": userInfo.Email,
		})
		http.Error(w, "Failed to process user", http.StatusInternalServerError)
		return
	}

	// Create session using your existing session management
	session.CreateSession(w, r, token.RefreshToken, user.Email)

	g.logger.LogInfo("User authenticated successfully", map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	})

	// Return success response
	frontendUrl := fmt.Sprintf("%s/callback?success=true&email=%s", config.GetUIAddress(), user.Email)
	http.Redirect(w, r, frontendUrl, http.StatusFound)
}

func (g *GoogleOAuth) generatePKCE() (*PKCEConfig, error) {
	// Generate code verifier (43-128 characters)
	verifier := make([]byte, 32)
	if _, err := rand.Read(verifier); err != nil {
		return nil, err
	}
	codeVerifier := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(verifier)

	// Generate code challenge (SHA256 hash of verifier)
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hash[:])

	return &PKCEConfig{
		CodeVerifier:        codeVerifier,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: "S256",
	}, nil
}

func (g *GoogleOAuth) getUserInfo(ctx context.Context, token *oauth2.Token) (*GoogleUserInfo, error) {
	client := g.config.Client(ctx, token)

	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info, status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user info: %w", err)
	}

	return &userInfo, nil
}

func (g *GoogleOAuth) createOrGetUser(userInfo *GoogleUserInfo) (*models.User, error) {
	// TODO: Implement database operations
	// This should interact with your user repository

	// For now, return a mock user
	now := time.Now()
	status := "active"

	user := &models.User{
		ID:        uuid.New(),
		Email:     userInfo.Email,
		Name:      &userInfo.Name,
		GoogleID:  &userInfo.ID,
		Picture:   &userInfo.Picture,
		Status:    &status,
		CreatedAt: &now,
		UpdatedAt: &now,
	}

	g.logger.LogInfo("User created/retrieved", map[string]interface{}{
		"email":     userInfo.Email,
		"name":      userInfo.Name,
		"google_id": userInfo.ID,
	})

	return user, nil
}
