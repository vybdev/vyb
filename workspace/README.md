# workspace Package

Everything related to the **local file system** lives here: matching,
selection, metadata generation and evolution.

## Sub-packages

| Package    | Responsibility                                     |
|------------|-----------------------------------------------------|
| `matcher`  | Lightweight implementation of `.gitignore` wildcards|
| `selector` | Walks the project applying inclusion/exclusion rules |
| `project`  | Creates/updates `.vyb/metadata.yaml` & annotations   |
| `context`  | Runtime-only struct capturing paths for a command    |

### File selection flow

1. A `context.ExecutionContext` pins *project root*, *working dir* and
   (optionally) a *target file*.
2. `selector.Select` starts at `TargetDir` and walks down, pruning:
   * directories excluded by user patterns or inherited `.gitignore`s;
   * files outside inclusion patterns.
3. Relative paths of the remaining files are returned for payload
   construction.

### Metadata lifecycle

* **Create (`vyb init`)** – scans the workspace and writes a brand-new
  `metadata.yaml` with empty annotations.
* **Update (`vyb update`)** – regenerates structural data, merges
  untouched annotations and refreshes/creates missing ones via the LLM.
* **Remove (`vyb remove`)** – deletes the whole `.vyb` folder.
