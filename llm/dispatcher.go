package llm

import (
    "fmt"
    "strings"

    "github.com/vybdev/vyb/config"
    "github.com/vybdev/vyb/llm/internal/gemini"
    "github.com/vybdev/vyb/llm/internal/openai"
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
    GetWorkspaceChangeProposals(fam config.ModelFamily, sz config.ModelSize, systemMessage, userMessage string) (*payload.WorkspaceChangeProposal, error)
    GetModuleContext(systemMessage, userMessage string) (*payload.ModuleSelfContainedContext, error)
    GetModuleExternalContexts(systemMessage, userMessage string) (*payload.ModuleExternalContextResponse, error)
}

type openAIProvider struct{}

type geminiProvider struct{}

func (*openAIProvider) GetWorkspaceChangeProposals(fam config.ModelFamily, sz config.ModelSize, sysMsg, userMsg string) (*payload.WorkspaceChangeProposal, error) {
    return openai.GetWorkspaceChangeProposals(fam, sz, sysMsg, userMsg)
}

func (*openAIProvider) GetModuleContext(sysMsg, userMsg string) (*payload.ModuleSelfContainedContext, error) {
    return openai.GetModuleContext(sysMsg, userMsg)
}

func (*openAIProvider) GetModuleExternalContexts(sysMsg, userMsg string) (*payload.ModuleExternalContextResponse, error) {
    return openai.GetModuleExternalContexts(sysMsg, userMsg)
}

// -----------------------------------------------------------------------------
//  Gemini provider implementation – WorkspaceChangeProposals hooked up
// -----------------------------------------------------------------------------

func mapGeminiModel(fam config.ModelFamily, sz config.ModelSize) (string, error) {
    switch sz {
    case config.ModelSizeSmall:
        return "gemini-2.5-flash-preview-05-20", nil
    case config.ModelSizeLarge:
        return "gemini-2.5-pro-preview-06-05", nil
    default:
        return "", fmt.Errorf("gemini: unsupported model size %s", sz)
    }
}

func (*geminiProvider) GetWorkspaceChangeProposals(fam config.ModelFamily, sz config.ModelSize, sysMsg, userMsg string) (*payload.WorkspaceChangeProposal, error) {
    return gemini.GetWorkspaceChangeProposals(fam, sz, sysMsg, userMsg)
}

func (*geminiProvider) GetModuleContext(sysMsg, userMsg string) (*payload.ModuleSelfContainedContext, error) {
    return gemini.GetModuleContext(sysMsg, userMsg)
}

func (*geminiProvider) GetModuleExternalContexts(sysMsg, userMsg string) (*payload.ModuleExternalContextResponse, error) {
    return gemini.GetModuleExternalContexts(sysMsg, userMsg)
}

// -----------------------------------------------------------------------------
//  Public façade helpers remain unchanged (dispatcher section).
// -----------------------------------------------------------------------------

func GetModuleExternalContexts(cfg *config.Config, sysMsg, userMsg string) (*payload.ModuleExternalContextResponse, error) {
    if provider, err := resolveProvider(cfg); err != nil {
        return nil, err
    } else {
        return provider.GetModuleExternalContexts(sysMsg, userMsg)
    }
}

func GetModuleContext(cfg *config.Config, sysMsg, userMsg string) (*payload.ModuleSelfContainedContext, error) {
    if provider, err := resolveProvider(cfg); err != nil {
        return nil, err
    } else {
        return provider.GetModuleContext(sysMsg, userMsg)
    }
}
func GetWorkspaceChangeProposals(cfg *config.Config, fam config.ModelFamily, sz config.ModelSize, sysMsg, userMsg string) (*payload.WorkspaceChangeProposal, error) {
    if provider, err := resolveProvider(cfg); err != nil {
        return nil, err
    } else {
        return provider.GetWorkspaceChangeProposals(fam, sz, sysMsg, userMsg)
    }
}

func resolveProvider(cfg *config.Config) (provider, error) {
    switch strings.ToLower(cfg.Provider) {
    case "openai":
        return &openAIProvider{}, nil
    case "gemini":
        return &geminiProvider{}, nil
    default:
        return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
    }
}
