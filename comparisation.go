package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
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
	OS      []DockerComponent `json:"os"`
	Library []DockerComponent `json:"library"`
	Binary  []DockerComponent `json:"binary"`
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
}

type Output struct {
	OSComponents      []CategorizedComponent `json:"os_components"`
	LibraryComponents []CategorizedComponent `json:"library_components"`
}

func normalize(s string) string {

	s = strings.ToLower(s)

	repl := []string{
		"github.com/",
		"pkg:",
		"golang/",
		"npm/",
		"deb/debian/",
		"%40",
		"@",
		"/v5",
		"/v4",
		"/v3",
		"/v2",
	}

	for _, r := range repl {
		s = strings.ReplaceAll(s, r, "")
	}

	return strings.TrimSpace(s)
}

func lastToken(s string) string {

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

	// owner/repo  e.g docker/cli
	return normalize(parts[0] + "/" + parts[1])
}

func strongMatch(sbomName string, dockerName string) bool {

	a := normalize(sbomName)
	b := normalize(dockerName)

	// exact full match
	if a == b {
		return true
	}

	// safer token match (exclude generic names)
	atok := lastToken(a)
	btok := lastToken(b)

	generic := map[string]bool{
		"cli":    true,
		"core":   true,
		"api":    true,
		"sdk":    true,
		"lib":    true,
		"common": true,
		"utils":  true,
	}

	if atok != "" &&
		atok == btok &&
		!generic[atok] {
		return true
	}

	// strongest github owner/repo comparison
	aproj := extractGitHubProject(sbomName)
	bproj := extractGitHubProject(dockerName)

	if aproj != "" && aproj == bproj {
		return true
	}

	return false
}

func matchOS(
	sb SBOMComponent,
	docker []DockerComponent,
) (bool, int, string) {

	for _, d := range docker {

		if strongMatch(
			sb.Component,
			d.ComponentName,
		) {
			return true,
				d.LineNumber,
				d.Raw
		}
	}

	return false, 0, ""
}

func matchLibrary(
	sb SBOMComponent,
	libs []DockerComponent,
	bins []DockerComponent,
) (bool, int, string) {

	for _, d := range libs {

		if strongMatch(
			sb.Component,
			d.ComponentName,
		) {
			return true,
				d.LineNumber,
				d.Raw
		}
	}

	for _, d := range bins {

		if strongMatch(
			sb.Component,
			d.ComponentName,
		) {
			return true,
				d.LineNumber,
				d.Raw
		}
	}

	return false, 0, ""
}

func buildOutput(
	sbom SBOM,
	docker DockerParse,
) Output {

	var out Output

	for _, c := range sbom.OSComponents {

		entry := CategorizedComponent{
			Component:    c.Component,
			Version:      c.Version,
			Purl:         c.Purl,
			Category:     c.Category,
			InDockerfile: "no",
		}

		ok, line, raw :=
			matchOS(c, docker.OS)

		if ok {
			entry.InDockerfile = "yes"
			entry.LineNumber = line
			entry.DockerInstruction = raw

			// override category if this is actually a binary download
			if strings.Contains(raw, "releases/download") {
				entry.Category = "Binary"
			}
		}

		out.OSComponents = append(
			out.OSComponents,
			entry,
		)
	}

	for _, c := range sbom.LibraryComponents {

		entry := CategorizedComponent{
			Component:    c.Component,
			Version:      c.Version,
			Purl:         c.Purl,
			Category:     c.Category,
			InDockerfile: "no",
		}

		ok, line, raw := matchLibrary(
			c,
			docker.Library,
			docker.Binary,
		)

		if ok {
			entry.InDockerfile = "yes"
			entry.LineNumber = line
			entry.DockerInstruction = raw
			// override category if library is actually used as binary download
			if strings.Contains(raw, "releases/download") {
				entry.Category = "Binary"
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