package service

import (
	"context"
	"fmt"

	"github.com/opsmx/ai-guardian-api/pkg/client"
	"github.com/opsmx/ai-guardian-api/pkg/config"
	"github.com/opsmx/ai-guardian-api/pkg/repository"
	"github.com/opsmx/ai-guardian-api/pkg/utils"
)

type RemediationService struct {
	SSDService      *SSDService
	CSPMService     *CSPMService
	SSEClient       *client.SSEClient
	remediationRepo *repository.RemediationRepository
	logger          *utils.ErrorLogger
}

func NewRemediationService() *RemediationService {
	restConfig := client.RESTClientConfig{
		BaseURL: config.GetRemediationURL(),
		Timeout: 0,
	}

	RESTClient := client.NewRESTClient(restConfig)

	return &RemediationService{
		SSEClient:       client.NewSSEClient(RESTClient),
		SSDService:      NewSSDService(),
		logger:          utils.NewErrorLogger("remediation_service"),
		CSPMService:     NewCSPMService(),
		remediationRepo: repository.NewRemediationRepository(),
	}
}

type SASTRemediationRequest struct {
	ID              string  `json:"id"`
	HubID           string  `json:"hub_id"`
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
	if req.HubID == "" {
		return fmt.Errorf("hub_id is required")
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

	return s.SSEClient.SSERequest(ctx, "/sast-remediation/v1/fix", "POST", req, options, false)
}

type CVERemediationRequest struct {
	ID              string  `json:"id"`
	HubID           string  `json:"hub_id"`
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
	if req.HubID == "" {
		return fmt.Errorf("hub_id is required")
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

	return s.SSEClient.SSERequest(ctx, "/cve-remediation/v1/fix", "POST", req, options, false)
}

type RemediationConversationResponse struct {
	Conversation any `json:"conversation"`
}

func (s *RemediationService) Conversation(ctx context.Context, remId string) (*RemediationConversationResponse, error) {

	conversation, err := s.remediationRepo.GetConversation(ctx, remId)
	if err != nil {
		return nil, err
	}

	return &RemediationConversationResponse{
		Conversation: conversation,
	}, nil
}

func (s *RemediationService) NLI(ctx context.Context, req map[string]interface{}, headers, queryParams map[string][]string) (*client.SSEResponse, error) {
	options := client.MakeRequestOptions(headers, queryParams)

	return s.SSEClient.SSERequest(ctx, "/api/v1/nli/stream", "POST", &req, options, true)
}

type CSPMRemediationRequest struct {
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
	LineNo          *int    `json:"line_no,omitempty"`
	MessageType     string  `json:"message_type"`
	UserMessage     *string `json:"user_message,omitempty"`
	UserEmail       string  `json:"user_email,omitempty"`
	ArtifactSha     string  `json:"artifact_sha"`
	CommitSHA       string  `json:"commit_sha"`
}

func (s *RemediationService) ValidateCSPMRequest(req *CSPMRemediationRequest) error {
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
	return nil
}

func (s *RemediationService) CSPM(ctx context.Context, req *CSPMRemediationRequest, projectId string, headers, queryParams map[string][]string, commitsha string) (*client.SSEResponse, error) {
	options := client.MakeRequestOptions(headers, queryParams)

	token, err := s.SSDService.getIntegratorToken(ctx, projectId)
	if err != nil {
		return nil, err
	}
	req.Token = token

	req.ArtifactSha = s.SSDService.GetArtifactSha(ctx, req.Organization, req.Repository, commitsha)
	req.CommitSHA = commitsha
	return s.SSEClient.SSERequest(ctx, "/cspm-remediation/v1/fix", "POST", req, options, false)
}
