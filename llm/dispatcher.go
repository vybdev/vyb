package llm

import (
    "fmt"
    "strings"

    "github.com/vybdev/vyb/config"
    "github.com/vybdev/vyb/llm/openai"
    "github.com/vybdev/vyb/llm/payload"
)

// provider captures the common operations expected from any LLM backend.
// It is intentionally unexported so that the public surface of the llm
// package stays minimal while allowing internal dispatch based on user
// configuration.
//
// Additional methods should be appended here whenever new high-level
// helpers are added to the llm façade.
type provider interface {
    GetWorkspaceChangeProposals(systemMessage, userMessage string) (*payload.WorkspaceChangeProposal, error)
    GetModuleContext(systemMessage, userMessage string) (*payload.ModuleSelfContainedContext, error)
    GetModuleExternalContexts(systemMessage, userMessage string) (*payload.ModuleExternalContextResponse, error)
}

// resolveProvider inspects `.vyb/config.yaml` located at projectRoot and
// instantiates the matching provider implementation.  The caller is
// responsible for passing the *workspace root* so that config.Load can
// locate the file.
func resolveProvider(projectRoot string) (provider, error) {
    cfg, err := config.Load(projectRoot)
    if err != nil {
        return nil, err
    }

    switch strings.ToLower(cfg.Provider) {
    case "openai":
        return openAIProvider{}, nil
    default:
        return nil, fmt.Errorf("unsupported LLM provider %q", cfg.Provider)
    }
}

// openAIProvider is a thin wrapper around the existing llm/openai helpers
// so they fulfil the provider interface.  No additional logic is needed
// at this stage.
type openAIProvider struct{}

func (openAIProvider) GetWorkspaceChangeProposals(sysMsg, userMsg string) (*payload.WorkspaceChangeProposal, error) {
    return openai.GetWorkspaceChangeProposals(sysMsg, userMsg)
}

func (openAIProvider) GetModuleContext(sysMsg, userMsg string) (*payload.ModuleSelfContainedContext, error) {
    return openai.GetModuleContext(sysMsg, userMsg)
}

func (openAIProvider) GetModuleExternalContexts(sysMsg, userMsg string) (*payload.ModuleExternalContextResponse, error) {
    return openai.GetModuleExternalContexts(sysMsg, userMsg)
}