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
	GetWorkspaceChangeProposals(fam config.ModelFamily, sz config.ModelSize, systemMessage string, request *payload.WorkspaceChangeRequest) (*payload.WorkspaceChangeProposal, error)
	GetModuleContext(systemMessage string, request *payload.ModuleContextRequest) (*payload.ModuleSelfContainedContext, error)
	GetModuleExternalContexts(systemMessage string, request *payload.ExternalContextsRequest) (*payload.ModuleExternalContextResponse, error)
}

type openAIProvider struct{}

type geminiProvider struct{}

type unknownProvider struct{}

func (*openAIProvider) GetWorkspaceChangeProposals(fam config.ModelFamily, sz config.ModelSize, sysMsg string, request *payload.WorkspaceChangeRequest) (*payload.WorkspaceChangeProposal, error) {
	return openai.GetWorkspaceChangeProposals(fam, sz, sysMsg, request)
}

func (*openAIProvider) GetModuleContext(sysMsg string, request *payload.ModuleContextRequest) (*payload.ModuleSelfContainedContext, error) {
	return openai.GetModuleContext(sysMsg, request)
}

func (*openAIProvider) GetModuleExternalContexts(sysMsg string, request *payload.ExternalContextsRequest) (*payload.ModuleExternalContextResponse, error) {
	return openai.GetModuleExternalContexts(sysMsg, request)
}

// -----------------------------------------------------------------------------
//  Gemini provider implementation
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

func (*geminiProvider) GetWorkspaceChangeProposals(fam config.ModelFamily, sz config.ModelSize, sysMsg string, request *payload.WorkspaceChangeRequest) (*payload.WorkspaceChangeProposal, error) {
	return gemini.GetWorkspaceChangeProposals(fam, sz, sysMsg, request)
}

func (*geminiProvider) GetModuleContext(sysMsg string, request *payload.ModuleContextRequest) (*payload.ModuleSelfContainedContext, error) {
	return gemini.GetModuleContext(sysMsg, request)
}

func (*geminiProvider) GetModuleExternalContexts(sysMsg string, request *payload.ExternalContextsRequest) (*payload.ModuleExternalContextResponse, error) {
	return gemini.GetModuleExternalContexts(sysMsg, request)
}

// -----------------------------------------------------------------------------
//	Unknown Provider is a throwing stub
// -----------------------------------------------------------------------------

func (*unknownProvider) GetWorkspaceChangeProposals(_ config.ModelFamily, _ config.ModelSize, _ string, _ *payload.WorkspaceChangeRequest) (*payload.WorkspaceChangeProposal, error) {
	return nil, fmt.Errorf("unknown provider")
}

func (*unknownProvider) GetModuleContext(_ string, _ *payload.ModuleContextRequest) (*payload.ModuleSelfContainedContext, error) {
	return nil, fmt.Errorf("unknown provider")
}

func (*unknownProvider) GetModuleExternalContexts(_ string, _ *payload.ExternalContextsRequest) (*payload.ModuleExternalContextResponse, error) {
	return nil, fmt.Errorf("unknown provider")
}

// -----------------------------------------------------------------------------
//  Public façade helpers remain unchanged (dispatcher section).
// -----------------------------------------------------------------------------

func GetModuleExternalContexts(cfg *config.Config, sysMsg string, request *payload.ExternalContextsRequest) (*payload.ModuleExternalContextResponse, error) {
	return resolveProvider(cfg).GetModuleExternalContexts(sysMsg, request)
}

func GetModuleContext(cfg *config.Config, sysMsg string, request *payload.ModuleContextRequest) (*payload.ModuleSelfContainedContext, error) {
	return resolveProvider(cfg).GetModuleContext(sysMsg, request)

}
func GetWorkspaceChangeProposals(cfg *config.Config, fam config.ModelFamily, sz config.ModelSize, sysMsg string, request *payload.WorkspaceChangeRequest) (*payload.WorkspaceChangeProposal, error) {
	return resolveProvider(cfg).GetWorkspaceChangeProposals(fam, sz, sysMsg, request)
}

// resolveProvider resolves the value of cfg.Provider to one of the known providers.
// Returns a throwing stub if it can't map the value to any known provider.
func resolveProvider(cfg *config.Config) provider {
	switch strings.ToLower(cfg.Provider) {
	case "openai":
		return &openAIProvider{}
	case "gemini":
		return &geminiProvider{}
	default:
		return &unknownProvider{}
	}
}
