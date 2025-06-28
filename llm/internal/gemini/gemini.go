package gemini

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/vybdev/vyb/config"
	"github.com/vybdev/vyb/llm/internal/gemini/internal/schema"
	"github.com/vybdev/vyb/llm/payload"
	"io"
	"net/http"
	"os"
	"strings"
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
	userMessage, err := serializeWorkspaceChangeRequest(request)
	if err != nil {
		return nil, fmt.Errorf("gemini: failed to serialize workspace change request: %w", err)
	}
	model, err := mapModel(fam, sz)
	if err != nil {
		return nil, err
	}

	if os.Getenv("GEMINI_API_KEY") == "" {
		return nil, errors.New("GEMINI_API_KEY is not set")
	}

	resp, err := callGemini([]string{systemMessage, userMessage}, schema.GetWorkspaceChangeProposalSchema(), model)
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
	userMessage, err := serializeModuleContextRequest(request)
	if err != nil {
		return nil, fmt.Errorf("gemini: failed to serialize module context request: %w", err)
	}
	model, err := mapModel(config.ModelFamilyReasoning, config.ModelSizeSmall)
	if err != nil {
		return nil, err
	}

	resp, err := callGemini([]string{systemMessage, userMessage}, schema.GetModuleContextSchema(), model)
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
	userMessage, err := serializeExternalContextsRequest(request)
	if err != nil {
		return nil, fmt.Errorf("gemini: failed to serialize external contexts request: %w", err)
	}
	model, err := mapModel(config.ModelFamilyReasoning, config.ModelSizeSmall)
	if err != nil {
		return nil, err
	}

	resp, err := callGemini([]string{systemMessage, userMessage}, schema.GetModuleExternalContextSchema(), model)
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
//
//	Request Serializers
//
// -----------------------------------------------------------------------------

func serializeWorkspaceChangeRequest(request *payload.WorkspaceChangeRequest) (string, error) {
	if request == nil {
		return "", fmt.Errorf("WorkspaceChangeRequest must not be nil")
	}
	if request.TargetModule == "" {
		return "", fmt.Errorf("TargetModule is required")
	}
	if request.TargetDirectory == "" {
		return "", fmt.Errorf("TargetDirectory is required")
	}

	var sb strings.Builder

	// Write target module information (these are now required)
	sb.WriteString(fmt.Sprintf("# Target Module: `%s`\n", request.TargetModule))
	sb.WriteString("## Target Module Context\n")
	sb.WriteString(fmt.Sprintf("%s\n\n", request.TargetModuleContext))
	sb.WriteString(fmt.Sprintf("## Target Directory: `%s`\n\n", request.TargetDirectory))

	// Write parent module contexts
	if len(request.ParentModuleContexts) > 0 {
		sb.WriteString("# Parent Module Contexts\n")
		for _, mc := range request.ParentModuleContexts {
			ctx := &payload.ModuleSelfContainedContext{
				Name:          mc.Name,
				PublicContext: mc.Content,
			}
			writeModule(&sb, mc.Name, ctx)
		}
		sb.WriteString("\n")
	}

	// Write sub-module contexts
	if len(request.SubModuleContexts) > 0 {
		sb.WriteString("# Sub-Module Contexts\n")
		for _, mc := range request.SubModuleContexts {
			ctx := &payload.ModuleSelfContainedContext{
				Name:          mc.Name,
				PublicContext: mc.Content,
			}
			writeModule(&sb, mc.Name, ctx)
		}
		sb.WriteString("\n")
	}

	// Write files
	if len(request.Files) > 0 {
		sb.WriteString("# Files\n")
		for _, f := range request.Files {
			writeFile(&sb, f.Path, f.Content)
		}
	}

	return sb.String(), nil
}

func serializeModuleContextRequest(request *payload.ModuleContextRequest) (string, error) {
	if request == nil {
		return "", fmt.Errorf("ModuleContextRequest must not be nil")
	}

	var sb strings.Builder
	rootPrefix := request.TargetModuleName

	// Only spend these tokens if we need to teach the LLM that a directory != module.
	if len(request.TargetModuleDirectories) > 1 {
		sb.WriteString(fmt.Sprintf("## Directories in module `%s`\n", rootPrefix))
		sb.WriteString(fmt.Sprintf("The following is a list of directories that are part of the module `%s`\n.", rootPrefix))
		sb.WriteString(fmt.Sprintf("These ARE NOT MODULES, they are directories within the module. When summarizing their file contents, include them in the summary of `%s`, do not make up modules for them.\n", rootPrefix))
		for _, dir := range request.TargetModuleDirectories {
			sb.WriteString(fmt.Sprintf("- %s\n", dir))
		}
	}

	sb.WriteString(fmt.Sprintf("## Files in module `%s`\n", rootPrefix))
	// Emit root-module files.
	for _, file := range request.TargetModuleFiles {
		writeFile(&sb, file.Path, file.Content)
	}

	// Emit public context of immediate sub-modules.
	for _, sub := range request.SubModulesPublicContexts {
		// We only expose the public context of immediate sub-modules.
		if sub.Content == "" && sub.Name == "" {
			continue
		}

		trimmedCtx := &payload.ModuleSelfContainedContext{
			Name:          sub.Name,
			PublicContext: sub.Content,
		}
		writeModule(&sb, trimmedCtx.Name, trimmedCtx)
	}

	return sb.String(), nil
}

