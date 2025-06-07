# TODO(vyb): THIS IS YOUR HIGHEST PRIORITY TASK

## What You Should Do
`vyb` has three main concepts for context inclusion:

- `project_root`: the root location of the `.vyb` directory. Only one exists per project. Every file is relative to it.
- `working_dir`: the directory from which a command is executed. `vyb` can be executed from any directory under a `project_root`.
- `target_dir`: some `vyb` commands can given a file parameter, this is the `target_file`. The `target_file` must be under the `working_dir` or one of its sub-directories. If a `target_file` is given to the `vyb` command, its parent directory is the `target_dir`. If no `target_file` is given, the `target_dir` is the same as the `working_dir`.

File inclusions are based on those concepts.
- Every file is relative to the `project_root`;
- Every file under the `target_dir` is included in the request;
- Only files under the `working_dir` can be changed;

Module summaries should leverage these concepts as well:

- Every request should start with the `External Context` of the module that contains the `working_dir`;
- If the `target_dir` is contained in a module that is not the same as the `working_dir`, the request should include:
  -  the `Internal Context` of every module between `working_dir` and `target_dir`;
  - the `Public Context` or every topmost module under `working_dir` that is not a parent of `target_dir`;
- Only files within the module that holds the `target_dir` should be included, if that module has sub-modules, the `Public Context` of the topmost sub-modules will also be included in the request;

## TODO List

Below is a sequenced backlog of **atomic** changes required to reach the
behaviour specified above.  Each bullet should result in a single,
self-contained commit.

1. [ ] 🔧 **Introduce `ExecutionContext` struct**
    - Fields: `ProjectRoot`, `WorkingDir`, `TargetDir` (all *relative*
      paths).
    - Add helper constructor that validates the invariants described in
      *What You Should Do*.
    - Place the new type in `workspace/context` (new package) so it can
      be reused by selector, template and tests.

2. [ ] 🧹 **Refactor `cmd/template.prepareExecutionContext`**
    - Replace current tuple return with the new `ExecutionContext`.
    - Ensure CLI commands fail fast when `target_file` is outside
      `working_dir`.

3. [ ] 📁 **Update `selector.Select` signature**
    - Accept `ExecutionContext` instead of loose params.
    - Implement inclusion logic: *all files under `target_dir`*.
    - Keep exclusion / inclusion pattern processing unchanged.
    - Add unit tests covering edge-cases (same dir, sibling, parent).

4. [ ] 🚦 **Enforce write-scope restrictions**
    - In `cmd/template.execute` replace `isPathUnderDir` logic with a
      check against `ExecutionContext.WorkingDir`.
    - Remove the standalone `isPathUnderDir` helper once migrated.

5. [ ] 🧩 **Module context wiring**
    - Enhance `payload.BuildModuleContextUserMessage` (or a new helper)
      to compose module contexts according to the bullets under
      *Module summaries should leverage*.
    - Add exhaustive unit tests using in-memory module trees.

6. [ ] 🛡️ **Strengthen matcher & selector tests**
    - Add cases ensuring that files outside `target_dir` are never
      included.
    - Add cases verifying that proposed modifications outside
      `working_dir` are rejected.

7. [ ] 📚 **Update documentation**
    - Amend `README.md` and command help to explain the three path
      concepts and new safety guarantees.

8. [ ] ✅ **Cleanup**
    - Remove obsolete helpers and dead code (e.g. the old
      `isPathUnderDir`).
    - Run `go test ./...` and ensure full pass.

## How You Should Do it
- One by one, implement each of the tasks listed in the "TODO List of this file";
- For every change you make, include updated tests and documentation. If you decide not to change tests or documentation as part of a given TODO, include a justification in the `description` of the `workspace_change_proposal`;
- ALWAYS return the full content of any file you change. The content you return will be used to *replace* the content of the file. If you send back just a partial delta, the file will be broken.
