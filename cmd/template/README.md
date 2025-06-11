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
| `model` *(opt)*                 | Tuple `{family, size}` selecting the LLM  |

At runtime the loader merges three sources (by precedence):

1. Embedded templates bundled at compile time (`embedded/*.vyb`).
2. User-wide templates under `$VYB_HOME/cmd`.
3. (planned) Project-local templates under `.vyb/cmd`.

Templates use Mustache placeholders to inject dynamic data (e.g. the
command-specific prompt gets embedded into a global *system* prompt).

### `model` field

Every template can optionally override the default model by specifying the
following YAML fragment:

```yaml
model:
  family: reasoning   # one of: gpt, reasoning
  size:   small       # large or small
```

When absent the loader falls back to `{family: reasoning, size: large}`.
The exact resolution to a concrete model string is handled by the active
provider (see `.vyb/config.yaml`).
