# workspace Directory

The code in this directory manages project-specific details, such as
finding and validating the project's root, matching and selecting
files for inclusion, and handling various metadata operations.

## Subdirectories

- matcher: Implements .gitignore-style file matching logic.
- project: Manages project metadata (the .vyb directory) and checks
  for correct project root location.
- selector: Orchestrates how files are scanned within the workspace,
  applying inclusion/exclusion patterns as well as .gitignore rules.

This abstraction allows the `vyb` CLI to work in a straightforward way
across diverse project structures by consistently determining which
files should be processed or ignored.
