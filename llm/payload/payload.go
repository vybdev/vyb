package payload

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// fileEntry represents a file with its relative path and content.
type fileEntry struct {
	Path    string
	Content string
}

// BuildUserMessage constructs a Markdown-formatted string that includes the content of all files in scope.
// projectRoot represents the base directory for this project, and all file paths in the given filePaths parameter are relative to projectRoot.
func BuildUserMessage(projectRoot fs.FS, filePaths []string) (string, error) {
	var files []fileEntry
	for _, path := range filePaths {
		data, err := fs.ReadFile(projectRoot, path)
		if err != nil {
			return "", err
		}
		files = append(files, fileEntry{
			Path:    path,
			Content: string(data),
		})
	}
	markdown := buildPayload(files)
	return markdown, nil
}

// ---------------------
//  Data abstractions
// ---------------------

// --- Request Payloads ---

// FileContent holds the path and content of a file.
type FileContent struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// WorkspaceChangeRequest contains all the necessary context and files for
// proposing workspace changes.
type WorkspaceChangeRequest struct {
	// ModuleContexts provides contextual information from various related modules.
	// The contexts should be ordered as they are intended to appear in the prompt.
	ModuleContexts []ModuleContext `json:"module_contexts"`

	// Files contains the content of files relevant to the task.
	Files []FileContent `json:"files"`
}

// ModuleContext represents a piece of named context from a module.
type ModuleContext struct {
	Name string `json:"name"`
	// Type can be "External", "Internal", or "Public".
	Type    string `json:"type"`
	Content string `json:"content"`
}

// SubModuleContext holds the public context for a submodule.
type SubModuleContext struct {
	Name    string `json:"name"`
	Context string `json:"context"`
}

// ModuleContextRequest provides the necessary information to generate
// the internal and public contexts for a single module.
type ModuleContextRequest struct {
	// TargetModuleFiles are the files within the module to be summarized.
	TargetModuleFiles []FileContent `json:"target_module_files"`

	// TargetModuleDirectories are the directories within the module.
	TargetModuleDirectories []string `json:"target_module_directories"`

	// SubModulesPublicContexts are the public contexts of immediate sub-modules.
	SubModulesPublicContexts []SubModuleContext `json:"sub_modules_public_contexts"`
}

// ExternalContextsRequest contains information about a module hierarchy
// needed to generate external contexts for each module.
type ExternalContextsRequest struct {
	Modules []ModuleInfoForExternalContext `json:"modules"`
}

// ModuleInfoForExternalContext holds the data for a single module.
type ModuleInfoForExternalContext struct {
	Name            string `json:"name"`
	ParentName      string `json:"parent_name,omitempty"`
	InternalContext string `json:"internal_context,omitempty"`
	PublicContext   string `json:"public_context,omitempty"`
}

// --- Response Payloads ---

// WorkspaceChangeProposal is a concrete description of proposed workspace
// changes coming from the LLM.
type WorkspaceChangeProposal struct {
	Description string               `json:"description"`
	Summary     string               `json:"summary"`
	Proposals   []FileChangeProposal `json:"proposals"`
}

// FileChangeProposal represents a single file modification.
type FileChangeProposal struct {
	FileName string `json:"file_name"`
	Content  string `json:"content"`
	Delete   bool   `json:"delete"`
}

// ModuleSelfContainedContext captures the context of a module and its sub-modules.
type ModuleSelfContainedContext struct {
	Name            string `json:"name,omitempty"`
	ExternalContext string `json:"external_context,omitempty"`
	InternalContext string `json:"internal_context,omitempty"`
	PublicContext   string `json:"public_context,omitempty"`
}

// ModuleExternalContext captures the context of a module and its sub-modules.
type ModuleExternalContext struct {
	Name            string `json:"name,omitempty"`
	ExternalContext string `json:"external_context,omitempty"`
}
type ModuleSelfContainedContextRequest struct {
	FilePaths   []string
	Directories []string
	ModuleCtx   *ModuleSelfContainedContext
	SubModules  []*ModuleSelfContainedContextRequest
}

