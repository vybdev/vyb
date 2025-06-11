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
[unchanged – trimmed for brevity]

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
- [x] Break down the work into atomic changes. Each step must leave the
      repository in a compiling, tested and documented state.

- [x] `feat(llm): add ModelFamily & ModelSize enums`
   * create `llm/types.go` with the two string types and constants.
   * add unit test validating `String()` behaviour (compile-time safety).
   * docs: update `llm/README.md` enumerations.

- [x] `feat(config): introduce .vyb/config.yaml loader`
   * new package `config` with struct `Config` and `Load()` helper (reads
     YAML or returns default).
   * unit tests with in-memory `fstest.MapFS`.
   * docs: add Configuration section to root README.
   * mark this task as completed

- [x] `feat(cmd/init): prompt for provider & write config.yaml`
   * extend `cmd/init.go` to ask user via `survey` (fallback to openai).
   * write default YAML when non-interactive (tests use env var to skip).
   * update unit tests; adjust workflow to ensure binary still builds.
   * mark this task as completed

- [x] refactor the init cmd so the list of providers comes from the llm package, the provider selection happens before project.Create is called. Move the logic to persist provider information into project.Create.

- [x] `refactor(llm): create provider interface & dispatcher`
   * add private `provider` interface mirroring façade helpers.
   * implement `resolveProvider()` using `config.Load()`.
   * move existing openai helpers to satisfy the interface.
   * compile-time stubs for future providers.
   * mark this task as completed

- [x] `refactor(llm/openai): map family/size to concrete model`
   * implement `mapModel` as specified.
   * update exported helpers (`GetWorkspaceChangeProposals`, etc.) to
     accept model spec and project config structs
   * adapt unit tests.
   * mark this task as completed

- [x] `chore(openai): remove direct usages from business code`
   * search & replace openai.* calls outside llm/openai → switch to llm
     package.
   * ensure no import cycles.
   * mark this task as completed

- [x] `test: integration – provider dispatch`
   * add table-driven test that mocks `config.Load()` and checks correct
     provider is picked (uses testdouble implementing provider iface).
   * mark this task as completed

- [ ] `docs(templates): update README & example snippets`
    * reflect new `model:` structure & provider logic.
    * mark this task as completed

- [ ] `ci: run go vet & tests on new packages`
    * update GitHub action matrix if necessary.
    * mark this task as completed

- [ ] `cleanup: remove obsolete TODOs & dead code`
    * delete Model string fields, old helpers, and outdated comments.
    * mark this task as completed
