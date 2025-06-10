# TODO(vyb): THIS IS YOUR HIGHEST PRIORITY TASK!

## What you will do
Right now the application has an llm package, but inside it we only have
an openai implementation that is exported and called directly by the
business logic that requires the functionality. Additionally, even the
templated commands have a field for model name, which is directly tied
to a model provider.

A more robust and flexible solution would be for the model providers to
be completely abstracted away from the business logic. All the exported
code should be at the llm package, and based on user configuration, the
code then decides whether to delegate the calls to OpenAi, or any of the
other--to be supported--model providers.

Instead of calling out the model name directly in the commands and
templates, we could refer to them as model families (GPT and reasoning),
and model size (Large,Small). And let the provider-specific code map
these concepts to its own domain.

The configuration should come from .vyb/config.yaml. This file should be
created during vyb init execution, and default the LLM-provider to
OpenAI.

## How you will do it
Perform the next task listed under "What is left to do" in the order
they are listed. You are expected to accomplish no more and no less than
one task at a time. Mark with an [x] the task you have finished.

## What you need to know
- Question: Where exactly should `.vyb/config.yaml` live relative to the
  project root?  Inside the existing `.vyb/` directory or alongside it?
  - Answer: the `config.yaml` will live inside the `.vyb/` directory that
    is created as part of the `vyb init` execution.

- Question: What *schema* is expected for this file?  At minimum it needs
  the provider name (`openai`, `anthropic`, *etc.*) plus the mapping from
  (family, size) → provider-specific model string; is anything else
  required (e.g. API keys, temperature defaults, retries)?
  - Answer: Model provider should be a field within the config data
    structure (which you will create as well). The value of model
    provider should be a data structure that only has the model
    provider's name, for now. We may make this more robust in the
    future. Do not include family and size mapping in the config, this
    should be in the code of the model specific package.

- Question: Which "model families" and "sizes" must be supported in the
  first iteration?  The task mentions *GPT / reasoning* and *Large /
  Small*; please confirm the full matrix we should encode.
  - Answer: This logic should live in the provider package, since the
    mapping will change from one to another.
    - family(GPT) + size(L) -> "GPT-4.1"
    - family(GPT) + size(S) -> "GPT-4.1-mini"
    - family(reasoning) + size(L) -> "o3"
    - family(reasoning) + size(S) -> "o4-mini" 

- Question: The current code uses explicit model strings (`o3`,
  `o4-mini`) in several places.  Should those be removed entirely or only
  hidden behind a resolver while keeping the literals?
  - Answer: functions within the openai module that already have models
    hardcoded can continue to do so. The functions that receive the
    model as a parameter should now receive model family and size
    instead. 

- Question: Is backwards compatibility with existing templates important
  (i.e. should `model:` in `.vyb` template files still work)?
  - Answer: backward compatibility is not important, but you should
    convert the templates you find to match the new structure

- Question: Which public surface should the refactored `llm` package
  expose?  A single `Call` function, higher-level helpers similar to the
  current `GetWorkspaceChangeProposals`, or both?
  - Answer:Functions like `GetWorkspaceChangeProposals`, which
    encapsulate business logic

- Question: Do we need a mechanism to override the provider at runtime
  via an environment variable/flag, or is the YAML file authoritative?
  - Answer:Design the code in a way that this can be added in the
    future. But for now there is no need to provide additional ways of
    loading the provider.

- Question: Should `vyb init` *prompt* the user for the desired provider
  or silently create the default (`openai`) config?
  - Answer: Yes, it should provide a list of supported providers for the
    user to select. For now the list will only have openai, but we will
    add to it later.

- Question: Apart from OpenAI, are there already providers on the
  roadmap that we should stub out (e.g. `anthropic`, `azure-openai`)?
  - Answer: Not yet

- Question: Any preference for the dependency injection approach?
  (interface in `llm` + provider registration vs. simple `switch` on
  config).
  - Answer: it can be a simple switch in the `llm` package, forwarding
    the calls to provider specific internal packages

## What will it look like
Below is the *updated* vision for the codebase once the refactor is
completed.  It incorporates all clarifications and feedback received so
far.

