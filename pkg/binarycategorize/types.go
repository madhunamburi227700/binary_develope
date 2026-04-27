package binarycategorize

import "strings"

// INPUT – SBOM (multiple layouts supported)
type SBOMInput struct {
	// CycloneDX / flat
	Components []RawComponent `json:"components"`

	// Syft
	Artifacts []RawComponent `json:"artifacts"`

	// SPDX
	Packages []RawComponent `json:"packages"`

	// CycloneDX metadata root component
	Metadata struct {
		Component RawComponent `json:"component"`
	} `json:"metadata"`

	// CycloneDX vulnerability section
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`

	// Dependency graph
	Dependencies []Dependency `json:"dependencies"`
}

// RAW COMPONENT
type RawComponent struct {
	Name       string `json:"name"`
	CompAlt    string `json:"component"`
	Version    string `json:"version"`
	Purl       string `json:"purl"`
	PackageURL string `json:"packageUrl"`
	PurlAlt    string `json:"PackageURL"`
}

// Same logic as individual script
func (c RawComponent) Resolved() (
	name string,
	version string,
	purl string,
) {
	name = firstNonEmpty(
		c.Name,
		c.CompAlt,
	)

	version = c.Version

	purl = firstNonEmpty(
		c.Purl,
		c.PackageURL,
		c.PurlAlt,
	)

	return
}

// INPUT – DOCKER PARSER OUTPUT
type DockerfileInput struct {
	OS          []DockerComponent `json:"os"`
	Binary      []DockerComponent `json:"binary"`
	Library     []DockerComponent `json:"library"`
	Application []DockerComponent `json:"application"`
}

type DockerComponent struct {
	Source        string `json:"source"`
	Type          string `json:"type"`
	LineNumber    int    `json:"line_number"`
	ComponentName string `json:"component_name"`
	Raw           string `json:"raw"`
}

// INTERMEDIATE
type CategorizedSBOM struct {
	OSComponents      []CategorizedComponent `json:"os_components"`
	LibraryComponents []CategorizedComponent `json:"library_components"`
}

// FINAL OUTPUT
type ComparisonOutput struct {
	OSComponents      []CategorizedComponent `json:"os_components"`
	LibraryComponents []CategorizedComponent `json:"library_components"`
	BinaryComponents  []CategorizedComponent `json:"binary_components"`
}

// Shared structure used through call1-call5

type CategorizedComponent struct {
	Component         string `json:"component"`
	Version           string `json:"version"`
	Purl              string `json:"purl"`
	Category          string `json:"category"`
	InDockerfile      string `json:"in_dockerfile"`

	LineNumber        int    `json:"line_number,omitempty"`
	DockerInstruction string `json:"docker_instruction,omitempty"`
	MatchScore        int    `json:"-"` // "match_score,omitempty" not exported, used for scoring matches internally but not part of final output
	MatchedWith       string `json:"matched_with,omitempty"`

	DependsOn         []string `json:"depends_on"`
	TransitiveOf      []string `json:"transitive_of"`

	Vulnerabilities   []VulnInfo `json:"vulnerabilities,omitempty"`
}

// VULNERABILITIES

type VulnInfo struct {
	ID   string `json:"id"`
	Purl string `json:"purl"`
}

type Vulnerability struct {
	ID      string   `json:"id"`
	Affects []Affect `json:"affects"`
}

type Affect struct {
	Ref      string    `json:"ref"`
	Versions []Version `json:"versions"`
}

type Version struct {
	Version string `json:"version"`
	Status  string `json:"status"`
}

// DEPENDENCY GRAPH
type Dependency struct {
	Ref       string   `json:"ref"`
	DependsOn []string `json:"dependsOn"`
}

// HELPERS
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
