package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// -------- SBOM STRUCTS --------

type SBOMFlat struct {
	Components []Component `json:"components"`
}

type SBOMSyft struct {
	Artifacts []Component `json:"artifacts"`
}

type SBOMSDPX struct {
	Packages []Component `json:"packages"`
}

// -------- COMPONENT --------

type Component struct {
	Name       string `json:"name"`
	CompAlt    string `json:"component"`
	Version    string `json:"version"`
	Purl       string `json:"purl"`
	PackageURL string `json:"packageUrl"`
	PurlAlt    string `json:"PackageURL"`
}

func (c Component) Resolved() (name, version, purl string) {
	name = firstNonEmpty(c.Name, c.CompAlt)
	version = c.Version
	purl = firstNonEmpty(c.Purl, c.PackageURL, c.PurlAlt)
	return
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// -------- OUTPUT --------

type SBOMComponent struct {
	Component string `json:"component"`
	Version   string `json:"version"`
	Purl      string `json:"purl"`
	Category  string `json:"category"`
}

type FinalOutput struct {
	SBOMComponents []SBOMComponent `json:"sbom_components"`
}

// -------- LOAD --------

func loadComponents(data []byte) ([]Component, error) {

	var flat SBOMFlat
	var syft SBOMSyft
	var spdx SBOMSDPX

	json.Unmarshal(data, &flat)
	json.Unmarshal(data, &syft)
	json.Unmarshal(data, &spdx)

	candidates := [][]Component{
		flat.Components,
		syft.Artifacts,
		spdx.Packages,
	}

	var best []Component
	for _, c := range candidates {
		if len(c) > len(best) {
			best = c
		}
	}

	if len(best) == 0 {
		var raw []Component
		if err := json.Unmarshal(data, &raw); err == nil {
			return raw, nil
		}
		return nil, fmt.Errorf("no components found")
	}

	return best, nil
}

// -------- BUILD sbom_components[] --------

func buildSBOMComponents(components []Component) FinalOutput {

	seen := map[string]bool{}
	var sbomList []SBOMComponent

	for _, c := range components {

		name, version, purl := c.Resolved()

		if purl == "" {
			continue
		}

		if seen[purl] {
			continue
		}
		seen[purl] = true

		category := "Library"

		if strings.HasPrefix(purl, "pkg:deb") ||
			strings.HasPrefix(purl, "pkg:rpm") ||
			strings.HasPrefix(purl, "pkg:apk") {
			category = "OS"
		}

		sbomList = append(sbomList, SBOMComponent{
			Component: name,
			Version:   version,
			Purl:      purl,
			Category:  category,
		})
	}

	return FinalOutput{
		SBOMComponents: sbomList,
	}
}

// -------- MAIN --------

func main() {

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run sbom.go <file>")
		return
	}

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	components, err := loadComponents(data)
	if err != nil {
		panic(err)
	}

	result := buildSBOMComponents(components)

	// Print output only (no JSON file save)
	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		panic(err)
	}

	fmt.Println(string(out))
}