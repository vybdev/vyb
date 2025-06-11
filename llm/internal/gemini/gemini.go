package gemini

// Package gemini provides an abstraction layer over the Google Gemini
// API similar to the llm/internal/openai module. The implementation is
// progressing incrementally – at the moment we expose internal helpers
// to build the JSON request body so the rest of the application can be
// integrated and tested without performing real network calls.

import (
    "bytes"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net/http"
    "os"
)

import "github.com/vybdev/vyb/config"
import "github.com/vybdev/vyb/llm/payload"

// -----------------------------------------------------------------------------
// Public stubs kept until full integration lands
// -----------------------------------------------------------------------------

// ErrNotImplemented is returned by helpers that are still pending
// implementation so callers can gracefully handle the missing feature.
var ErrNotImplemented = errors.New("gemini: not implemented")

// GetWorkspaceChangeProposals mirrors the OpenAI helper and will, once
// implemented, send the conversation to Gemini and unmarshal a
// WorkspaceChangeProposal.
func GetWorkspaceChangeProposals(_ config.ModelFamily, _ config.ModelSize, _ string, _ string) (*payload.WorkspaceChangeProposal, error) {
    return nil, ErrNotImplemented
}

// GetModuleContext will request an internal & public context summary for a
// single module. Not implemented yet.
func GetModuleContext(_ string, _ string) (*payload.ModuleSelfContainedContext, error) {
    return nil, ErrNotImplemented
}

// GetModuleExternalContexts will request external contexts for a set of
// modules. Not implemented yet.
func GetModuleExternalContexts(_ string, _ string) (*payload.ModuleExternalContextResponse, error) {
    return nil, ErrNotImplemented
}

// -----------------------------------------------------------------------------
// Provider-specific data structures & helpers (non-exported)
// -----------------------------------------------------------------------------

// NOTE: baseEndpoint is a var (not const) to allow test overrides.
var baseEndpoint = "https://generativelanguage.googleapis.com/v1beta"

// generateContentTmpl is the relative path (fmt formatted) used to call
// the "generateContent" method on a specific model, e.g.:
//   fmt.Sprintf(generateContentTmpl, "gemini-2.5-flash", apiKey)
const generateContentTmpl = "/models/%s:generateContent?key=%s"

type part struct {
    Text string `json:"text,omitempty"`
}

type content struct {
    Role  string `json:"role,omitempty"`
    Parts []part `json:"parts,omitempty"`
}

type generationConfig struct {
    ResponseMimeType string      `json:"responseMimeType,omitempty"`
    ResponseSchema   interface{} `json:"responseSchema,omitempty"`
}

type requestPayload struct {
    Contents         []content        `json:"contents"`
    GenerationConfig generationConfig `json:"generationConfig"`
}

// geminiResponse mirrors the minimal subset of the response envelope we
// care about. The actual schema will be expanded once streaming/network
// wiring is added.
//
// { "candidates": [ { "content": {"parts": [ {"text": "..."} ] } } ] }

type geminiResponse struct {
    Candidates []struct {
        Content struct {
            Parts []struct {
                Text string `json:"text"`
            } `json:"parts"`
        } `json:"content"`
    } `json:"candidates"`
}

// geminiErrorResponse captures error payloads returned by Gemini.
// Example:
// {
//   "error": {
//     "code": 400,
//     "message": "...",
//     "status": "INVALID_ARGUMENT"
//   }
// }

type geminiErrorResponse struct {
    Err struct {
        Code    int    `json:"code"`
        Message string `json:"message"`
        Status  string `json:"status"`
    } `json:"error"`
}

func (e geminiErrorResponse) Error() string {
    return fmt.Sprintf("Gemini API error (%d %s): %s", e.Err.Code, e.Err.Status, e.Err.Message)
}

// buildRequest constructs the request body expected by the Gemini
// generateContent endpoint given the system & user messages and the
// JSON schema that should be enforced in the response.
//
// The function is internal to the package but kept separate to allow
// focused unit-testing without touching network code.
func buildRequest(systemMessage, userMessage string, schema interface{}) ([]byte, error) {
    if userMessage == "" {
        return nil, errors.New("gemini: user message must not be empty")
    }

    var msgs []content
    if systemMessage != "" {
        msgs = append(msgs, content{
            Role:  "system",
            Parts: []part{{Text: systemMessage}},
        })
    }
    msgs = append(msgs, content{
        Role:  "user",
        Parts: []part{{Text: userMessage}},
    })

    payload := requestPayload{
        Contents: msgs,
        GenerationConfig: generationConfig{
            ResponseMimeType: "application/json",
            ResponseSchema:   schema,
        },
    }

    return json.Marshal(payload)
}

// callGemini performs a synchronous REST call to the Gemini
// generateContent endpoint and returns the decoded response.
//
// The helper validates required environment variables, handles non-200
// responses and unmarshals the successful JSON body into geminiResponse.
func callGemini(systemMessage, userMessage string, schema interface{}, model string) (*geminiResponse, error) {
    apiKey := os.Getenv("GEMINI_API_KEY")
    if apiKey == "" {
        return nil, errors.New("GEMINI_API_KEY is not set")
    }

    if model == "" {
        return nil, errors.New("gemini: model must not be empty")
    }

    // Build request body.
    bodyBytes, err := buildRequest(systemMessage, userMessage, schema)
    if err != nil {
        return nil, err
    }

    // Compose endpoint URL.
    url := fmt.Sprintf("%s"+generateContentTmpl, baseEndpoint, model, apiKey)

    req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
    if err != nil {
        return nil, fmt.Errorf("gemini: failed to create request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("gemini: request failed: %w", err)
    }
    defer resp.Body.Close()

    respBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("gemini: failed to read response body: %w", err)
    }

    if resp.StatusCode != http.StatusOK {
        // Try to decode structured error first.
        var gErr geminiErrorResponse
        if jsonErr := json.Unmarshal(respBytes, &gErr); jsonErr == nil && gErr.Err.Message != "" {
            return nil, gErr
        }
        return nil, fmt.Errorf("gemini: http %d – %s", resp.StatusCode, string(respBytes))
    }

    var out geminiResponse
    if err := json.Unmarshal(respBytes, &out); err != nil {
        return nil, fmt.Errorf("gemini: failed to unmarshal response: %w", err)
    }

    return &out, nil
}
