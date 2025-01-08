package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

// findRootPath returns the absolute path of the repository root
// This is retrievable with: go list -m -f {{.Dir}}
func findRootPath() (string, error) {
	// rootPath, _ := exec.Command("go list -m -f {{.Dir}}").Output()
	// This returns the path to the root of the project.
	rootPathBz, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}").Output()
	if err != nil {
		return "", err
	}
	rootPath := strings.Trim(string(rootPathBz), "\n")
	return rootPath, nil
}

func main() {
	// Define the input and output directories
	rootPath, err := findRootPath()
	if err != nil {
		log.Fatalf("Unable to find repo root path: %s", err)
	}
	inputDir := path.Join(rootPath, "x/evm/embeds/artifacts/contracts/")
	outputDir := path.Join(rootPath, "x/evm/embeds/abi/")

	// Ensure the output directory exists
	err = os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Walk through the input directory
	err = filepath.Walk(inputDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Process only...
		if !info.IsDir() &&
			filepath.Ext(path) == ".json" && // .json files that
			!strings.Contains(path, ".dbg") && // are NOT "dbg" files
			!strings.HasPrefix(info.Name(), "Test") { // are NOT for tests
			// Read the JSON file
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %v", path, err)
			}

			// Parse the JSON file
			var parsed map[string]interface{}
			err = json.Unmarshal(data, &parsed)
			if err != nil {
				return fmt.Errorf("failed to parse JSON in file %s: %v", path, err)
			}

			// Extract the "abi" field
			abi, ok := parsed["abi"]
			if !ok {
				fmt.Printf("No 'abi' field found in file %s, skipping...\n", path)
				return nil
			}

			// Marshal the ABI field back to JSON
			abiData, err := json.MarshalIndent(abi, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal 'abi' field from file %s: %v", path, err)
			}

			// Create the output file name
			outputFileName := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name())) + ".json"
			outputPath := filepath.Join(outputDir, outputFileName)

			// Write the ABI JSON to the output directory
			err = os.WriteFile(outputPath, abiData, 0o644)
			if err != nil {
				return fmt.Errorf("failed to write ABI to file %s: %v", outputPath, err)
			}

			fmt.Printf("Processed and saved ABI: %s\n", outputPath)
		}

		return nil
	})
	if err != nil {
		log.Fatalf("Error processing files: %v", err)
	}
}
