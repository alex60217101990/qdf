package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// loadTyped reads one localmachine JSON file into the typed Info struct.
func loadTyped(path string) (*Info, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var info Info
	if err := json.Unmarshal(raw, &info); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return &info, nil
}

// loadMap reads one localmachine JSON file into a dynamic map[string]any.
func loadMap(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return m, nil
}
