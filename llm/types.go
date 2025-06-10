package llm

// ModelFamily represents the generic family of a language model.
//
// The enumeration is intentionally small for now – new families can be
// added without touching the public API surface that depends on the
// underlying string values.
//
// NOTE: keep the string literals all-lowercase as they are used for
// YAML/JSON marshaling and command-line flags.
//
// Example usage:
//   var f ModelFamily = ModelFamilyGPT
//   fmt.Println(f) // -> "gpt"
//
// The zero value is an empty string and therefore invalid. Always
// initialise the variable with one of the provided constants.
//
// New families MUST be handled in every provider implementation – use
// exhaustive switch checks to ensure compile-time safety.
type ModelFamily string

const (
    // ModelFamilyGPT groups together GPT-style chat models optimised for
    // general purpose coding / conversation (e.g. GPT-4).
    ModelFamilyGPT ModelFamily = "gpt"

    // ModelFamilyReasoning corresponds to models targeted at long
    // reasoning or planning tasks.
    ModelFamilyReasoning ModelFamily = "reasoning"
)

func (m ModelFamily) String() string { return string(m) }

// ModelSize captures the coarse size tier of a model within the same
// family.  Providers translate these buckets to concrete model names
// (e.g. "large" → "gpt-4o", "small" → "gpt-3.5-turbo-0125").
type ModelSize string

const (
    // ModelSizeLarge is the higher-capability (and more expensive)
    // variant within a given family.
    ModelSizeLarge ModelSize = "large"

    // ModelSizeSmall is the cheaper and faster sibling of Large.
    ModelSizeSmall ModelSize = "small"
)

func (m ModelSize) String() string { return string(m) }
