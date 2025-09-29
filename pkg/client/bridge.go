package client

import (
	"fmt"

	oauthBridge "github.com/OpsMx/oauth-bridge-client"
)

func GetGithubTokenFromInstallationId(installationId string) (string, error) {
	// Initialize client from environment variables
	bridgeClient, err := oauthBridge.NewClient("supplychain-api")
	if err != nil {
		return "", fmt.Errorf("unable to fetch github token, error while initializing oauth client: %s", err.Error())
	}

	// Generate token with all app permissions
	tokenResponse, err := bridgeClient.GenerateToken(installationId)
	if err != nil {
		return "", fmt.Errorf("unable to fetch github token, error while receiving token response: %s", err.Error())
	}

	return tokenResponse.Token, nil
}
