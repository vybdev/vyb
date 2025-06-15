# TODO(vyb): THIS IS YOUR HIGHEST PRIORITY TASK

## What will you do
### Remove the serialization logic from the `payload` package
The logic in the `payload` package can be specific to an LLM provider, and it is better if it moves closer to those implementations.

Instead of passing a string to the exported functions of the LLM providers, we should build data structures that represent the content that these functions take, and let the provider-specific function determine how to serialize these data structures into messages that work best for their backend.

Specifically, we currently have the following exported functions:

```go
func GetWorkspaceChangeProposals(fam config.ModelFamily, sz config.ModelSize, systemMessage, userMessage string) (*payload.WorkspaceChangeProposal, error)

func GetModuleContext(systemMessage, userMessage string) (*payload.ModuleSelfContainedContext, error)

func GetModuleExternalContexts(systemMessage, userMessage string) (*payload.ModuleExternalContextResponse, error)
```

In all of these functions, the userMessage should be replaced with a data structure that represents the content that each of these functions need. These new data structures should live in the `payload` package.

## How will you do it
You will execute only one step from the TODO List section at a time, in the order they are presented.
Always complete the entire TODO, and mark it as done ([x]) to demonstrate the TODO has indeed been achieved.
Do not mention the TODO itself in the change description, only the changes you've made.

## What you should know
- Question: Should this change include addition of a new dependency to the project?
    - Answer: No

## What the solution looks like
This document outlines the design for refactoring how user messages are constructed and passed to LLM providers. The goal is to move serialization logic from shared packages into provider-specific implementations, promoting better separation of concerns.

### 1. Problem Statement

Currently, functions in the `llm` package (`GetWorkspaceChangeProposals`, `GetModuleContext`, `GetModuleExternalContexts`) accept a pre-formatted `userMessage` string. This string is constructed in different parts of the application (`cmd/template`, `workspace/project`). This approach has several drawbacks:

- **Tight Coupling:** The calling code is responsible for serializing data into a Markdown format that the LLM expects. This couples the callers to the specific prompt format.
- **Inflexibility:** If a provider (e.g., Gemini) prefers a different message format, we would need to add conditional logic in the calling code, further complicating it.
- **Poor Abstraction:** The `llm` package should abstract away provider details, including how to format messages. Accepting a raw string breaks this abstraction.

### 2. Proposed Solution

The solution is to replace the `userMessage` string parameter in the `llm` dispatcher functions with new, purpose-built request structs. These structs will live in the `llm/payload` package and will contain the structured data needed for each type of request.

The provider-specific packages (`llm/internal/openai`, `llm/internal/gemini`) will then be responsible for serializing these request structs into the final message format (e.g., Markdown) before calling the LLM API.

### 3. Detailed Changes

#### 3.1. New Request Structs in `llm/payload`

The following new structs will be introduced in `llm/payload/payload.go`:

A common struct to hold file information:
```go
// FileContent holds the path and content of a file.
type FileContent struct {
    Path    string
    Content string
}
```

For `GetWorkspaceChangeProposals`:
```go
// WorkspaceChangeRequest contains all the necessary context and files for
// proposing workspace changes.
type WorkspaceChangeRequest struct {
    // ModuleContexts provides contextual information from various related modules.
    // The contexts should be ordered as they are intended to appear in the prompt.
    ModuleContexts []ModuleContext

    // Files contains the content of files relevant to the task.
    Files []FileContent
}

// ModuleContext represents a piece of named context from a module.
type ModuleContext struct {
    Name    string
    // Type can be "External", "Internal", or "Public".
    Type    string
    Content string
}
```

For `GetModuleContext`:
```go
// ModuleContextRequest provides the necessary information to generate
// the internal and public contexts for a single module.
type ModuleContextRequest struct {
    // TargetModuleFiles are the files within the module to be summarized.
    TargetModuleFiles []FileContent

    // TargetModuleDirectories are the directories within the module.
    TargetModuleDirectories []string

    // SubModulesPublicContexts are the public contexts of immediate sub-modules.
    SubModulesPublicContexts []struct {
        Name    string
        Context string
    }
}
```

