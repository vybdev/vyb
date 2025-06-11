# vyb

`vyb` is an AI-assisted command-line companion that helps developers
iterate on code, documentation and specifications **without having to
craft prompts by hand**.  It wraps common workflows (code generation,
refactoring, documentation, spec maintenance, …) into deterministic CLI
commands that:

1. **Select the right context** – only the files relevant for the task
   are sent to the LLM.
2. **Apply guarded changes** – proposed modifications are validated
   against allow/deny patterns before touching the working tree.
3. **Generate commit messages** – every LLM reply already contains a
   Conventional Commit message so you can just `git commit -F` it.  (WIP)

---

## Installation

Requirements:

* Go ≥ 1.24
* A valid API key for your chosen provider (`OPENAI_API_KEY` for OpenAI,
  `GEMINI_API_KEY` for Gemini).

```bash
# set your API key
$ export OPENAI_API_KEY="sk-..." # or...
$ export GEMINI_API_KEY="..."

# install the latest directly from github
$ go install github.com/vybdev/vyb@latest

# make the binary discoverable
# if #GOPATH is not set in your environment, it is usually under $HOME/go
$ export PATH=$GOPATH/bin:$PATH
```

> TIP Run `vyb --help` at any time to see all registered commands.

---

## Quick-start

```bash
# initialize repository configuration. 
# This will analyze the project files, and summarize them using your LLM provider of choice.
$ vyb init

# ask the LLM to implement a TODO in the current module
$ vyb code my/pkg/handler.go

# refresh documentation after a big refactor
$ vyb document -a   # -a ⇒ include *all* modules
```

All commands only **stage changes** locally; no commit is created.  After
reviewing the alterations you can commit them with the message suggested
by `vyb`.

---

## Built-in commands

| Command        | Purpose                                                    |
|----------------|------------------------------------------------------------|
| `init`         | Create `.vyb/metadata.yaml` in the project root            |
| `update`       | Re-scan workspace, merge & (re)generate annotations        |
| `remove`       | Delete `.vyb` completely                                   |
| `code`         | Implement `TODO(vyb)`s or the file passed as argument      |
| `document`     | Generate / refresh `README.md` files                       |
| `refine`       | Polish `SPEC.md` content                                   |
| `inferspec`    | Make spec match the *current* codebase                     |

Flags accepted by **all** AI-driven commands:

* `-a, --all` – include every file in the project, not only the current
  module.

---

## Core concepts

### Project metadata (`.vyb/metadata.yaml`)

A hierarchical representation of the workspace:

* **Module** – a folder treated as a logical unit.  Stores token counts,
  MD5 digests and an *Annotation* (see below).
* **FileRef** – path, checksum and token count for each file.

The metadata is fully derived from the file system; you should never
edit it manually.


### Project Configuration (`.vyb/config.yaml`)

`vyb init` creates a **config file** alongside the project metadata so the
CLI knows which LLM backend to call:

```yaml
provider: openai # or "gemini"
```

Only one key is defined for now but the document might grow in the future
(temperature defaults, retries, …).  The provider string is case-insensitive
and must match one of the options returned by `vyb llm.SupportedProviders()`.

### Model abstraction – family & size

Instead of hard-coding provider-specific model identifiers in every template
we use a two-part specification:

* **Family** – logical grouping (`gpt`, `reasoning`, …)
* **Size**   – `large` or `small`

The active provider maps the tuple to its concrete model name.

For example, the **OpenAI** implementation currently resolves to:

| Family / Size | Resolved model |
|---------------|----------------|
| gpt   / large | GPT-4.1        |
| gpt   / small | GPT-4.1-mini   |
| reasoning / large | o3         |
| reasoning / small | o4-mini    |

The **Gemini** provider maps both families to the same models:

| Family / Size | Resolved model                 |
|---------------|--------------------------------|
| *any* / large | gemini-2.5-pro-preview-06-05   |
| *any* / small | gemini-2.5-flash-preview-05-20 |

This indirection keeps templates provider-agnostic and allows you to switch
backends without touching prompt definitions.

### Annotations

`vyb` records three complementary summaries for every module:

* **Internal context** – what lives *inside* the module (private view).
* **Public context** – what the module (and its children) expose.
* **External context** – how the module fits in the *overall* application.

These texts are generated with the help of the LLM and later injected
into prompts to reduce the number of files that need to be submitted in each request.

---

## Architecture overview

```
cmd/            entry-points and Cobra command wiring
  template/     YAML + Mustache definitions used by AI commands
llm/            LLM provider wrappers + strongly typed JSON payloads
workspace/      file selection, .gitignore handling, metadata evolution
```

Flow of an AI command (`vyb code` for instance):

1. "template" loads the prompt YAML, computes inclusion/exclusion sets.
2. "selector" walks the workspace to gather the right files.
3. The user & system messages are built, then sent to `llm`.
4. The JSON reply is validated and applied to the working tree.

---

## Extending `vyb`

Put additional `.vyb` YAML templates under:

* `$VYB_HOME/cmd/` – globally available commands.
* `.vyb/cmd/` inside your project – repo-local commands *(planned)*.

See `cmd/template/embedded/code.vyb` for the field reference.

---

## Development & Testing

* Unit tests: `go test ./...`
* Lint / CI:   see `.github/workflows/go.yml`

Feel free to open issues or PRs – all contributions are welcome!
