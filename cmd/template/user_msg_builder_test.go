package template

import (
	"fmt"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/vybdev/vyb/llm/payload"
	"github.com/vybdev/vyb/workspace/context"
	"github.com/vybdev/vyb/workspace/project"
)

func Test_buildExtendedUserMessage(t *testing.T) {
	ann := func(s string) *project.Annotation {
		return &project.Annotation{
			PublicContext:   fmt.Sprintf("%s public", s),
			ExternalContext: fmt.Sprintf("%s external", s),
			InternalContext: fmt.Sprintf("%s internal", s),
		}
	}
	// Build minimal module tree: root -> work (w) -> mid -> tgt (w/child)
	root := &project.Module{Name: "."}
	work := &project.Module{Name: "w", Parent: root, Annotation: ann("W")}
	mid := &project.Module{Name: "w/mid", Parent: work, Annotation: ann("Mid")}
	tgt := &project.Module{Name: "w/mid/child", Parent: mid, Annotation: ann("Target")}
	sib := &project.Module{Name: "w/mid/sibling", Parent: mid, Annotation: ann("Sibling")}
	cous := &project.Module{Name: "w/cousin", Parent: work, Annotation: ann("Cousin")}
	out := &project.Module{Name: "out", Parent: root, Annotation: ann("Out")}

	work.Modules = []*project.Module{mid, cous}
	mid.Modules = []*project.Module{tgt, sib}
	root.Modules = []*project.Module{work, out}

	meta := &project.Metadata{Modules: root}

	// in-memory fs with one file inside target module.
	mfs := fstest.MapFS{
		"w/mid/child/file.txt":   &fstest.MapFile{Data: []byte("hello")},
		"w/mid/file.txt":         &fstest.MapFile{Data: []byte("mid content")},
		"w/mid/sibling/file.txt": &fstest.MapFile{Data: []byte("sibling content")},
		"w/file.txt":             &fstest.MapFile{Data: []byte("w content")},
		"w/cousin/file.txt":      &fstest.MapFile{Data: []byte("cousin content")},
		"out/file.txt":           &fstest.MapFile{Data: []byte("out content")},
	}

	ec := &context.ExecutionContext{
		ProjectRoot: ".",
		WorkingDir:  "w",
		TargetDir:   "w/mid/child",
	}

	req, err := buildWorkspaceChangeRequest(mfs, meta, ec, []string{"w/mid/child/file.txt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Basic assertions – ensure expected contexts are present.
	expectedFiles := []payload.FileContent{
		{Path: "w/mid/child/file.txt", Content: "hello"},
	}
	if !reflect.DeepEqual(req.Files, expectedFiles) {
		t.Errorf("Files mismatch: got %+v, want %+v", req.Files, expectedFiles)
	}

	// Verify target module information
	if req.TargetModule != "w/mid/child" {
		t.Errorf("TargetModule mismatch: got %q, want %q", req.TargetModule, "w/mid/child")
	}

	if req.TargetDirectory != "w/mid/child" {
		t.Errorf("TargetDirectory mismatch: got %q, want %q", req.TargetDirectory, "w/mid/child")
	}

	expectedParentContexts := []payload.ModuleContext{
		{Name: "w/mid/sibling", Content: "Sibling public"},
		{Name: "w/cousin", Content: "Cousin public"},
	}

	if !reflect.DeepEqual(req.ParentModuleContexts, expectedParentContexts) {
		t.Errorf("ParentModuleContexts mismatch:\ngot:  %+v\nwant: %+v", req.ParentModuleContexts, expectedParentContexts)
	}

	// Should be empty since target module has no sub-modules
	if len(req.SubModuleContexts) != 0 {
		t.Errorf("SubModuleContexts should be empty, got: %+v", req.SubModuleContexts)
	}
}

func Test_buildExtendedUserMessage_nilValidation(t *testing.T) {
	mfs := fstest.MapFS{
		"file.txt": &fstest.MapFile{Data: []byte("content")},
	}

	ec := &context.ExecutionContext{
		ProjectRoot: ".",
		WorkingDir:  ".",
		TargetDir:   ".",
	}

	// Test nil metadata
	_, err := buildWorkspaceChangeRequest(mfs, nil, ec, []string{"file.txt"})
	if err == nil || err.Error() != "metadata cannot be nil" {
		t.Errorf("Expected 'metadata cannot be nil' error, got: %v", err)
	}

	// Test nil modules
	meta := &project.Metadata{Modules: nil}
	_, err = buildWorkspaceChangeRequest(mfs, meta, ec, []string{"file.txt"})
	if err == nil || err.Error() != "metadata.Modules cannot be nil" {
		t.Errorf("Expected 'metadata.Modules cannot be nil' error, got: %v", err)
	}
}