For `GetModuleExternalContexts`:
```go
// ExternalContextsRequest contains information about a module hierarchy
// needed to generate external contexts for each module.
type ExternalContextsRequest struct {
    Modules []ModuleInfoForExternalContext
}

// ModuleInfoForExternalContext holds the data for a single module.
type ModuleInfoForExternalContext struct {
    Name            string
    ParentName      string
    InternalContext string
    PublicContext   string
}
```

#### 3.2. LLM Dispatcher and Provider Interface Changes

The signatures of the `llm` dispatcher functions and the `provider` interface will be updated to use the new request structs.

`llm/dispatcher.go`:
```go
// provider interface
type provider interface {
    GetWorkspaceChangeProposals(fam config.ModelFamily, sz config.ModelSize, systemMessage string, request *payload.WorkspaceChangeRequest) (*payload.WorkspaceChangeProposal, error)
    GetModuleContext(systemMessage string, request *payload.ModuleContextRequest) (*payload.ModuleSelfContainedContext, error)
    GetModuleExternalContexts(systemMessage string, request *payload.ExternalContextsRequest) (*payload.ModuleExternalContextResponse, error)
}

// Public functions
func GetWorkspaceChangeProposals(cfg *config.Config, fam config.ModelFamily, sz config.ModelSize, sysMsg string, request *payload.WorkspaceChangeRequest) (*payload.WorkspaceChangeProposal, error)
// ... and so on for the other functions.
```
The provider implementations (`openAIProvider`, `geminiProvider`) will be updated to match the new interface.

#### 3.3. Refactoring Message Construction Logic

- **`cmd/template/user_msg_builder.go`**: `buildExtendedUserMessage` will be modified. Instead of returning a `string`, it will return a `*payload.WorkspaceChangeRequest`. It will be responsible for gathering the module contexts and file contents to populate this struct.

- **`workspace/project/annotation.go`**:
    - In `addOrUpdateSelfContainedContext`, the logic that calls `payload.BuildModuleContextUserMessage` will be replaced. It will now construct a `*payload.ModuleContextRequest`. This will involve modifying `buildModuleContextRequest` to accept an `fs.FS` and read file contents.
    - In `addOrUpdateExternalContext`, the manual string building will be replaced with logic to populate a `*payload.ExternalContextsRequest`.

- **`llm/payload/payload.go`**: `BuildUserMessage` and `BuildModuleContextUserMessage` will be removed, as their serialization responsibilities are moving to the providers.

#### 3.4. Provider-Specific Serialization

Inside `llm/internal/openai/openai.go` and `llm/internal/gemini/gemini.go`, new internal functions will be added to handle the serialization of the request structs (`WorkspaceChangeRequest`, `ModuleContextRequest`, `ExternalContextsRequest`) into the final Markdown string format before making the API call.

This isolates the serialization logic within each provider, achieving the primary goal of this refactoring.

### 4. Impact

This change affects several packages (`cmd`, `llm`, `workspace`). However, it's a pure refactoring and will not change the application's behavior. The prompts sent to the LLMs will be identical to what is currently generated. It will improve code quality and maintainability. No new dependencies are required.

## TODO List
- [x] Evaluate the task description and code in the application, and all the questions already present in the "What you should know" section of this document, and elaborate all remaining questions you may have about the task. If you don't have any remaining question, only mark this TODO as completed.
- [x] Evaluate all the information you have so far and elaborate design for what the system should look like once the change is completed. The design should represent a delta between the system we have now, and that the system will look like when the task is completed. Included the design in "What the solution looks like" section of this document;
- [ ] Evaluate all the information you have so far and elaborate a plan describing each coding step needed to take the system from its current state to what is described in the section "What the solution looks like". The plan should describe individual changes to the code. Each change should leave the code in a consistent state, compiling and passing tests. Avoid leaving tests to the end of the list, and instead, add them alongside the new and updated code you are introducing. Don't change any code yet, only build a plan and add it to this list. Each step should be added as an individual item in the "TODO List" section of this document;  