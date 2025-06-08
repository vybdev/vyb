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
2. [x] 🧹 **Refactor `cmd/template.prepareExecutionContext`**
3. [x] 📁 **Update `selector.Select` signature**
4. [x] 🚦 **Enforce write-scope restrictions**
5. [x] 🧩 **Module context wiring**
6. [x] Introduce `--all` flag 
7. [x] 🛡️ **Strengthen matcher & selector tests**
8. [x] Rely on modules for final file inclusion within a request
       After loading project `Metadata` from the `.vyb` directory, also load a fresh representation of the 
       filesystem's `Metadata` using the `metadata.buildMetadata` function. Merge both instances of metadata, 
       making sure that the final struct maintains the original annotations loaded from `.vyb`, but the new file lists, 
       token counts and hashes loaded from the filesystem. Exit in error if you find the module names loaded from the 
       filesystem don't match the modules persisted in the `.vyb` directory. 

9.  [ ] Tidy up the code and remove any unnecessary logic related to this task list. 
        Move all the `Metadata` and `Module` parsing logic from the `cmd/template` module back into the `workspace/metadata` module. Unexport everything that can be safely unexported, and remove every function that is no longer used.

10. [ ] Ensure that when the `target_dir` is the same as the `project_root` all files directly located in the root 
module are included in the request. As of now, this is not the case. The only way as of now to include the files from 
the root module in the request is with the `--all` flag, but that also includes files from sub-modules of the root 
module, which is inefficient.
 
11. [ ] 📚 **Update documentation**
    - Amend `README.md` and command help to explain the three path
      concepts and new safety guarantees.

12. [ ] ✅ **Cleanup**
    - Remove obsolete helpers and dead code (e.g. the old
      `isPathUnderDir`).
    - Run `go test ./...` and ensure full pass.
