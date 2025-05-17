package matcher

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// mockFileInfo is a helper struct used to simulate file info when the
// path does not exist on disk.
type mockFileInfo struct {
	name  string
	isDir bool
}

func (m mockFileInfo) Name() string       { return m.name }
func (m mockFileInfo) Size() int64        { return 0 }
func (m mockFileInfo) Mode() os.FileMode {
	if m.isDir {
		return os.ModeDir | 0755
	}
	return 0644
}
func (m mockFileInfo) ModTime() time.Time {
	return time.Time{}
}
func (m mockFileInfo) IsDir() bool { return m.isDir }
func (m mockFileInfo) Sys() any    { return nil }

// IsIncluded takes a file path and a `.gitignore` style matching pattern slice and returns true only if the file
// does not match the exclusion patterns AND matches the inclusion patterns.
func IsIncluded(projectRoot fs.FS, filePath string, exclusionPatterns, inclusionPatterns []string) bool {
	fileInfo, err := fs.Stat(projectRoot, filePath)
	if err != nil {
		if os.IsNotExist(err) {
			isDir := strings.HasSuffix(filePath, "/")
			mockFi := mockFileInfo{
				name:  filepath.Base(strings.TrimSuffix(filePath, "/")),
				isDir: isDir,
			}
			return isIncluded(mockFi, filePath, exclusionPatterns, inclusionPatterns)
		}

		fmt.Printf("Couldn't stat %s\n", filePath)
		return false
	}
	return isIncluded(fileInfo, filePath, exclusionPatterns, inclusionPatterns)
}

// IsExcluded takes a file path and a `.gitignore` style matching pattern slice and returns true if the file
// does matches the exclusion patterns.
func IsExcluded(projectRoot fs.FS, filePath string, exclusionPatterns []string) bool {
	fileInfo, err := fs.Stat(projectRoot, filePath)
	if err != nil {
		fmt.Printf("Couldn't stat %s\n", filePath)
		return false
	}
	return matchesExclusionPatterns(fileInfo, filePath, exclusionPatterns)
}

// isIncluded applies exclusion patterns first with support for negation.
// Exclusion patterns are processed in order: if a non-negated pattern matches, the file is marked as excluded;
// if a later negated pattern matches, it reverses the exclusion.
// After applying exclusions, if the file is not excluded, it is included only if it matches at least one
// non-negated inclusion pattern (or is not explicitly negated by an inclusion pattern).
func isIncluded(fileInfo fs.FileInfo, filePath string, exclusionPatterns, inclusionPatterns []string) bool {
	if matchesExclusionPatterns(fileInfo, filePath, exclusionPatterns) {
		return false
	}

	return matchesInclusionPatterns(fileInfo, filePath, inclusionPatterns)
}

// matchesExclusionPatterns returns true if the filePath matches any of the given exclusionPatterns.
func matchesExclusionPatterns(fileInfo fs.FileInfo, filePath string, exclusionPatterns []string) bool {
	excluded := false
	for _, pattern := range exclusionPatterns {
		if pattern == "" {
			continue
		}
		if strings.HasPrefix(pattern, "!") {
			actualPattern := pattern[1:]
			if matchesPattern(fileInfo, filePath, actualPattern, false) {
				excluded = false
			}
		} else {
			if matchesPattern(fileInfo, filePath, pattern, false) {
				// When evaluating exclusion patterns, if a directory matching pattern matches the file path,
				// then it immediately exits with a match.
				if isDirMatcher(pattern) {
					return true
				}
				excluded = true
			}
		}
	}
	return excluded
}

func matchesInclusionPatterns(fileInfo fs.FileInfo, filePath string, inclusionPatterns []string) bool {
	if len(inclusionPatterns) > 0 {
		// Process inclusion patterns
		for _, pattern := range inclusionPatterns {
			if pattern == "" {
				continue
			}
			if strings.HasPrefix(pattern, "!") {
				actualPattern := pattern[1:]
				if matchesPattern(fileInfo, filePath, actualPattern, true) {
					return false
				}
				continue
			}
			if matchesPattern(fileInfo, filePath, pattern, true) {
				return true
			}
		}
		// If inclusion patterns were provided but none matched, do not include the file.
		return false
	}
	return false
}

