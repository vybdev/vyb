package selector

import (
	"fmt"
	"github.com/dangazineu/vyb/workspace/context"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestSelect(t *testing.T) {
	fsys := fstest.MapFS{
		"base/file1.txt":                {Data: []byte("content1")},
		"base/dir1/.gitignore":          {Data: []byte("ignored.txt\n")},
		"base/dir1/ignored.txt":         {Data: []byte("ignored content")},
		"base/dir1/file1.txt":           {Data: []byte("content1")},
		"base/dir1/subdir1/.gitignore":  {Data: []byte("# no ignore here\n")},
		"base/dir1/subdir1/ignored.txt": {Data: []byte("ignored also as it's inherited from parent\n")},
		"base/dir1/subdir1/file2.txt":   {Data: []byte("content2")},
		"base/dir1/subdir2/.gitignore":  {Data: []byte("*\n")},
		"base/dir1/subdir2/file3.txt":   {Data: []byte("this file should never be included\n")},
		"base/dir1/subdir3/file4.txt":   {Data: []byte("content4\n")},
		"base/dir2/file5.txt":           {Data: []byte("content5")},
		"base/dir2/file6.md":            {Data: []byte("# content6")},
		"base/dir2/subdir1/file7.md":    {Data: []byte("# content7")},
	}

	tests := []struct {
		baseDir     string
		target      *string
		exclusions  []string
		inclusions  []string
		want        []string
		explanation string
	}{
		{
			baseDir:    "base/dir1",
			target:     target("base/dir1/file1.txt"),
			exclusions: []string{".gitignore"},
			inclusions: []string{"*"},
			want: []string{
				"base/dir1/file1.txt",
				"base/dir1/subdir1/file2.txt",
				"base/dir1/subdir3/file4.txt",
			},
		},
		{
			baseDir:    "base/dir1",
			target:     target("base/dir1/file1.txt"),
			exclusions: []string{".gitignore", "file2.txt"},
			inclusions: []string{"*"},
			want: []string{
				"base/dir1/file1.txt",
				"base/dir1/subdir3/file4.txt",
			},
		},
		{
			baseDir:    "base/dir1/subdir1",
			exclusions: []string{".gitignore"},
			inclusions: []string{"*"},
			want: []string{
				"base/dir1/subdir1/file2.txt",
			},
		},
		{
			baseDir:    "base",
			target:     target("base/dir2/file5.txt"),
			exclusions: []string{".gitignore"},
			inclusions: []string{"*"},
			want: []string{
				"base/dir2/file5.txt",
				"base/dir2/file6.md",
				"base/dir2/subdir1/file7.md",
			},
		},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("TestSelect[%d]", i), func(t *testing.T) {
			ec := &context.ExecutionContext{ProjectRoot: ".", WorkingDir: tc.baseDir, TargetDir: func() string {
				if tc.target != nil {
					return filepath.Dir(*tc.target)
				}
				return tc.baseDir
			}()}

			got, err := Select(fsys, ec, tc.exclusions, tc.inclusions)
			if err != nil {
				t.Fatalf("Got an error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("(-want, +got):\n%s", diff)
			}
		})
	}
}

// TestSelect_TargetDirIsolation ensures that Select never returns files that
// live outside ec.TargetDir – even when they share part of the path or match
// the inclusion patterns. This guards against accidental context leakage to
// the LLM caused by future regressions in the traversal logic.
func TestSelect_TargetDirIsolation(t *testing.T) {
	fsys := fstest.MapFS{
		"root/work/a.txt":     {Data: []byte("w a")},
		"root/work/b.txt":     {Data: []byte("w b")},
		"root/work/sub/c.txt": {Data: []byte("w sub c")},
		"root/other/x.txt":    {Data: []byte("o x")},
	}

	// Simulate: project_root = root, working_dir = root/work, target = work/sub/c.txt
	// We expect only files under work/sub to be selected.
	targetFile := "root/work/sub/c.txt"
	ec := &context.ExecutionContext{
		ProjectRoot: ".",
		WorkingDir:  "root/work",
		TargetDir:   filepath.Dir(targetFile),
	}

	got, err := Select(fsys, ec, []string{}, []string{"*"})

	if err != nil {
		t.Fatalf("Select returned error: %v", err)
	}

	want := []string{
		"root/work/sub/c.txt",
	}

	if diff := cmp.Diff(want, got, cmpopts.EquateEmpty()); diff != "" {
		t.Fatalf("selected paths mismatch (-want +got):\n%s", diff)
	}
}

func target(t string) *string {
	return &t
}
