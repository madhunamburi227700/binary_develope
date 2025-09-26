package client

import (
	"fmt"
	"time"
)

// SSDConfig holds configuration for connecting to SSD services
type SSDConfig struct {
	BaseURL   string        `yaml:"base_url" json:"base_url"`
	OrgID     string        `yaml:"org_id" json:"org_id"`
	SessionID string        `yaml:"session_id" json:"session_id"`
	Timeout   time.Duration `yaml:"timeout" json:"timeout"`
	// Username  string        `yaml:"username" json:"username"`
	// Password  string        `yaml:"password" json:"password"`
}

// DefaultSSDConfig returns a default SSD configuration
func DefaultSSDConfig() *SSDConfig {
	return &SSDConfig{
		BaseURL: "https://july-dev.aoa.oes.opsmx.org",
		Timeout: 30 * time.Second,
	}
}

// Validate validates the SSD configuration
func (c *SSDConfig) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("base URL is required")
	}
	if c.OrgID == "" {
		return fmt.Errorf("organization ID is required")
	}
	if c.SessionID == "" {
		return fmt.Errorf("session ID is required")
	}
	return nil
}

// SSDClient provides access to SSD (OpsMx) APIs
type SSDClient struct {
	restClient *RESTClient
	orgID      string
	sessionID  string
}

// GraphQL Request/Response structures
type GraphQLRequest struct {
	Query string `json:"query"`
}

type GraphQLResponse struct {
	Data       interface{} `json:"data"`
	Extensions interface{} `json:"extensions,omitempty"`
	Errors     []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

// Organization structures
type Organization struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Roles []struct {
		Permission string `json:"permission"`
	} `json:"roles"`
	Teams []Hub `json:"teams"`
}

