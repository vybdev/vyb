# cmd Directory

This folder contains the main entry point for the `vyb` CLI and its
subcommands, implemented using the Cobra library.

## Subcommands

- init: Creates a .vyb directory in the current project root with basic
  metadata (metadata.yaml).
- remove: Deletes all .vyb metadata from the current project root
  (or forcibly from the entire directory hierarchy using --force-root).
- root: The root command that prints help if no subcommand is specified.
- template: Registers specialized commands for AI-based tasks such
  as 'refine', 'code', 'summarize', etc.
