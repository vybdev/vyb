# selector sub-package

Responsible for discovering which files should be **sent to the LLM**
after all inclusion/exclusion rules are applied.

## Walk strategy

1. Determine `relStart` (directory that contains the *target* file or,
   if no target is given, the current working directory).
2. Walk the `fs.FS` from root.
   * Skip directories not relevant to the target (cheap pruning).
   * Merge inherited exclusion patterns with any `.gitignore` found on
     the way.
3. Every non-excluded file that matches inclusion patterns and lives
   *under* the target subtree is returned.

## Key invariants

* **Isolation** – never leaks files outside `TargetDir` into the prompt.
* **Consistency** – all paths are returned relative to workspace root
  using forward slashes.
* **Composable** – pure functions working on `fs.FS`, facilitating unit
  tests with `fstest.MapFS`.

### Interaction with other packages

* Delegates pattern checks to `workspace/matcher`.
* Relies on `workspace/context.ExecutionContext` for validated path
  boundaries.

See `selector_test.go` for practical scenarios, including protection
against context leakage.
