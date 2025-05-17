// +build !windows

package matcher

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// createFile is a utility function that creates a file with some content
// relative to the base directory. It also ensures that any parent directory exists.
func createFile(base, relPath, content string) error {
	fullPath := filepath.Join(base, relPath)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}
	return nil
}

func Test_IsIncluded(t *testing.T) {
	tests := []struct {
		name        string
		pathToTest  string // relative path to validate
		exclusions  []string
		inclusions  []string
		want        bool
		explanation string
	}{
		{
			name:        "no patterns",
			pathToTest:  "foo.txt",
			exclusions:  []string{},
			inclusions:  []string{},
			want:        false,
			explanation: "No exclusion and no inclusion means file is not included.",
		},
		{
			name:        "simple inclusion",
			pathToTest:  "foo.txt",
			exclusions:  []string{},
			inclusions:  []string{"foo.txt"},
			want:        true,
			explanation: "Exact inclusion of the file.",
		},
		{
			name:        "exclusion takes precedence",
			pathToTest:  "foo.txt",
			exclusions:  []string{"*.txt"},
			inclusions:  []string{"*"},
			want:        false,
			explanation: "Exclusion matching *.txt prevents inclusion even though * would include it.",
		},
		{
			name:        "exclusion not matching",
			pathToTest:  "foo.txt",
			exclusions:  []string{"*.log"},
			inclusions:  []string{"*"},
			want:        true,
			explanation: "File is not excluded and matches the inclusion pattern.",
		},
		{
			name:        "negated exclusion for nested file with wildcard inclusion",
			pathToTest:  "dir/foo.txt",
			exclusions:  []string{"dir/*", "!dir/foo.txt"},
			inclusions:  []string{"*"},
			want:        true,
			explanation: "Exclusion removes all files in dir but negated for foo.txt, so inclusion applies. Inclusion is the * wildcard, so it matches everything that is not excluded.",
		},
		{
			// [vyb] TODO(user): Please clarify if inclusion patterns should match the entire relative path from the project root or only the basename? Implementation currently uses the entire relative path.
			name:        "negated exclusion for nested file",
			pathToTest:  "dir/foo.txt",
			exclusions:  []string{"dir/*", "!dir/foo.txt"},
			inclusions:  []string{"dir/*"},
			want:        true,
			explanation: "Exclusion removes all files in dir but negated for foo.txt, so inclusion applies. Inclusion is the dir/*, so it matches everything that is not excluded within the dir/ folder",
		},
		{
			name:        "negated exclusion fallback fails",
			pathToTest:  "dir/bar.txt",
			exclusions:  []string{"dir/*", "!dir/foo.txt"},
			inclusions:  []string{"dir/bar.txt"},
			want:        false,
			explanation: "bar.txt remains excluded since negation did not apply.",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			base := t.TempDir()
			if err := createFile(base, tc.pathToTest, "content"); err != nil {
				t.Fatalf("setup failed: %v", err)
			}
			got := IsIncluded(os.DirFS(base), tc.pathToTest, tc.exclusions, tc.inclusions)
			if got != tc.want {
				t.Fatalf("For %s, want %v, got %v. Explanation: %s", tc.pathToTest, tc.want, got, tc.explanation)
			}
		})
	}
}

func Test_matchesPattern(t *testing.T) {
	tests := []struct {
		path        string
		isDir       bool
		template    string
		matchAll    bool
		want        bool
		explanation string
	}{
		{"foo.md", false, "*", true, true, ""},
		{"foo.md", false, "*.md", true, true, ""},
		{"foo.md", false, "foo.*", true, true, ""},
		{"foo.MD", false, "*.md", true, false, "extensions are case-sensitive"},
		{"foo.MD", false, "Foo.*", true, false, "file names are case-sensitive"},
		{"foo/bar.txt", false, "foo/", false, true, "When matchAll is false and the template is a directory, it should match the directory hierarchy, not the entire file path"},
		{"foo/baz/bar.txt", false, "foo/", false, true, "Partial match on a directory matching matches the entire directory hierarchy"},
		{"baz/foo/bar.txt", false, "foo/", false, false, "Partial match on a directory matching pattern must start from the beginning of the path"},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("(%s, %s)", tc.path, tc.template), func(t *testing.T) {
			got := matchesPattern(&NameAndDir{tc.path, tc.isDir}, tc.path, tc.template, tc.matchAll)
			if tc.want != got {
				t.Fatalf("matchesPattern(%s, %s) -> %v, want %v: %s", tc.path, tc.template, got, tc.want, tc.explanation)
			}
		})
	}
}

