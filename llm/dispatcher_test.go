package llm

import (
    "testing"

    "github.com/vybdev/vyb/config"
)

// The following checks ensure that the provider implementations adhere to the
// provider interface.
var _ provider = (*openAIProvider)(nil)
var _ provider = (*geminiProvider)(nil)

// TestMapGeminiModel ensures that the (family,size) tuple is translated to
// the correct concrete model identifier and that unsupported sizes are
// properly rejected.
func TestMapGeminiModel(t *testing.T) {
    t.Parallel()

    cases := []struct {
        fam  config.ModelFamily
        size config.ModelSize
        want string
    }{
        {config.ModelFamilyGPT, config.ModelSizeSmall, "gemini-2.5-flash-preview-05-20"},
        {config.ModelFamilyGPT, config.ModelSizeLarge, "gemini-2.5-pro-preview-06-05"},
        {config.ModelFamilyReasoning, config.ModelSizeSmall, "gemini-2.5-flash-preview-05-20"},
        {config.ModelFamilyReasoning, config.ModelSizeLarge, "gemini-2.5-pro-preview-06-05"},
    }

    for _, c := range cases {
        got, err := mapGeminiModel(c.fam, c.size)
        if err != nil {
            t.Fatalf("mapGeminiModel(%s,%s) returned unexpected error: %v", c.fam, c.size, err)
        }
        if got != c.want {
            t.Fatalf("mapGeminiModel(%s,%s) = %q, want %q", c.fam, c.size, got, c.want)
        }
    }

    // Ensure an unsupported size triggers an error.
    if _, err := mapGeminiModel(config.ModelFamilyGPT, config.ModelSize("medium")); err == nil {
        t.Fatalf("expected error for unsupported model size, got nil")
    }
}