type Hub struct { //team
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Roles []struct {
		Permission string `json:"permission"`
	} `json:"roles"`
	Labels []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"labels"`
	Applications []struct {
		Environments []struct {
			Environment struct {
				Purpose string `json:"purpose"`
				ID      string `json:"id"`
			} `json:"environment"`
		} `json:"environments"`
	} `json:"applications"`
}

type OrganizationResponse struct {
	QueryOrganization []Organization `json:"queryOrganization"`
}

// Hub structures
type CreateHubRequest struct {
	Name  string `json:"name"`
	Tag   string `json:"tag"`
	Email string `json:"email"`
}

type CreateHubResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Integration structures
type Integration struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	IntegratorType    string                 `json:"integratorType"`
	Category          string                 `json:"category"`
	Status            string                 `json:"status"`
	AuthType          string                 `json:"authType"`
	URL               string                 `json:"url"`
	Team              []string               `json:"team"`
	Environments      []string               `json:"environments"`
	FeatureConfigs    map[string]interface{} `json:"featureConfigs"`
	IntegratorConfigs map[string]interface{} `json:"integratorConfigs"`
}

type CreateIntegrationRequest struct {
	Name              string                 `json:"name"`
	IntegratorType    string                 `json:"integratorType"`
	Category          string                 `json:"category"`
	FeatureConfigs    map[string]interface{} `json:"featureConfigs"`
	IntegratorConfigs map[string]interface{} `json:"integratorConfigs"`
	Team              []TeamAssignment       `json:"team"`
	ID                string                 `json:"id"`
}

type TeamAssignment struct {
	TeamName string `json:"teamName"`
	TeamID   string `json:"teamId"`
}

type ValidateIntegrationRequest struct {
	Name              string                 `json:"name"`
	IntegratorType    string                 `json:"integratorType"`
	Category          string                 `json:"category"`
	FeatureConfigs    map[string]interface{} `json:"featureConfigs"`
	IntegratorConfigs map[string]interface{} `json:"integratorConfigs"`
	Team              []TeamAssignment       `json:"team"`
	ID                string                 `json:"id"`
}

type ValidateIntegrationResponse struct {
	Message string `json:"Message"`
}

// Resource structures
type ResourceResponse struct {
	Integrations int `json:"integrations"`
	Rules        int `json:"rules"`
}

// project
type ProjectSummaryRequest struct {
	TeamIDs     string `json:"team_ids"`
	PageNo      int    `json:"page_no"`
	PageLimit   int    `json:"page_limit"`
	ProjectName string `json:"project_name"`
	Platform    string `json:"platform"`
	Status      string `json:"status"`
}

type ProjectSummaryResponse struct {
	ProjectSummaryResponse []ProjectSummary `json:"projectSummaryResponse"`
	TotalSize              int              `json:"totalSize"`
}

type ProjectSummary struct {
	ProjectID       string          `json:"projectId"`
	SummaryMetaData SummaryMetaData `json:"summaryMetaData"`
	Data            interface{}     `json:"data"`
}

type SummaryMetaData struct {
	ProjectName string `json:"projectName"`
	Platform    string `json:"platform"`
	// Organisation string `json:"organisation"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// type ProjectDetailsRequest struct {
// 	ProjectID string `json:"project_id"`
// 	TeamIDs   string `json:"team_ids"`
// }

// type ProjectDetailsResponse map[string]map[string][]ProjectBranch

type ProjectBranch struct {
	Branch           string  `json:"branch"`
	LastScanDuration float64 `json:"lastScanDuration"`
	LastScannedAt    string  `json:"lastScannedAt"`
	TriggeredBy      string  `json:"triggeredBy"`
	TriggerType      string  `json:"triggerType"`
	Status           string  `json:"status"`
	Error            string  `json:"error,omitempty"`
}

// count summary
type SummaryCountResponse struct {
	SourceScanSummaryCount SourceScanSummaryCount `json:"sourceScanSummaryCount"`
}

type SourceScanSummaryCount struct {
	AutoScanEnabledRepos int `json:"autoScanEnabledRepos"`
	ReposRegistered      int `json:"reposRegistered"`
	TotalBranches        int `json:"totalBranches"`
	TotalScans           int `json:"totalScans"`
	TotalProjects        int `json:"totalProjects"`
}

// vuln
type ScanResultDataRequest struct {
	Repository string `json:"repository"`
	TeamID     string `json:"teamId"`
	ProjectID  string `json:"projectId"`
	Type       string `json:"type"`
	Branch     string `json:"branch,omitempty"`
}

type ScanResultDataResponse struct {
	ScanID           string           `json:"scanId"`
	Branch           string           `json:"branch"`
	HeadCommit       string           `json:"headCommit"`
	LastScanDuration float64          `json:"lastScanDuration"`
	LastScannedAt    string           `json:"lastScannedAt"`
	TriggeredBy      string           `json:"triggeredBy"`
	TriggerType      string           `json:"triggerType"`
	ProjectID        string           `json:"projectId"`
	ProjectName      string           `json:"projectName"`
	ScanTool         string           `json:"scanTool"`
	ScanType         string           `json:"scanType"`
	Repository       string           `json:"repository"`
	ScannedFiledData ScannedFiledData `json:"scannedFiledData"`
	Platform         string           `json:"platform"`
	Status           string           `json:"status"`
	ArtifactName     string           `json:"artifactName"`
	ArtifactTag      string           `json:"artifactTag"`
	ArtifactSha      string           `json:"artifactSha"`
	SbomTool         string           `json:"sbomTool"`
}

// ScannedFiledData represents the scanned file data structure
type ScannedFiledData struct {
	OpenSSF OpenSSFData `json:"OpenSSF"`
	SAST    SASTData    `json:"SAST"`
	// SCA     SCAData     `json:"SCA"`
}

// OpenSSFData represents OpenSSF scan data
type OpenSSFData struct {
	Openssf ScanFiles `json:"openssf"`
}

// SASTData represents SAST scan data
type SASTData struct {
	Semgrep ScanFiles `json:"semgrep"`
}

// SemgrepScanData represents Semgrep scan data
type ScanFiles struct {
	ScanName   string `json:"scanName"`
	ScanTool   string `json:"scanTool"`
	ResultFile string `json:"resultFile"`
	Status     string `json:"status"`
	Error      string `json:"error"`
}

// SCAData represents SCA scan data
// type SCAData struct {
// 	CodeLicense CodeLicenseScanData `json:"codelicense"`
// 	CodeSecret  CodeSecretScanData  `json:"codesecret"`
// 	Sbom        SbomScanData        `json:"sbom"`
// }

// vuln findings
// VulnerabilityDataRequest represents the request for getting vulnerability data
type VulnerabilityDataRequest struct {
	Type      string `json:"type"`
	ProjectID string `json:"projectId"`
	ScanID    string `json:"scanId"`
}

// VulnerabilityDataResponse represents the response for vulnerability data
type VulnerabilityDataResponse struct {
	Results  []VulnerabilityScanResult `json:"results"`
	ScanID   string                    `json:"scanId"`
	Platform string                    `json:"platform"`
}

// VulnerabilityScanResult represents a single scan result
type VulnerabilityScanResult struct {
	ScanName      string                 `json:"scanName"`
	ScanTool      string                 `json:"scanTool"`
	Status        string                 `json:"status"`
	Error         string                 `json:"error"`
	Metadata      map[string]interface{} `json:"metadata"`
	Data          []VulnerabilityFinding `json:"data"`
	SummaryResult SummaryResult          `json:"summaryResult"`
}

// VulnerabilityFinding represents a single vulnerability finding
type VulnerabilityFinding struct {
	Severity    string                `json:"severity"`
	Confidence  string                `json:"confidence"`
	RuleName    string                `json:"rule_name"`
	RuleMessage string                `json:"rule_message"`
	Metadata    VulnerabilityMetadata `json:"metadata"`
	State       string                `json:"state"`
	CWE         []string              `json:"cwe"`
	OWASP       []string              `json:"owasp"`
	Fix         string                `json:"fix"`
}

// VulnerabilityMetadata represents metadata for a vulnerability finding
type VulnerabilityMetadata struct {
	FilePath string `json:"file_path"`
	Line     int    `json:"line"`
}

// SummaryResult represents the summary of scan results
type SummaryResult struct {
	Malicious        int `json:"malicious"`
	Suspicious       int `json:"suspicious"`
	Undetected       int `json:"undetected"`
	Harmless         int `json:"harmless"`
	Timeout          int `json:"timeout"`
	ConfirmedTimeout int `json:"confirmed-timeout"`
	Failure          int `json:"failure"`
	TypeUnsupported  int `json:"type-unsupported"`
}

// Rescan Request
type RescanRequest struct {
	ProjectID   string `json:"projectId"`
	ProjectName string `json:"projectName"`
	Platform    string `json:"platform"`
	ScanID      string `json:"scanId"`
	ScanType    string `json:"scanType"`
}

type RescanResponse struct {
	Message string `json:"message"`
}

// SCA
// VulnerabilityListRequest represents the request for getting vulnerability list
type VulnerabilityListRequest struct {
	OrgID       string `json:"orgId"`
	TeamID      string `json:"teamId"`
	PageNo      int    `json:"pageNo"`
	PageLimit   int    `json:"pageLimit"`
	SortBy      string `json:"sortBy"`
	SortOrder   string `json:"sortOrder"`
	Artifacts   string `json:"artifacts"`
	ArtifactSha string `json:"artifactSha"`
	Tools       string `json:"tools"`
}

// VulnerabilityListResponse represents the response for vulnerability list
type VulnerabilityListResponse struct {
	VulnerabilityList []VulnerabilityItem `json:"vulnerabilityList"`
	TotalSize         int                 `json:"totalSize"`
}

// VulnerabilityItem represents a single vulnerability item
type VulnerabilityItem struct {
	Component        []string `json:"component"`
	Severity         string   `json:"severity"`
	Priority         string   `json:"priority"`
	CVSS             float64  `json:"cvss"`
	EPSS             float64  `json:"epss"`
	InstalledVersion []string `json:"installedVersion"`
	FixedVersion     []string `json:"fixedVersion"`
	Title            string   `json:"title"`
	Vulnerability    string   `json:"vulnerability"`
	PublishedAt      string   `json:"publishedAt"`
	CWEList          []string `json:"cweList"`
	Artifacts        []string `json:"artifacts"`
	TimeScanned      string   `json:"timeScanned"`
	ArtifactSha      string   `json:"artifactSha"`
	Tool             string   `json:"tool"`
	Exploitation     string   `json:"exploitation,omitempty"`
	Automatable      string   `json:"automatable,omitempty"`
	TechnicalImpact  string   `json:"technicalImpact,omitempty"`
}
