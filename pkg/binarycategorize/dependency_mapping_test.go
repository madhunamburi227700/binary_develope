package binarycategorize

import "testing"

// cleanVersion
func TestCleanVersion(t *testing.T) {

	got := cleanVersion(
		"12.2.0-14+deb12u1",
	)

	want := "12.2.0-14"

	if got != want {
		t.Fatalf(
			"got %s want %s",
			got,
			want,
		)
	}
}

// normalizePurl
func TestNormalizePurl(t *testing.T) {

	got := normalizePurl(
		"pkg:deb/debian/gcc@12.2.0-14+deb12u1?package-id=abc",
	)

	want := "deb/debian/gcc@12.2.0-14"

	if got != want {
		t.Fatalf(
			"got %s want %s",
			got,
			want,
		)
	}
}

// stripPackageID
func TestStripPackageID(t *testing.T) {

	in := "pkg:pypi/requests@2.31&package-id=123"

	got := stripPackageID(in)

	if got == in {
		t.Fatal("package-id was not stripped")
	}
}

// direct dependency map
func TestBuildDirectDependencyMap(t *testing.T) {

	deps := []Dependency{
		{
			Ref:"pkg:pypi/app@1.0",
			DependsOn: []string{
				"pkg:pypi/requests@2.31",
				"pkg:pypi/flask@3.0",
			},
		},
	}

	m := buildDirectDependencyMap(deps)

	if len(m["pypi/app@1.0"]) != 2 {
		t.Fatal("expected 2 direct deps")
	}
}

// reverse dependency map
func TestBuildReverseDependencyMap(t *testing.T) {

	deps := []Dependency{
		{
			Ref:"pkg:pypi/app@1.0",
			DependsOn: []string{
				"pkg:pypi/requests@2.31",
			},
		},
	}

	m := buildReverseDependencyMap(deps)

	if len(m["pypi/requests@2.31"]) != 1 {
		t.Fatal("reverse dependency missing")
	}
}

// addDependencies
func TestAddDependencies(t *testing.T) {

	item := CategorizedComponent{
		Purl:"pkg:pypi/app@1.0",
	}

	direct := map[string][]string{
		"pypi/app@1.0": {
			"pkg:pypi/requests@2.31",
		},
	}

	addDependencies(
		&item,
		direct,
	)

	if len(item.DependsOn) != 1 {
		t.Fatal("dependsOn not enriched")
	}
}

// addTransitiveParents-------------------------------------------------

func TestAddTransitiveParents(t *testing.T) {

	item := CategorizedComponent{
		Purl:"pkg:pypi/requests@2.31",
	}

	reverse := map[string][]string{
		"pypi/requests@2.31":{
			"pkg:pypi/app@1.0",
		},
	}

	addTransitiveParents(
		&item,
		reverse,
	)

	if len(item.TransitiveOf) != 1 {
		t.Fatal("transitiveOf missing")
	}
}

// enrichComponents

func TestEnrichComponents(t *testing.T) {

	items := []CategorizedComponent{
		{
			Purl:"pkg:pypi/app@1.0",
		},
	}

	direct := map[string][]string{
		"pypi/app@1.0":{
			"pkg:pypi/requests@2.31",
		},
	}

	reverse := map[string][]string{}

	out := enrichComponents(
		items,
		direct,
		reverse,
	)

	if len(out[0].DependsOn) != 1 {
		t.Fatal("enrichment failed")
	}
}

// full integration enrich
func TestEnrichDependencies(t *testing.T) {

	out := ComparisonOutput{
		LibraryComponents: []CategorizedComponent{
			{
				Component:"app",
				Purl:"pkg:pypi/app@1.0",
			},
			{
				Component:"requests",
				Purl:"pkg:pypi/requests@2.31",
			},
		},
	}

	deps := []Dependency{
		{
			Ref:"pkg:pypi/app@1.0",
			DependsOn: []string{
				"pkg:pypi/requests@2.31",
			},
		},
	}

	enriched := EnrichDependencies(
		out,
		deps,
	)

	if len(
		enriched.LibraryComponents[0].DependsOn,
	) != 1 {
		t.Fatal("dependsOn missing")
	}

	if len(
		enriched.LibraryComponents[1].TransitiveOf,
	) != 1 {
		t.Fatal("transitiveOf missing")
	}
}