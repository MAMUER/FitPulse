package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// isSafeYAMLPath validates that a path is safe for reading YAML files
func isSafeYAMLPath(path string) bool {
	// Must not contain path traversal
	if strings.Contains(path, "..") {
		return false
	}

	// Must not be absolute path
	if filepath.IsAbs(path) {
		return false
	}

	// Must be a YAML file
	if !strings.HasSuffix(path, ".yml") && !strings.HasSuffix(path, ".yaml") {
		return false
	}

	// Path should be reasonable length
	if len(path) > 500 {
		return false
	}

	// Check for invalid characters
	invalidChars := "<>\"|?*"
	for _, char := range invalidChars {
		if strings.ContainsRune(path, char) {
			return false
		}
	}

	return true
}

// readYAMLFile safely reads a YAML file with additional validation
func readYAMLFile(path string) ([]byte, error) {
	// Double-check path safety
	if !isSafeYAMLPath(path) {
		return nil, fmt.Errorf("unsafe path rejected: %s", path)
	}

	// Open file safely. Path is validated before this call.
	file, err := os.Open(path) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("error opening file %s: %w", path, err)
	}
	defer func() {
		_ = file.Close() // Ignore close error in defer
	}()

	// Check file size using Seek
	const maxYAMLSize = 10 * 1024 * 1024 // 10MB
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("error getting file info %s: %w", path, err)
	}
	if stat.Size() > maxYAMLSize {
		return nil, fmt.Errorf("file too large: %s (%d bytes)", path, stat.Size())
	}

	// Read file content using bufio.Scanner
	var content strings.Builder
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		content.WriteString(scanner.Text())
		content.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file %s: %w", path, err)
	}

	return []byte(content.String()), nil
}

func main() {
	// Collect all YAML files first
	var yamlFiles []string
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".yaml") {
			if !strings.Contains(path, "api/gen/") && isSafeYAMLPath(path) {
				yamlFiles = append(yamlFiles, path)
			}
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Validate each file
	for _, path := range yamlFiles {
		data, err := readYAMLFile(path)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		var m interface{}
		decoder := yaml.NewDecoder(strings.NewReader(string(data)))
		decoder.KnownFields(true)
		if err := decoder.Decode(&m); err != nil {
			fmt.Printf("Error: YAML syntax error in %s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Printf("OK: %s\n", filepath.Base(path))
	}
}
