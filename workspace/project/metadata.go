package project

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/vybdev/vyb/config"
	"github.com/vybdev/vyb/workspace/context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/vybdev/vyb/workspace/selector"
)

// Metadata represents the project-specific metadata file. Only one Metadata
// file should exist within a given vyb project, and it should be located in
// the .vyb/ directory under the project root directory.
type Metadata struct {
	Modules *Module `yaml:"modules"`
}

func newModule(name string, parent *Module, modules []*Module, files []*FileRef, annotation *Annotation) *Module {
	return &Module{
		Name:            name,
		Parent:          parent,
		Modules:         modules,
		Files:           files,
		Directories:     deriveDirectoriesFromFiles(files),
		Annotation:      annotation,
		MD5:             computeHashFromChildren(modules, files),
		localTokenCount: computeTokenCountFromChildren(nil, files),
		TokenCount:      computeTokenCountFromChildren(modules, files),
	}
}

// deriveDirectoriesFromFiles gets a list of files and returns a list of unique directories holding those files
func deriveDirectoriesFromFiles(files []*FileRef) []string {
	dirs := make(map[string]struct{})
	for _, f := range files {
		dir := filepath.Dir(f.Name)
		dirs[dir] = struct{}{}
	}

	var result []string
	for dir := range dirs {
		result = append(result, dir)
	}
	sort.Strings(result)
	return result
}

// Module represents a hierarchical grouping of information within a vyb
// project structure.
type Module struct {
	// Name stores the *full* relative path of the module from the workspace
	// root – e.g. "dirA/dirB".  The root module has Name equal to ".".
	Name            string      `yaml:"name"`
	Parent          *Module     `yaml:"-"`
	Modules         []*Module   `yaml:"modules"`
	Files           []*FileRef  `yaml:"files"`
	Directories     []string    `yaml:"-"`
	Annotation      *Annotation `yaml:"annotation,omitempty"`
	TokenCount      int64       `yaml:"token_count"`
	MD5             string      `yaml:"md5"`
	localTokenCount int64       `yaml:"-"`
}

func computeTokenCountFromChildren(modules []*Module, files []*FileRef) int64 {
	var count int64
	for _, m := range modules {
		count += m.TokenCount
	}
	for _, f := range files {
		count += f.TokenCount
	}
	return count
}

func computeHashFromChildren(modules []*Module, files []*FileRef) string {
	var hashes []string
	for _, m := range modules {
		hashes = append(hashes, m.MD5)
	}
	for _, f := range files {
		hashes = append(hashes, f.MD5)
	}
	sort.Strings(hashes)
	return computeHashFromBytes([]byte(strings.Join(hashes, "")))
}

func computeHashFromBytes(bytes []byte) string {
	h := md5.Sum(bytes)
	return hex.EncodeToString(h[:])
}

type FileRef struct {
	// Name holds the full relative path to the file from the workspace root.
	Name         string    `yaml:"name"`
	LastModified time.Time `yaml:"last_modified"`
	TokenCount   int64     `yaml:"token_count"`
	MD5          string    `yaml:"md5"`
}

func newFileRef(name string, lastModified time.Time, tokenCount int64, md5 string) *FileRef {
	return &FileRef{
		Name:         name,
		LastModified: lastModified,
		TokenCount:   tokenCount,
		MD5:          md5,
	}
}

var systemExclusionPatterns = []string{
	".git/",
	".gitignore",
	".vyb/",
	"LICENSE",
	"go.sum",
}

// Create creates the project metadata configuration at the project root.
// The function now also persists .vyb/config.yaml with the chosen LLM
// provider so callers do not have to duplicate that logic.
//
// Returns an error if the metadata cannot be created, or if it already
// exists.  If a ".vyb" folder exists in the root directory or any of its
// subdirectories, this function returns an error.
func Create(projectRoot string, provider string) error {

	if provider == "" {
		provider = config.Default().Provider
	}

	rootFS := os.DirFS(projectRoot)
	existingFolders, err := findAllConfigWithinRoot(rootFS)
	if err != nil {
		return err
	}
	if len(existingFolders) > 0 {
		return fmt.Errorf("failed to create a project configuration because there is already a configuration within the given root: %s", existingFolders[0])
	}

	configDir := filepath.Join(projectRoot, ".vyb")
	if err := os.Mkdir(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create .vyb directory: %w", err)
	}

	// ------------------------------------------------------------------
	// 1. Persist configuration – this must happen before metadata so that
	//    later code relying on config.Load() works even during init.
	// ------------------------------------------------------------------
	cfg := &config.Config{Provider: provider}
	cfgBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config.yaml: %w", err)
	}
	if len(cfgBytes) == 0 || cfgBytes[len(cfgBytes)-1] != '\n' {
		cfgBytes = append(cfgBytes, '\n')
	}
	cfgPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(cfgPath, cfgBytes, 0644); err != nil {
		return fmt.Errorf("failed to write config.yaml: %w", err)
	}

	// ------------------------------------------------------------------
	// 2. Build and annotate metadata as before.
	// ------------------------------------------------------------------
	metadata, err := buildMetadata(rootFS)
	if err != nil {
		return fmt.Errorf("failed to build metadata: %w", err)
	}

	err = annotate(cfg, metadata, rootFS)
	if err != nil {
		return fmt.Errorf("failed to annotate metadata: %w", err)
	}

	data, err := yaml.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata.yaml: %w", err)
	}

	metaFilePath := filepath.Join(configDir, "metadata.yaml")
	if err := os.WriteFile(metaFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata.yaml: %w", err)
	}

	return nil
}