### Configuration layout
```
.vyb/
├── metadata.yaml   # existing – DO NOT TOUCH BY HAND
└── config.yaml     # NEW – written by `vyb init`
```

`config.yaml` (initial contents)
```yaml
# .vyb/config.yaml
provider:
  name: openai  # future-proof: structure allows nested settings later
```
Only the provider name is stored for now; credentials continue to be
supplied through environment variables (e.g. `OPENAI_API_KEY`).

### Public API (`llm` package)
```go
// enum-like helpers – exported so templates & callers share the same type
package llm

type ModelFamily string // “GPT”, “Reasoning”, …
const (
    GPT       ModelFamily = "GPT"
    Reasoning ModelFamily = "Reasoning"
)

type ModelSize string // “Large”, “Small”
const (
    Large ModelSize = "Large"
    Small ModelSize = "Small"
)

// Provider-agnostic helpers
func GetWorkspaceChangeProposals(sysMsg, userMsg string,
    fam ModelFamily, sz ModelSize) (*payload.WorkspaceChangeProposal, error)

// More façade helpers will follow the same pattern.
```
Only these high-level functions are reachable by the rest of the
application.  They internally choose the concrete provider based on the
user configuration.

### Provider resolution
```go
func resolveProvider() provider { // provider is an unexported interface
    cfg, _ := config.Load() // cached read of .vyb/config.yaml
    switch strings.ToLower(cfg.Provider.Name) {
    case "openai", "":
        return openai.Provider{}
    // future: case "anthropic": …
    default:
        return openai.Provider{} // safe fallback
    }
}
```
The returned object satisfies a private `provider` interface mirroring
all façade helpers, so delegation is trivial.

### Template changes
Old:
```yaml
model: o3
```
New:
```yaml
model:
  family: GPT        # or Reasoning
  size:   Large      # or Small
```
The loader now unmarshals into:
```go
type ModelSpec struct {
    Family llm.ModelFamily `yaml:"family"`
    Size   llm.ModelSize   `yaml:"size"`
}
```
and hands it straight to the façade call.

### Mapping family/size → provider model (OpenAI example)
```go
// openai/provider.go (unexported helper)
func mapModel(fam llm.ModelFamily, sz llm.ModelSize) string {
    switch fam {
    case llm.GPT:
        if sz == llm.Large {
            return "GPT-4.1"
        }
        return "GPT-4.1-mini"
    case llm.Reasoning:
        if sz == llm.Large {
            return "o3"
        }
        return "o4-mini"
    default:
        return "o3" // sane fallback
    }
}
```
This logic lives **inside** the provider; business code never sees raw
model strings.

### CLI (`vyb init`)
At initialisation time we now:
1. Ask: *“Select LLM provider”* – currently the only option is **OpenAI**.
2. Persist the choice to `.vyb/config.yaml`.
3. Leave `metadata.yaml` untouched.

### Testing strategy
* Config loader round-trip & defaults.
* Provider mapping unit tests (one table-driven test per provider).
* Integration stub verifying façade dispatch according to `config.yaml`.

### Documentation
* README gains a *Configuration* section.
* Template README updated to new `model:` structure.

## What is left to do
- [x] First, evaluate the code in this project, and the task
      description in "What you will do". Then ask as many questions as
      you need to have full certainty about what is being asked. Ask
      your questions under "What you need to know" section.
- [x] Once your questions have been answered, propose a design for your
      solution. Replace the contents under "What will it look like" with
      the proposed changes to the system. This is not a list of tasks,
      it is a vision for the final state of the system to satisfy all
      the requirements.
- [x] Review and update the proposed design taking into consideration
      any feedback and TODO notes I left.
- [ ] Now review everything you know about this task, and break it down
      into a list of atomic changes, and add them to this list here.
      Each change should be selfcontained and leave the system one step
      closer to the desired state. At the end of each change, the system 
      should still compile, pass tests, and be fully functional.
      Make sure to include tests and documentation changes alongside 
      each step, since the repository should not get into an inconsistent 
      state in between these changes. 
