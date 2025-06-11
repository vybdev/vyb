package gemini

// Package gemini provides an abstraction layer over the Google Gemini
// API similar to the llm/internal/openai module. The implementation is
// still a work-in-progress – at the moment all helpers return
// ErrNotImplemented so that the rest of the application can compile
// while incremental tasks add real functionality.

import (
    "errors"

    "github.com/vybdev/vyb/config"
    "github.com/vybdev/vyb/llm/payload"
)

// ErrNotImplemented is returned by every helper in this package until
// the real Gemini integration is completed.
var ErrNotImplemented = errors.New("gemini: not implemented")

// GetWorkspaceChangeProposals mirrors the OpenAI helper and will, once
// implemented, send the conversation to Gemini and unmarshal a
// WorkspaceChangeProposal.  Currently it returns ErrNotImplemented so
// callers can gracefully handle the missing feature.
func GetWorkspaceChangeProposals(_ config.ModelFamily, _ config.ModelSize, _ string, _ string) (*payload.WorkspaceChangeProposal, error) {
    return nil, ErrNotImplemented
}

// GetModuleContext will request an internal & public context summary
// for a single module. Not implemented yet.
func GetModuleContext(_ string, _ string) (*payload.ModuleSelfContainedContext, error) {
    return nil, ErrNotImplemented
}

// GetModuleExternalContexts will request external contexts for a set
// of modules. Not implemented yet.
func GetModuleExternalContexts(_ string, _ string) (*payload.ModuleExternalContextResponse, error) {
    return nil, ErrNotImplemented
}
