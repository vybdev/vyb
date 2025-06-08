package template

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/dangazineu/vyb/workspace/context"
	"github.com/dangazineu/vyb/workspace/project"
)

func Test_buildExtendedUserMessage(t *testing.T) {
	ann := func(s string) *project.Annotation {
		return &project.Annotation{
			PublicContext:   fmt.Sprintf("%s public", s),
			ExternalContext: fmt.Sprintf("%s external", s),
			InternalContext: fmt.Sprintf("%s internal", s),
		}
	}
	// Build minimal module tree: root -> work (w) -> tgt (w/child)
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

	msg, err := buildExtendedUserMessage(mfs, meta, ec, []string{"w/mid/child/file.txt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Basic assertions – ensure expected contexts are present.
	mustContain := []string{"W external", "Mid internal", "Sibling public", "Cousin public", "hello"}
	for _, s := range mustContain {
		if !strings.Contains(msg, s) {
			t.Fatalf("expected message to contain %q", s)
		}
	}

	mustNotContain := []string{
		"W public", "W internal",
		"Mid public", "Mid external",
		"Sibling internal", "Sibling external",
		"Cousin internal", "Cousin external",
		"Out public", "Out internal", "Out external",
		"mid content", "sibling content", "w content", "cousin content", "out content",
	}

	for _, s := range mustNotContain {
		if strings.Contains(msg, s) {
			t.Fatalf("should not include contexts for target module itself, got message:\n%s", msg)
		}
	}
}
