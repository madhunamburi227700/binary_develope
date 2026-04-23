package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type DockerComponent struct {
	Source        string `json:"source"`
	Type          string `json:"type"`
	LineNumber    int    `json:"line_number"`
	ComponentName string `json:"component_name"`
	Raw           string `json:"raw"`
}

type DockerParse struct {
	OS          []DockerComponent `json:"os"`
	Library     []DockerComponent `json:"library"`
	Binary      []DockerComponent `json:"binary"`
	Application []DockerComponent `json:"application"`
}

type SBOMComponent struct {
	Component string `json:"component"`
	Version   string `json:"version"`
	Purl      string `json:"purl"`
	Category  string `json:"category"`
}

type SBOM struct {
	OSComponents      []SBOMComponent `json:"os_components"`
	LibraryComponents []SBOMComponent `json:"library_components"`
}

type CategorizedComponent struct {
	Component         string `json:"component"`
	Version           string `json:"version"`
	Purl              string `json:"purl"`
	Category          string `json:"category"`
	InDockerfile      string `json:"in_dockerfile"`
	LineNumber        int    `json:"line_number,omitempty"`
	DockerInstruction string `json:"docker_instruction,omitempty"`
	MatchScore        int    `json:"match_score,omitempty"` // ✅ Added for debugging
	MatchedWith       string `json:"matched_with,omitempty"` // ✅ Added for debugging
}

type Output struct {
	OSComponents      []CategorizedComponent `json:"os_components"`
	LibraryComponents []CategorizedComponent `json:"library_components"`
	BinaryComponents  []CategorizedComponent `json:"binary_components"`
}

// ✅ NEW: Extract version specifier from component name
// Examples: "scoutsuite>=5.12.0" → "scoutsuite"
//           "ruamel.yaml<0.19.0" → "ruamel.yaml"
//           "azure-identity>=1.16.1" → "azure-identity"
func stripVersionSpecifier(s string) string {
	// Match patterns like >=1.0.0, <2.0, ==1.2.3, ~=1.0, !=1.5, >1.0, <1.0, etc.
	re := regexp.MustCompile(`([><=!~]+|[@#]|\s|[\[\{]).*`)
	return re.ReplaceAllString(s, "")
}

// ✅ IMPROVED: Better normalization that strips versions
func normalize(s string) string {
	// Step 1: Strip version specifiers first
	s = stripVersionSpecifier(s)

	// Step 2: Lowercase
	s = strings.ToLower(s)

	// Step 3: Remove URL and packaging prefixes
	repl := []string{
		"github.com/",
		"pkg:",
		"golang/",
		"npm/",
		"deb/debian/",
		"pypi/",          // ✅ Added
		"maven-central/", // ✅ Added
		"%40",
		"@",
		"/v5",
		"/v4",
		"/v3",
		"/v2",
		"/v1",
	}

	for _, r := range repl {
		s = strings.ReplaceAll(s, r, "")
	}

	// Step 4: Remove trailing slashes and trim
	s = strings.TrimRight(s, "/")
	return strings.TrimSpace(s)
}

// ✅ IMPROVED: Extract last token (package name) better
func lastToken(s string) string {
	// Step 1: Strip version first
	s = stripVersionSpecifier(s)

	// Step 2: Get last part after /
	s = s[strings.LastIndex(s, "/")+1:]

	// Step 3: Remove @ version marker
	if idx := strings.Index(s, "@"); idx != -1 {
		s = s[:idx]
	}

	// Step 4: Remove # hash marker
	if idx := strings.Index(s, "#"); idx != -1 {
		s = s[:idx]
	}

	return normalize(s)
}

// ✅ NEW: Extract GitHub project with better logic
func extractGitHubProject(s string) string {
	// Step 1: Strip version specifiers
	s = stripVersionSpecifier(s)

	if !strings.Contains(s, "github.com/") {
		return ""
	}

	x := strings.Split(s, "github.com/")
	if len(x) < 2 {
		return ""
	}

	parts := strings.Split(x[1], "/")

	if len(parts) < 2 {
		return ""
	}

	// owner/repo e.g docker/cli
	return normalize(parts[0] + "/" + parts[1])
}

