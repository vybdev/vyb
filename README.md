# vyb

`vyb` is a CLI that helps you iteratively develop applications, faster. 
With 'vyb', you can refine your application spec, generate or update code, or provide your own command extensions to 
perform tasks that are relevant to your application workspace.

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
