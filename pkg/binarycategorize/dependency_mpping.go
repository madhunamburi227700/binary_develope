package binarycategorize

import "strings"


func EnrichDependencies(
	out ComparisonOutput,
	deps []Dependency,
) ComparisonOutput {

	directMap := buildDirectDependencyMap(
		deps,
	)

	reverseMap := buildReverseDependencyMap(
		deps,
	)

	out.OSComponents = enrichComponents(
		out.OSComponents,
		directMap,
		reverseMap,
	)

	out.LibraryComponents = enrichComponents(
		out.LibraryComponents,
		directMap,
		reverseMap,
	)

	out.BinaryComponents = enrichComponents(
		out.BinaryComponents,
		directMap,
		reverseMap,
	)

	return out
}

func buildDirectDependencyMap(
	deps []Dependency,
) map[string][]string {

	direct := make(
		map[string][]string,
	)

	for _, d := range deps {

		ref := normalizePurl(
			d.Ref,
		)

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

	reverse := make(
		map[string][]string,
	)

	for _, d := range deps {

		parent := stripPackageID(
			d.Ref,
		)

		for _, dep := range d.DependsOn {

			child := normalizePurl(
				dep,
			)

			reverse[child] = append(
				reverse[child],
				parent,
			)
		}
	}

	return reverse
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

func addDependencies(
	item *CategorizedComponent,
	direct map[string][]string,
) {

	p := normalizePurl(
		item.Purl,
	)

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

	p := normalizePurl(
		item.Purl,
	)

	if parents, ok := reverse[p]; ok {

		item.TransitiveOf = parents

	} else {

		item.TransitiveOf = []string{}
	}
}


func stripPackageID(
	p string,
) string {

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

func normalizePurl(
	p string,
) string {

	p = strings.ToLower(
		strings.TrimSpace(p),
	)

	if idx := strings.Index(
		p,
		"?package-id=",
	); idx != -1 {

		p = p[:idx]
	}

	if idx := strings.Index(
		p,
		"?",
	); idx != -1 {

		p = p[:idx]
	}

	p = strings.TrimPrefix(
		p,
		"pkg:",
	)

	if at := strings.Index(
		p,
		"@",
	); at != -1 {

		p = p[:at+1] +
			cleanVersion(
				p[at+1:],
			)
	}

	return p
}

func cleanVersion(
	v string,
) string {

	// keeps:
	// 12.2.0-14+deb12u1 -> 12.2.0-14
	if idx := strings.Index(
		v,
		"+",
	); idx != -1 {

		return v[:idx]
	}

	return v
}