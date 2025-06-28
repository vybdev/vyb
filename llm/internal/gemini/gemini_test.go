package gemini

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/vybdev/vyb/config"
	"github.com/vybdev/vyb/llm/payload"
)

func TestGetWorkspaceChangeProposals(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{
								"text": `{"summary":"s","description":"d","proposals":[]}`,
							},
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldBase := baseEndpoint
	baseEndpoint = srv.URL
	defer func() { baseEndpoint = oldBase }()

	os.Setenv("GEMINI_API_KEY", "x")
	defer os.Unsetenv("GEMINI_API_KEY")

	req := &payload.WorkspaceChangeRequest{
		TargetModule: "test-module",
		TargetModuleContext: "Test module context",
		TargetDirectory: "src/",
		Files: []payload.FileContent{
			{Path: "test.go", Content: "package main"},
		},
	}
	got, err := GetWorkspaceChangeProposals(config.ModelFamilyGPT, config.ModelSizeSmall, "sys", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := &payload.WorkspaceChangeProposal{Summary: "s", Description: "d", Proposals: []payload.FileChangeProposal{}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected proposal: got %+v, want %+v", got, want)
	}
}

func TestGetModuleContext(t *testing.T) {
	// Dummy server returning minimal module context JSON.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{
								"text": `{"internal_context":"i","public_context":"p"}`,
							},
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldBase := baseEndpoint
	baseEndpoint = srv.URL
	defer func() { baseEndpoint = oldBase }()

	os.Setenv("GEMINI_API_KEY", "x")
	defer os.Unsetenv("GEMINI_API_KEY")

	req := &payload.ModuleContextRequest{
		TargetModuleName: "test-module",
	}

	got, err := GetModuleContext("sys", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := &payload.ModuleSelfContainedContext{InternalContext: "i", PublicContext: "p"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected ctx: %+v", got)
	}
}

func TestGetModuleExternalContexts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{
								"text": `{"modules":[{"name":"foo","external_context":"bar"}]}`,
							},
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	oldBase := baseEndpoint
	baseEndpoint = srv.URL
	defer func() { baseEndpoint = oldBase }()

	os.Setenv("GEMINI_API_KEY", "x")
	defer os.Unsetenv("GEMINI_API_KEY")

	req := &payload.ExternalContextsRequest{
		Modules: []payload.ModuleInfoForExternalContext{
			{Name: "foo"},
		},
	}

	got, err := GetModuleExternalContexts("sys", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := &payload.ModuleExternalContextResponse{Modules: []payload.ModuleExternalContext{{Name: "foo", ExternalContext: "bar"}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected ext ctx: %+v", got)
	}
}
