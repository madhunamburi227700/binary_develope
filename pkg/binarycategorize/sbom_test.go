package binarycategorize

import "testing"

// isOSPurl Tests
func TestIsOSPurl(t *testing.T) {

	tests := []struct {
		name string
		purl string
		want bool
	}{
		{
			name: "deb package",
			purl: "pkg:deb/debian/adduser@3.134",
			want: true,
		},
		{
			name: "rpm package",
			purl: "pkg:rpm/redhat/openssl@1.1",
			want: true,
		},
		{
			name: "apk package",
			purl: "pkg:apk/alpine/bash@5",
			want: true,
		},
		{
			name: "golang library",
			purl: "pkg:golang/github.com/dgryski/go-rendezvous@v1",
			want: false,
		},
		{
			name: "pypi library",
			purl: "pkg:pypi/requests@2.0",
			want: false,
		},
		{
			name: "empty purl",
			purl: "",
			want: false,
		},
	}

	for _, tc := range tests {

		t.Run(tc.name, func(t *testing.T) {

			got := isOSPurl(tc.purl)

			if got != tc.want {
				t.Errorf(
					"got %v want %v",
					got,
					tc.want,
				)
			}
		})
	}
}

// pickLargestList Tests
func TestPickLargestList(t *testing.T) {

	sbom := SBOMInput{
		Components: []RawComponent{
			{Purl: "a"},
		},
		Artifacts: []RawComponent{
			{Purl: "a"},
			{Purl: "b"},
			{Purl: "c"},
		},
		Packages: []RawComponent{
			{Purl: "a"},
			{Purl: "b"},
		},
	}

	got := pickLargestList(sbom)

	if len(got) != 3 {
		t.Fatalf(
			"expected 3 got %d",
			len(got),
		)
	}
}


func TestPickLargestListEmpty(t *testing.T) {

	got := pickLargestList(SBOMInput{})

	if len(got) != 0 {
		t.Fatalf(
			"expected empty list got %d",
			len(got),
		)
	}
}

// bucketize Tests
func TestBucketizeClassification(t *testing.T) {

	raw := []RawComponent{
		{
			Name:    "adduser",
			Version: "3.134",
			Purl: "pkg:deb/debian/adduser@3.134",
		},
		{
			Name:    "go-rendezvous",
			Version: "v1",
			Purl: "pkg:golang/github.com/dgryski/go-rendezvous@v1",
		},
	}

	out := bucketize(raw)

	if len(out.OSComponents) != 1 {
		t.Fatalf(
			"expected 1 os component got %d",
			len(out.OSComponents),
		)
	}

	if len(out.LibraryComponents) != 1 {
		t.Fatalf(
			"expected 1 library got %d",
			len(out.LibraryComponents),
		)
	}
}


func TestBucketizeDedup(t *testing.T) {

	raw := []RawComponent{
		{
			Name:"adduser",
			Purl:"pkg:deb/debian/adduser@3.134",
		},
		{
			Name:"adduser",
			Purl:"pkg:deb/debian/adduser@3.134",
		},
	}

	out := bucketize(raw)

	if len(out.OSComponents) != 1 {
		t.Fatalf(
			"duplicate removal failed got %d",
			len(out.OSComponents),
		)
	}
}


func TestBucketizeSkipEmptyPurl(t *testing.T) {

	raw := []RawComponent{
		{
			Name:"bad-component",
			Purl:"",
		},
	}

	out := bucketize(raw)

	if len(out.OSComponents) != 0 {
		t.Fatal("should skip empty purl")
	}

	if len(out.LibraryComponents) != 0 {
		t.Fatal("should skip empty purl")
	}
}


func TestBucketizeCategoryField(t *testing.T) {

	raw := []RawComponent{
		{
			Name:"adduser",
			Purl:"pkg:deb/debian/adduser@3.134",
		},
	}

	out := bucketize(raw)

	if out.OSComponents[0].Category != "OS" {
		t.Fatal("category not set to OS")
	}
}

// Full Integration Test
func TestCategorizeSBOM(t *testing.T) {

	sbom := SBOMInput{
		Components: []RawComponent{
			{
				Name:"adduser",
				Version:"3.134",
				Purl:"pkg:deb/debian/adduser@3.134",
			},
			{
				Name:"go-rendezvous",
				Version:"v1",
				Purl:"pkg:golang/github.com/dgryski/go-rendezvous@v1",
			},
		},
	}

	out := CategorizeSBOM(sbom)

	if len(out.OSComponents) != 1 {
		t.Fatal("OS classification failed")
	}

	if len(out.LibraryComponents) != 1 {
		t.Fatal("Library classification failed")
	}
}

// Integration: largest list selected first
func TestCategorizeUsesLargestList(t *testing.T) {

	sbom := SBOMInput{
		Components: []RawComponent{
			{
				Purl:"pkg:deb/debian/a@1",
			},
		},

		Artifacts: []RawComponent{
			{
				Purl:"pkg:deb/debian/b@1",
			},
			{
				Purl:"pkg:golang/lib@1",
			},
		},
	}

	out := CategorizeSBOM(sbom)

	if len(out.OSComponents) != 1 {
		t.Fatal("expected 1 os component")
	}

	if len(out.LibraryComponents) != 1 {
		t.Fatal("expected 1 library component")
	}
}