func Test_matchesInclusionPatterns(t *testing.T) {
	gitignoreExample := []string{
		"*",      // matches every file's basename
		"!/foo",  // negation: explicitly exclude directory "foo"
		"/foo/*", // matches files and subdirectories immediately under "foo"
		"!/foo/bar",
	}

	tests := []struct {
		path        string
		isDir       bool
		templates   []string
		want        bool
		explanation string
	}{
		{"a.md", false, gitignoreExample, true, "matches first rule and no other"},
		{"foo/a.md", false, gitignoreExample, true, "exclusion in /foo then re-inclusion in /foo/*"},
		{"foo/bar/a.md", false, []string{"*"}, true, "* should match every file name in every directory"},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("(%s)", tc.path), func(t *testing.T) {
			got := matchesInclusionPatterns(&NameAndDir{tc.path, tc.isDir}, tc.path, tc.templates)
			if tc.want != got {
				t.Fatalf("matchesInclusionPatterns(%s, %v) -> %v, want %v: %s", tc.path, tc.templates, got, tc.want, tc.explanation)
			}
		})
	}
}

func Test_matchesExclusionPatterns(t *testing.T) {
	tests := []struct {
		path        string
		isDir       bool
		templates   []string
		want        bool
		explanation string
	}{
		{
			path:        "bar/foo/baz.txt",
			isDir:       false,
			templates:   []string{"bar/"},
			want:        true,
			explanation: "A directory exclusion pattern excludes the directory and all of its contents.",
		},
		{
			path:        "bar/foo.txt",
			isDir:       false,
			templates:   []string{"bar/", "!bar/*.txt"},
			want:        true,
			explanation: "Once a directory is excluded, the exclusion cannot be negated for any file or subdirectory within it.",
		},
		{
			path:        "bar/foo.txt",
			isDir:       false,
			templates:   []string{"bar/*", "!bar/*.txt"},
			want:        false,
			explanation: "Files within a directory are excluded, but the exclusion can be negated.",
		},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("(%s, %v)", tc.path, tc.templates), func(t *testing.T) {
			got := matchesExclusionPatterns(&NameAndDir{tc.path, tc.isDir}, tc.path, tc.templates)
			if tc.want != got {
				t.Fatalf("matchesExclusionPatterns(%s, %v) -> %v, want %v: %s", tc.path, tc.templates, got, tc.want, tc.explanation)
			}
		})
	}
}

func Test_isIncluded(t *testing.T) {
	// Tests for the new isIncluded logic using exclusion and inclusion patterns separately.
	tests := []struct {
		path        string
		isDir       bool
		exclusions  []string
		inclusions  []string
		want        bool
		explanation string
	}{
		{
			path:        "foo.txt",
			isDir:       false,
			exclusions:  []string{},
			inclusions:  []string{},
			want:        false,
			explanation: "File is not excluded, but there is no inclusion pattern, so it isn't included.",
		},
		{
			path:        "foo.txt",
			isDir:       false,
			exclusions:  []string{},
			inclusions:  []string{"foo.txt"},
			want:        true,
			explanation: "File is not excluded, and is included through an exact match.",
		},
		{
			path:        "foo.txt",
			isDir:       false,
			exclusions:  []string{"*.txt"},
			inclusions:  []string{"*"},
			want:        false,
			explanation: "File is excluded by matching exclusion pattern, even though it matches inclusion.",
		},
		{
			path:        "foo.txt",
			isDir:       false,
			exclusions:  []string{"*.log"},
			inclusions:  []string{"*"},
			want:        true,
			explanation: "File is not excluded and matches the inclusion pattern.",
		},
		{
			path:        "foo.log",
			isDir:       false,
			exclusions:  []string{"*.log"},
			inclusions:  []string{"foo.log"},
			want:        false,
			explanation: "File is excluded by matching exclusion pattern.",
		},
		{
			path:        "dir/foo.txt",
			isDir:       false,
			exclusions:  []string{"dir/*", "!dir/foo.txt"},
			inclusions:  []string{"dir/foo.txt"},
			want:        true,
			explanation: "Exclusion pattern negated for foo.txt makes it eligible for inclusion.",
		},
		{
			path:        "dir/bar.txt",
			isDir:       false,
			exclusions:  []string{"dir/*"},
			inclusions:  []string{"dir/bar.txt"},
			want:        false,
			explanation: "File remains excluded and cannot match the inclusion pattern.",
		},
		{
			path:        "dir/bar.txt",
			isDir:       false,
			exclusions:  []string{"dir/", "!dir/bar.txt"},
			inclusions:  []string{"dir/foo.txt"},
			want:        false,
			explanation: "File remains excluded and is not re-included.",
		},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("(%s, %v, %v)", tc.path, tc.exclusions, tc.inclusions), func(t *testing.T) {
			got := isIncluded(&NameAndDir{tc.path, tc.isDir}, tc.path, tc.exclusions, tc.inclusions)
			if tc.want != got {
				t.Fatalf("isIncluded(%s, %v, %v) -> %v, want %v: %s",
					tc.path, tc.exclusions, tc.inclusions, got, tc.want, tc.explanation)
			}
		})
	}
}

type NameAndDir struct {
	name  string
	isDir bool
}

func (m *NameAndDir) Sys() any {
	return nil
}

func (m *NameAndDir) Name() string {
	return m.name
}

func (m *NameAndDir) Size() int64 {
	return 0
}

func (m *NameAndDir) Mode() os.FileMode {
	return os.ModeTemporary
}

func (m *NameAndDir) ModTime() time.Time {
	return time.Now()
}

func (m *NameAndDir) IsDir() bool {
	return m.isDir
}
