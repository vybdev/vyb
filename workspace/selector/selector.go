package selector

import (
	"bufio"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/dangazineu/vyb/workspace/matcher"
)

// Select selects which files should be in scope, based on a series of criteria:
// - the return value of the function is a slice with a list of file names, relative to the projectRoot (only files, no directories);
// - if no target is provided, every file directly and indirectly under the baseDir will be evaluated;
// - if a target is provided, and it is a directory, only files directly and indirectly under the target folder will be evaluated;
// - if a target is provided, and it is a file, only files directly and indirectly under the target file's enclosing directory will be evaluated;
// - A file is included in the response if matcher.IsIncluded returns true;
// - If a directory is excluded if matcher.IsExcluded returns true;
// - If a directory is excluded, none of its contents will be evaluated;
// - For each directory that is not excluded, if a .gitignore file is present, it will be read, and its contents will be appended to the exclusionPatterns for this and all its sub-directories;
// - All arguments (commandBaseDir, target, exclusionPatterns, and inclusionPatterns) are relative to the projectRoot;
// - .gitignore patterns are relative to the directory where the .gitignore file was found;
func Select(projectRoot fs.FS, commandBaseDir string, target *string, exclusionPatterns, inclusionPatterns []string) ([]string, error) {
	// Determine the startDir based on commandBaseDir and target.
	startDir := path.Clean(commandBaseDir)
	if target != nil {
		t := path.Clean(*target)
		info, err := fs.Stat(projectRoot, t)
		if err == nil {
			if !info.IsDir() {
				// target is a file, use its parent directory as the starting directory
				startDir = path.Dir(t)
			} else {
				startDir = t
			}
		}
		// If error, fallback to commandBaseDir
	}

	var results []string

	// effectiveExclusions holds the accumulated exclusion patterns for each directory.
	effectiveExclusions := make(map[string][]string)

	// Helper function to check if two paths are related.
	// Two paths are considered related if one is an ancestor of the other.
	// Always returns true for the root directory.
	isRelated := func(curr, target string) bool {
		curr = path.Clean(curr)
		target = path.Clean(target)
		if curr == "." || target == "." {
			return true
		}
		// Append "/" to ensure we match whole directory segments.
		currPrefix := curr + "/"
		targetPrefix := target + "/"
		res := strings.HasPrefix(targetPrefix, currPrefix) || strings.HasPrefix(currPrefix, targetPrefix)
		return res
	}

	// Walk the entire project from the project root to accumulate all .gitignore exclusions.
	err := fs.WalkDir(projectRoot, ".", func(currPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Only process directories (and their files) that are related to startDir.
		// That is, the directory is either an ancestor of startDir or is under startDir.
		if d.IsDir() {
			if !isRelated(currPath, startDir) {
				return fs.SkipDir
			}
		} else {
			// Only process files that are under the startDir
			rel, relErr := filepath.Rel(startDir, currPath)
			if relErr != nil || rel == ".." || strings.HasPrefix(rel, "../") {
				return nil
			}
		}

		// Determine parent's effective exclusions.
		parentDir := path.Dir(currPath)
		parentExclusions, ok := effectiveExclusions[parentDir]
		if !ok {
			// Fallback in case parent's exclusions are missing.
			parentExclusions = exclusionPatterns
		}

		// When processing a directory, first check if it should be excluded.
		if d.IsDir() {
			if matcher.IsExcluded(projectRoot, currPath, parentExclusions) {
				return fs.SkipDir
			}
			// Compute current directory's effective exclusions.
			effectiveExclusions[currPath] = computeEffectiveExclusions(projectRoot, currPath, parentExclusions)
		} else { // File
			if matcher.IsIncluded(projectRoot, currPath, parentExclusions, inclusionPatterns) {
				results = append(results, currPath)
			}
		}
		return nil
	})
	return results, err
}

// computeEffectiveExclusions extracts the effective exclusion patterns for a directory.
// It starts with the provided baseExclusions and appends patterns from a .gitignore file, if present.
func computeEffectiveExclusions(projectRoot fs.FS, dir string, baseExclusions []string) []string {
	exclusions := append([]string{}, baseExclusions...)
	gitignorePath := path.Join(dir, ".gitignore")
	if data, err := fs.ReadFile(projectRoot, gitignorePath); err == nil {
		exclusions = append(exclusions, parseGitignore(string(data))...)
	}
	return exclusions
}

// parseGitignore parses the content of a .gitignore file and returns a slice of patterns.
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
		// Additional parsing rules can be added here if needed.
		patterns = append(patterns, line)
		if err == io.EOF {
			break
		}
	}
	return patterns
}
