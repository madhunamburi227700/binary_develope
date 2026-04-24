package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/opsmx/ai-guardian-api/pkg/binarycategorize"
)

func main() {

	// Load SBOM
	sbomData, err := os.ReadFile("sbom.json")
	if err != nil {
		panic(err)
	}

	var sbom binarycategorize.SBOMInput

	err = json.Unmarshal(sbomData, &sbom)
	if err != nil {
		panic(err)
	}

	// Load Dockerfile
	dockerData, err := os.ReadFile("Dockerfile")
	if err != nil {
		panic(err)
	}

	dockerfile := string(dockerData)

	// Run package flow
	result := binarycategorize.FilterComponents(
		sbom,
		dockerfile,
	)

	// Write JSON same as your standalone scripts
	var buf bytes.Buffer

	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false) // keeps & instead of \u0026

	err = enc.Encode(result)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(
		"output.json",
		buf.Bytes(),
		0644,
	)
	if err != nil {
		panic(err)
	}

	fmt.Println("✅ Output saved to output.json")
}