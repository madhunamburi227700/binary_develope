package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// -------- SBOM STRUCTS (multiple layouts) --------

type SBOMFlat struct {
	Components []Component `json:"components"`
}

type SBOMSyft struct {
	Artifacts []Component `json:"artifacts"`
}

type SBOMSDPX struct {
	Packages []Component `json:"packages"`
}

type SBOMCycloneDX struct {
	Components []Component `json:"components"`
	Metadata   struct {
		Component Component `json:"component"`
	} `json:"metadata"`
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

type Categorized struct {
	Component string `json:"component"`
	Version   string `json:"version"`
	Purl      string `json:"purl"`
	Category  string `json:"category"`
}

type FinalOutput struct {
	OSComponents      []Categorized `json:"os_components"`
	LibraryComponents []Categorized `json:"library_components"`
}

// -------- LANGUAGE --------

func extractLanguage(purl string) string {
	if !strings.HasPrefix(purl, "pkg:") {
		return ""
	}
	trimmed := strings.TrimPrefix(purl, "pkg:")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) == 0 {
		return ""
	}
	return strings.ToUpper(parts[0])
}

// -------- LOAD --------

func loadComponents(data []byte) ([]Component, error) {

	var flat SBOMFlat
	var syft SBOMSyft
	var spdx SBOMSDPX
	var cdx SBOMCycloneDX

	json.Unmarshal(data, &flat)
	json.Unmarshal(data, &syft)
	json.Unmarshal(data, &spdx)
	json.Unmarshal(data, &cdx)

	candidates := []struct {
		label string
		comps []Component
	}{
		{"components", flat.Components},
		{"artifacts", syft.Artifacts},
		{"packages", spdx.Packages},
	}

	best := candidates[0]
	for _, c := range candidates {
		if len(c.comps) > len(best.comps) {
			best = c
		}
	}

	if len(best.comps) == 0 {
		var raw []Component
		if err := json.Unmarshal(data, &raw); err == nil {
			return raw, nil
		}
		return nil, fmt.Errorf("no components found")
	}

	return best.comps, nil
}

// -------- CATEGORIZE --------

func categorize(components []Component) FinalOutput {

	seen := map[string]bool{}

	var osList []Categorized
	var libList []Categorized

	for _, c := range components {

		name, version, purl := c.Resolved()

		if purl == "" {
			continue
		}

		if seen[purl] {
			continue
		}
		seen[purl] = true

		record := Categorized{
			Component: name,
			Version:   version,
			Purl:      purl,
		}

		if strings.HasPrefix(purl, "pkg:deb") ||
			strings.HasPrefix(purl, "pkg:rpm") ||
			strings.HasPrefix(purl, "pkg:apk") {

			record.Category = "OS"
			osList = append(osList, record)

		} else {
			record.Category = "Library"
			libList = append(libList, record)
		}
	}

	return FinalOutput{
		OSComponents:      osList,
		LibraryComponents: libList,
	}
}

// -------- MAIN (ONLY FIX HERE) --------

func main() {

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run sbom.go <file>")
		return
	}

	file := os.Args[1]

	data, err := os.ReadFile(file)
	if err != nil {
		panic(err)
	}

	components, err := loadComponents(data)
	if err != nil {
		panic(err)
	}

	result := categorize(components)

	// 🔥 ONLY FIX IS HERE (IMPORTANT)

	var buf strings.Builder
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false) // ⭐ FIX: stops \u0026 escaping

	if err := enc.Encode(result); err != nil {
		panic(err)
	}

	err = os.WriteFile("categorized.json", []byte(buf.String()), 0644)
	if err != nil {
		panic(err)
	}

	fmt.Println("✅ Done → categorized.json")
}