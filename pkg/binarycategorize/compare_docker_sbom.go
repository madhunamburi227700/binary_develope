package binarycategorize

import (
	"regexp"
	"strings"
)

// -------------------------------------------------------------------
// ENTRY
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
// STRONG MATCH LOGIC
// -------------------------------------------------------------------

func strongMatch(
	sbomComponent string,
	dockerComponent string,
	purl string,
) (bool, int) {

	normalizedSBOM := normalize(sbomComponent)
	normalizedDocker := normalize(dockerComponent)

	// Exact normalized match
	if normalizedSBOM != "" &&
		normalizedSBOM == normalizedDocker {
		return true, 100
	}

	// Last token match
	tokenSBOM := lastToken(sbomComponent)
	tokenDocker := lastToken(dockerComponent)

	if tokenSBOM != "" &&
		tokenSBOM == tokenDocker &&
		!isGenericName(tokenSBOM) {
		return true, 85
	}

	// PURL package match
	purlName := extractFromPurl(purl)

	if purlName != "" &&
		purlName == normalizedDocker {
		return true, 90
	}

	// github owner/repo match
	githubSBOM := extractGitHubProject(sbomComponent)

	if githubSBOM != "" &&
		githubSBOM == normalizedDocker {
		return true, 80
	}

	// substring
	if len(normalizedSBOM) > 4 &&
		len(normalizedDocker) > 4 {

		if strings.Contains(
			normalizedDocker,
			normalizedSBOM,
		) ||
			strings.Contains(
				normalizedSBOM,
				normalizedDocker,
			) {

			return true, 60
		}
	}

	// prefix fallback
	if len(normalizedSBOM) > 3 &&
		len(normalizedDocker) > 3 {

		if strings.HasPrefix(
			normalizedDocker,
			normalizedSBOM,
		) ||
			strings.HasPrefix(
				normalizedSBOM,
				normalizedDocker,
			) {

			return true, 50
		}
	}

	return false, 0
}

// -------------------------------------------------------------------
// NORMALIZATION
// -------------------------------------------------------------------

var versionRe = regexp.MustCompile(
	`([><=!~]+|[@#]|\s|[\[\{]).*`,
)

func stripVersionSpecifier(s string) string {
	return versionRe.ReplaceAllString(
		s,
		"",
	)
}

func normalize(s string) string {

	s = stripVersionSpecifier(s)

	s = strings.ToLower(s)

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
		s = strings.ReplaceAll(
			s,
			r,
			"",
		)
	}

	s = strings.TrimRight(s, "/")

	return strings.TrimSpace(s)
}

func lastToken(s string) string {

	s = stripVersionSpecifier(s)

	s = s[strings.LastIndex(s, "/")+1:]

	if idx := strings.Index(s, "@"); idx != -1 {
		s = s[:idx]
	}

	if idx := strings.Index(s, "#"); idx != -1 {
		s = s[:idx]
	}

	return normalize(s)
}

func extractGitHubProject(s string) string {

	s = stripVersionSpecifier(s)

	if !strings.Contains(s, "github.com/") {
		return ""
	}

	x := strings.Split(
		s,
		"github.com/",
	)

	if len(x) < 2 {
		return ""
	}

	parts := strings.Split(
		x[1],
		"/",
	)

	if len(parts) < 2 {
		return ""
	}

	return normalize(
		parts[0] + "/" + parts[1],
	)
}

func extractFromPurl(purl string) string {

	if !strings.Contains(
		purl,
		"pkg:",
	) {
		return ""
	}

	purl = strings.TrimPrefix(
		purl,
		"pkg:",
	)

	parts := strings.Split(
		purl,
		"@",
	)

	name := parts[0]

	if idx := strings.LastIndex(
		name,
		"/",
	); idx != -1 {
		name = name[idx+1:]
	}

	name = strings.TrimPrefix(
		name,
		"@",
	)

	return normalize(name)
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

// -------------------------------------------------------------------
// BINARY PROMOTION DETECTION
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