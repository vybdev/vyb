package openai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/vybdev/vyb/config"
	"github.com/vybdev/vyb/llm/internal/openai/internal/schema"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/vybdev/vyb/llm/payload"
	"time"
)

// message represents a single message in the chat conversation.
type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// request defines the request payload sent to the OpenAI API.
type request struct {
	Model          string         `json:"model"`
	Messages       []message      `json:"messages"`
	ResponseFormat responseFormat `json:"response_format"`
}

type responseFormat struct {
	Type       string                        `json:"type"`
	JSONSchema schema.StructuredOutputSchema `json:"json_schema"`
}

// openaiResponse defines the expected response structure from the OpenAI API.
type openaiResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
}

type openaiErrorResponse struct {
	OpenAIError struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Param   string `json:"param"`
		Code    string `json:"code"`
	} `json:"error"`
}

func (o openaiErrorResponse) Error() string {
	return fmt.Sprintf("OpenAI API error: %s", o.OpenAIError.Message)
}

// -----------------------------------------------------------------------------
//
//	Model resolver
//
// -----------------------------------------------------------------------------
// mapModel converts a generic (family,size) pair into a concrete OpenAI model
// string.  The mapping is local to this provider so business-level code never
// depends on provider-specific identifiers.
func mapModel(fam config.ModelFamily, sz config.ModelSize) (string, error) {
	switch fam {
	case config.ModelFamilyGPT:
		switch sz {
		case config.ModelSizeLarge:
			return "GPT-4.1", nil
		case config.ModelSizeSmall:
			return "GPT-4.1-mini", nil
		}
	case config.ModelFamilyReasoning:
		switch sz {
		case config.ModelSizeLarge:
			return "o3", nil
		case config.ModelSizeSmall:
			return "o4-mini", nil
		}
	}
	return "", fmt.Errorf("openai: unsupported model mapping for family=%s size=%s", fam, sz)
}

// GetModuleContext calls the LLM and returns a parsed ModuleSelfContainedContext
// value using the model derived from family/size.
func GetModuleContext(systemMessage string, request *payload.ModuleContextRequest) (*payload.ModuleSelfContainedContext, error) {
	userMessage, err := serializeModuleContextRequest(request)
	if err != nil {
		return nil, fmt.Errorf("openai: failed to serialize module context request: %w", err)
	}
	model := "o4-mini"
	openaiResp, err := callOpenAI(systemMessage, userMessage, schema.GetModuleContextSchema(), model)
	if err != nil {
		var openAIErrResp openaiErrorResponse
		if errors.As(err, &openAIErrResp) {
			if openAIErrResp.OpenAIError.Code == "rate_limit_exceeded" {
				fmt.Printf("Rate limit exceeded, retrying after 30s\n")
				<-time.After(30 * time.Second)
				return GetModuleContext(systemMessage, request)
			}
		}
		return nil, err
	}
	var ctx payload.ModuleSelfContainedContext
	if err := json.Unmarshal([]byte(openaiResp.Choices[0].Message.Content), &ctx); err != nil {
		return nil, err
	}
	return &ctx, nil
}

// GetWorkspaceChangeProposals sends the given messages to the OpenAI API and
// returns the structured workspace change proposal.
func GetWorkspaceChangeProposals(fam config.ModelFamily, sz config.ModelSize, systemMessage string, request *payload.WorkspaceChangeRequest) (*payload.WorkspaceChangeProposal, error) {
	userMessage, err := serializeWorkspaceChangeRequest(request)
	if err != nil {
		return nil, fmt.Errorf("openai: failed to serialize workspace change request: %w", err)
	}
	model, err := mapModel(fam, sz)
	if err != nil {
		return nil, err
	}

	openaiResp, err := callOpenAI(systemMessage, userMessage, schema.GetWorkspaceChangeProposalSchema(), model)
	if err != nil {
		return nil, err
	}

	var proposal payload.WorkspaceChangeProposal
	if err := json.Unmarshal([]byte(openaiResp.Choices[0].Message.Content), &proposal); err != nil {
		return nil, err
	}
	return &proposal, nil
}

