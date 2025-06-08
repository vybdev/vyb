package project

import "testing"

func TestFindModule_RootFile(t *testing.T) {
    // build a tiny hierarchy: root (.) with child "dir"
    root := &Module{Name: "."}
    child := &Module{Name: "dir", Parent: root}
    root.Modules = []*Module{child}

    // Case 1: file directly in root – expect root module.
    if got := FindModule(root, "README.md"); got != root {
        t.Fatalf("expected root module for file in project root, got %v", got)
    }

    // Case 2: file inside dir – expect child module.
    if got := FindModule(root, "dir/file.txt"); got != child {
        t.Fatalf("expected child module for nested file, got %v", got)
    }
}
