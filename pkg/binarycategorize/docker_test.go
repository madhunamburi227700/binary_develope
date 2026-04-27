package binarycategorize

import "testing"

// normalizeLines
func TestNormalizeLinesMultiline(t *testing.T) {

	raw := []string{
		"RUN apt-get install curl \\",
		"vim",
	}

	lines := normalizeLines(raw)

	t.Log(lines)

	if len(lines) != 1 {
		t.Fatalf("expected 1 merged line got %d", len(lines))
	}
}

// Parse FROM base image
func TestParseBaseImage(t *testing.T) {

	dockerfile := `
FROM debian:bookworm-slim AS final
`

	out := ParseDockerfile(dockerfile)

	if len(out.OS) != 1 {
		t.Fatal("expected 1 OS component")
	}

	if out.OS[0].Type != "base-image" {
		t.Fatal("expected base-image type")
	}
}

// APT install classification

func TestParseAptInstall(t *testing.T) {

	dockerfile := `
FROM debian:bookworm
RUN apt-get install curl vim
`

	out := ParseDockerfile(dockerfile)

	if len(out.OS) != 3 {
		t.Fatalf(
			"expected base image + 2 packages got %d",
			len(out.OS),
		)
	}
}

// pip install classification
func TestParsePipInstall(t *testing.T) {

	dockerfile := `
FROM python:3.11
RUN pip install requests flask
`

	out := ParseDockerfile(dockerfile)

	if len(out.Library) != 2 {
		t.Fatalf(
			"expected 2 libraries got %d",
			len(out.Library),
		)
	}
}

// npm install classification
func TestParseNpmInstall(t *testing.T) {

	dockerfile := `
FROM node:20
RUN npm install react express
`

	out := ParseDockerfile(dockerfile)

	if len(out.Library) != 2 {
		t.Fatalf(
			"expected 2 npm libraries got %d",
			len(out.Library),
		)
	}
}

// Binary download detection
func TestBinaryURLDetection(t *testing.T) {

	dockerfile := `
FROM alpine
RUN curl -sSf https://github.com/anchore/grype/releases/latest/download/grype.tar.gz | tar xz
`

	out := ParseDockerfile(dockerfile)

	if len(out.Binary) != 1 {
		t.Fatalf(
			"expected 1 binary got %d",
			len(out.Binary),
		)
	}
}

// COPY classification
func TestCopyClassification(t *testing.T) {

	dockerfile := `
FROM alpine
COPY app.py /opt/app.py
`

	out := ParseDockerfile(dockerfile)

	if len(out.Application) != 1 {
		t.Fatal("expected 1 application copy")
	}
}

// Skip apt update commands
func TestSkipCommands(t *testing.T) {

	dockerfile := `
FROM debian
RUN apt-get update
`

	out := ParseDockerfile(dockerfile)

	// only base image should exist
	if len(out.OS) != 1 {
		t.Fatal("apt-get update should be skipped")
	}
}

// extractAptPackages helper
func TestExtractAptPackages(t *testing.T) {

	pkgs := extractAptPackages(
		"apt-get install -y curl vim",
	)

	if len(pkgs) != 2 {
		t.Fatalf(
			"expected 2 packages got %d",
			len(pkgs),
		)
	}
}

// extractPipPackages helper
func TestExtractPipPackages(t *testing.T) {

	pkgs := extractPipPackages(
		"pip install requests flask",
	)

	if len(pkgs) != 2 {
		t.Fatalf(
			"expected 2 packages got %d",
			len(pkgs),
		)
	}
}

// Full mixed integration test
func TestParseDockerfileIntegration(t *testing.T) {

	dockerfile := `
FROM debian:bookworm-slim

RUN apt-get install curl vim

RUN pip install requests

RUN curl -sSf https://github.com/anchore/grype/releases/latest/download/grype.tar.gz | tar xz

COPY app.py /app.py
`

	out := ParseDockerfile(dockerfile)

	if len(out.OS) != 3 {
		t.Fatalf("expected 3 os entries got %d", len(out.OS))
	}

	if len(out.Library) != 1 {
		t.Fatalf("expected 1 library got %d", len(out.Library))
	}

	if len(out.Binary) != 1 {
		t.Fatalf("expected 1 binary got %d", len(out.Binary))
	}

	if len(out.Application) != 1 {
		t.Fatalf("expected 1 application got %d", len(out.Application))
	}
}