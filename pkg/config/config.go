package config

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v2"
)

type configType struct {
	AppName     string `yaml:"appName"`
	LogLevel    string `yaml:"logLevel"`
	Timezone    string `yaml:"timezone"`
	ApiHost     string `yaml:"apiHost"`
	ApiPort     string `yaml:"apiPort"`
	UIAddress   string `yaml:"uiAddr"`
	ShowVersion bool   `yaml:"showVersion"`
	// TODO: Remove
	Token   string `yaml:"githubToken"`
	CORSStr string `yaml:"cors_str,omitempty"`
	Pg      struct {
		Address  string `yaml:"address"`
		Port     string `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
		SSLMode  string `yaml:"sslMode"`
	}
	Redis struct {
		Address  string `yaml:"address"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
	}
	SSD struct {
		BaseURL  string `yaml:"baseURL"`
		UserName string `yaml:"userName"`
		Password string `yaml:"password"`
	}
	S3 struct {
		BucketName string `yaml:"bucketName"`
	} `yaml:"s3"`

	ScanningService struct {
		Addr string `yaml:"addr"`
	}
	RemediationService struct {
		Addr string `yaml:"addr"`
	} `yaml:"remediation_service"`

	Polling struct {
		Enabled         bool `yaml:"enabled"`
		IntervalSeconds int  `yaml:"intervalSeconds"`
	} `yaml:"polling"`

	Google struct {
		ClientID     string `yaml:"clientID"`
		ClientSecret string `yaml:"clientSecret"`
	}
	Auth struct {
		Type       string `yaml:"type"` // "google_oidc"
		GoogleOIDC struct {
			ClientID     string   `yaml:"client_id"`
			ClientSecret string   `yaml:"client_secret"`
			RedirectURL  string   `yaml:"redirect_url"`
			Scopes       []string `yaml:"scopes"`
			PKCE         bool     `yaml:"pkce"`
		} `yaml:"google_oidc"`
	} `yaml:"auth"`

	StartUpMessages  []string `yaml:"startUpMessages"`
	HomePage         string   `yaml:"homePage"`
	SessionStoreType string   `yaml:"sessionStoreType"`
	SMTP             struct {
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
	} `yaml:"smtp"`
	Feedback struct {
		AdminEmails       []string `yaml:"adminEmails"`
		EmailSubject      string   `yaml:"emailSubject"`
		EmailBodyTemplate string   `yaml:"emailBodyTemplate"`
	} `yaml:"feedback"`
	Notification struct {
		Enabled bool     `yaml:"enabled"`
		Type    string   `yaml:"type"` // "email"
		Emails  []string `yaml:"emails"`
	} `yaml:"notification"`
	AuditUsers []string `yaml:"auditUsers"`
}

var config configType
var SessionTimeout uint = 3600

func validateConfigPath(path string) error {
	s, err := os.Stat(path)
	if err != nil {
		return err
	}
	if s.IsDir() {
		return fmt.Errorf("'%s' is a directory, not a normal file", path)
	}
	return nil
}

// ParsesConfig the config returns error if fails
func ParseConfig(configPath string) error {
	// validate config path before decoding
	if err := validateConfigPath(configPath); err != nil {
		return err
	}

	// open config file
	file, err := os.Open(configPath)
	if err != nil {
		return err
	}
	//nolint: errcheck
	defer file.Close()

	// init new YAML decoder with file
	d := yaml.NewDecoder(file)

	// start YAML decoding from file
	if err := d.Decode(&config); err != nil {
		return err
	}

	return nil
}

func GetAppName() string {
	return config.AppName
}

func GetCORSStr() string {
	return config.CORSStr
}

func GetUIAddress() string {
	return config.UIAddress
}

func GetCorsOrigin() string {
	if config.CORSStr != "" {
		return config.CORSStr
	}
	// Default to localhost:3000 if not configured
	return "http://localhost:3000"
}

func GetLogLevel() string {
	return config.LogLevel
}

func GetTimezone() string {
	return config.Timezone
}

func GetApiHost() string {
	return config.ApiHost
}

func GetApiPort() string {
	return config.ApiPort
}

func GetSessionStoreType() string {
	return config.SessionStoreType
}

// Add getter functions
func GetAuthType() string {
	return config.Auth.Type
}

func GetGoogleOIDCClientID() string {
	return config.Auth.GoogleOIDC.ClientID
}

func GetGoogleOIDCClientSecret() string {
	return config.Auth.GoogleOIDC.ClientSecret
}

func GetGoogleOIDCRedirectURL() string {
	return config.Auth.GoogleOIDC.RedirectURL
}

func GetGoogleOIDCScopes() []string {
	return config.Auth.GoogleOIDC.Scopes
}

func GetGoogleOIDCPKCE() bool {
	return config.Auth.GoogleOIDC.PKCE
}

func GetPgAddress() string {
	return fmt.Sprintf("%s:%s@%s:%s/%s", config.Pg.User,
		config.Pg.Password, config.Pg.Address, config.Pg.Port, config.Pg.Database)
}

func GetS3BucketName() string {
	return config.S3.BucketName
}

func GetScanningServiceAddr() string {
	return config.ScanningService.Addr
}

// GetShowVersion returns whether the version should be shown
func GetShowVersion() bool {
	return config.ShowVersion
}

func GetGoogleClientID() string {
	return config.Google.ClientID
}

func GetGoogleClientSecret() string {
	return config.Google.ClientSecret
}

func GetPgConfig() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", config.Pg.User,
		config.Pg.Password, config.Pg.Address, config.Pg.Port, config.Pg.Database, config.Pg.SSLMode)
}

// GetRedisConfig returns the Redis configuration
func GetRedisConfig() (string, string, string) {
	return config.Redis.User, config.Redis.Password, config.Redis.Address
}

func GetSSDBaseURL() string {
	return config.SSD.BaseURL
}

// GetSSDConfig returns the complete SSD configuration
func GetSSDConfig() string {
	return config.SSD.BaseURL
}

func GetUserOrgName() string {
	return config.SSD.UserName
}

func GetUserOrgPassword() string {
	return config.SSD.Password
}

func GetRemediationURL() string {
	return config.RemediationService.Addr
}

// TODO: Remove later on
func GetGithubTokenTemp() string {
	return config.Token
}

// GetPollingEnabled returns whether polling is enabled
func GetPollingEnabled() bool {
	return config.Polling.Enabled
}

// GetPollingIntervalSeconds returns the polling interval in seconds
func GetPollingIntervalSeconds() int {
	if config.Polling.IntervalSeconds <= 0 {
		return 300 // Default to 5 minutes
	}
	return config.Polling.IntervalSeconds
}

// GetSMTPConfig returns smtp configuration
func GetSMTPConfig() (smtpHost, smtpPort, smtpUser, smtpPassword string) {
	return config.SMTP.Host,
		config.SMTP.Port,
		config.SMTP.User,
		config.SMTP.Password
}

// GetFeedbackConfig returns feedback configuration
func GetFeedbackConfig() (emailSubject, emailBodyTemplate string, adminEmails []string) {
	return config.Feedback.EmailSubject,
		config.Feedback.EmailBodyTemplate,
		config.Feedback.AdminEmails
}

// GetNotificationEnabled returns whether polling is enabled
func GetNotificationEnabled() bool {
	return config.Notification.Enabled
}

// GetNotificationConfig returns notification configuration
func GetNotificationConfig() (string, []string) {
	return config.Notification.Type,
		config.Notification.Emails
}

// GetAuditUsers the users which are allowed to do audit
func GetAuditUsers() []string {
	return config.AuditUsers
}
