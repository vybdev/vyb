package llm

import (
    "testing"

    "github.com/vybdev/vyb/config"
)

// TestResolveProvider verifies that the dispatcher returns the expected
// concrete implementation for known providers and fails for unknown ones.
func TestResolveProvider(t *testing.T) {
    // 1. Happy-path – "openai" should map to *openAIProvider.
    cfg := &config.Config{Provider: "openai"}

    p, err := resolveProvider(cfg)
    if err != nil {
        t.Fatalf("unexpected error resolving provider: %v", err)
    }
    if _, ok := p.(*openAIProvider); !ok {
        t.Fatalf("resolveProvider returned %T, want *openAIProvider", p)
    }

    // 2. Unknown provider should surface an error.
    cfg.Provider = "doesnotexist"
    if _, err := resolveProvider(cfg); err == nil {
        t.Fatalf("expected error for unknown provider, got nil")
    }
}
