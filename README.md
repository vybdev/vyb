# vyb

`vyb` is a CLI that helps you iteratively develop applications, faster. 
With 'vyb', you can refine your application spec, generate or update code, or provide your own command extensions to 
perform tasks that are relevant to your application workspace.

## Vision
The vision for `vyb` is that it will provide a workflow for developers to collaborate with AI, through well-defined 
commands that encapsulate intent and avoid repetitive activities like drafting a prompt or choosing which files to 
include in every interaction with the LLM. 

Each `vyb` command has a filter for which workspace files should be included in the request to the LLM, and which files 
are allowed to be modified by the LLM's response.

### Workflow
The envisioned workflow for `vyb` is as follows:
- `vyb refine` will help the developer draft requirements for their application, marking which features are already 
implemented and which still need to be implemented. This could be connected with GitHub API, to link the issue # to the 
spec element. In this case, the first step in implementing an issue would be to add it as a TODO in the SPEC;
-  `vyb code` implements either a `TODO(vyb)` in the code, or a TODO SPEC element. As of now, this command relies 
heavily on `TODO(vyb)` for prioritization, and doesn't really know how to pick a SPEC element to implement. It would be 
good to find a better way to tell the CLI which specific task needs to be executed each time the command is called;
- `vyb document` updates README.md files to reflect the application code. Ideally, this activity would be done 
atomically when `vyb code` runs, but the prompt isn't there yet. But even after that is resolved, this command will 
continue to be useful because the developer may manually change code and allow it to drift over time;
- `vyb inferspec` ensures any manual modifications made to the code are reflected back in the SPEC.md files;

### Git Integration
Every command execution gets a list of files to be modified, along with a commit message. For now, the commit message is 
only printed in the output for the user to see, but the goal is to store it in the `.vyb/` folder, alongside the list of 
modified files. After reviewing the proposed modifications, the user will then run `vyb accept`, at which point `vyb` 
would stage and commit the changes, using the commit message provided by the LLM.

### LLM Quotas
`vyb` only includes files that match the command filtering criteria under the directory in which it is executed, plus 
any of its sub-directories. This is to limit the number of tokens sent to the LLM, but it could also hurt the LLM's 
ability to resolve a problem if it doesn't have all the context it needs.  
Still to be developed, is a summarization logic that will produce embeddings at each relevant application folder,
allowing for better contextualization even when executing `vyb` within a module and not passing it the entire codebase.

This is a high priority feature to be implemented soon, since even vyb's codebase is already reaching o1's token limits 
for certain prompts.

## Features

- Iterative specification refinement and code generation
- Automated project file detection and filtering
- Integration with OpenAI for advanced language-based tasks
- Modular command structure using Cobra

## Getting Started

1. Install Go 1.24 or later.
2. Clone this repository.
3. Build the CLI using:
   go build -o vyb
4. Obtain an OpenAI API key and store it in the OPENAI_API_KEY env var.

## Basic Usage

Below are some key commands:

• Initialize project:
  vyb init

• Remove project metadata:
  vyb remove [--force-root]

• Summarize code and docs:
  vyb summarize

• Refine specifications:
  vyb refine [SPEC.md]

• Implement code:
  vyb code [SPEC.md]

Check out the subdirectory README files for deeper insights into the
application's architecture.
