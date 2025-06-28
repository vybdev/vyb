package project

import (
	"fmt"
	"github.com/vybdev/vyb/config"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// collectModuleMap traverses a module tree and records every module by
// its Name into dst.
func collectModuleMap(mod *Module, dst map[string]*Module) {
	if mod == nil {
		return
	}
	dst[mod.Name] = mod
	for _, child := range mod.Modules {
		collectModuleMap(child, dst)
	}
}

// mergeAnnotations walks the freshly generated module tree (fresh) and,
// using oldMap, copies annotations from the previous metadata when the
// module name exists and its MD5 hash is unchanged.
func mergeAnnotations(fresh *Module, oldMap map[string]*Module) {
	if fresh == nil {
		return
	}

	if old, ok := oldMap[fresh.Name]; ok {
		if old.MD5 == fresh.MD5 && old.Annotation != nil {
			fresh.Annotation = old.Annotation
		}
	}
	for _, child := range fresh.Modules {
		mergeAnnotations(child, oldMap)
	}
}

// Update refreshes the .vyb/metadata.yaml content to reflect the current
// workspace state while preserving valid annotations.
//
// Algorithm:
//  1. Load the stored metadata (with annotations).
//  2. Produce a fresh metadata snapshot from the file system.
//  3. Patch the stored metadata with the fresh snapshot.
//  4. Run annotate so missing/invalid annotations are regenerated.
//  5. Persist the updated metadata back to disk.
func Update(projectRoot string) error {
	// Ensure we have an absolute project root path.
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to determine absolute project root: %w", err)
	}

	rootFS := os.DirFS(absRoot)

	// load existing metadata (with annotations).
	stored, err := loadStoredMetadata(rootFS)
	if err != nil {
		return err
	}

	// build a fresh snapshot.
	fresh, err := buildMetadata(rootFS)
	if err != nil {
		return err
	}

	// patch stored metadata with the fresh structure.
	stored.Patch(fresh)

	cfg, err := config.Load(absRoot)
	if err != nil {
		return err
	}
	// (re)annotate modules missing or with invalid annotations.
	if err := annotate(cfg, stored, rootFS); err != nil {
		return err
	}

	// persist back to .vyb/metadata.yaml.
	data, err := yaml.Marshal(stored)
	if err != nil {
		return fmt.Errorf("failed to marshal updated metadata: %w", err)
	}

	metaFilePath := filepath.Join(absRoot, ".vyb", "metadata.yaml")
	if err := os.WriteFile(metaFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write updated metadata.yaml: %w", err)
	}

	return nil
}
