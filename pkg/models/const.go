package models

const (
	// keep this in sync with middleware.HeaderXUser
	HeaderXUser = "X-User"

	// SAST tool priority: Semgrep is first priority, OpenGrep is second.
	// Used in polling (persist only first-priority tool's data), stats, and scan_type updates.
	ToolSemgrep  = "Semgrep"
	ToolOpengrep = "Opengrep"
)

const (
	ScanTypeSAST    = "sast"
	ScanTypeSCA     = "sca"
	ScanTypeSBOM    = "sbom"
	ScanTypeUnknown = "unknown"
)
