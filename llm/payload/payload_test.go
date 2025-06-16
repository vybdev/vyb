package payload

import (
	"encoding/json"
	"reflect"
	"testing"
	"testing/fstest"
)

func context(name string) *ModuleSelfContainedContext  { return &ModuleSelfContainedContext{Name: name} }
func pcontext(name string) *ModuleSelfContainedContext { return context(name) }

func TestBuildModuleContextUserMessage(t *testing.T) {
	// Files arranged in a nested module hierarchy:
	//   - root.txt (root module / no module name)
	//   - moduleA/a.go
	//   - moduleA/subB/b.md
	mfs := fstest.MapFS{
		"root.txt":          &fstest.MapFile{Data: []byte("root")},
		"moduleA/a.go":      &fstest.MapFile{Data: []byte("package foo\n")},
		"moduleA/subB/b.md": &fstest.MapFile{Data: []byte("Markdown content")},
	}

	// Construct the ModuleSelfContainedContextRequest tree that mirrors the hierarchy.
	req := &ModuleSelfContainedContextRequest{
		ModuleCtx: &ModuleSelfContainedContext{Name: "."},
		FilePaths: []string{"root.txt"},
		SubModules: []*ModuleSelfContainedContextRequest{
			{
				ModuleCtx: &ModuleSelfContainedContext{Name: "moduleA", PublicContext: "moduleA public"},
				FilePaths: []string{"moduleA/a.go"},
				SubModules: []*ModuleSelfContainedContextRequest{
					{
						ModuleCtx: &ModuleSelfContainedContext{Name: "moduleA/subB", PublicContext: "subB public"},
						FilePaths: []string{"moduleA/subB/b.md"},
					},
				},
			},
		},
	}

	got, err := BuildModuleContextUserMessage(mfs, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "# Module: `.`\n## Files in module `.`\n" +
		"### root.txt\n```text\nroot\n```\n\n" +
		"# Module: `moduleA`\n## Public Context\nmoduleA public\n"

	if got != expected {
		t.Errorf("payload mismatch.\nGot:\n%s\nExpected:\n%s", got, expected)
	}
}

func TestBuildModuleContextUserMessage_FileNotFound(t *testing.T) {
	// Empty filesystem – any file access should fail.
	mfs := fstest.MapFS{}

	req := &ModuleSelfContainedContextRequest{
		FilePaths: []string{"does_not_exist.txt"},
	}

	if _, err := BuildModuleContextUserMessage(mfs, req); err == nil {
		t.Fatalf("expected error for missing file, got nil")
	}
}

// New test validating selective context inclusion semantics.
func TestBuildModuleContextUserMessage_Selectivity(t *testing.T) {
	/*
	   Module hierarchy used in this test:
	       A (root)
	         ├── B
	         └── C
	             └── D
	*/
	mfs := fstest.MapFS{
		"A/a.go":   &fstest.MapFile{Data: []byte("package a\n")},
		"A/C/c.go": &fstest.MapFile{Data: []byte("package c\n")},
	}

	// Full tree rooted at A.
	treeA := &ModuleSelfContainedContextRequest{
		ModuleCtx: &ModuleSelfContainedContext{Name: "A"},
		FilePaths: []string{"A/a.go"},
		SubModules: []*ModuleSelfContainedContextRequest{
			{
				ModuleCtx: &ModuleSelfContainedContext{Name: "A/B", PublicContext: "This is B"},
			},
			{
				ModuleCtx: &ModuleSelfContainedContext{Name: "A/C", PublicContext: "This is C", InternalContext: "This is C's internal context."},
				FilePaths: []string{"A/C/c.go"},
				SubModules: []*ModuleSelfContainedContextRequest{{
					ModuleCtx: &ModuleSelfContainedContext{Name: "A/C/D", PublicContext: "This is D. It won't be included."},
				}},
			},
		},
	}

	gotA, err := BuildModuleContextUserMessage(mfs, treeA)
	if err != nil {
		t.Fatalf("unexpected error building payload for A: %v", err)
	}

	expectedA := "# Module: `A`\n## Files in module `A`\n" +
		"### A/a.go\n```go\npackage a\n```\n\n" +
		"# Module: `A/B`\n## Public Context\nThis is B\n" +
		"# Module: `A/C`\n## Public Context\nThis is C\n"

	if gotA != expectedA {
		t.Errorf("payload for A mismatch.\nGot:\n%s\nExpected:\n%s", gotA, expectedA)
	}

	// Sub-tree rooted at C.
	treeC := &ModuleSelfContainedContextRequest{
		ModuleCtx: &ModuleSelfContainedContext{Name: "A/C"},
		FilePaths: []string{"A/C/c.go"},
		SubModules: []*ModuleSelfContainedContextRequest{{
			ModuleCtx: &ModuleSelfContainedContext{Name: "A/C/D", PublicContext: "This is D"},
		}},
	}

	gotC, err := BuildModuleContextUserMessage(mfs, treeC)
	if err != nil {
		t.Fatalf("unexpected error building payload for C: %v", err)
	}

	expectedC := "# Module: `A/C`\n## Files in module `A/C`\n" +
		"### A/C/c.go\n```go\npackage c\n```\n\n" +
		"# Module: `A/C/D`\n## Public Context\nThis is D\n"

	if gotC != expectedC {
		t.Errorf("payload for C mismatch.\nGot:\n%s\nExpected:\n%s", gotC, expectedC)
	}
}

func TestRequestPayloads_JSONMarshalling(t *testing.T) {
	testcases := []struct {
		name    string
		payload interface{}
		newInst func() interface{}
	}{
		{
			name: "WorkspaceChangeRequest",
			payload: &WorkspaceChangeRequest{
				ModuleContexts: []ModuleContext{
					{Name: "mod1", Type: "External", Content: "ctx1"},
				},
				Files: []FileContent{
					{Path: "file1.go", Content: "package main"},
				},
			},
			newInst: func() interface{} { return &WorkspaceChangeRequest{} },
		},
		{
			name: "ModuleContextRequest",
			payload: &ModuleContextRequest{
				TargetModuleFiles: []FileContent{
					{Path: "file1.go", Content: "package main"},
				},
				TargetModuleDirectories: []string{"dir1"},
				SubModulesPublicContexts: []SubModuleContext{
					{Name: "sub1", Context: "pub_ctx"},
				},
			},
			newInst: func() interface{} { return &ModuleContextRequest{} },
		},
		{
			name: "ExternalContextsRequest",
			payload: &ExternalContextsRequest{
				Modules: []ModuleInfoForExternalContext{
					{
						Name:            "mod1",
						ParentName:      "",
						InternalContext: "int_ctx",
						PublicContext:   "pub_ctx",
					},
				},
			},
			newInst: func() interface{} { return &ExternalContextsRequest{} },
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.payload)
			if err != nil {
				t.Fatalf("json.Marshal() failed: %v", err)
			}

			unmarshaled := tc.newInst()
			if err := json.Unmarshal(data, unmarshaled); err != nil {
				t.Fatalf("json.Unmarshal() failed: %v", err)
			}

			if !reflect.DeepEqual(tc.payload, unmarshaled) {
				t.Errorf("round-trip mismatch.\nGot:  %#v\nWant: %#v", unmarshaled, tc.payload)
			}
		})
	}
}
