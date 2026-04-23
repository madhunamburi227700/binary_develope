package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ---------------- STRUCT ----------------

type Component struct {
	Source        string `json:"source"`
	Type          string `json:"type"`
	LineNumber    int    `json:"line_number"`
	ComponentName string `json:"component_name"`
	Raw           string `json:"raw"`
}

type Output struct {
	OS          []Component `json:"os"`
	Binary      []Component `json:"binary"`
	Library     []Component `json:"library"`
	Application []Component `json:"application"`
}

type Line struct {
	Content string
	Number  int
}

var fromRegex = regexp.MustCompile(`^FROM\s+([^\s]+)(?:\s+AS\s+([a-zA-Z0-9_-]+))?`)

// Known binary tools that are installed via shell scripts
var binaryToolPatterns = map[string]bool{
	"syft":      true,
	"trivy":     true,
	"grype":     true,
	"scorecard": true,
	"snyk":      true,
	"tfsec":     true,
	"opengrep":  true,
	"codacy":    true,
	"helm":      true,
	"kubescape": true,
	"aws":       true,
}

// Known pip/npm packages
var libraryPackagePatterns = map[string]bool{
	"ruamel.yaml":      true,
	"azure-identity":   true,
	"semgrep":          true,
	"pipx":             true,
	"cdxgen":           true,
	"scoutsuite":       true,
}

// ---------------- NORMALIZE ----------------

func normalizeLines(lines []string) []Line {
	var result []Line
	var current string
	var startLine int

	for i, line := range lines {
		lineNum := i + 1
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if current == "" {
			startLine = lineNum
		}

		if strings.HasSuffix(line, "\\") {
			current += strings.TrimSuffix(line, "\\") + " "
		} else {
			current += line
			result = append(result, Line{
				Content: current,
				Number:  startLine,
			})
			current = ""
		}
	}

	return result
}

// ---------------- LAST FROM DETECTION ----------------

func getLastFromIndex(lines []Line) int {
	lastIndex := -1

	for i, l := range lines {
		if fromRegex.MatchString(l.Content) {
			lastIndex = i
		}
	}

	return lastIndex
}

// ---------------- PACKAGE EXTRACTION ----------------

func extractAptPackages(cmd string) []string {
	var pkgs []string

	idx := strings.Index(cmd, "install")
	if idx == -1 {
		return pkgs
	}

	parts := strings.Fields(cmd[idx+len("install"):])

	for _, p := range parts {
		if strings.HasPrefix(p, "-") {
			continue
		}

		p = strings.Trim(p, "'\"")
		pkgs = append(pkgs, p)
	}

	return pkgs
}

// Extract pip/pipx packages properly
func extractPipPackages(cmd string) []string {
	var pkgs []string
	
	l := strings.ToLower(cmd)
	var installIdx int
	
	if strings.Contains(l, "pip install") {
		installIdx = strings.Index(l, "install") + len("install")
	} else if strings.Contains(l, "pipx install") {
		installIdx = strings.Index(l, "install") + len("install")
	} else if strings.Contains(l, "pip3 install") {
		installIdx = strings.Index(l, "install") + len("install")
	} else {
		return pkgs
	}

	parts := strings.Fields(cmd[installIdx:])

	skipWords := map[string]bool{
		"pip":                      true,
		"pip3":                     true,
		"pipx":                     true,
		"-r":                       true,
		"--upgrade":                true,
		"--no-cache-dir":           true,
		"-g":                       true,
		"--break-system-packages":  true,
		"--unsafe-perm=false":      true,
		"-b":                       true,
		"-s":                       true,
		"--":                       true,
		"-q":                       true,
		"--quiet":                  true,
	}

	for _, p := range parts {
		p = strings.Trim(p, "'\"")
		p = strings.TrimSpace(p)
		
		// Skip flags, empty strings, and paths
		if p == "" || skipWords[p] || strings.HasPrefix(p, "-") || strings.Contains(p, "/") {
			continue
		}

		pkgs = append(pkgs, p)
	}

	return pkgs
}

func extractAllURLs(cmd string) []string {
	re := regexp.MustCompile(`https?://[^\s"'\\)]+`)
	return re.FindAllString(cmd, -1)
}

// Identify if URL/tool is a known binary tool
func isBinaryTool(url string, cmd string) bool {
	l := strings.ToLower(url)
	
	for tool := range binaryToolPatterns {
		if strings.Contains(l, tool) {
			return true
		}
	}
	
	return false
}

// Identify if it's a library installation script
func isLibraryInstall(url string, cmd string) bool {
	l := strings.ToLower(url)
	
	// Known library installation scripts
	libraryInstallPatterns := []string{
		"anchore/syft",
		"nodejs",
		"nodesource",
	}
	
	for _, pattern := range libraryInstallPatterns {
		if strings.Contains(l, pattern) {
			return true
		}
	}
	
	return false
}

// ---------------- CLASSIFY ----------------

