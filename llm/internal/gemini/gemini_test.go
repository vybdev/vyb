package gemini

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "os"
    "reflect"
    "strings"
    "testing"

    "github.com/vybdev/vyb/config"
    "github.com/vybdev/vyb/llm/payload"
)

func TestGetWorkspaceChangeProposals(t *testing.T) {
    // ------------------------------------------------------------------
    // 1. Prepare fake Gemini server.
    // ------------------------------------------------------------------
    var capturedReq struct {
        Method string
        Path   string
        Body   map[string]any
    }

    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        capturedReq.Method = r.Method
        capturedReq.Path = r.URL.Path + "?" + r.URL.RawQuery
        dec := json.NewDecoder(r.Body)
        if err := dec.Decode(&capturedReq.Body); err != nil {
            t.Fatalf("failed decoding body: %v", err)
        }
        // Craft minimal valid Gemini response.
        resp := map[string]any{
            "candidates": []any{
                map[string]any{
                    "content": map[string]any{
                        "parts": []any{
                            map[string]any{
                                "text": `{"proposals":[],"summary":"sum","description":"desc"}`,
                            },
                        },
                    },
                },
            },
        }
        _ = json.NewEncoder(w).Encode(resp)
    }))
    defer srv.Close()

    // Override package-level endpoint and restore afterwards.
    oldBase := baseEndpoint
    baseEndpoint = srv.URL
    defer func() { baseEndpoint = oldBase }()

    // Ensure API key so helper doesn’t abort early.
    os.Setenv("GEMINI_API_KEY", "testkey")
    defer os.Unsetenv("GEMINI_API_KEY")

    // ------------------------------------------------------------------
    // 2. Call helper under test.
    // ------------------------------------------------------------------
    got, err := GetWorkspaceChangeProposals(config.ModelFamilyGPT, config.ModelSizeSmall, "sys", "usr")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    want := &payload.WorkspaceChangeProposal{Summary: "sum", Description: "desc", Proposals: []payload.FileChangeProposal{}}
    if !reflect.DeepEqual(got, want) {
        t.Fatalf("unexpected proposal: %+v", got)
    }

    // ------------------------------------------------------------------
    // 3. Validate request basics.
    // ------------------------------------------------------------------
    if capturedReq.Method != http.MethodPost {
        t.Fatalf("expected POST, got %s", capturedReq.Method)
    }
    if !strings.Contains(capturedReq.Path, "/models/gemini-2.5-flash-preview-05-20:generateContent") {
        t.Fatalf("unexpected request path %s", capturedReq.Path)
    }

    // Check presence of system & user parts.
    contents, ok := capturedReq.Body["contents"].([]any)
    if !ok || len(contents) != 2 {
        t.Fatalf("request body missing contents array: %#v", capturedReq.Body)
    }
 }
