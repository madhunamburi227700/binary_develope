package binarycategorize

import "strings"


func CategorizeSBOM(
	sbom SBOMInput,
) CategorizedSBOM {

	raw := pickLargestList(sbom)

	return bucketize(raw)
}


func pickLargestList(
	sbom SBOMInput,
) []RawComponent {

	candidates := [][]RawComponent{
		sbom.Components,
		sbom.Artifacts,
		sbom.Packages,
	}

	best := candidates[0]

	for _, c := range candidates[1:] {

		if len(c) > len(best) {
			best = c
		}
	}

	return best
}


func bucketize(
	components []RawComponent,
) CategorizedSBOM {

	seen := map[string]bool{}

	var osList []CategorizedComponent
	var libList []CategorizedComponent

	for _, c := range components {

		name, version, purl := c.Resolved()

		if purl == "" {
			continue
		}

		if seen[purl] {
			continue
		}

		seen[purl] = true

		record := CategorizedComponent{
			Component:    name,
			Version:      version,
			Purl:         purl,
			Category:     "",
			InDockerfile: "no",
			DependsOn:    []string{},
			TransitiveOf: []string{},
		}

		if isOSPurl(purl) {

			record.Category = "OS"

			osList = append(
				osList,
				record,
			)

		} else {

			record.Category = "Library"

			libList = append(
				libList,
				record,
			)
		}
	}

	return CategorizedSBOM{
		OSComponents:      osList,
		LibraryComponents: libList,
	}
}


func isOSPurl(
	purl string,
) bool {

	p := strings.ToLower(
		strings.TrimSpace(purl),
	)

	return strings.HasPrefix(p, "pkg:deb") ||
		strings.HasPrefix(p, "pkg:rpm") ||
		strings.HasPrefix(p, "pkg:apk")
}