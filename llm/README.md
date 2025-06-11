# llm Package

`llm` wraps all interaction with OpenAI and exposes strongly typed data
structures so the rest of the codebase never has to deal with raw JSON.

## Model abstractions ⚙️

| Type           | Constants                | Purpose                              |
|----------------|--------------------------|--------------------------------------|
| `ModelFamily`  | `gpt`, `reasoning`       | High-level family/category of models |
| `ModelSize`    | `large`, `small`         | Coarse size tier inside a family     |

The `(family, size)` tuple is later resolved by the active provider into a
concrete model string (e.g. `GPT+Large → "GPT-4.1"`).

## Sub-packages

### `llm/openai`

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

### `llm/payload`

Pure data & helper utilities:

* `BuildUserMessage` – turns a list of files into a Markdown payload.
* `BuildModuleContextUserMessage` – embeds annotations into the payload
  according to precise inclusion rules.
* Go structs mirroring every JSON schema (WorkspaceChangeProposal,
  ModuleSelfContainedContext, …).

## JSON Schema enforcement

The JSON responses expected from the LLM are described under
`llm/openai/internal/schema/schemas/*.json`.  Each request sets the
`json_schema` field so GPT returns **validatable, deterministic** output
that can be unmarshalled straight into Go types.
