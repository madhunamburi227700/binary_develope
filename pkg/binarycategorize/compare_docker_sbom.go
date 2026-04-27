package binarycategorize

import (
	"fmt"
	"regexp"
	"strings"
)

// ENTRYPOINT

func CompareComponents(
	sbom CategorizedSBOM,
	docker DockerfileInput,
) ComparisonOutput {

	var out ComparisonOutput

	// OS COMPONENTS

	for _, c := range sbom.OSComponents {

		entry := makeEntry(c)

		ok, line, raw, score := matchOS(c, docker.OS)

		if ok {
			entry.InDockerfile = "yes"
			entry.LineNumber = line
			entry.DockerInstruction = raw
			entry.MatchScore = score

			// Check if it's a binary installation
			if isBinaryInstallation(raw) {
				entry.Category = "Binary"
			}
		}

		out.OSComponents = append(
			out.OSComponents,
			entry,
		)
	}

	// LIBRARY COMPONENTS

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

			// IMPROVED: Multi-layer binary detection
			isBinary := false

			// Check 1: Is it a shell/bash installation pattern?
			// This catches: curl | sh, wget | bash, /install.sh, etc.
			if isBinaryInstallation(raw) {
				isBinary = true
			}

			// Check 2: Does the component name appear in the installation URL?
			// This catches cases like "uv" in "https://astral.sh/uv/install.sh"
			if componentInInstallURL(c.Component, raw) && isBinaryInstallation(raw) {
				isBinary = true
			}

			// Check 3: Was it found in docker binary section?
			if entry.MatchScore > 0 {
				for _, binComp := range docker.Binary {
					if binComp.Raw == raw {
						isBinary = true
						break
					}
				}
			}

			// Check 4: SBOM category is already Binary
			if c.Category == "Binary" && isBinaryInstallation(raw) {
				isBinary = true
			}

			if isBinary {
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

// MATCH OS

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

// MATCH LIBRARIES

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

// STRONG MATCH LOGIC

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

	// NEW: Strategy 2a: Extract from install URL and match
	// This handles: "uv" from SBOM matching "https://astral.sh/uv/install.sh"
	urlComponentName := extractFromInstallURL(dockerComponent)
	if urlComponentName != "" && urlComponentName == normalizedSBOM {
		score = 88
		return true, score
	}

	// Strategy 2b: Last token match with word boundary check (strong)
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

// Extract component name from install URLs
// Extracts "uv" from "https://astral.sh/uv/install.sh"
func extractFromInstallURL(url string) string {
	url = strings.ToLower(url)

	// Remove common prefixes
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")

	// Split by / and look for install.sh, setup.sh patterns
	parts := strings.Split(url, "/")

	// Pattern: domain/component/install.sh or domain/component/setup.sh
	// Look for segments before common installer names
	for i := 0; i < len(parts); i++ {
		part := parts[i]

		// Skip domain-like parts (contain dots)
		if strings.Contains(part, ".") {
			continue
		}

		// Check if next part is an installer script
		if i+1 < len(parts) {
			nextPart := parts[i+1]
			if nextPart == "install.sh" ||
				nextPart == "setup.sh" ||
				nextPart == "get.sh" ||
				strings.HasSuffix(nextPart, ".sh") {
				// Found it!
				if part != "" && !isGenericName(part) {
					return normalize(part)
				}
			}
		}
	}

	return ""
}

// Check if component name appears in the curl/install command
// This catches things like "uv" in "https://astral.sh/uv/install.sh"
func componentInInstallURL(componentName, dockerRaw string) bool {
	// Normalize component name for matching
	normalized := normalize(componentName)

	if normalized == "" {
		return false
	}

	// Check if it appears in the raw docker instruction
	// Use word boundaries to avoid partial matches
	pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(normalized))
	re := regexp.MustCompile(pattern)

	return re.MatchString(strings.ToLower(dockerRaw))
}

// Added more generic names to prevent false positives
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
		"bin":    true,
		"src":    true,
		"test":   true,
	}

	return generic[s]
}

// Better binary installation detection - checks entire curl/sh/bash command structure
func isBinaryInstallation(raw string) bool {
	raw = strings.ToLower(raw)

	// Pattern 1: curl + pipe + sh/bash
	if strings.Contains(raw, "curl") &&
		(strings.Contains(raw, "| sh") ||
			strings.Contains(raw, "| bash") ||
			strings.Contains(raw, "sh -c") ||
			strings.Contains(raw, "bash -c")) {
		return true
	}

	// Pattern 2: Direct download with releases/download
	if strings.Contains(raw, "releases/download") {
		return true
	}

	// Pattern 3: Direct .sh file execution
	if strings.Contains(raw, "/install.sh") ||
		strings.Contains(raw, "/setup.sh") ||
		strings.Contains(raw, ".sh\"") ||
		strings.Contains(raw, ".sh'") {
		return true
	}

	// Pattern 4: wget + pipe + sh/bash
	if strings.Contains(raw, "wget") &&
		(strings.Contains(raw, "| sh") || strings.Contains(raw, "| bash")) {
		return true
	}

	// Pattern 5: Direct tar/zip extraction with curl/wget
	if (strings.Contains(raw, "curl") || strings.Contains(raw, "wget")) &&
		(strings.Contains(raw, "tar") ||
			strings.Contains(raw, "unzip") ||
			strings.Contains(raw, "gzip")) {
		return true
	}

	// Pattern 6: apt-get/yum/pacman install from external repos
	if (strings.Contains(raw, "apt-get") ||
		strings.Contains(raw, "yum install") ||
		strings.Contains(raw, "pacman")) &&
		(strings.Contains(raw, "curl") ||
			strings.Contains(raw, "wget") ||
			strings.Contains(raw, "python") ||
			strings.Contains(raw, "npm") ||
			strings.Contains(raw, "go") ||
			strings.Contains(raw, "github")) {
		return true
	}

	// Pattern 7: pip/npm/go install with specific versions or URLs
	if (strings.Contains(raw, "pip install") ||
		strings.Contains(raw, "npm install") ||
		strings.Contains(raw, "go install")) &&
		(strings.Contains(raw, "github") ||
			strings.Contains(raw, "http://") ||
			strings.Contains(raw, "https://")) {
		return true
	}

	// Pattern 8: Direct executable download (e.g., /usr/local/bin)
	if (strings.Contains(raw, "/usr/local/bin") ||
		strings.Contains(raw, "/usr/bin")) &&
		(strings.Contains(raw, "curl") || strings.Contains(raw, "wget")) {
		return true
	}

	return false
}

// LEGACY: Kept for backward compatibility (delegates to improved version)
func isBinaryInstall(raw string) bool {
	return isBinaryInstallation(raw)
}

// HELPER

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