// matchesPattern matches a file path to a given matcher pattern.
// The following pattern format spec was copied from https://git-scm.com/docs/gitignore
// PATTERN FORMAT
// - A blank pattern matches no files, so it can serve as a separator for readability.
// - A pattern starting with # serves as a comment. Put a backslash ("\") in front of the first hash for patterns that begin with a hash.
// - Trailing spaces are ignored unless they are quoted with backslash ("\").
// - An optional prefix "!" which negates the pattern; any matching file excluded by a previous pattern will become included again. It is not possible to re-include a file if a parent directory of that file is excluded. Git doesn’t list excluded directories for performance reasons, so any patterns on contained files have no effect, no matter where they are defined. Put a backslash ("\") in front of the first "!" for patterns that begin with a literal "!", for example, "\!important!.txt".
// - The slash "/" is used as the directory separator. Separators may occur at the beginning, middle or end of the .gitignore search pattern.
// - If there is a separator at the beginning or middle (or both) of the pattern, then the pattern is relative to the directory level of the particular .gitignore file itself. Otherwise the pattern may also match at any level below the .gitignore level.
// - If there is a separator at the end of the pattern then the pattern will only match directories, otherwise the pattern can match both files and directories.
// - For example, a pattern doc/frotz/ matches the doc/frotz directory, but not a/doc/frotz directory; however frotz/ matches frotz and a/frotz that is a directory (all paths are relative from the .gitignore file).
// - An asterisk "*" matches anything except a slash. The character "?" matches any one character except "/". The range notation, e.g. [a-zA-Z], can be used to match one of the characters in a range. See fnmatch(3) and the FNM_PATHNAME flag for a more detailed description.
// - Two consecutive asterisks ("**") in patterns matched against full pathname may have special meaning:
//   - A leading "**" followed by a slash means match in all directories. For example, "**/foo" matches file or directory "foo" anywhere, the same as pattern "foo". "**/foo/bar" matches file or directory "bar" anywhere that is directly under directory "foo".
//   - A trailing "/**" matches everything inside. For example, "abc/**" matches all files inside directory "abc", relative to the location of the .gitignore file, with infinite depth.
//   - A slash followed by two consecutive asterisks then a slash matches zero or more directories. For example, "a/**/b" matches "a/b", "a/x/b", "a/x/y/b" and so on.
//   - Other consecutive asterisks are considered regular asterisks and will match according to the previous rules.
func matchesPattern(fileInfo fs.FileInfo, filePath string, matcher string, matchAll bool) bool {
	dirMatcher := isDirMatcher(matcher)
	// Shortcut: a directory matching pattern should only match directories.
	if fileInfo.IsDir() && !dirMatcher {
		return false
	}

	// Use the provided filePath (which is relative) and ensure it uses "/" as separator.
	normalizedPath := filepath.ToSlash(filePath)

	// Handle directory matcher when matchAll is false.
	if dirMatcher && !matchAll {
		trimmed := strings.TrimSuffix(matcher, "/")
		// Match if the normalizedPath is exactly the directory or is inside the directory.
		if normalizedPath == trimmed || strings.HasPrefix(normalizedPath, trimmed+"/") {
			return true
		}
		return false
	}

	// If the pattern does not contain a slash, it should be matched against the basename only.
	if !strings.Contains(matcher, "/") {
		return matchSingleSegment(filepath.Base(normalizedPath), matcher)
	}

	// If the pattern starts with a '/', it is relative to the root,
	// so remove the leading '/' to align with normalizedPath tokens.
	if strings.HasPrefix(matcher, "/") {
		matcher = matcher[1:]
	}

	// Split into tokens.
	fileTokens := strings.Split(normalizedPath, "/")
	patternTokens := strings.Split(matcher, "/")

	// Kick off recursive matching logic.
	return matchTokens(fileTokens, patternTokens)
}

// isDirMatcher returns true if the matcher pattern ends with a slash,
// indicating it should only match directories.
func isDirMatcher(matcher string) bool {
	return strings.HasSuffix(matcher, "/")
}

// matchTokens checks if path tokens match pattern tokens, including handling
// "**" as a wildcard for zero or more segments.
func matchTokens(pathTokens, patternTokens []string) bool {
	// If the pattern is exhausted, it's a match only if the path is also exhausted.
	if len(patternTokens) == 0 {
		return len(pathTokens) == 0
	}

	// Get the first pattern token.
	head := patternTokens[0]

	// Handle "**" (match zero or more segments).
	if head == "**" {
		// Option 1: Skip the "**" and see if the rest matches.
		if matchTokens(pathTokens, patternTokens[1:]) {
			return true
		}
		// Option 2: If there are segments left, consume one and keep trying.
		if len(pathTokens) > 0 {
			return matchTokens(pathTokens[1:], patternTokens)
		}
		return false
	}

	// If we're out of path tokens at this point, no match unless the pattern token is "**".
	if len(pathTokens) == 0 {
		return false
	}

	// Match current segment and pattern, then advance.
	if matchSingleSegment(pathTokens[0], head) {
		return matchTokens(pathTokens[1:], patternTokens[1:])
	}

	// Otherwise, no match.
	return false
}

// matchSingleSegment matches a single path segment (no slashes) against a
// .gitignore-style pattern containing possible "*" or "?" characters.
func matchSingleSegment(segment, pattern string) bool {
	si, pi := 0, 0

	for si < len(segment) && pi < len(pattern) {
		switch pattern[pi] {
		case '*':
			pi++
			if pi == len(pattern) {
				return true
			}
			for matchStart := si; matchStart <= len(segment); matchStart++ {
				if matchSingleSegment(segment[matchStart:], pattern[pi:]) {
					return true
				}
			}
			return false
		case '?':
			si++
			pi++
		default:
			if segment[si] != pattern[pi] {
				return false
			}
			si++
			pi++
		}
	}

	for pi < len(pattern) {
		if pattern[pi] != '*' {
			return false
		}
		pi++
	}

	return si == len(segment) && pi == len(pattern)
}