func serializeExternalContextsRequest(request *payload.ExternalContextsRequest) (string, error) {
	if request == nil {
		return "", fmt.Errorf("ExternalContextsRequest must not be nil")
	}

	var sb strings.Builder

	// Write each module with H1 headers
	for _, module := range request.Modules {
		if module.Name == "" {
			continue
		}
		sb.WriteString(fmt.Sprintf("# Module: `%s`\n", module.Name))
		if module.ParentName != "" {
			sb.WriteString(fmt.Sprintf("Parent Module: `%s`\n\n", module.ParentName))
		}
		if module.InternalContext != "" {
			sb.WriteString("## Internal Context\n")
			sb.WriteString(fmt.Sprintf("%s\n\n", module.InternalContext))
		}
		if module.PublicContext != "" {
			sb.WriteString("## Public Context\n")
			sb.WriteString(fmt.Sprintf("%s\n\n", module.PublicContext))
		}
	}

	return sb.String(), nil
}

func writeModule(sb *strings.Builder, path string, context *payload.ModuleSelfContainedContext) {
	if sb == nil {
		return
	}
	if path == "" && (context == nil || (context.ExternalContext == "" && context.InternalContext == "" && context.PublicContext == "")) {
		return
	}
	sb.WriteString(fmt.Sprintf("# Module: `%s`\n", path))
	if context != nil {
		if context.ExternalContext != "" {
			sb.WriteString("## External Context\n")
			sb.WriteString(fmt.Sprintf("%s\n", context.ExternalContext))
		}
		if context.InternalContext != "" {
			sb.WriteString("## Internal Context\n")
			sb.WriteString(fmt.Sprintf("%s\n", context.InternalContext))
		}
		if context.PublicContext != "" {
			sb.WriteString("## Public Context\n")
			sb.WriteString(fmt.Sprintf("%s\n", context.PublicContext))
		}
	}
}

func writeFile(sb *strings.Builder, filepath, content string) {
	if sb == nil {
		return
	}
	lang := getLanguageFromFilename(filepath)
	sb.WriteString(fmt.Sprintf("### %s\n", filepath))
	sb.WriteString(fmt.Sprintf("```%s\n", lang))
	sb.WriteString(content)
	// Ensure a trailing newline before closing the code block.
	if !strings.HasSuffix(content, "\n") {
		sb.WriteString("\n")
	}
	sb.WriteString("```\n\n")
}

// getLanguageFromFilename returns a language identifier based on file extension.
func getLanguageFromFilename(filename string) string {
	if strings.HasSuffix(filename, ".go") {
		return "go"
	} else if strings.HasSuffix(filename, ".md") {
		return "markdown"
	} else if strings.HasSuffix(filename, ".json") {
		return "json"
	} else if strings.HasSuffix(filename, ".txt") {
		return "text"
	}
	// Default: no language specified.
	return ""
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

func buildRequest(messages []string, schema interface{}) ([]byte, error) {
	if len(messages) == 0 {
		return nil, errors.New("gemini: messages cannot be empty")
	}

	// Create a part for each message
	var parts []part
	for _, msg := range messages {
		if msg != "" {
			parts = append(parts, part{Text: msg})
		}
	}

	if len(parts) == 0 {
		return nil, errors.New("gemini: all messages are empty")
	}

	r := requestPayload{
		Contents: []content{
			{
				Role:  "user",
				Parts: parts,
			},
		},
		GenerationConfig: generationConfig{
			ResponseMimeType: "application/json",
			ResponseSchema:   schema,
		},
	}

	return json.Marshal(r)
}

func callGemini(messages []string, schema interface{}, model string) (*geminiResponse, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, errors.New("GEMINI_API_KEY is not set")
	}

	if model == "" {
		return nil, errors.New("gemini: model must not be empty")
	}

	// Build request body.
	bodyBytes, err := buildRequest(messages, schema)
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
