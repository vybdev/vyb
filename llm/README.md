# llm Directory

This folder houses logic for interacting with the OpenAI API as well as
for building and parsing the requests and responses used by the `vyb` CLI.

## openai Package

Implements the HTTP calls to the OpenAI API, using a structured request
and response format. The `CallOpenAI` function returns proposed file
changes that the CLI then applies.

## payload Package

Defines the data structures for building the user message payload and
for interpreting the AI's proposed modifications to the workspace.
