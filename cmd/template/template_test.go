package template

import "testing"

func TestIsPathUnderDir(t *testing.T) {
	tests := []struct {
		workDir string
		path    string
		want    bool
	}{
		{".", "foo/bar.go", true},                // root WD allows all
		{"", "foo/bar.go", true},                 // empty behaves like root
		{"foo", "foo/bar.go", true},              // direct child
		{"foo", "foo", true},                     // same directory
		{"foo", "foobar/bar.go", false},          // sibling with common prefix
		{"foo", "bar/foo.go", false},             // different branch
		{"foo/bar", "foo/bar/baz.txt", true},     // nested deeper
		{"foo/bar", "foo/barbaz/baz.txt", false}, // false positive guard
	}

	for _, tc := range tests {
		got := isPathUnderDir(tc.workDir, tc.path)
		if got != tc.want {
			t.Fatalf("isPathUnderDir(%q, %q) = %v, want %v", tc.workDir, tc.path, got, tc.want)
		}
	}
}
