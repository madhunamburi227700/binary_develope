package client

import (
	"time"

	"github.com/opsmx/ai-guardian-api/pkg/config"
)

// SSDClientParams holds runtime parameters for SSD client (from service layer)
type SSDClientParams struct {
	OrgID     string
	SessionID string
}

// NewSSDClient creates a new SSD client using config for base URL and service layer params
func NewSSDClient() *SSDClient {
	baseURL := config.GetSSDBaseURL()
	if baseURL == "" {
		baseURL = "https://july-dev.aoa.oes.opsmx.org" // fallback
	}

	restConfig := RESTClientConfig{
		BaseURL: baseURL,
		Timeout: 30 * time.Second,
		Headers: map[string]string{
			"Accept":          "application/json, text/plain, */*",
			"Accept-Language": "en-US,en;q=0.9",
			"Content-Type":    "application/json",
			"User-Agent":      "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36 OPR/119.0.0.0",
		},
		Cookies: map[string]string{
			"SESSION": config.GetCurrentSessionID(),
		},
	}

	restClient := NewRESTClient(restConfig)

	return &SSDClient{
		restClient: restClient,
		orgID:      config.GetCurrentOrgID(),
		sessionID:  config.GetCurrentSessionID(),
	}
}
