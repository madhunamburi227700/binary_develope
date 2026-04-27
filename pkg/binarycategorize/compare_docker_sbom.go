package binarycategorize

import (
	"fmt"
	"regexp"
	"strings"
)

// -------------------------------------------------------------------
// ENTRY POINT
// -------------------------------------------------------------------

func CompareComponents(
	sbom CategorizedSBOM,
	docker DockerfileInput,
) ComparisonOutput {

	var out ComparisonOutput

	//----------------------------------------------------
	// OS COMPONENTS
	//----------------------------------------------------

	for _, c := range sbom.OSComponents {

		entry := makeEntry(c)

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

		out.OSComponents = append(
			out.OSComponents,
			entry,
		)
	}

	//----------------------------------------------------
	// LIBRARY COMPONENTS
	//----------------------------------------------------

	for _, c := range sbom.LibraryComponents {

		entry := makeEntry(c)

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

			// Promote to binary if installed as downloadable tool
			if isBinaryInstall(raw) {
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

// -------------------------------------------------------------------
// MATCH OS
// -------------------------------------------------------------------

func matchOS(
	sb CategorizedComponent,
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

// -------------------------------------------------------------------
// MATCH LIBRARIES
// -------------------------------------------------------------------

func matchLibrary(
	sb CategorizedComponent,
	libs []DockerComponent,
	bins []DockerComponent,
) (bool, int, string, int) {

	bestScore := 0
	bestLine := 0
	bestRaw := ""

	// Search library section
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

	// Search binary section too
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

// -------------------------------------------------------------------
// STRONG MATCH LOGIC (WITH ALL FIXES FROM INDIVIDUAL SCRIPT)
// -------------------------------------------------------------------

func strongMatch(
	sbomComponent string,
	dockerComponent string,
	purl string,
) (bool, int) {

	score := 0
	maxScore := 100

	normalizedSBOM := normalize(sbomComponent)
	normalizedDocker := normalize(dockerComponent)

	// Quick validation: both must be non-empty
	if normalizedSBOM == "" || normalizedDocker == "" {
		return false, 0
	}

	// Strategy 1: Exact normalized match (highest score)
	if normalizedSBOM == normalizedDocker {
		return true, maxScore
	}

	// Strategy 2: Last token match with word boundary check (strong)
	tokenSBOM := lastToken(sbomComponent)
	tokenDocker := lastToken(dockerComponent)

	if tokenSBOM != "" && tokenDocker != "" && tokenSBOM == tokenDocker && !isGenericName(tokenSBOM) {
		// Check that it's not a prefix false positive
		if !isExactPrefixFalsePositive(tokenSBOM, tokenDocker) {
			score = 85
			return true, score
		}
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

	// Strategy 5: Substring match with word boundary (medium)
	// Only match if it's a word boundary match, not embedded in another word
	if len(normalizedSBOM) > 4 && len(normalizedDocker) > 4 {
		if isWordBoundaryMatch(normalizedSBOM, normalizedDocker) ||
			isWordBoundaryMatch(normalizedDocker, normalizedSBOM) {
			score = 60
			return true, score
		}
	}

	// Strategy 6: Partial token match with STRICT word boundary check (weaker)
	// For names like "python3-dev" vs "python3"
	// BUT reject matches like "security" in "aquasecurity" or "http" in "https"
	if len(normalizedSBOM) > 3 && len(normalizedDocker) > 3 {
		// CRITICAL: Reject if sbom is just a substring within docker without boundaries
		if (strings.HasPrefix(normalizedDocker, normalizedSBOM) ||
			strings.HasPrefix(normalizedSBOM, normalizedDocker)) &&
			!isExactPrefixFalsePositive(normalizedSBOM, normalizedDocker) {
			score = 50
			return true, score
		}
	}

	return false, 0
}

// Example: "security" should NOT match inside "aquasecurity"
func isWordBoundaryMatch(sbom, docker string) bool {
	// Create regex with word boundaries
	// Escape special regex characters in sbom
	escaped := regexp.QuoteMeta(sbom)

	// Pattern: word boundary + exact match + word boundary
	pattern := fmt.Sprintf(`\b%s\b`, escaped)
	re := regexp.MustCompile(pattern)

	return re.MatchString(docker)
}

// Prevent exact prefix false positives like "http" matching "https"
func isExactPrefixFalsePositive(sbom, docker string) bool {
	normSBOM := normalize(sbom)
	normDocker := normalize(docker)

	// If docker starts with sbom but has more characters after
	if strings.HasPrefix(normDocker, normSBOM) {
		// Check if the next character after the match is alphanumeric
		if len(normDocker) > len(normSBOM) {
			nextChar := rune(normDocker[len(normSBOM)])
			// If next character is alphanumeric or hyphen, it's a false positive
			// Examples: "http" + "s" = "https" (false), "http" + "-" = "http-server" (false)
			if (nextChar >= 'a' && nextChar <= 'z') ||
				(nextChar >= 'A' && nextChar <= 'Z') ||
				(nextChar >= '0' && nextChar <= '9') ||
				nextChar == '-' || nextChar == '_' || nextChar == '.' {
				return true
			}
		}
	}

	return false
}

// Extract version specifier with proper word boundaries
func stripVersionSpecifier(s string) string {
	// Match patterns like >=1.0.0, <2.0, ==1.2.3, ~=1.0, !=1.5, >1.0, <1.0, etc.
	// But NOT dots that are part of the package name itself
	re := regexp.MustCompile(`([><=!~]+|[@#]|\s+|[\[\{]).*`)
	return re.ReplaceAllString(s, "")
}

// Better normalization with word boundary awareness
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
		"pypi/",
		"maven-central/",
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

// Extract last token better with word boundaries
func lastToken(s string) string {

	// Step 1: Strip version first
	s = stripVersionSpecifier(s)

	// Step 2: Get last part after /
	if idx := strings.LastIndex(s, "/"); idx != -1 {
		s = s[idx+1:]
	}

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

// Extract GitHub project with better logic
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

// Extract package name from PURL with word boundaries
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

// IMPROVED: Added more generic names to prevent false positives
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
		"http":   true,
		"https":  true,
		"net":    true,
		"main":   true,
	}

	return generic[s]
}

// -------------------------------------------------------------------
// BINARY INSTALLATION DETECTION
// -------------------------------------------------------------------

func isBinaryInstall(raw string) bool {

	l := strings.ToLower(raw)

	return strings.Contains(
		raw,
		"releases/download",
	) ||
		strings.Contains(
			raw,
			"/install.sh",
		) ||
		(strings.Contains(l, "curl") &&
			strings.Contains(l, "| bash")) ||
		(strings.Contains(l, "curl") &&
			strings.Contains(l, "| sh"))
}

// -------------------------------------------------------------------
// HELPER
// -------------------------------------------------------------------

func makeEntry(
	c CategorizedComponent,
) CategorizedComponent {

	return CategorizedComponent{
		Component:    c.Component,
		Version:      c.Version,
		Purl:         c.Purl,
		Category:     c.Category,
		InDockerfile: "no",

		DependsOn:    []string{},
		TransitiveOf: []string{},
	}
}