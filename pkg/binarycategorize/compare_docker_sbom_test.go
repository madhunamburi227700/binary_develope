package binarycategorize

import "testing"

// normalize

func TestNormalize(t *testing.T) {

	got := normalize(
		"github.com/docker/cli@v1.2.0",
	)

	want := "docker/cli"

	if got != want {
		t.Fatalf(
			"got %s want %s",
			got,
			want,
		)
	}
}

// exact strong match

func TestStrongMatchExact(t *testing.T) {

	ok, score := strongMatch(
		"requests",
		"requests",
		"",
	)

	if !ok {
		t.Fatal("should match")
	}

	if score != 100 {
		t.Fatal("expected score 100")
	}
}

// PURL extraction match

func TestStrongMatchPurl(t *testing.T) {

	ok, _ := strongMatch(
		"something",
		"scoutsuite",
		"pkg:pypi/scoutsuite@5.0",
	)

	if !ok {
		t.Fatal("purl match failed")
	}
}

// Prevent prefix false positives

func TestNoHttpHttpsFalsePositive(t *testing.T) {

	ok, _ := strongMatch(
		"http",
		"https",
		"",
	)

	if ok {
		t.Fatal("http should not match https")
	}
}

// Prevent embedded false positives

func TestNoSecurityAquasecurityFalsePositive(t *testing.T) {

	ok, _ := strongMatch(
		"security",
		"aquasecurity",
		"",
	)

	if ok {
		t.Fatal("security should not match aquasecurity")
	}
}

// Match OS

func TestMatchOS(t *testing.T) {

	sb := CategorizedComponent{
		Component: "curl",
		Purl: "pkg:deb/debian/curl@7",
	}

	docker := []DockerComponent{
		{
			ComponentName: "curl",
			LineNumber: 8,
			Raw: "RUN apt-get install curl",
		},
	}

	ok, line, _, _ := matchOS(
		sb,
		docker,
	)

	if !ok {
		t.Fatal("os match failed")
	}

	if line != 8 {
		t.Fatal("wrong line number")
	}
}

// Match library

func TestMatchLibrary(t *testing.T) {

	sb := CategorizedComponent{
		Component: "requests",
		Purl: "pkg:pypi/requests@2",
	}

	libs := []DockerComponent{
		{
			ComponentName:"requests",
			LineNumber:12,
			Raw:"RUN pip install requests",
		},
	}

	ok, _, _, _ := matchLibrary(
		sb,
		libs,
		nil,
	)

	if !ok {
		t.Fatal("library match failed")
	}
}

// binary install detection

func TestBinaryInstallDetection(t *testing.T) {

	raw := `curl -sSf https://github.com/anchore/grype/releases/download/latest/grype.tar.gz | tar xz`

	if !isBinaryInstall(raw) {
		t.Fatal("should detect binary install")
	}
}

// Compare promotes library -> binary

func TestComparePromotesToBinary(t *testing.T) {

	sbom := CategorizedSBOM{
		LibraryComponents: []CategorizedComponent{
			{
				Component:"grype",
				Purl:"pkg:golang/grype@1",
				Category:"Library",
			},
		},
	}

	docker := DockerfileInput{
		Binary: []DockerComponent{
			{
				ComponentName:"grype",
				LineNumber:20,
				Raw:"curl https://github.com/anchore/grype/releases/download/x | tar xz",
			},
		},
	}

	out := CompareComponents(
		sbom,
		docker,
	)

	if len(out.BinaryComponents) != 1 {
		t.Fatal("should promote to binary")
	}
}

// Full integration compare

func TestCompareComponentsIntegration(t *testing.T) {

	sbom := CategorizedSBOM{
		OSComponents: []CategorizedComponent{
			{
				Component:"curl",
				Purl:"pkg:deb/debian/curl@7",
				Category:"OS",
			},
		},
		LibraryComponents: []CategorizedComponent{
			{
				Component:"requests",
				Purl:"pkg:pypi/requests@2",
				Category:"Library",
			},
		},
	}

	docker := DockerfileInput{
		OS: []DockerComponent{
			{
				ComponentName:"curl",
				LineNumber:5,
				Raw:"RUN apt-get install curl",
			},
		},
		Library: []DockerComponent{
			{
				ComponentName:"requests",
				LineNumber:9,
				Raw:"RUN pip install requests",
			},
		},
	}

	out := CompareComponents(
		sbom,
		docker,
	)

	if out.OSComponents[0].InDockerfile != "yes" {
		t.Fatal("OS should match")
	}

	if out.LibraryComponents[0].InDockerfile != "yes" {
		t.Fatal("library should match")
	}
}