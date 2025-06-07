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

## How You Should Do it
- Start by making a plan of self-contained atomic changes to the codebase that will take you to the final result described above. Store that change in the TODO List section of this document.
- ALWAYS return the full content of any file you change. The content you return will be used to *replace* the content of the file. If you send back just a partial delta, the file will be broken.