func classify(cmd string, lineNum int) []Component {

	l := strings.ToLower(strings.TrimSpace(cmd))

	// ✅ FIXED: Be more specific with filters - don't filter commands that contain packages
	// Only filter apt-get cleanup commands and basic system commands
	if strings.HasPrefix(l, "apt-get update") ||
		strings.HasPrefix(l, "apt-get upgrade") ||
		strings.HasPrefix(l, "apt-get purge") ||
		strings.HasPrefix(l, "rm ") ||
		strings.HasPrefix(l, "mkdir ") ||
		strings.HasPrefix(l, "chmod ") ||
		strings.HasPrefix(l, "echo ") ||
		strings.HasPrefix(l, "set ") {
		return nil
	}

	var result []Component

	// ---------- OS ----------
	if strings.Contains(l, "apt-get install") || strings.Contains(l, "apt install") {
		for _, p := range extractAptPackages(cmd) {
			result = append(result, Component{
				Source:        "os",
				Type:          "install",
				LineNumber:    lineNum,
				ComponentName: p,
				Raw:           cmd,
			})
		}
	}

	// ---------- LIBRARIES (pip/pipx) ----------
	if strings.Contains(l, "pip install") ||
		strings.Contains(l, "pip3 install") ||
		strings.Contains(l, "pipx install") {

		for _, p := range extractPipPackages(cmd) {
			result = append(result, Component{
				Source:        "library",
				Type:          "install",
				LineNumber:    lineNum,
				ComponentName: p,
				Raw:           cmd,
			})
		}
	}

	// ---------- NPM PACKAGES ----------
	if strings.Contains(l, "npm install") {
		fields := strings.Fields(cmd)

		skipWords := map[string]bool{
			"npm":                 true,
			"install":             true,
			"-g":                  true,
			"--unsafe-perm=false": true,
		}

		for _, p := range fields {
			p = strings.Trim(p, "'\"")

			if skipWords[p] || strings.HasPrefix(p, "-") {
				continue
			}

			result = append(result, Component{
				Source:        "library",
				Type:          "install",
				LineNumber:    lineNum,
				ComponentName: p,
				Raw:           cmd,
			})
		}
	}

	// ---------- URLS / BINARIES ----------
	urls := extractAllURLs(cmd)

	for _, u := range urls {
		source := "binary"
		ctype := "download"

		// Better classification based on URL and context
		if isBinaryTool(u, cmd) {
			// This is a known binary tool
			source = "binary"
			ctype = "download"
			
			if strings.Contains(l, "| tar") {
				ctype = "install"
			}
		} else if isLibraryInstall(u, cmd) {
			// This is a library installation script
			source = "library"
			ctype = "install"
		} else {
			// Default classification for unknown URLs
			if strings.Contains(l, "| tar") {
				ctype = "install"
			} else if strings.Contains(l, "| sh") || strings.Contains(l, "| bash") {
				// Could be binary or library - check if it's a known pattern
				source = "binary"
				ctype = "install"
			}
		}

		result = append(result, Component{
			Source:        source,
			Type:          ctype,
			LineNumber:    lineNum,
			ComponentName: u,
			Raw:           cmd,
		})
	}

	// ---------- COPY ----------
	if strings.HasPrefix(l, "copy") {

		parts := strings.Fields(cmd)

		var clean []string

		for _, p := range parts[1:] {
			if strings.HasPrefix(p, "--") {
				continue
			}
			clean = append(clean, p)
		}

		if len(clean) >= 2 {

			src := clean[0]
			dst := clean[len(clean)-1]

			name := src

			if src == "." {
				name = "current-dir"
			} else if dst == "." {
				name = src
			} else {
				name = src + " -> " + dst
			}

			result = append(result, Component{
				Source:        "application",
				Type:          "copy",
				LineNumber:    lineNum,
				ComponentName: name,
				Raw:           cmd,
			})
		}
	}

	return result
}

// ---------------- PARSE ----------------

func parseDockerfile(lines []Line) []Component {

	lastFromIndex := getLastFromIndex(lines)

	var output []Component

	for i, line := range lines {

		// ✅ Only process last stage
		if i < lastFromIndex {
			continue
		}

		if m := fromRegex.FindStringSubmatch(line.Content); len(m) > 0 {

			image := m[1]

			output = append(output, Component{
				Source:        "os",
				Type:          "base-image",
				LineNumber:    line.Number,
				ComponentName: image,
				Raw:           line.Content,
			})

			continue
		}

		if strings.HasPrefix(line.Content, "RUN") {

			cmd := strings.TrimPrefix(line.Content, "RUN")

			parts := strings.FieldsFunc(cmd, func(r rune) bool {
				return r == '&' || r == ';'
			})

			for _, p := range parts {
				p = strings.TrimSpace(p)
				output = append(output, classify(p, line.Number)...)
			}
		}

		if strings.HasPrefix(line.Content, "COPY") {
			output = append(output, classify(line.Content, line.Number)...)
		}
	}

	return output
}

// ---------------- GROUP ----------------

func groupComponents(components []Component) Output {

	var out Output

	for _, c := range components {

		switch c.Source {

		case "os":
			out.OS = append(out.OS, c)

		case "binary":
			out.Binary = append(out.Binary, c)

		case "library":
			out.Library = append(out.Library, c)

		case "application":
			out.Application = append(out.Application, c)
		}
	}

	return out
}

// ---------------- MAIN ----------------

func main() {

	file, err := os.Open("Dockerfile")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var rawLines []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		rawLines = append(rawLines, scanner.Text())
	}

	lines := normalizeLines(rawLines)

	components := parseDockerfile(lines)

	grouped := groupComponents(components)

	var buf bytes.Buffer

	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)

	err = enc.Encode(grouped)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile("docker_output.json", buf.Bytes(), 0644)
	if err != nil {
		panic(err)
	}

	fmt.Println("✅ docker_output.json generated successfully")
}