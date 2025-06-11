package llm

import "testing"

func TestSupportedProvidersContainsGemini(t *testing.T) {
    providers := SupportedProviders()
    found := false
    for _, p := range providers {
        if p == "gemini" {
            found = true
            break
        }
    }
    if !found {
        t.Fatalf("SupportedProviders() = %v, want to contain 'gemini'", providers)
    }
}
