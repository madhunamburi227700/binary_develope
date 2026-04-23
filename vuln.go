package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

//////////////////////////////////////////////////////
// STRUCTS
//////////////////////////////////////////////////////

type VulnInfo struct {
	ID   string `json:"id"`
	Purl string `json:"purl"`
}

type CategorizedComponent struct {
	Component         string   `json:"component"`
	Version           string   `json:"version"`
	Purl              string   `json:"purl"`
	Category          string   `json:"category"`
	InDockerfile      string   `json:"in_dockerfile"`
	LineNumber        int      `json:"line_number,omitempty"`
	DockerInstruction string   `json:"docker_instruction,omitempty"`

	DependsOn       []string   `json:"depends_on"`
	TransitiveOf    []string   `json:"transitive_of"`
	Vulnerabilities []VulnInfo `json:"vulnerabilities,omitempty"`
}

type ComparisonOutput struct {
	OSComponents      []CategorizedComponent `json:"os_components"`
	LibraryComponents []CategorizedComponent `json:"library_components"`
	BinaryComponents  []CategorizedComponent `json:"binary_components"`
}

type Comparison struct {
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
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

//////////////////////////////////////////////////////
// MATCHING LOGIC (UPDATED)
//////////////////////////////////////////////////////

func matchComponents(
	components []CategorizedComponent,
	comp Comparison,
) []CategorizedComponent {

	for i := range components {

		componentPurl := normalizePurl(components[i].Purl)

		for _, vuln := range comp.Vulnerabilities {

			for _, affect := range vuln.Affects {

				vulnPurl := normalizePurl(affect.Ref)

				// ---------- DIRECT ----------
				if componentPurl == vulnPurl {

					if isVersionAffected(
						components[i].Version,
						affect.Versions,
					) {
						addVuln(&components[i], vuln.ID, components[i].Purl)
					}
				}

				// ---------- DEPENDS ----------
				for _, dep := range components[i].DependsOn {

					if normalizePurl(dep) == vulnPurl {

						depVersion := extractVersionFromPurl(dep)

						if isVersionAffected(depVersion, affect.Versions) {
							addVuln(&components[i], vuln.ID, dep)
						}
					}
				}

				// ---------- TRANSITIVE ----------
				for _, trans := range components[i].TransitiveOf {

					if normalizePurl(trans) == vulnPurl {

						transVersion := extractVersionFromPurl(trans)

						if isVersionAffected(transVersion, affect.Versions) {
							addVuln(&components[i], vuln.ID, trans)
						}
					}
				}
			}
		}
	}

	return components
}

//////////////////////////////////////////////////////
// ADD VULNERABILITY (UPDATED)
//////////////////////////////////////////////////////

func addVuln(c *CategorizedComponent, vulnID, purl string) {

	for _, v := range c.Vulnerabilities {
		if v.ID == vulnID && v.Purl == purl {
			return
		}
	}

	c.Vulnerabilities = append(c.Vulnerabilities, VulnInfo{
		ID:   vulnID,
		Purl: purl,
	})
}

//////////////////////////////////////////////////////
// HELPERS
//////////////////////////////////////////////////////

func normalizePurl(purl string) string {

	p := strings.ToLower(strings.TrimSpace(purl))

	if idx := strings.Index(p, "?"); idx != -1 {
		p = p[:idx]
	}

	if idx := strings.Index(p, "@"); idx != -1 {
		p = p[:idx]
	}

	return p
}

func extractVersionFromPurl(purl string) string {
	if idx := strings.Index(purl, "@"); idx != -1 {
		v := purl[idx+1:]
		if q := strings.Index(v, "?"); q != -1 {
			v = v[:q]
		}
		return v
	}
	return ""
}

func isVersionAffected(
	version string,
	versions []Version,
) bool {

	for _, v := range versions {
		if v.Version == version &&
			v.Status == "affected" {
			return true
		}
	}

	return false
}

//////////////////////////////////////////////////////
// LOADERS
//////////////////////////////////////////////////////

func loadComponents(file string) ComparisonOutput {

	var c ComparisonOutput

	data, err := os.ReadFile(file)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(data, &c)
	if err != nil {
		panic(err)
	}

	return c
}

func loadVulnerabilities(file string) Comparison {

	var v Comparison

	data, err := os.ReadFile(file)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(data, &v)
	if err != nil {
		panic(err)
	}

	return v
}

//////////////////////////////////////////////////////
// SAVE OUTPUT
//////////////////////////////////////////////////////

func writeOutput(path string, out ComparisonOutput) {

	var buf bytes.Buffer

	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)

	err := enc.Encode(out)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(path, buf.Bytes(), 0644)
	if err != nil {
		panic(err)
	}
}

//////////////////////////////////////////////////////
// MAIN
//////////////////////////////////////////////////////

func main() {

	sbom := loadComponents("comparison_with_dependencies.json")

	vulns := loadVulnerabilities("sbom.json")

	sbom.OSComponents = matchComponents(
		sbom.OSComponents,
		vulns,
	)

	sbom.LibraryComponents = matchComponents(
		sbom.LibraryComponents,
		vulns,
	)

	sbom.BinaryComponents = matchComponents(
		sbom.BinaryComponents,
		vulns,
	)

	writeOutput("final_output.json", sbom)

	fmt.Println("✅ final_output.json generated")
}