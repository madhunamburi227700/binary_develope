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

type IntegrationStatus struct {
	IntegratorTypeId string `json:"integratorTypeId,omitempty"`
	IntegratorType   string `yaml:"integratorType,omitempty" json:"integratorType,omitempty"`
	Status           string `yaml:"status,omitempty" json:"status,omitempty"`
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

type GetIntegratorConfigResponse struct {
	QueryProject []Project `json:"queryProject"`
}

type Project struct {
	ID                string           `json:"id"`
	Name              string           `json:"name"`
	Platform          string           `json:"platform"`
	IntegratorConfigs IntegratorConfig `json:"integratorConfigs"`
}

type IntegratorConfig struct {
	Name    string   `json:"name"`
	Status  string   `json:"status"`
	Configs []Config `json:"configs"`
}

type Config struct {
	ID      string `json:"id"`
	Key     string `json:"key"`
	Value   string `json:"value"`
	Encrypt bool   `json:"encrypt"`
}

// Resource structures
type ResourceResponse struct {
	Integrations int `json:"integrations"`
	Rules        int `json:"rules"`
}

// RiskStatus tells us what risk a current application instance or a deployment is at.
type RiskStatus string

const (
	RiskStatusLowrisk        RiskStatus = "lowrisk"
	RiskStatusMediumrisk     RiskStatus = "mediumrisk"
	RiskStatusHighrisk       RiskStatus = "highrisk"
	RiskStatusApocalypserisk RiskStatus = "apocalypserisk"
	RiskStatusScanning       RiskStatus = "scanning"
	RiskStatusCompleted      RiskStatus = "completed"
	RiskStatusFail           RiskStatus = "fail"
	RiskStatusPending        RiskStatus = "pending"
)

// project
type ProjectRef struct {
	ID            string             `json:"id,omitempty" yaml:"id,omitempty"`
	Name          string             `json:"name,omitempty" yaml:"name,omitempty"`
	ScanType      string             `json:"scanType,omitempty" yaml:"scanType,omitempty"`
	Type          string             `json:"type,omitempty" yaml:"type,omitempty"`
	Platform      string             `json:"platform,omitempty" yaml:"platform,omitempty"`
	AccountID     string             `json:"accountId,omitempty" yaml:"accountId,omitempty"`
	TeamID        string             `json:"teamId" yaml:"teamId,omitempty"`
	TeamName      string             `json:"teamName,omitempty" yaml:"teamName,omitempty"`
	AccountName   string             `json:"accountName,omitempty" yaml:"accountName,omitempty"`
	Organisation  string             `json:"organisation,omitempty" yaml:"organisation,omitempty"`
	ScanLevel     string             `json:"scanLevel,omitempty" yaml:"scanLevel,omitempty"`
	Level         string             `json:"level,omitempty" yaml:"level,omitempty"`
	ProjectConfig []ProjectConfigRef `json:"projectConfigs,omitempty" yaml:"projectConfigs,omitempty"`
	Scans         []*ScanTargetRef   `json:"scans,omitempty"`
	CreatedAt     *time.Time         `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
	UpdatedAt     *time.Time         `json:"updatedAt,omitempty" yaml:"updatedAt,omitempty"`
}

type ProjectDetailsResponse struct {
	RiskStatus RiskStatus   `json:"riskStatus"`
	Error      string       `json:"error,omitempty"`
	ID         string       `json:"id,omitempty"`
	Team       *TeamDetails `json:"team,omitempty"`
	Scans      []*Scans     `json:"scans,omitempty"`
}

type Scans struct {
	Branch string `json:"branch"`
}

type TeamDetails struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ProjectConfigRef struct {
	ID                 string     `json:"id,omitempty" yaml:"id,omitempty"`
	Organisation       string     `json:"organisation,omitempty" yaml:"organisation,omitempty"`
	Repository         string     `json:"repository,omitempty" yaml:"repository,omitempty"`
	Branch             []string   `json:"branch,omitempty" yaml:"branch,omitempty"`
	BranchPattern      string     `json:"branchPattern,omitempty" yaml:"branchPattern,omitempty"`
	ArtifactTag        []string   `json:"tag,omitempty" yaml:"tag,omitempty"`
	ArtifactTagPattern string     `json:"tagPattern,omitempty" yaml:"tagPattern,omitempty"`
	ScheduleTime       *int       `json:"scheduleTime,omitempty" yaml:"scheduleTime,omitempty"`
	ScheduledScan      bool       `json:"scheduledScan,omitempty" yaml:"scheduledScan,omitempty"`
	CreatedAt          *time.Time `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
	UpdatedAt          *time.Time `json:"updatedAt,omitempty" yaml:"updatedAt,omitempty"`
	ScanUpto           *int       `json:"scanUpto,omitempty" yaml:"scanUpto,omitempty"`
}

type ScanTargetRef struct {
	Id                *string          `json:"id"`
	Projects          []*ProjectRef    `json:"projects,omitempty"`
	Organization      string           `json:"organization"`
	Repository        string           `json:"repository"`
	Branch            string           `json:"branch"`
	LastTriggeredBy   string           `json:"lastTriggeredBy"`
	LastScannedTime   *time.Time       `json:"lastScannedTime"`
	LastAttemptedTime *time.Time       `json:"lastAttemptedTime"`
	CreatedAt         *time.Time       `json:"createdAt"`
	UpdatedAt         *time.Time       `json:"updatedAt"`
	ScanResults       []*ScanResultRef `json:"scanResults,omitempty"`
	Artifact          *ArtifactRef     `json:"artifact,omitempty"`
	Error             string           `json:"error"`
	RiskStatus        RiskStatus       `json:"riskStatus"`
}

type ScanResultRef struct {
	Id           *string        `json:"id"`
	Group        string         `json:"group"`
	HeadCommit   string         `json:"headCommit"`
	TriggerdBy   string         `json:"triggerdBy"`
	TriggerType  string         `json:"triggerType"`
	ScanType     string         `json:"scanType"`
	ResultFile   string         `json:"resultFile"`
	ScanTool     string         `json:"scanTool"`
	ScannedAt    *time.Time     `json:"scannedAt"`
	ScanDuration *time.Time     `json:"scanDuration"`
	RiskStatus   RiskStatus     `json:"riskStatus"`
	ScanTarget   *ScanTargetRef `json:"scanTarget,omitempty"`
	Error        string         `json:"error"`
}

type ArtifactRef struct {
	Id           string                 `json:"id"`
	Attempt      *int                   `json:"attempt"`
	ArtifactType string                 `json:"artifactType"`
	ArtifactName string                 `json:"artifactName"`
	ArtifactTag  string                 `json:"artifactTag"`
	ArtifactSha  string                 `json:"artifactSha"`
	ScanData     []*ArtifactScanDataRef `json:"scanData,omitempty"`
	ScanTarget   []*ScanTargetRef       `json:"scanTarget,omitempty"`
}

type ArtifactScanDataRef struct {
	Id string `json:"id"`
	// platform: String! @search(by: [exact]) -> add later
	ArtifactSha       string       `json:"artifactSha"`
	ArtifactNameTag   string       `json:"artifactNameTag"`
	Tool              string       `json:"tool"`
	ArtifactDetails   *ArtifactRef `json:"artifactDetails,omitempty"`
	LastScannedAt     *time.Time   `json:"lastScannedAt"`
	CreatedAt         *time.Time   `json:"createdAt"`
	VulnTrackingId    string       `json:"vulnTrackingId"`
	VulnScanState     string       `json:"vulnScanState"`
	ScanState         string       `json:"scanState"`
	VulnCriticalCount *int         `json:"vulnCriticalCount"`
	VulnHighCount     *int         `json:"vulnHighCount"`
	VulnMediumCount   *int         `json:"vulnMediumCount"`
	VulnLowCount      *int         `json:"vulnLowCount"`
	VulnInfoCount     *int         `json:"vulnInfoCount"`
	VulnUnknownCount  *int         `json:"vulnUnknownCount"`
	VulnNoneCount     *int         `json:"vulnNoneCount"`
	VulnTotalCount    *int         `json:"vulnTotalCount"`
}

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
	TeamID      string `json:"teamId"`
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
	SCA     SCAData     `json:"SCA"`
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
type SCAData struct {
	CodeLicense ScanFiles `json:"codelicense"`
	CodeSecret  ScanFiles `json:"codesecret"`
	Sbom        ScanFiles `json:"sbom"`
}

// vuln findings
// VulnerabilityDataRequest represents the request for getting vulnerability data
type VulnerabilityDataRequest struct {
	Type      string `json:"type"`
	ProjectID string `json:"projectId"`
	ScanID    string `json:"scanId"`
}

// VulnerabilityDataResponse represents the response for vulnerability data
type VulnerabilityDataResponse struct {
	Results   []VulnerabilityScanResult `json:"results"`
	ScanID    string                    `json:"scanId"`
	Platform  string                    `json:"platform"`
	TotalSize int                       `json:"totalSize"`
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
	ScanID            string              `json:"scanId"`
	Platform          string              `json:"platform"`
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

// VulnerabilityData represents the response structure for vulnerability endpoints
// VulnerabilityOptimization represents the optimization data structure
type VulnerabilityOptimization struct {
	AllVulnerabilities    int `json:"allVulnerabilities"`
	UniqueVulnerabilities int `json:"uniqueVulnerabilities"`
	TopPriority           int `json:"topPriority"`
}

// VulnerabilityPriority represents the priority data structure
type VulnerabilityPriority struct {
	Vulnerabilities                 int            `json:"vulnerabilities"`
	Priority                        map[string]int `json:"priority"`
	VulnerabilityPrioritisationData []struct {
		Name        string  `json:"name"`
		Severity    string  `json:"severity"`
		CVSS        float64 `json:"cvss"`
		EPSS        float64 `json:"epss"`
		PriorityInt int     `json:"prirorityInt"`
	} `json:"vulnerabilityPrioritisationData"`
}

type DeleteIntegrationRequest struct {
	IntegrationID   string `json:"integrationId"`
	IntegrationName string `json:"integrationName"`
	IntegrationType string `json:"integrationType"`
	Level           string `json:"level"`
	TeamID          string `json:"teamId"`
}

type SASTScanRequest struct {
	Semgrep SASTScanToolDetails `json:"semgrep"`
}

type SASTScanToolDetails struct {
	ScanName   string `json:"scanName"`
	ScanTool   string `json:"scanTool"`
	ResultFile string `json:"resultFile"`
	Status     string `json:"status"`
}

type SASTScanResult struct {
	ScanName string `json:"scanName"`
	Data     []struct {
		Severity string `json:"severity"`
	} `json:"data"`
}
