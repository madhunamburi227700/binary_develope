package binarycategorize

import (
	"regexp"
	"strings"
)


func ParseDockerfile(content string) DockerfileInput {
	rawLines := strings.Split(content, "\n")
	lines := normalizeLines(rawLines)
	components := parseLines(lines)
	return groupComponents(components)
}


type line struct {
	content string
	number  int
}

var fromRegex = regexp.MustCompile(`(?i)^FROM\s+([^\s]+)(?:\s+AS\s+([a-zA-Z0-9_-]+))?`)

func normalizeLines(raw []string) []line {

	var result []line
	var current string
	var startLine int

	for i, l := range raw {

		num := i + 1
		l = strings.TrimSpace(l)

		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}

		if current == "" {
			startLine = num
		}

		if strings.HasSuffix(l, "\\") {
			current += strings.TrimSuffix(l, "\\") + " "
		} else {
			current += l
			result = append(result, line{
				content: current,
				number:  startLine,
			})
			current = ""
		}
	}

	return result
}


func parseLines(lines []line) []DockerComponent {

	lastFrom := -1

	for i, l := range lines {
		if fromRegex.MatchString(l.content) {
			lastFrom = i
		}
	}

	var out []DockerComponent

	for i, l := range lines {

		if i < lastFrom {
			continue
		}

		if m := fromRegex.FindStringSubmatch(l.content); len(m) > 0 {

			out = append(out, DockerComponent{
				Source:        "os",
				Type:          "base-image",
				LineNumber:    l.number,
				ComponentName: m[1],
				Raw:           l.content,
			})

			continue
		}

		if strings.HasPrefix(l.content, "RUN") {

			cmd := strings.TrimPrefix(l.content, "RUN")

			parts := strings.FieldsFunc(cmd, func(r rune) bool {
				return r == '&' || r == ';'
			})

			for _, p := range parts {
				out = append(out, classifyCmd(strings.TrimSpace(p), l.number)...)
			}
		}

		if strings.HasPrefix(l.content, "COPY") {
			out = append(out, classifyCopy(l.content, l.number)...)
		}
	}

	return out
}

// CLASSIFICATION
var binaryTools = map[string]bool{
	"syft": true,
	"trivy": true,
	"grype": true,
	"scorecard": true,
	"snyk": true,
	"tfsec": true,
	"opengrep": true,
	"codacy": true,
	"helm": true,
	"kubescape": true,
	"aws": true,
}

var skipCmds = []string{
	"apt-get update",
	"apt-get upgrade",
	"apt-get purge",
	"rm ",
	"mkdir ",
	"chmod ",
	"echo ",
	"set ",
}

func classifyCmd(cmd string, lineNum int) []DockerComponent {

	l := strings.ToLower(strings.TrimSpace(cmd))

	for _, s := range skipCmds {
		if strings.HasPrefix(l, s) {
			return nil
		}
	}

	var result []DockerComponent

	// ---------------- OS ----------------

	if strings.Contains(l, "apt-get install") ||
		strings.Contains(l, "apt install") {

		for _, p := range extractAptPackages(cmd) {
			result = append(result, DockerComponent{
				Source: "os",
				Type: "install",
				LineNumber: lineNum,
				ComponentName: p,
				Raw: cmd,
			})
		}
	}

	// ---------------- PIP ----------------

	if strings.Contains(l, "pip install") ||
		strings.Contains(l, "pip3 install") ||
		strings.Contains(l, "pipx install") {

		for _, p := range extractPipPackages(cmd) {

			result = append(result, DockerComponent{
				Source: "library",
				Type: "install",
				LineNumber: lineNum,
				ComponentName: p,
				Raw: cmd,
			})
		}
	}

	// ---------------- NPM ----------------

	if strings.Contains(l, "npm install") {

		skipNpm := map[string]bool{
			"npm": true,
			"install": true,
			"-g": true,
			"--unsafe-perm=false": true,
		}

		for _, p := range strings.Fields(cmd) {

			p = strings.Trim(p, "'\"")

			if skipNpm[p] || strings.HasPrefix(p, "-") {
				continue
			}

			result = append(result, DockerComponent{
				Source: "library",
				Type: "install",
				LineNumber: lineNum,
				ComponentName: p,
				Raw: cmd,
			})
		}
	}

	// ---------------- URLS ----------------

	for _, u := range extractAllURLs(cmd) {

		src := "binary"
		ctype := "download"

		if isBinaryTool(u) {

			src = "binary"

			if strings.Contains(l, "| tar") {
				ctype = "install"
			}

		} else if isLibraryInstall(u) {

			src = "library"
			ctype = "install"

		} else {

			if strings.Contains(l, "| tar") {
				ctype = "install"

			} else if strings.Contains(l, "| sh") ||
				strings.Contains(l, "| bash") {

				src = "binary"
				ctype = "install"
			}
		}

		result = append(result, DockerComponent{
			Source: src,
			Type: ctype,
			LineNumber: lineNum,
			ComponentName: u,
			Raw: cmd,
		})
	}

	return result
}

