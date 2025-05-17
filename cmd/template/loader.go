package template

import (
	"embed"
	"gopkg.in/yaml.v3"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed embedded/*
var embedded embed.FS

// loadConfigs takes an fs.FS instance, reads all top-level *.yml or *.yaml
// files in its root, unmarshals them into Definition, and returns
// []*Definition.
func loadConfigs(rootFS fs.FS) []*Definition {
	var cmdDefinitions []*Definition

	entries, err := fs.ReadDir(rootFS, ".")
	if err != nil {
		// Handle or log error as needed
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Check file extension
		if ext := strings.ToLower(filepath.Ext(entry.Name())); ext == ".vyb" {
			data, err := fs.ReadFile(rootFS, entry.Name())
			if err != nil {
				// Handle or log error as needed
				continue
			}

			var cmdDef Definition
			if err := yaml.Unmarshal(data, &cmdDef); err != nil {
				// Handle or log error as needed
				continue
			}

			cmdDefinitions = append(cmdDefinitions, &cmdDef)
		}
	}

	return cmdDefinitions
}

// loadEmbeddedConfigs reads configuration files from the embedded directory.
func loadEmbeddedConfigs() []*Definition {
	subFS, err := fs.Sub(embedded, "embedded")
	if err != nil {
		// Handle or log error as needed
		return nil
	}
	return loadConfigs(subFS)
}

// loadGlobalConfigs reads configuration files from the directory specified
// by the VYB_HOME environment variable, if set.
func loadGlobalConfigs() []*Definition {
	vybHome := os.Getenv("VYB_HOME")
	if vybHome == "" {
		return nil
	}
	cmdPath := filepath.Join(vybHome, "cmd")
	if _, err := os.Stat(cmdPath); err != nil {
		return nil
	}
	return loadConfigs(os.DirFS(cmdPath))
}

// toMap converts a slice of *Definition into a map where the key is the Name field
// and the value is the corresponding Definition struct.
func toMap(cmdDefinitions []*Definition) map[string]*Definition {
	result := make(map[string]*Definition)
	for _, cmdDef := range cmdDefinitions {
		if cmdDef != nil && cmdDef.Name != "" {
			result[cmdDef.Name] = cmdDef
		}
	}
	return result
}

// load combines the results of loadEmbeddedConfigs, loadGlobalConfigs,
// and loadLocalConfigs in order of precedence: embedded < global < local.
func load() []*Definition {
	// Combine results using precedence
	combinedMap := toMap(loadEmbeddedConfigs())

	// Override with global configs
	for name, cmdDef := range toMap(loadGlobalConfigs()) {
		combinedMap[name] = cmdDef
	}

	// Convert the final map back to a slice
	finalConfigs := make([]*Definition, 0, len(combinedMap))
	for _, cmdDef := range combinedMap {
		finalConfigs = append(finalConfigs, cmdDef)
	}

	return finalConfigs
}
