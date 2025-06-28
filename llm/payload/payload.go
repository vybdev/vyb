// Package payload contains data structures for LLM requests and responses.
package payload

// --- Request Payloads ---

// FileContent holds the path and content of a file.
type FileContent struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// WorkspaceChangeRequest contains all the necessary context and files for
// proposing workspace changes.
type WorkspaceChangeRequest struct {

	// WorkingModule represents the topmost module whose context is included in the workspace request
	WorkingModule string `json:"working_module"`
	// WorkingModuleContext is the context of the working module
	WorkingModuleContext string `json:"working_module_context"`

	// TargetModule represents the module that contains the target directory
	TargetModule string `json:"target_module"`
	// TargetModuleContext contains the context of the target module
	TargetModuleContext string `json:"target_module_context"`
	// TargetDirectory is the root directory from which the change request
	// should be applied (no change is expected outside of this directory or its subdirectories)
	TargetDirectory string `json:"target_directory"`

	// ParentModuleContexts contains the context of the parent and sibling modules
	// of the TargetModule contained within the working module, if any
	ParentModuleContexts []ModuleContext `json:"parent_module_contexts"`

	// SubModuleContexts contains the context of all the direct submodules of the TargetModule, if any.
	SubModuleContexts []ModuleContext `json:"submodule_contexts"`

	// Files contains the content of files relevant to the task.
	Files []FileContent `json:"files"`
}

// ModuleContext represents a piece of named context from a module.
type ModuleContext struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// ModuleContextRequest provides the necessary information to generate
// the internal and public contexts for a single module.
type ModuleContextRequest struct {
	// TargetModuleName is the name of the module being processed.
	TargetModuleName string `json:"target_module_name"`

	// TargetModuleFiles are the files within the module to be summarized.
	TargetModuleFiles []FileContent `json:"target_module_files"`

	// TargetModuleDirectories are the directories within the module.
	TargetModuleDirectories []string `json:"target_module_directories"`

	// SubModulesPublicContexts are the public contexts of immediate sub-modules.
	SubModulesPublicContexts []ModuleContext `json:"sub_modules_public_contexts"`
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

// ModuleExternalContextResponse captures the LLM response when generating
// external contexts for a set of modules.
type ModuleExternalContextResponse struct {
	Modules []ModuleExternalContext `json:"modules"`
}