// ✅ NEW: Extract package name from PURL
// Examples:
//   pkg:pypi/scoutsuite@5.14.0 → scoutsuite
//   pkg:npm/@cyclonedx/cdxgen@10.0.0 → @cyclonedx/cdxgen or cdxgen
//   pkg:deb/debian/curl@7.68.0-1+deb10u5 → curl
func extractFromPurl(purl string) string {
	if !strings.Contains(purl, "pkg:") {
		return ""
	}

	// Remove pkg: prefix
	purl = strings.TrimPrefix(purl, "pkg:")

	// Split on @ to separate name from version
	parts := strings.Split(purl, "@")
	if len(parts) < 1 {
		return ""
	}

	name := parts[0] // e.g., "pypi/scoutsuite" or "npm/@cyclonedx/cdxgen"

	// Extract just the package name (after the last /)
	if idx := strings.LastIndex(name, "/"); idx != -1 {
		name = name[idx+1:]
	}

	// Remove @ if present (for scoped packages)
	name = strings.TrimPrefix(name, "@")

	return normalize(name)
}

// ✅ IMPROVED: Better matching with multiple strategies and scoring
func strongMatch(sbomComponent string, dockerComponent string, purl string) (bool, int) {
	score := 0
	maxScore := 100

	// Strategy 1: Exact normalized match (highest score)
	normalizedSBOM := normalize(sbomComponent)
	normalizedDocker := normalize(dockerComponent)

	if normalizedSBOM != "" && normalizedSBOM == normalizedDocker {
		return true, maxScore
	}

	// Strategy 2: Last token match (strong)
	tokenSBOM := lastToken(sbomComponent)
	tokenDocker := lastToken(dockerComponent)

	if tokenSBOM != "" && tokenSBOM == tokenDocker && !isGenericName(tokenSBOM) {
		score = 85
		return true, score
	}

	// Strategy 3: Extract from PURL if available (strong)
	purlName := extractFromPurl(purl)
	if purlName != "" && purlName == normalizedDocker {
		score = 90
		return true, score
	}

	// Strategy 4: GitHub project match (strong)
	githubSBOM := extractGitHubProject(sbomComponent)
	if githubSBOM != "" && githubSBOM == normalizedDocker {
		score = 80
		return true, score
	}

	// Strategy 5: Substring match for longer names (medium)
	// Only if both names are reasonably long and one contains the other
	if len(normalizedSBOM) > 4 && len(normalizedDocker) > 4 {
		if strings.Contains(normalizedDocker, normalizedSBOM) ||
			strings.Contains(normalizedSBOM, normalizedDocker) {
			score = 60
			return true, score
		}
	}

	// Strategy 6: Partial token match (weaker)
	// For names like "python3-dev" vs "python3"
	if strings.HasPrefix(normalizedDocker, normalizedSBOM) ||
		strings.HasPrefix(normalizedSBOM, normalizedDocker) {
		if len(normalizedSBOM) > 3 && len(normalizedDocker) > 3 {
			score = 50
			return true, score
		}
	}

	return false, 0
}

func isGenericName(s string) bool {
	generic := map[string]bool{
		"cli":    true,
		"core":   true,
		"api":    true,
		"sdk":    true,
		"lib":    true,
		"common": true,
		"utils":  true,
		"dev":    true,
		"tools":  true,
	}
	return generic[s]
}

// ✅ IMPROVED: Better OS component matching
func matchOS(
	sb SBOMComponent,
	docker []DockerComponent,
) (bool, int, string, int) {
	bestScore := 0
	bestLine := 0
	bestRaw := ""

	for _, d := range docker {
		ok, score := strongMatch(
			sb.Component,
			d.ComponentName,
			sb.Purl,
		)

		if ok && score > bestScore {
			bestScore = score
			bestLine = d.LineNumber
			bestRaw = d.Raw
		}
	}

	if bestScore > 0 {
		return true, bestLine, bestRaw, bestScore
	}

	return false, 0, "", 0
}

