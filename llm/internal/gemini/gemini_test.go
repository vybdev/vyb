package gemini

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/vybdev/vyb/llm/payload"
)

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

	got, err := GetModuleContext("sys", "usr")
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

	got, err := GetModuleExternalContexts("sys", "usr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := &payload.ModuleExternalContextResponse{Modules: []payload.ModuleExternalContext{{Name: "foo", ExternalContext: "bar"}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected ext ctx: %+v", got)
	}
}