// COPY

func classifyCopy(cmd string, lineNum int) []DockerComponent {

	parts := strings.Fields(cmd)

	var clean []string

	for _, p := range parts[1:] {
		if !strings.HasPrefix(p, "--") {
			clean = append(clean, p)
		}
	}

	if len(clean) < 2 {
		return nil
	}

	src := clean[0]
	dst := clean[len(clean)-1]

	name := src

	if src == "." {
		name = "current-dir"
	} else if dst != "." {
		name = src + " -> " + dst
	}

	return []DockerComponent{{
		Source: "application",
		Type: "copy",
		LineNumber: lineNum,
		ComponentName: name,
		Raw: cmd,
	}}
}

// HELPERS

func extractAllURLs(cmd string) []string {
	re := regexp.MustCompile(`https?://[^\s"'\\)]+`)
	return re.FindAllString(cmd, -1)
}

func isBinaryTool(url string) bool {

	u := strings.ToLower(url)

	for tool := range binaryTools {
		if strings.Contains(u, tool) {
			return true
		}
	}

	return false
}

func isLibraryInstall(url string) bool {

	u := strings.ToLower(url)

	patterns := []string{
		"anchore/syft",
		"nodejs",
		"nodesource",
	}

	for _, p := range patterns {
		if strings.Contains(u, p) {
			return true
		}
	}

	return false
}

func extractAptPackages(cmd string) []string {

	idx := strings.Index(cmd, "install")
	if idx == -1 {
		return nil
	}

	var pkgs []string

	for _, p := range strings.Fields(cmd[idx+len("install"):]) {

		if strings.HasPrefix(p, "-") {
			continue
		}

		pkgs = append(pkgs, strings.Trim(p, "'\""))
	}

	return pkgs
}

func extractPipPackages(cmd string) []string {

	l := strings.ToLower(cmd)

	idx := strings.Index(l, "install")
	if idx == -1 {
		return nil
	}

	skip := map[string]bool{
		"pip": true,
		"pip3": true,
		"pipx": true,
		"-r": true,
		"--upgrade": true,
		"--no-cache-dir": true,
		"-g": true,
		"--break-system-packages": true,
		"--unsafe-perm=false": true,
		"-b": true,
		"-s": true,
		"--": true,
		"-q": true,
		"--quiet": true,
	}

	var pkgs []string

	for _, p := range strings.Fields(cmd[idx+len("install"):]) {

		p = strings.Trim(p, "'\"")

		if p == "" ||
			skip[p] ||
			strings.HasPrefix(p, "-") ||
			strings.Contains(p, "/") {
			continue
		}

		pkgs = append(pkgs, p)
	}

	return pkgs
}

// GROUP

func groupComponents(components []DockerComponent) DockerfileInput {

	var out DockerfileInput

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