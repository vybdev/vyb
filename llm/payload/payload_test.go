package payload

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestRequestPayloads_JSONMarshalling(t *testing.T) {
	testcases := []struct {
		name    string
		payload interface{}
		newInst func() interface{}
	}{
		{
			name: "WorkspaceChangeRequest",
			payload: &WorkspaceChangeRequest{
				TargetModule: "my-module",
				TargetModuleContext: "context info",
				TargetDirectory: "src/",
				ParentModuleContexts: []ModuleContext{
					{Name: "parent1", Content: "parent context"},
				},
				SubModuleContexts: []ModuleContext{
					{Name: "sub1", Content: "sub context"},
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
				TargetModuleName: "my-module",
				TargetModuleFiles: []FileContent{
					{Path: "file1.go", Content: "package main"},
				},
				TargetModuleDirectories: []string{"dir1"},
				SubModulesPublicContexts: []ModuleContext{
					{Name: "sub1", Content: "pub_ctx"},
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