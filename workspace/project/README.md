# project sub-package

This folder contains everything related to **workspace metadata**.
The main artifact managed here is `.vyb/metadata.yaml`, a structured
inventory of the project that helps `vyb` decide *what* to send to the
LLM and *when* to regenerate summaries.

## Core concepts

| Concept  | Purpose |
|--------- | ---------------------------------------------------------------- |
| Module   | Logical grouping that mirrors a directory (e.g. `api/user`).      |
| FileRef  | Lightweight descriptor for a single file (path, MD5, token cnt). |
| Metadata | Root object that keeps the full Module tree.                      |
| Annotation | Three complementary summaries (internal, public, external).    |

### Token accounting & hashing

Each Module aggregates token counts from its children and computes an
MD5 digest of their hashes.  When two Module objects share the same MD5
we can safely reuse previous annotations.

### Annotation workflow (high level)

1. `vyb init`  – creates metadata **and** calls the LLM to fill missing
   annotations bottom-up (leaf modules first).
2. `vyb update` – rebuilds a fresh snapshot from disk, *patches* it into
   the stored tree preserving still-valid annotations and asks the LLM
   to fill only the gaps.
3. `vyb remove` – deletes the whole `.vyb` folder.

### Files of interest

| File                            | Responsibility |
|--------------------------------|------------------------------------------------|
| metadata.go                     | CRUD helpers + `Update` logic                  |
| filesystem.go                   | Walks `fs.FS`, builds Module/FileRef objects   |
| annotation.go                   | Parallel LLM calls that populate annotations   |
| root.go                         | Utility to locate project root from any path   |

### Example `metadata.yaml` (truncated)

```yaml
modules:
  name: .
  modules:
    - name: api
      token_count: 3712
      annotation:
        public-context: >-
          Package `api` exposes HTTP handlers …
```

> NOTE: The file is **fully generated** – do not edit by hand.