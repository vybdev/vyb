# cmd/template Directory

This folder contains the *prompt templates* that power every AI-driven
`vyb` command.  Each template is a `.vyb` YAML file with the following
fields:

| Field                           | Description                               |
|---------------------------------|-------------------------------------------|
| `name`                          | The CLI sub-command to register           |
| `prompt`                        | User-facing task description (Markdown)   |
| `targetSpecificPrompt` *(opt)*  | Extra instructions when a file is passed  |
| `argInclusionPatterns`          | Glob patterns accepted as CLI arguments   |
| `argExclusionPatterns`          | … patterns that cannot be passed          |
| `requestInclusionPatterns`      | Files to embed in the LLM payload         |
| `requestExclusionPatterns`      | Files to never embed                      |
| `modificationInclusionPatterns` | Files the LLM is allowed to touch         |
| `modificationExclusionPatterns` | Guard-rails against accidental edits      |

At runtime the loader merges three sources (by precedence):

1. Embedded templates bundled at compile time (`embedded/*.vyb`).
2. User-wide templates under `$VYB_HOME/cmd`.
3. (planned) Project-local templates under `.vyb/cmd`.

Templates use Mustache placeholders to inject dynamic data (e.g. the
command-specific prompt gets embedded into a global *system* prompt).
