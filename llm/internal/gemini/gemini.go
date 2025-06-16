package gemini

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/vybdev/vyb/config"
	gemschema "github.com/vybdev/vyb/llm/internal/gemini/internal/schema"
	"github.com/vybdev/vyb/llm/payload"
	"io"
	"net/http"
	"os"
)

// mapModel converts the (family,size) tuple into the concrete Gemini
// model identifier expected by the REST endpoint.
func mapModel(fam config.ModelFamily, sz config.ModelSize) (string, error) {
	// The same resolution logic lives also inside llm/dispatcher for the
	// compile-time tests that exercise dispatch mapping. Keep both in
	// sync until the refactor that centralises it lands.
	switch sz {
	case config.ModelSizeSmall:
		return "gemini-2.5-flash-preview-05-20", nil
	case config.ModelSizeLarge:
		return "gemini-2.5-pro-preview-06-05", nil
	default:
		return "", fmt.Errorf("gemini: unsupported model size %s", sz)
	}
}

// GetWorkspaceChangeProposals composes the request, sends it to Gemini and
// converts the response into a strongly-typed WorkspaceChangeProposal.
//
// The function mirrors the public surface exposed by the OpenAI provider so
// callers can remain provider-agnostic.
func GetWorkspaceChangeProposals(fam config.ModelFamily, sz config.ModelSize, systemMessage string, request *payload.WorkspaceChangeRequest) (*payload.WorkspaceChangeProposal, error) {
	_ = request // TODO(vyb): serialize request payload
	model, err := mapModel(fam, sz)
	if err != nil {
		return nil, err
	}

	if os.Getenv("GEMINI_API_KEY") == "" {
		return nil, errors.New("GEMINI_API_KEY is not set")
	}

	schema := gemschema.GetWorkspaceChangeProposalSchema()

	userMessage := "placeholder"
	resp, err := callGemini(systemMessage, userMessage, schema, model)
	if err != nil {
		return nil, err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("gemini: empty response")
	}

	raw := resp.Candidates[0].Content.Parts[0].Text

	var proposal payload.WorkspaceChangeProposal
	if err := json.Unmarshal([]byte(raw), &proposal); err != nil {
		return nil, fmt.Errorf("gemini: failed to unmarshal WorkspaceChangeProposal: %w", err)
	}
	return &proposal, nil
}

func GetModuleContext(systemMessage string, request *payload.ModuleContextRequest) (*payload.ModuleSelfContainedContext, error) {
	_ = request // TODO(vyb): serialize request payload
	model, err := mapModel(config.ModelFamilyReasoning, config.ModelSizeSmall)
	if err != nil {
		return nil, err
	}

	schema := gemschema.GetModuleContextSchema()

	userMessage := "placeholder"
	resp, err := callGemini(systemMessage, userMessage, schema, model)
	if err != nil {
		return nil, err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("gemini: empty response")
	}

	raw := resp.Candidates[0].Content.Parts[0].Text

	var ctx payload.ModuleSelfContainedContext
	if err := json.Unmarshal([]byte(raw), &ctx); err != nil {
		return nil, fmt.Errorf("gemini: failed to unmarshal ModuleSelfContainedContext: %w", err)
	}
	return &ctx, nil
}

func GetModuleExternalContexts(systemMessage string, request *payload.ExternalContextsRequest) (*payload.ModuleExternalContextResponse, error) {
	_ = request // TODO(vyb): serialize request payload
	model, err := mapModel(config.ModelFamilyReasoning, config.ModelSizeSmall)
	if err != nil {
		return nil, err
	}

	schema := gemschema.GetModuleExternalContextSchema()

	userMessage := "placeholder"
	resp, err := callGemini(systemMessage, userMessage, schema, model)
	if err != nil {
		return nil, err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("gemini: empty response")
	}

	raw := resp.Candidates[0].Content.Parts[0].Text

	var ext payload.ModuleExternalContextResponse
	if err := json.Unmarshal([]byte(raw), &ext); err != nil {
		return nil, fmt.Errorf("gemini: failed to unmarshal ModuleExternalContextResponse: %w", err)
	}
	return &ext, nil
}

// -----------------------------------------------------------------------------
// Provider-specific data structures & helpers (non-exported)
// -----------------------------------------------------------------------------

// NOTE: baseEndpoint is a var (not const) to allow test overrides.
var baseEndpoint = "https://generativelanguage.googleapis.com/v1beta"

// generateContentTmpl is the relative path (fmt formatted) used to call
// the "generateContent" method on a specific model, e.g.:
//
//	fmt.Sprintf(generateContentTmpl, "gemini-2.5-flash", apiKey)
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
// care about.
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

func buildRequest(systemMessage, userMessage string, schema interface{}) ([]byte, error) {
	if userMessage == "" {
		return nil, errors.New("gemini: user message must not be empty")
	}

	r := requestPayload{
		Contents: []content{
			{
				Role:  "user",
				Parts: []part{{Text: systemMessage + "\n\n" + userMessage}},
			},
		},
		GenerationConfig: generationConfig{
			ResponseMimeType: "application/json",
			ResponseSchema:   schema,
		},
	}

	return json.Marshal(r)
}

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

	// ---------------------------------------------------------------------
	// Persist request/response pair for debugging – same approach as OpenAI.
	// ---------------------------------------------------------------------
	logEntry := struct {
		Request  json.RawMessage `json:"request"`
		Response json.RawMessage `json:"response"`
	}{
		Request:  bodyBytes,
		Response: respBytes,
	}

	if logBytes, err := json.MarshalIndent(logEntry, "", "  "); err == nil {
		if f, err := os.CreateTemp("", "vyb-gemini-*.json"); err == nil {
			if _, wErr := f.Write(logBytes); wErr == nil {
				_ = f.Close()
			}
		}
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
