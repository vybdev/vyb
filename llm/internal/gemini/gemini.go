package gemini

// Package gemini provides an abstraction layer over the Google Gemini
// API similar to the llm/internal/openai module. The implementation is
// progressing incrementally – at the moment we expose internal helpers
// to build the JSON request body so the rest of the application can be
// integrated and tested without performing real network calls.

import (
    "encoding/json"
    "errors"
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

// baseEndpoint is the common prefix for every Gemini REST call.
const baseEndpoint = "https://generativelanguage.googleapis.com/v1beta"

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
