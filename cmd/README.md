# cmd Directory

This folder contains the main entry point for the `vyb` CLI and its
subcommands, implemented using the Cobra library.

## Subcommands

- init: Creates a .vyb directory in the current project root with basic
  metadata (metadata.yaml).
- remove: Deletes all .vyb metadata from the current project root
  (or forcibly from the entire directory hierarchy using --force-root).
- update: Updates the vyb project metadata.
- version: Prints the vyb CLI version.
- template-based commands: A dynamic set of commands for AI-based tasks
  such as 'refine', 'code', 'document', etc., are registered from `.vyb`
  template files.