// ✅ IMPROVED: Better library component matching (searches both Library and Binary)
func matchLibrary(
	sb SBOMComponent,
	libs []DockerComponent,
	bins []DockerComponent,
) (bool, int, string, int) {
	bestScore := 0
	bestLine := 0
	bestRaw := ""

	// Search library section first
	for _, d := range libs {
		ok, score := strongMatch(
			sb.Component,
			d.ComponentName,
			sb.Purl,
		)

		if ok && score > bestScore {
			bestScore = score
			bestLine = d.LineNumber
			bestRaw = d.Raw
		}
	}

	// Also search binary section (for tools installed as binaries)
	for _, d := range bins {
		ok, score := strongMatch(
			sb.Component,
			d.ComponentName,
			sb.Purl,
		)

		if ok && score > bestScore {
			bestScore = score
			bestLine = d.LineNumber
			bestRaw = d.Raw
		}
	}

	if bestScore > 0 {
		return true, bestLine, bestRaw, bestScore
	}

	return false, 0, "", 0
}

// ✅ IMPROVED: Build output with better matching logic
func buildOutput(
	sbom SBOM,
	docker DockerParse,
) Output {

	var out Output

	//////////////////////////////////////////////////
	// OS COMPONENTS
	//////////////////////////////////////////////////

	for _, c := range sbom.OSComponents {

		entry := CategorizedComponent{
			Component:    c.Component,
			Version:      c.Version,
			Purl:         c.Purl,
			Category:     c.Category,
			InDockerfile: "no",
		}

		ok, line, raw, score := matchOS(c, docker.OS)

		if ok {
			entry.InDockerfile = "yes"
			entry.LineNumber = line
			entry.DockerInstruction = raw
			entry.MatchScore = score

			if strings.Contains(raw, "releases/download") {
				entry.Category = "Binary"
			}
		}

		out.OSComponents = append(out.OSComponents, entry)
	}

	//////////////////////////////////////////////////
	// LIBRARY COMPONENTS
	//////////////////////////////////////////////////

	for _, c := range sbom.LibraryComponents {

		entry := CategorizedComponent{
			Component:    c.Component,
			Version:      c.Version,
			Purl:         c.Purl,
			Category:     c.Category,
			InDockerfile: "no",
		}

		ok, line, raw, score := matchLibrary(
			c,
			docker.Library,
			docker.Binary,
		)

		if ok {
			entry.InDockerfile = "yes"
			entry.LineNumber = line
			entry.DockerInstruction = raw
			entry.MatchScore = score

			// ✅ IMPROVED: Better detection of binary components
			// Check if it was found in binary section OR has download pattern
			if strings.Contains(raw, "releases/download") ||
				strings.Contains(raw, "/install.sh") ||
				strings.Contains(raw, "curl") && strings.Contains(raw, "| bash") ||
				strings.Contains(raw, "curl") && strings.Contains(raw, "| sh") {
				entry.Category = "Binary"

				out.BinaryComponents = append(
					out.BinaryComponents,
					entry,
				)

				continue
			}
		}

		out.LibraryComponents = append(
			out.LibraryComponents,
			entry,
		)
	}

	return out
}

func main() {

	var docker DockerParse
	var sbom SBOM

	d1, err := os.ReadFile("docker_output.json")
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(d1, &docker)
	if err != nil {
		panic(err)
	}

	d2, err := os.ReadFile("categorized.json")
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(d2, &sbom)
	if err != nil {
		panic(err)
	}

	out := buildOutput(sbom, docker)

	var buf bytes.Buffer

	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)

	err = enc.Encode(out)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(
		"comparisation_output.json",
		buf.Bytes(),
		0644,
	)

	if err != nil {
		panic(err)
	}

	fmt.Println("✅ comparisation_output.json generated")
}