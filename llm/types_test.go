package llm

import "testing"

func TestModelFamilyString(t *testing.T) {
    cases := []struct {
        in   ModelFamily
        want string
    }{
        {ModelFamilyGPT, "gpt"},
        {ModelFamilyReasoning, "reasoning"},
    }

    for _, c := range cases {
        if got := c.in.String(); got != c.want {
            t.Fatalf("ModelFamily.String() = %s, want %s", got, c.want)
        }
    }

    // Compile-time exhaustiveness – if a new constant is added the
    // switch below must be updated or the build will fail.
    var fam ModelFamily = ModelFamilyGPT
    switch fam {
    case ModelFamilyGPT, ModelFamilyReasoning:
        // ok
    default:
        t.Fatalf("unhandled ModelFamily constant %q", fam)
    }
}

func TestModelSizeString(t *testing.T) {
    cases := []struct {
        in   ModelSize
        want string
    }{
        {ModelSizeLarge, "large"},
        {ModelSizeSmall, "small"},
    }

    for _, c := range cases {
        if got := c.in.String(); got != c.want {
            t.Fatalf("ModelSize.String() = %s, want %s", got, c.want)
        }
    }

    // Compile-time exhaustiveness guard.
    var sz ModelSize = ModelSizeLarge
    switch sz {
    case ModelSizeLarge, ModelSizeSmall:
        // ok
    default:
        t.Fatalf("unhandled ModelSize constant %q", sz)
    }
}
