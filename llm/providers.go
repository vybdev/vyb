package llm

// SupportedProviders returns the list of LLM providers that can be chosen
// when initialising a new vyb project.  The slice is a copy – callers may
// modify it without affecting the package-level data.
func SupportedProviders() []string {
    return append([]string(nil), supportedProviders...) // defensive copy
}

// supportedProviders holds the hard-coded list of providers until dynamic
// registration lands.  Keep the strings in lowercase as they are written
// verbatim to .vyb/config.yaml.
var supportedProviders = []string{"openai"}
