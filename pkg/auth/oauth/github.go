package oauth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	oauthBridge "github.com/OpsMx/oauth-bridge-client"
	"github.com/jackc/pgx/v5"
	"github.com/opsmx/ai-guardian-api/pkg/auth/session"
	"github.com/opsmx/ai-guardian-api/pkg/config"
	"github.com/opsmx/ai-guardian-api/pkg/models"
	"github.com/opsmx/ai-guardian-api/pkg/repository"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

// GithubLoginData represents request data
type GithubLoginData struct {
	Token     string `json:"token"`
	Timestamp int64  `json:"timestamp"`
}

// GitHubUser represents a github user details
type GitHubUser struct {
	Login string `json:"login"`
	ID    int    `json:"id"`
	Type  string `json:"type"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// GitHubEmail represents a github user email
type GitHubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

type GithubOAuth struct {
	uiAddr string
	logger *utils.ErrorLogger

	userRepo *repository.UserRepository
}

func NewGithubOAuth() *GithubOAuth {
	return &GithubOAuth{
		uiAddr: config.GetUIAddress(),
		logger: utils.NewErrorLogger("github_oauth"),

		userRepo: repository.NewUserRepository(),
	}
}

func (g *GithubOAuth) HandleLogin(w http.ResponseWriter, r *http.Request) {
	loginData := &GithubLoginData{}
	if err := json.NewDecoder(r.Body).Decode(loginData); err != nil {
		g.logger.LogError(err, "failed to read request data", nil)
		http.Error(w, "failed to read request data", http.StatusBadRequest)
		return
	}
	dToken, err := decryptToken(loginData.Token, loginData.Timestamp)
	if err != nil {
		g.logger.LogError(err, "failed to decrypt token", map[string]interface{}{
			"encrypted_token": loginData.Token,
			"timestamp":       loginData.Timestamp,
		})
		http.Error(w, "failed to decrypt token", http.StatusInternalServerError)
		return
	}

	githubUser, err := getGithubUser(dToken)
	if err != nil {
		g.logger.LogError(err, "failed to authenticate via github", nil)
		http.Error(w, "failed to authenticate via github", http.StatusForbidden)
		return
	}

	if githubUser.Email == "" {
		// fetching email if not present
		emails, err := getGithubUserEmails(dToken)
		if err != nil {
			g.logger.LogError(err, "failed to fetch user emails", nil)
		} else {
			// assigning email id giving priority to verified
			// primary email
			for _, email := range emails {
				if githubUser.Email == "" {
					githubUser.Email = email.Email
				}
				if email.Primary && email.Verified {
					githubUser.Email = email.Email
					break
				}
			}
		}
	}

	// making github email id based on id
	// id for a user would always be same
	// whereas username, login, email can update
	userEmail := fmt.Sprintf("%d@github.com", githubUser.ID)

	// making user exits in db
	ctx := context.TODO()
	dbUser, err := g.userRepo.GetByProviderUserID(ctx, userEmail)
	if err != nil {
		if err != pgx.ErrNoRows {
			g.logger.LogError(err, "failed to get user", nil)
			http.Error(w, "failed to authenticate", http.StatusInternalServerError)
			return
		}
		// Handling new user case
		dbUser = &models.User{
			Email:          sql.NullString{String: githubUser.Email, Valid: githubUser.Email != ""},
			Name:           sql.NullString{String: githubUser.Name, Valid: true},
			Provider:       "github",
			ProviderUserID: userEmail,
		}
		err = g.userRepo.Create(ctx, dbUser)
		if err != nil {
			g.logger.LogError(err, "failed to create user", nil)
			http.Error(w, "failed to authenticate", http.StatusInternalServerError)
			return
		}
	}

	// if on first user creation email wasn't set then we will update
	if !dbUser.Email.Valid && githubUser.Email != "" {
		dbUser.Email = sql.NullString{String: githubUser.Email, Valid: githubUser.Email != ""}
		err = g.userRepo.Update(ctx, dbUser)
		if err != nil {
			g.logger.LogError(err, "failed to update user", nil)
			http.Error(w, "failed to authenticate", http.StatusInternalServerError)
			return
		}
	}

	// Create session using your existing session management
	session.CreateSession(w, r, "", userEmail)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    map[string]string{"email": userEmail},
	})
}

func decryptToken(eToken string, timestamp int64) (string, error) {
	bridgeClient, err := oauthBridge.NewClient("ai-guardian")
	if err != nil {
		return "", fmt.Errorf("failed to initialize oauth client: %s", err.Error())
	}

	oauthDecryptedToken, err := bridgeClient.DecryptToken(eToken, timestamp)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt oauth token: %s", err.Error())
	}

	return oauthDecryptedToken, nil
}

func getGithubUser(dtoken string) (*GitHubUser, error) {
	url := "https://api.github.com/user"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %s", err.Error())
	}

	req.Header.Set("Authorization", "Bearer "+dtoken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %s", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gitHub API error: %sResponse body: %s", resp.Status, string(body))
	}

	user := &GitHubUser{}
	if err := json.NewDecoder(resp.Body).Decode(user); err != nil {
		fmt.Println("Error decoding response:", err)
		return nil, err
	}

	return user, nil
}

func getGithubUserEmails(dtoken string) ([]GitHubEmail, error) {
	url := "https://api.github.com/user/emails"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %s", err.Error())
	}

	req.Header.Set("Authorization", "Bearer "+dtoken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %s", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gitHub API error: %s Response body: %s", resp.Status, string(body))
	}

	var emails []GitHubEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return nil, fmt.Errorf("error decoding response: %s", err.Error())
	}

	return emails, nil
}
