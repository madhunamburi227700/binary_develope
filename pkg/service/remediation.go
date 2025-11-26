package service

import (
	"context"
	"fmt"

	"github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/config"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

type RemediationService struct {
	SSDService *SSDService
	SSEClient  *client.SSEClient
	logger     *utils.ErrorLogger
}

func NewRemediationService() *RemediationService {
	restConfig := client.RESTClientConfig{
		BaseURL: config.GetRemediationURL(),
		Timeout: 0,
	}

	RESTClient := client.NewRESTClient(restConfig)

	return &RemediationService{
		SSEClient:  client.NewSSEClient(RESTClient),
		SSDService: NewSSDService(),
		logger:     utils.NewErrorLogger("remediation_service"),
	}
}

type SASTRemediationRequest struct {
	ID              string  `json:"id"`
	VulnerabilityID string  `json:"vulnerability_id"`
	SessionID       *string `json:"session_id,omitempty"`
	Platform        string  `json:"platform"`
	Organization    string  `json:"organization"`
	Repository      string  `json:"repository"`
	Token           string  `json:"token"`
	Branch          string  `json:"branch"`
	Rule            string  `json:"rule"`
	RuleMessage     string  `json:"rule_message"`
	FilePath        string  `json:"file_path"`
	LineNo          *int    `json:"line_no"`
	MessageType     string  `json:"message_type"`
	UserMessage     *string `json:"user_message,omitempty"`
	UserEmail       string  `json:"user_email,omitempty"`
}

func (s *RemediationService) ValidateSASTRequest(req *SASTRemediationRequest) error {
	if req.ID == "" {
		return fmt.Errorf("id is required")
	}
	if req.VulnerabilityID == "" {
		return fmt.Errorf("vulnerability_id is required")
	}
	if req.Platform == "" {
		return fmt.Errorf("platform is required")
	}
	if req.Organization == "" {
		return fmt.Errorf("organization is required")
	}
	if req.Repository == "" {
		return fmt.Errorf("repository is required")
	}
	if req.Branch == "" {
		return fmt.Errorf("branch is required")
	}
	if req.Rule == "" {
		return fmt.Errorf("rule is required")
	}
	if req.RuleMessage == "" {
		return fmt.Errorf("rule_message is required")
	}
	if req.FilePath == "" {
		return fmt.Errorf("file_path is required")
	}
	if req.LineNo == nil {
		return fmt.Errorf("line_no is required")
	}
	return nil
}

func (s *RemediationService) SAST(ctx context.Context, req *SASTRemediationRequest, projectId string, headers, queryParams map[string][]string) (*client.SSEResponse, error) {
	options := client.MakeRequestOptions(headers, queryParams)

	// TODO: Temp remove
	if config.GetGithubTokenTemp() != "" {
		req.Token = config.GetGithubTokenTemp()
	} else {
		token, err := s.SSDService.getIntegratorToken(ctx, projectId)
		if err != nil {
			return nil, err
		}
		req.Token = token
	}

	return s.SSEClient.SSERequest(ctx, "/sast-remediation/v1/fix", "POST", req, options)
}

type CVERemediationRequest struct {
	ID              string  `json:"id"`
	VulnerabilityID string  `json:"vulnerability_id"`
	SessionID       *string `json:"session_id,omitempty"`
	Token           string  `json:"token"`
	Platform        string  `json:"platform"`
	Organization    string  `json:"organization"`
	Repository      string  `json:"repository"`
	CVEID           string  `json:"cve_id"`
	Package         string  `json:"package"`
	Branch          *string `json:"branch,omitempty"`
	MessageType     string  `json:"message_type"`
	UserMessage     *string `json:"user_message,omitempty"`
	UserEmail       string  `json:"user_email,omitempty"`
}

func (s *RemediationService) ValidateCVERequest(req *CVERemediationRequest) error {
	if req.ID == "" {
		return fmt.Errorf("id is required")
	}
	if req.VulnerabilityID == "" {
		return fmt.Errorf("vulnerability_id is required")
	}
	if req.Platform == "" {
		return fmt.Errorf("platform is required")
	}
	if req.Organization == "" {
		return fmt.Errorf("organization is required")
	}
	if req.Repository == "" {
		return fmt.Errorf("repository is required")
	}
	if req.CVEID == "" {
		return fmt.Errorf("cve_id is required")
	}
	if req.Package == "" {
		return fmt.Errorf("package is required")
	}
	if req.MessageType == "" {
		return fmt.Errorf("message_type is required")
	}
	if req.MessageType != "start_generate" && req.MessageType != "start_apply" && req.MessageType != "followup" {
		return fmt.Errorf("message_type must be one of: start_generate, start_apply, followup")
	}

	return nil
}

func (s *RemediationService) CVE(ctx context.Context, req *CVERemediationRequest, projectId string, headers, queryParams map[string][]string) (*client.SSEResponse, error) {
	options := client.MakeRequestOptions(headers, queryParams)

	// TODO: Temp remove
	if config.GetGithubTokenTemp() != "" {
		req.Token = config.GetGithubTokenTemp()
	} else {
		token, err := s.SSDService.getIntegratorToken(ctx, projectId)
		if err != nil {
			return nil, err
		}
		req.Token = token
	}

	return s.SSEClient.SSERequest(ctx, "/cve-remediation/v1/fix", "POST", req, options)
}
