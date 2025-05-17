package payload

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestBuildUserMessage(t *testing.T) {
	// Create a MapFS with two files.
	testFS := fstest.MapFS{
		"file1.md": &fstest.MapFile{
			Data: []byte("Hello, markdown!"),
		},
		"file2.go": &fstest.MapFile{
			Data: []byte("package main\n\nfunc main() {}"),
		},
	}

	// Define the file paths in the order to be processed.
	filePaths := []string{"file1.md", "file2.go"}

	// Call BuildUserMessage.
	result, err := BuildUserMessage(testFS, filePaths)
	if err != nil {
		t.Fatalf("BuildUserMessage returned an error: %v", err)
	}

	// Manually construct the expected markdown output.
	// For file1.md, language is "markdown" and content is appended with a trailing newline.
	// For file2.go, language is "go" and since the content does not end with a newline, one is appended.
	expected := strings.Join([]string{
		"# file1.md",
		"```markdown",
		"Hello, markdown!",
		"```",
		"",
		"# file2.go",
		"```go",
		"package main",
		"",
		"func main() {}",
		"```",
		"",
		"",
	}, "\n")

	// Compare the generated markdown with the expected output.
	if result != expected {
		t.Errorf("Unexpected markdown output:\nGot:\n%s\nExpected:\n%s", result, expected)
	}
}
