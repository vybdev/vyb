package context

import (
	"fmt"
	"os"
	"path/filepath"
)

// ExecutionContext captures the three key path concepts used by vyb
// commands. All fields are absolute, clean paths.
//
//   • ProjectRoot – directory that contains the .vyb folder.
//   • WorkingDir  – directory from which the command is executed. Must be
//                   the same as ProjectRoot or a descendant of it.
//   • TargetDir   – directory containing the target file (if one was
//                   provided to the command). When no target is given it
//                   equals WorkingDir. TargetDir is guaranteed to be the
//                   same as WorkingDir or a descendant of it.
//
// Invariants are enforced by the constructor – direct struct instantiation
// outside this package is discouraged.
//
// NOTE: This package purposefully sits outside the project/root package so
// it can be reused by matcher, selector and template with no import
// cycles.
//
// TODO(vyb): Add convenience helpers (e.g. Rel(path)) when required by
// later tasks.
type ExecutionContext struct {
	ProjectRoot string
	WorkingDir  string
	TargetDir   string
}

// NewExecutionContext validates and returns an ExecutionContext.
//
// Parameters must be *absolute* paths. If targetFile is nil it is treated
// as if no target was provided.
func NewExecutionContext(projectRoot, workingDir string, targetFile *string) (*ExecutionContext, error) {
	// Sanity-check that we received absolute paths.
	if !filepath.IsAbs(projectRoot) || !filepath.IsAbs(workingDir) {
		return nil, fmt.Errorf("projectRoot and workingDir must be absolute paths")
	}
	var targetAbs string
	if targetFile != nil {
		if !filepath.IsAbs(*targetFile) {
			return nil, fmt.Errorf("targetFile must be an absolute path when provided")
		}
		targetAbs = filepath.Clean(*targetFile)
	}

	root := filepath.Clean(projectRoot)
	work := filepath.Clean(workingDir)

	// Ensure .vyb exists inside projectRoot.
	if fi, err := os.Stat(filepath.Join(root, ".vyb")); err != nil || !fi.IsDir() {
		return nil, fmt.Errorf("%s is not a valid project root – missing .vyb directory", root)
	}

	// workingDir must be under projectRoot.
	if !isDescendant(root, work) {
		return nil, fmt.Errorf("workingDir %s is not within projectRoot %s", work, root)
	}

	// Derive/validate targetDir when a target file is provided.
	var targetDir string
	if targetFile != nil {
		fi, err := os.Stat(targetAbs)
		if err != nil {
			return nil, fmt.Errorf("target file %s does not exist: %w", targetAbs, err)
		}
		if fi.IsDir() {
			return nil, fmt.Errorf("target %s is a directory, expected a file", targetAbs)
		}

		if !isDescendant(work, targetAbs) {
			return nil, fmt.Errorf("target file %s is outside workingDir %s", targetAbs, work)
		}

		targetDir = filepath.Dir(targetAbs)
	} else {
		targetDir = work
	}

	return &ExecutionContext{
		ProjectRoot: root,
		WorkingDir:  work,
		TargetDir:   filepath.Clean(targetDir),
	}, nil
}

// isDescendant returns true when child == parent or child is somewhere
// below parent in the directory hierarchy.
func isDescendant(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel == "." || !startsWithDotDot(rel)
}

func startsWithDotDot(rel string) bool {
	return rel == ".." || filepath.HasPrefix(rel, ".."+string(os.PathSeparator))
}