// NOTE: baseEndpoint is a var (not const) to allow test overrides.
var baseEndpoint = "https://api.openai.com/v1/chat/completions"

// callOpenAI sends a request to OpenAI, returns the parsed response, and logs
// the request/response pair to a uniquely-named JSON file in the OS temp dir.
func callOpenAI(systemMessage, userMessage string, structuredOutput schema.StructuredOutputSchema, model string) (*openaiResponse, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, errors.New("OPENAI_API_KEY is not set")
	}

	// Construct request payload.
	reqPayload := request{
		Model: model,
		Messages: []message{
			{
				Role:    "system",
				Content: systemMessage,
			},
			{
				Role:    "user",
				Content: userMessage,
			},
		},
		ResponseFormat: responseFormat{
			Type:       "json_schema",
			JSONSchema: structuredOutput,
		},
	}

	reqBytes, err := json.MarshalIndent(reqPayload, "", "  ")
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", baseEndpoint, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	fmt.Printf("About to call OpenAI\n")
	client := &http.Client{}
	resp, err := client.Do(req)
	fmt.Printf("Fininshed calling OpenAI\n")

	if err != nil {
		fmt.Printf("Got an error back %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)

		var errorResp openaiErrorResponse
		if err := json.Unmarshal(bodyBytes, &errorResp); err != nil {
			fmt.Printf("Response code %d, aborting\nOpenAI API error: %s\n", resp.StatusCode, string(bodyBytes))
			return nil, fmt.Errorf("OpenAI API error: %s", string(bodyBytes))
		}

		return nil, errorResp
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error when reading response body%v\n", err)
		return nil, err
	}

	var openaiResp openaiResponse
	if err := json.Unmarshal(respBytes, &openaiResp); err != nil {
		return nil, err
	}

	if len(openaiResp.Choices) == 0 {
		return nil, errors.New("no choices returned from OpenAI")
	}

	// ------------------------------------------------------------
	// Persist request and response to a unique temp-file for debug.
	// ------------------------------------------------------------
	logEntry := struct {
		Request  json.RawMessage `json:"request"`
		Response json.RawMessage `json:"response"`
	}{
		Request:  reqBytes,
		Response: respBytes,
	}

	if logBytes, err := json.MarshalIndent(logEntry, "", "  "); err == nil {
		if f, err := os.CreateTemp("", "vyb-openai-*.json"); err == nil {
			if _, wErr := f.Write(logBytes); wErr == nil {
				_ = f.Close()
			} else {
				fmt.Printf("error writing OpenAI log file: %v\n", wErr)
			}
			fmt.Printf("Wrote OpenAI log file to %s\n", f.Name())
		} else {
			fmt.Printf("error creating OpenAI log file: %v\n", err)
		}
	} else {
		fmt.Printf("error marshalling OpenAI log entry: %v\n", err)
	}

	return &openaiResp, nil
}

// GetModuleExternalContexts calls the LLM and returns a list of external
// context strings – one per module.
func GetModuleExternalContexts(systemMessage string, request *payload.ExternalContextsRequest) (*payload.ModuleExternalContextResponse, error) {
	userMessage, err := serializeExternalContextsRequest(request)
	if err != nil {
		return nil, fmt.Errorf("openai: failed to serialize external contexts request: %w", err)
	}
	model := "o4-mini"
	openaiResp, err := callOpenAI(systemMessage, userMessage, schema.GetModuleExternalContextSchema(), model)
	if err != nil {
		return nil, err
	}

	var resp payload.ModuleExternalContextResponse
	if err := json.Unmarshal([]byte(openaiResp.Choices[0].Message.Content), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
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
				Name: mc.Name,
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
				Name: mc.Name,
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
