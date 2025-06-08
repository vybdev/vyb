package selector

import (
	"bufio"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/dangazineu/vyb/workspace/context"
	"github.com/dangazineu/vyb/workspace/matcher"
)

// Select walks the workspace starting from ec.TargetDir (relative to the
// project root) collecting every file that matches inclusion/exclusion
// patterns.  All parameters – except projectRoot – are derived from the
// provided ExecutionContext.
//
// Invariants:
//   - ec.ProjectRoot MUST correspond to the same filesystem represented by
//     projectRoot (no runtime cross-validation is performed).
//   - Only paths that are **under** ec.TargetDir are evaluated.
//
// - If a directory is excluded if matcher.IsExcluded returns true;
// - If a directory is excluded, none of its contents will be evaluated;
// - For each directory that is not excluded, if a .gitignore file is present, it will be read, and its contents will be appended to the exclusionPatterns for this and all its sub-directories;
// - All arguments (commandBaseDir, target, exclusionPatterns, and inclusionPatterns) are relative to the projectRoot;
// - .gitignore patterns are relative to the directory where the .gitignore file was found;
func Select(projectRoot fs.FS, ec *context.ExecutionContext, exclusionPatterns, inclusionPatterns []string) ([]string, error) {
	if ec == nil {
		return nil, fs.ErrInvalid
	}

	// Compute the directory (relative to project root) that will seed the
	// walk – guaranteed to be within the workspace as enforced by
	// ExecutionContext.
	relStart := "."
	if rel, err := filepath.Rel(ec.ProjectRoot, ec.TargetDir); err == nil {
		relStart = filepath.ToSlash(rel)
	}

	// effectiveExclusions keeps the accumulated exclusion patterns per dir.
	effectiveExclusions := map[string][]string{}

	var results []string

	// Helper: determines if a given path (directory) is *inside* relStart –
	// used to prune walkDir.
	isWithinStart := func(p string) bool {
		if relStart == "." {
			return true
		}
		p = path.Clean(p)
		relStartClean := path.Clean(relStart)
		return p == relStartClean || strings.HasPrefix(p+"/", relStartClean+"/")
	}

	err := fs.WalkDir(projectRoot, ".", func(currPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip anything outside targetDir just in case WalkDir goes up due to
		// relStart == ".".
		if !isWithinStart(currPath) {
			return fs.SkipDir
		}

		parentDir := path.Dir(currPath)
		parentExcl := effectiveExclusions[parentDir]
		parentExcl = append(parentExcl, exclusionPatterns...)

		// Directory handling.
		if d.IsDir() {
			// Apply parent exclusion patterns to decide whether to descend.
			if matcher.IsExcluded(projectRoot, currPath, parentExcl) {
				return fs.SkipDir
			}
			// Build this dir's exclusion list inheriting parent + .gitignore.
			effectiveExclusions[currPath] = computeEffectiveExclusions(projectRoot, currPath, parentExcl)
			return nil
		}

		// File: include if patterns allow.
		if matcher.IsIncluded(projectRoot, currPath, parentExcl, inclusionPatterns) {
			results = append(results, currPath)
		}
		return nil
	})

	return results, err
}

// computeEffectiveExclusions extracts the effective exclusion patterns for a
// directory. It starts with the provided baseExclusions and appends patterns
// from a .gitignore file, if present.
func computeEffectiveExclusions(projectRoot fs.FS, dir string, baseExclusions []string) []string {
	exclusions := append([]string{}, baseExclusions...)
	gitignorePath := path.Join(dir, ".gitignore")
	if data, err := fs.ReadFile(projectRoot, gitignorePath); err == nil {
		exclusions = append(exclusions, parseGitignore(string(data))...)
	}
	return exclusions
}

// parseGitignore parses the content of a .gitignore file and returns a slice
// of patterns.
func parseGitignore(data string) []string {
	var patterns []string
	reader := strings.NewReader(data)
	buf := bufio.NewReader(reader)
	for {
		line, err := buf.ReadString('\n')
		if err != nil && err != io.EOF {
			break
		}
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			if err == io.EOF {
				break
			}
			continue
		}
		patterns = append(patterns, line)
		if err == io.EOF {
			break
		}
	}
	return patterns
}
