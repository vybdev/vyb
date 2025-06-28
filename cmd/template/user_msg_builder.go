package template

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/vybdev/vyb/llm/payload"
	"github.com/vybdev/vyb/workspace/context"
	"github.com/vybdev/vyb/workspace/project"
)

// buildWorkspaceChangeRequest composes a payload.WorkspaceChangeRequest that will be
// sent to the LLM. It prepends module context information — as dictated
// by the specification — before the raw file contents. Both meta and
// meta.Modules must be non-nil.
func buildWorkspaceChangeRequest(rootFS fs.FS, meta *project.Metadata, ec *context.ExecutionContext, filePaths []string) (*payload.WorkspaceChangeRequest, error) {
	if meta == nil {
		return nil, fmt.Errorf("metadata cannot be nil")
	}
	if meta.Modules == nil {
		return nil, fmt.Errorf("metadata.Modules cannot be nil")
	}

	request := &payload.WorkspaceChangeRequest{}

	// Helper to clean/normalise relative paths
	rel := func(abs string) string {
		if abs == "" {
			return ""
		}
		r, _ := filepath.Rel(ec.ProjectRoot, abs)
		return filepath.ToSlash(r)
	}

	workingRel := rel(ec.WorkingDir)
	targetRel := rel(ec.TargetDir)

	request.TargetDirectory = targetRel

	// Find modules (metadata is guaranteed to be valid)
	workingMod := project.FindModule(meta.Modules, workingRel)
	targetMod := project.FindModule(meta.Modules, targetRel)

	if workingMod == nil || targetMod == nil {
		return nil, fmt.Errorf("failed to locate working and target modules")
	}

	// Set target module information
	request.TargetModule = targetMod.Name

	// Set target module context (combined internal and external context)
	var targetContext strings.Builder
	if ann := targetMod.Annotation; ann != nil {
		if ann.ExternalContext != "" {
			targetContext.WriteString("External Context: ")
			targetContext.WriteString(ann.ExternalContext)
			targetContext.WriteString("\n\n")
		}
		if ann.InternalContext != "" {
			targetContext.WriteString("Internal Context: ")
			targetContext.WriteString(ann.InternalContext)
		}
	}

	// Ensure TargetModuleContext is never empty
	if targetContext.Len() == 0 {
		targetContext.WriteString("No specific context available for this module.")
	}
	request.TargetModuleContext = targetContext.String()

	var parentModuleContexts []payload.ModuleContext
	var subModuleContexts []payload.ModuleContext

	// Collect parent and sibling module contexts
	isAncestor := func(a, b string) bool {
		return a == b || (a != "." && strings.HasPrefix(b, a+"/"))
	}

	for ancestor := targetMod.Parent; ancestor != nil; ancestor = ancestor.Parent {
		for _, child := range ancestor.Modules {
			// Skip the target itself and all its ancestor path.
			if isAncestor(child.Name, targetMod.Name) {
				continue
			}
			if ann := child.Annotation; ann != nil && ann.PublicContext != "" {
				parentModuleContexts = append(parentModuleContexts, payload.ModuleContext{
					Name:    child.Name,
					Content: ann.PublicContext,
				})
			}
		}
		if ancestor == workingMod {
			break
		}
	}

	// Collect immediate sub-modules of target module
	for _, child := range targetMod.Modules {
		if ann := child.Annotation; ann != nil && ann.PublicContext != "" {
			subModuleContexts = append(subModuleContexts, payload.ModuleContext{
				Name:    child.Name,
				Content: ann.PublicContext,
			})
		}
	}

	request.ParentModuleContexts = parentModuleContexts
	request.SubModuleContexts = subModuleContexts

	// Append file contents
	var files []payload.FileContent
	for _, path := range filePaths {
		content, err := fs.ReadFile(rootFS, path)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", path, err)
		}
		files = append(files, payload.FileContent{
			Path:    path,
			Content: string(content),
		})
	}
	request.Files = files

	return request, nil
}
