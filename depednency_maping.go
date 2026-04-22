package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type CategorizedComponent struct {
	Component         string   `json:"component"`
	Version           string   `json:"version"`
	Purl              string   `json:"purl"`
	Category          string   `json:"category"`
	InDockerfile      string   `json:"in_dockerfile"`
	LineNumber        int      `json:"line_number,omitempty"`
	DockerInstruction string   `json:"docker_instruction,omitempty"`

	// removed omitempty so these always show
	DependsOn    []string `json:"depends_on"`
	TransitiveOf []string `json:"transitive_of"`
}

type ComparisonOutput struct {
	OSComponents      []CategorizedComponent `json:"os_components"`
	LibraryComponents []CategorizedComponent `json:"library_components"`
}

type Dependency struct {
	Ref       string   `json:"ref"`
	DependsOn []string `json:"dependsOn"`
}

type SBOM struct {
	Dependencies []Dependency `json:"dependencies"`
}

/* ---------------- NORMALIZATION ---------------- */

func stripPackageID(p string) string {

	if idx := strings.Index(
		p,
		"&package-id=",
	); idx != -1 {

		end := strings.Index(
			p[idx+1:],
			"&",
		)

		if end != -1 {
			return p[:idx] +
				p[idx+1+end:]
		}

		return p[:idx]
	}

	return p
}

func normalize(p string) string {

	p = strings.ToLower(strings.TrimSpace(p))

	// remove package-id completely
	if idx := strings.Index(p, "?package-id="); idx != -1 {
		p = p[:idx]
	}

	// remove other query params
	if idx := strings.Index(p, "?"); idx != -1 {
		p = p[:idx]
	}

	// remove pkg: prefix
	p = strings.TrimPrefix(p, "pkg:")

	// remove version comparison noise after @
	if at := strings.Index(p, "@"); at != -1 {
		p = p[:at+1] + cleanVersion(p[at+1:])
	}

	return p
}
func cleanVersion(v string) string {
	// remove +build metadata like +deb12u13 etc
	if idx := strings.Index(v, "+"); idx != -1 {
		return v[:idx]
	}
	return v
}

/* ---------------- FILE LOADERS ---------------- */

func loadComparisonFile(path string) ComparisonOutput {

	var c ComparisonOutput

	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(data, &c)
	if err != nil {
		panic(err)
	}

	return c
}

func loadSBOMFile(path string) SBOM {

	var s SBOM

	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(data, &s)
	if err != nil {
		panic(err)
	}

	return s
}

/* ---------------- DEP MAPS ---------------- */

func buildDirectDependencyMap(
	deps []Dependency,
) map[string][]string {

	direct := make(map[string][]string)

	for _, d := range deps {

		ref := normalize(d.Ref)

		var children []string

		for _, dep := range d.DependsOn {
			children = append(
				children,
				stripPackageID(dep),
			)
		}

		direct[ref] = children
	}

	return direct
}

func buildReverseDependencyMap(
	deps []Dependency,
) map[string][]string {

	reverse := make(map[string][]string)

	for _, d := range deps {

		parent := stripPackageID(d.Ref)

		for _, dep := range d.DependsOn {

			child := normalize(dep)

			reverse[child] = append(
				reverse[child],
				parent,
			)
		}
	}

	return reverse
}

/* ---------------- ENRICHMENT ---------------- */

func addDependencies(
	item *CategorizedComponent,
	direct map[string][]string,
) {

	p := normalize(item.Purl)

	if deps, ok := direct[p]; ok {
		item.DependsOn = deps
	} else {
		item.DependsOn = []string{}
	}
}

func addTransitiveParents(
	item *CategorizedComponent,
	reverse map[string][]string,
) {

	p := normalize(item.Purl)

	if parents, ok := reverse[p]; ok {
		item.TransitiveOf = parents
	} else {
		item.TransitiveOf = []string{}
	}
}

func enrichComponents(
	items []CategorizedComponent,
	direct map[string][]string,
	reverse map[string][]string,
) []CategorizedComponent {

	for i := range items {

		addDependencies(
			&items[i],
			direct,
		)

		addTransitiveParents(
			&items[i],
			reverse,
		)
	}

	return items
}

/* ---------------- SAVE OUTPUT ---------------- */

func writeOutput(
	path string,
	out ComparisonOutput,
) {

	var buf bytes.Buffer

	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)

	err := enc.Encode(out)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(
		path,
		buf.Bytes(),
		0644,
	)

	if err != nil {
		panic(err)
	}
}

/* ---------------- MAIN ---------------- */

func main() {

	comparison :=
		loadComparisonFile(
			"comparisation_output.json",
		)

	sbom :=
		loadSBOMFile(
			"sbom.json",
		)

	directMap :=
		buildDirectDependencyMap(
			sbom.Dependencies,
		)

	reverseMap :=
		buildReverseDependencyMap(
			sbom.Dependencies,
		)

	comparison.OSComponents =
		enrichComponents(
			comparison.OSComponents,
			directMap,
			reverseMap,
		)

	comparison.LibraryComponents =
		enrichComponents(
			comparison.LibraryComponents,
			directMap,
			reverseMap,
		)

	writeOutput(
		"comparison_with_dependencies.json",
		comparison,
	)

	fmt.Println(
		"✅ comparison_with_dependencies.json generated",
	)
}