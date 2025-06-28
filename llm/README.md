# llm Package

`llm` wraps all interaction with LLM providers (currently OpenAI and Gemini)
and exposes strongly typed data structures so the rest of the codebase never
has to deal with raw JSON.

The active provider is selected based on `.vyb/config.yaml`.

## Model abstractions ⚙️

| Type           | Constants                | Purpose                              |
|----------------|--------------------------|--------------------------------------|
| `ModelFamily`  | `gpt`, `reasoning`       | High-level family/category of models |
| `ModelSize`    | `large`, `small`         | Coarse size tier inside a family     |

The `(family, size)` tuple is later resolved by the active provider into a
concrete model string (e.g. `GPT+Large → "GPT-4.1"` for OpenAI).

## Sub-packages

### `llm/internal/openai`

* Builds requests (`model`, messages, `response_format`).
* Retries on `rate_limit_exceeded`.
* Dumps every request/response pair to a temporary JSON file for easy
debugging.
* Public helpers:
  * `GetWorkspaceChangeProposals` – returns a list of file edits + commit
    message.
  * `GetModuleContext` – summarises a module into *internal* & *public*
    contexts.
  * `GetModuleExternalContexts` – produces *external* contexts in bulk.

### `llm/internal/gemini`

* Builds requests (`model`, messages, `generationConfig`).
* Dumps every request/response pair to a temporary JSON file for easy
debugging.
* Public helpers are the same as the OpenAI provider.

### `llm/payload`

Pure data structures for LLM communication:

* Go structs for request payloads (WorkspaceChangeRequest, ModuleContextRequest, ExternalContextsRequest)
* Go structs for response payloads (WorkspaceChangeProposal, ModuleSelfContainedContext, ModuleExternalContextResponse)
* All structs support JSON marshalling/unmarshalling for LLM interactions

## JSON Schema enforcement

The JSON responses expected from the LLM are described under
`llm/internal/<provider>/internal/schema/schemas/*.json`. Both providers
enforce structured JSON output to ensure responses can be unmarshalled
straight into Go types.

* **OpenAI** uses the `response_format` field with a `json_schema`.
* **Gemini** uses the `generationConfig` field with a `responseSchema`.
