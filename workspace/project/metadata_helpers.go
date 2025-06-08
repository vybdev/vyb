package project

import (
    "fmt"
    "gopkg.in/yaml.v3"
    "io/fs"
    "os"
    "path/filepath"
    "strings"
)

// LoadMetadata reads .vyb/metadata.yaml under the provided absolute
// project root directory and unmarshals it into a *Metadata.  The
// function returns an error when the metadata file cannot be found or
// parsed.
func LoadMetadata(projectRoot string) (*Metadata, error) {
    if projectRoot == "" {
        return nil, fmt.Errorf("projectRoot must not be empty")
    }
    return LoadMetadataFS(os.DirFS(projectRoot))
}

// LoadMetadataFS performs the same operation as LoadMetadata but takes an
// fs.FS rooted at workspace root. This is mostly useful for tests where
// an in-memory fs.FS is more convenient than an OS path.
func LoadMetadataFS(fsys fs.FS) (*Metadata, error) {
    data, err := fs.ReadFile(fsys, ".vyb/metadata.yaml")
    if err != nil {
        return nil, fmt.Errorf("failed to read metadata.yaml: %w", err)
    }
    var m Metadata
    if err := yaml.Unmarshal(data, &m); err != nil {
        return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
    }
    return &m, nil
}

// FindModule returns the deepest *Module whose Name is an ancestor (or
// equal) to relPath. Both parameters must use forward-slash separators
// and be relative to the workspace root (same convention used by the
// Module.Name field).  When no matching module exists the function now
// returns the root module. This guarantees that files living directly in
// the workspace root are associated with the root module instead of
// returning nil.
func FindModule(root *Module, relPath string) *Module {
    if root == nil {
        return nil
    }
    // Normalise to forward-slash for consistent matching.
    relPath = filepath.ToSlash(relPath)

    // By default, the best match is the root module itself – this covers
    // files that live directly in the workspace root and therefore do
    // not belong to any nested sub-module.
    best := root

    var dfs func(*Module)
    dfs = func(m *Module) {
        for _, c := range m.Modules {
            if relPath == c.Name || (c.Name != "." && strings.HasPrefix(relPath, c.Name+"/")) {
                best = c
            }
            dfs(c)
        }
    }
    dfs(root)
    return best
}
