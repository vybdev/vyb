# matcher sub-package

Tiny but powerful implementation of **.gitignore-style pattern
matching** used across `vyb` for:

* argument validation (`argInclusionPatterns`, `argExclusionPatterns`),
* request trimming (what gets embedded in the LLM prompt),
* write-guard rails (what the LLM is allowed to change).

## Highlights

* Supports `*`, `?`, ranges `[a-z]` and the double-star `**` semantics
  documented in the official Git spec.
* Implements negated rules (`!important.txt`) and directory-only
  patterns (`build/`).
* Works directly with `fs.FS` so it can be unit-tested against an
  in-memory filesystem.

### Public helpers

| Function           | Description |
|--------------------|-------------|
| `IsIncluded`       | True when a path **is not** excluded **and** matches at
|                    | least one inclusion rule.                                |
| `IsExcluded`       | Convenience wrapper that checks only the exclusion set.  |

The actual globbing logic lives in `matchesPattern`, which performs a
recursive token comparison to honour `**` behaviour without relying on
`filepath.Match` (that API lacks double-star support).

Extensive test coverage can be found in `matchers_test.go`.