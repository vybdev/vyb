package context

import (
	"os"
	"path/filepath"
	"testing"
)

// helper to create temp project with optional sub-dirs/files.
func setupProject(t *testing.T) string {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".vyb"), 0o755); err != nil {
		t.Fatalf("failed to create .vyb: %v", err)
	}
	return dir
}

func TestNewExecutionContext_ValidNoTarget(t *testing.T) {
	root := setupProject(t)

	ec, err := NewExecutionContext(root, root, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ec.ProjectRoot != root || ec.WorkingDir != root || ec.TargetDir != root {
		t.Fatalf("unexpected paths in context: %+v", ec)
	}
}

func TestNewExecutionContext_ValidWithTarget(t *testing.T) {
	root := setupProject(t)
	work := filepath.Join(root, "sub")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	targetFile := filepath.Join(work, "file.txt")
	if err := os.WriteFile(targetFile, []byte("data"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	ec, err := NewExecutionContext(root, work, &targetFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ec.TargetDir != work {
		t.Fatalf("expected TargetDir %s, got %s", work, ec.TargetDir)
	}
}

func TestNewExecutionContext_ErrWorkingDirOutsideRoot(t *testing.T) {
	root := setupProject(t)
	outside := filepath.Dir(root) // parent of root
	_, err := NewExecutionContext(root, outside, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestNewExecutionContext_ErrTargetOutsideWork(t *testing.T) {
	root := setupProject(t)
	work := filepath.Join(root, "some")
	target := filepath.Join(root, "other", "file.txt")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := NewExecutionContext(root, work, &target)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