// BuildModuleContextUserMessage constructs a Markdown-formatted string that
// includes the content of all files referenced by the provided
// ModuleSelfContainedContextRequest *root* and the public context of its immediate
// sub-modules.
//
// Behaviour rules:
//  1. The files listed in the root request are included verbatim.
//  2. For each *immediate* sub-module of the root request, only its
//     PublicContext (if any) is emitted – no files are rendered for those
//     sub-modules and no information is emitted for modules that are
//     grandchildren or deeper.
//
// If any referenced file cannot be read this function returns an error.
func BuildModuleContextUserMessage(projectRoot fs.FS, request *ModuleSelfContainedContextRequest) (string, error) {
	if projectRoot == nil {
		return "", fmt.Errorf("projectRoot fs.FS must not be nil")
	}
	if request == nil {
		return "", fmt.Errorf("ModuleSelfContainedContextRequest must not be nil")
	}

	var sb strings.Builder

	// Helper that resolves the absolute workspace path for a file declared in
	// the *root* module.
	resolvePath := func(rootPrefix, rel string) string {
		if rootPrefix == "" || rootPrefix == "." || strings.HasPrefix(rel, rootPrefix+string(filepath.Separator)) {
			return rel
		}
		return filepath.Join(rootPrefix, rel)
	}

	// -----------------------------
	// 1.  Emit root-module information.
	// -----------------------------
	rootPrefix := ""
	if request.ModuleCtx != nil && request.ModuleCtx.Name != "" {
		rootPrefix = request.ModuleCtx.Name
	}

	// Header for the root module if we have a name or any context data.
	if rootPrefix != "" || (request.ModuleCtx != nil && (request.ModuleCtx.ExternalContext != "" || request.ModuleCtx.InternalContext != "" || request.ModuleCtx.PublicContext != "")) {
		writeModule(&sb, rootPrefix, request.ModuleCtx)
	}
	// Only spend these tokens if we need to teach the LLM that a directory != module.
	if len(request.Directories) > 1 {
		sb.WriteString(fmt.Sprintf("## Directories in module `%s`\n", rootPrefix))
		sb.WriteString(fmt.Sprintf("The following is a list of directories that are part of the module `%s`\n.", rootPrefix))
		sb.WriteString(fmt.Sprintf("These ARE NOT MODULES, they are directories within the module. When summarizing their file contents, include them in the summary of `%s`, do not make up modules for them.\n", rootPrefix))
		for _, dir := range request.Directories {
			sb.WriteString(fmt.Sprintf("- %s\n", dir))
		}
	}

	sb.WriteString(fmt.Sprintf("## Files in module `%s`\n", rootPrefix))
	// Emit root-module files.
	for _, relFile := range request.FilePaths {
		fullPath := resolvePath(rootPrefix, relFile)
		data, err := fs.ReadFile(projectRoot, fullPath)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s: %w", fullPath, err)
		}
		writeFile(&sb, fullPath, string(data))
	}

	// -----------------------------
	// 2. Emit public context of immediate sub-modules.
	// -----------------------------
	for _, sub := range request.SubModules {
		if sub == nil || sub.ModuleCtx == nil {
			continue // nothing useful to emit
		}

		// We only expose the public context of immediate sub-modules.
		if sub.ModuleCtx.PublicContext == "" && sub.ModuleCtx.Name == "" {
			continue
		}

		trimmedCtx := &ModuleSelfContainedContext{
			Name:          sub.ModuleCtx.Name,
			PublicContext: sub.ModuleCtx.PublicContext,
		}
		writeModule(&sb, trimmedCtx.Name, trimmedCtx)
	}

	return sb.String(), nil
}

// buildPayload constructs a Markdown payload from a slice of fileEntry.
// Each file is represented with an H1 header for its relative path, followed by a code block.
func buildPayload(files []fileEntry) string {
	var sb strings.Builder
	for _, f := range files {
		writeFile(&sb, f.Path, f.Content)
	}
	return sb.String()
}

func writeModule(sb *strings.Builder, path string, context *ModuleSelfContainedContext) {
	if sb == nil {
		return
	}
	if path == "" && (context == nil || (context.ExternalContext == "" && context.InternalContext == "" && context.PublicContext == "")) {
		return
	}
	sb.WriteString(fmt.Sprintf("# Module: `%s`\n", path))
	if context != nil {
		if context.ExternalContext != "" {
			sb.WriteString("## External Context\n")
			sb.WriteString(fmt.Sprintf("%s\n", context.ExternalContext))
		}
		if context.InternalContext != "" {
			sb.WriteString("## Internal Context\n")
			sb.WriteString(fmt.Sprintf("%s\n", context.InternalContext))
		}
		if context.PublicContext != "" {
			sb.WriteString("## Public Context\n")
			sb.WriteString(fmt.Sprintf("%s\n", context.PublicContext))
		}
	}
}

func writeFile(sb *strings.Builder, filepath, content string) {
	if sb == nil {
		return
	}
	lang := getLanguageFromFilename(filepath)
	sb.WriteString(fmt.Sprintf("### %s\n", filepath))
	sb.WriteString(fmt.Sprintf("```%s\n", lang))
	sb.WriteString(content)
	// Ensure a trailing newline before closing the code block.
	if !strings.HasSuffix(content, "\n") {
		sb.WriteString("\n")
	}
	sb.WriteString("```\n\n")
}

// getLanguageFromFilename returns a language identifier based on file extension.
func getLanguageFromFilename(filename string) string {
	if strings.HasSuffix(filename, ".go") {
		return "go"
	} else if strings.HasSuffix(filename, ".md") {
		return "markdown"
	} else if strings.HasSuffix(filename, ".json") {
		return "json"
	} else if strings.HasSuffix(filename, ".txt") {
		return "text"
	}
	// Default: no language specified.
	return ""
}

// ModuleExternalContextResponse captures the LLM response when generating
// external contexts for a set of modules.
type ModuleExternalContextResponse struct {
	Modules []ModuleExternalContext `json:"modules"`
}
