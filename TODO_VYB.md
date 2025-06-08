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

`openai.GetWorkspaceChangeProposals` should leverage these concepts to incorporate Module summaries within the request, as follows:

- Every request should start with the `External Context` of the module that contains the `working_dir`;
- If the `target_dir` is contained in a module that is not the same as the `working_dir`'s module, the request should include:
  -  the `Internal Context` of every module between `working_dir` and `target_dir`;
  - the `Public Context` or every topmost module under `working_dir` that is not a parent of `target_dir`;
- Only files within the module that holds the `target_dir` should be included, if that module has sub-modules, the `Public Context` of the topmost sub-modules will also be included in the request;
- If a --all flag is provided, all files that pass inclusion/exclusion rules under the `target_dir` and any of its sub-modules are included in the request; 

## TODO List

Below is a sequenced backlog of **atomic** changes required to reach the
behaviour specified above.  Each bullet should result in a single,
self-contained commit.

1. [x] 🔧 **Introduce `ExecutionContext` struct**
    - Fields: `ProjectRoot`, `WorkingDir`, `TargetDir` (all *absolute*
      paths).
    - Add helper constructor that validates the invariants described in
      *What You Should Do*.
    - Place the new type in `workspace/context` (new package) so it can
      be reused by selector, template and tests.

2. [x] 🧹 **Refactor `cmd/template.prepareExecutionContext`**
    - Replace current tuple return with the new `ExecutionContext`.
    - Ensure CLI commands fail fast when `target_file` is outside
      `working_dir`.

3. [x] 📁 **Update `selector.Select` signature**
    - Accept `ExecutionContext` instead of loose params.
    - Implement inclusion logic: *all files under `target_dir`*.
    - Keep exclusion / inclusion pattern processing unchanged.
    - Add unit tests covering edge-cases (same dir, sibling, parent).

4. [x] 🚦 **Enforce write-scope restrictions**
    - In `cmd/template.execute` replace `isPathUnderDir` logic with a
      check against `ExecutionContext.WorkingDir`.
    - Remove the standalone `isPathUnderDir` helper once migrated.

5. [x] 🧩 **Module context wiring**
    - Enhance `openai.GetWorkspaceChangeProposals` (or the functions it calls)
      to compose module contexts according to the bullets under
      *`openai.GetWorkspaceChangeProposals` should leverage*.
    - Add exhaustive unit tests using in-memory module trees.

6. [ ] Introduce `--all` flag 
   - In the templated command execution (`cmd/template/template.go`), if `--all` is set, 
     include all files under the `target_dir` and any of its sub-modules.
   - Adjust tests accordingly.

7. [ ] 🛡️ **Strengthen matcher & selector tests**
    - Add cases ensuring that files outside `target_dir` are never
      included.
    - Add cases verifying that proposed modifications outside
      `working_dir` are rejected.

8. [ ] 📚 **Update documentation**
    - Amend `README.md` and command help to explain the three path
      concepts and new safety guarantees.

9. [ ] ✅ **Cleanup**
    - Remove obsolete helpers and dead code (e.g. the old
      `isPathUnderDir`).
    - Run `go test ./...` and ensure full pass.

## How You Should Do it
- One by one, implement each of the tasks listed in the "TODO List" of this file. Each execution should complete an entire task in that list, and mark it as completed, by updating this file and replacing "[ ]" with "[x]";
- For every change you make, include updated tests and documentation. If you decide not to change tests or documentation as part of a given TODO, include a justification in the `description` of the `workspace_change_proposal`;
- ALWAYS return the full content of any file you change. The content you return will be used to *replace* the content of the file. If you send back just a partial delta, the file will be broken.
