package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func main() {
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".yaml") {
			if strings.Contains(path, "api/gen/") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("error reading %s: %v", path, err)
			}
			var m interface{}
			decoder := yaml.NewDecoder(strings.NewReader(string(data)))
			decoder.KnownFields(true) // This makes it strict for unknown fields, but for duplicates, we need to handle differently
			if err := decoder.Decode(&m); err != nil {
				return fmt.Errorf("YAML syntax error in %s: %v", path, err)
			}
			fmt.Printf("OK: %s\n", filepath.Base(path))
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