// BuildMetadataFS exposes the internal buildMetadata helper so that external
// packages (e.g. cmd/template) can generate a *fresh* snapshot of the workspace
// file structure without losing the richer annotation data stored on disk.
//
// The behaviour is identical to buildMetadata – it walks the filesystem rooted
// at the provided fs.FS, produces a full Module/File hierarchy and returns the
// resulting *Metadata.
func BuildMetadataFS(fsys fs.FS) (*Metadata, error) {
	return buildMetadata(fsys)
}

// buildMetadata builds a metadata representation for the files available in
// the given filesystem
func buildMetadata(fsys fs.FS) (*Metadata, error) {
	// Build a minimal execution context anchored at workspace root so selector
	// includes *all* files. We bypass constructor to avoid filesystem checks
	// (unit-tests use fstest.MapFS).
	ec := &context.ExecutionContext{ProjectRoot: ".", WorkingDir: ".", TargetDir: "."}

	selected, err := selector.Select(fsys, ec, systemExclusionPatterns, []string{"*"})
	if err != nil {
		return nil, fmt.Errorf("failed during file selection: %w", err)
	}

	rootModule, err := buildModuleFromFS(fsys, selected)
	if err != nil {
		return nil, fmt.Errorf("failed to build summary module tree: %w", err)
	}

	metadata := &Metadata{
		Modules: rootModule,
	}
	return metadata, nil
}

// loadStoredMetadata reads the .vyb/metadata.yaml in the given fs.FS.
// It parses its contents into a Metadata struct. If the file is
// not found or if parsing fails, it returns an error.
func loadStoredMetadata(fsys fs.FS) (*Metadata, error) {
	data, err := fs.ReadFile(fsys, ".vyb/metadata.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata file .vyb/metadata.yaml: %w", err)
	}

	var meta Metadata
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata from .vyb/metadata.yaml: %w", err)
	}

	return &meta, nil
}

// WrongRootError is returned by Remove when the current directory is not a
// valid project root.
type WrongRootError struct {
	Root *string
}

func (w WrongRootError) Error() string {
	if w.Root == nil {
		return "Error: Folder has no project configuration. Project root is unknown."
	}
	return fmt.Sprintf("Error: Removal attempted on a folder that is not configured as the project root. Project root is %s", *w.Root)
}

func newWrongRootErr(root string) *WrongRootError {
	return &WrongRootError{
		Root: &root,
	}
}

// Remove removes the metadata folder directly under a given project root
// directory. projectRoot must point to a directory with a .vyb directory under
// it, otherwise the operation fails.
func Remove(projectRoot string) error {
	absPath, err := filepath.Abs(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to determine absolute path of project root: %w", err)
	}

	configDir := filepath.Join(absPath, ".vyb")
	info, err := os.Stat(configDir)
	if err != nil {
		if os.IsNotExist(err) {
			return newWrongRootErr(absPath)
		}
		return fmt.Errorf("failed to stat .vyb directory: %w", err)
	}

	if !info.IsDir() {
		return newWrongRootErr(absPath)
	}

	if err := os.RemoveAll(configDir); err != nil {
		return fmt.Errorf("failed to remove .vyb directory: %w", err)
	}

	return nil
}

// findAllConfigWithinRoot recursively scans the provided file system for directories named
// ".vyb". It returns a slice of paths (relative to the provided file system's root) where
// ".vyb" directories are found.
func findAllConfigWithinRoot(projectRoot fs.FS) ([]string, error) {
	var matches []string
	err := fs.WalkDir(projectRoot, ".", func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && info.Name() == ".vyb" {
			matches = append(matches, path)
			return fs.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking the file tree: %w", err)
	}
	sort.Strings(matches)
	return matches, nil
}

// -------------------- internal helpers --------------------

var minTokenCountPerModule int64 = 10000
var maxTokenCountPerModule int64 = 100000

// collapseByTokens walks the tree bottom-up, merging children whose cumulative
// token counts are < minTokenCountPerModule into their parent when this does not push the
// parent direct token count above maxTokenCountPerModule.
//
// The function mutates the provided module tree.
func collapseByTokens(m *Module) {
	// Recurse first so children are already processed.
	for _, child := range m.Modules {
		collapseByTokens(child)
	}

	// Don't collapse the root module.
	if m.Name == "." {
		return
	}

	// Iterate over children and merge the small ones.
	for i := 0; i < len(m.Modules); {
		child := m.Modules[i]

		if child.localTokenCount < minTokenCountPerModule {
			// Can we merge? Check direct token limit for parent.
			if m.localTokenCount+child.localTokenCount <= maxTokenCountPerModule {
				// Adopt child's files.
				m.Files = append(m.Files, child.Files...)
				// Remove child and adopt its sub-modules.
				m.Modules = append(m.Modules[:i], m.Modules[i+1:]...)
				m.Modules = append(m.Modules, child.Modules...)
				m.localTokenCount += child.localTokenCount
				// Do NOT advance i – re-evaluate new item in same index.
				continue
			}
		}
		i++
	}
}

// rebuildModule converts a pre-existing *Module hierarchy into a new
// tree where each node is produced via newModule so token counts and hashes
// are accurate.
func rebuildModule(old *Module, parent *Module) *Module {
	if old == nil {
		return nil
	}
	// Convert children first.
	var children []*Module
	for _, c := range old.Modules {
		children = append(children, rebuildModule(c, old))
	}
	return newModule(old.Name, parent, children, old.Files, old.Annotation)
}
