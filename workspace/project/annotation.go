package project

import (
	"fmt"
	"github.com/vybdev/vyb/config"
	"github.com/vybdev/vyb/llm"
	"github.com/vybdev/vyb/llm/payload"
	"github.com/vybdev/vyb/logging"
	"io/fs"
	"strings"
)

// Annotation holds context and summary for a Module.
// ExternalContext is an LLM-provided textual description of the context in which a given Module exists.
// InternalContext is an LLM-provided textual description of the content that lives within a given Module.
// PublicContext is an LLM-provided textual description of content that his Module exposes for other modules to use.
type Annotation struct {
	ExternalContext string `yaml:"external-context"`
	InternalContext string `yaml:"internal-context"`
	PublicContext   string `yaml:"public-context"`
}

// annotate navigates the modules graph, starting from the leaf-most
// modules back to the root. For each module that has no Annotation, it calls
// addOrUpdateSelfContainedContext for it after all its submodules are annotated. The creation of
// annotations is performed in parallel using goroutines.
func annotate(cfg *config.Config, metadata *Metadata, sysfs fs.FS) error {
	if metadata == nil || metadata.Modules == nil {
		return nil
	}

	// Collect modules in post-order so children come before parents.
	modules := collectModulesInPostOrder(metadata.Modules)
	// Channel to collect errors from annotation goroutines.
	errCh := make(chan error, len(modules))
	// Create a done channel for each module to signal completion of annotation.
	dones := make(map[*Module]chan struct{})
	for _, m := range modules {
		dones[m] = make(chan struct{})
	}
	// Pre-close done channels for modules already annotated.
	for _, m := range modules {
		if m.Annotation != nil {
			close(dones[m])
		}
	}

	// Launch annotation tasks.
	for _, m := range modules {
		if m.Annotation != nil {
			logging.Log.Infof("module %q already has an annotation, skipping...\n", m.Name)
			continue
		}
		logging.Log.Infof("module %q doesn't have annotation\n", m.Name)
		// Capture m for the goroutine.
		go func(mod *Module) {
			// Wait for all submodules to complete.
			for _, sub := range mod.Modules {
				<-dones[sub]
			}
			err := addOrUpdateSelfContainedContext(cfg, mod, sysfs)
			if err != nil {
				errCh <- fmt.Errorf("failed to create annotation for module %q: %w", mod.Name, err)
				// Signal done to avoid blocking parents.
				close(dones[mod])
				return
			}
			close(dones[mod])
		}(m)
	}

	// Wait for root module to finish annotation.
	root := metadata.Modules
	<-dones[root]
	close(errCh)

	// Check for errors.
	for err := range errCh {
		if err != nil {
			return err
		}
	}

	// Add all external context annotations in a single shot
	// In the future, we should make this take into consideration
	// the token count of the annotations and possibly split the calls.
	return addOrUpdateExternalContext(cfg, root)
}

// collectModulesInPostOrder gathers modules in a post-order traversal (children first).
func collectModulesInPostOrder(root *Module) []*Module {
	var result []*Module
	var traverse func(*Module)

	traverse = func(m *Module) {
		for _, sub := range m.Modules {
			traverse(sub)
		}
		result = append(result, m)
	}

	traverse(root)
	return result
}

// addOrUpdateSelfContainedContext calls the LLM to construct the internal and public context of a given module.
func addOrUpdateSelfContainedContext(cfg *config.Config, m *Module, sysfs fs.FS) error {
	// Build the ModuleContextRequest for this module.
	var targetFiles []payload.FileContent
	for _, fileRef := range m.Files {
		content, err := fs.ReadFile(sysfs, fileRef.Name)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", fileRef.Name, err)
		}
		targetFiles = append(targetFiles, payload.FileContent{
			Path:    fileRef.Name,
			Content: string(content),
		})
	}

	var subContexts []payload.ModuleContext
	for _, subMod := range m.Modules {
		var publicContext string
		if subMod.Annotation != nil && subMod.Annotation.PublicContext != "" {
			publicContext = subMod.Annotation.PublicContext
		}
		subContexts = append(subContexts, payload.ModuleContext{
			Name:    subMod.Name,
			Content: publicContext,
		})
	}

	req := &payload.ModuleContextRequest{
		TargetModuleName:         m.Name,
		TargetModuleFiles:        targetFiles,
		TargetModuleDirectories:  m.Directories,
		SubModulesPublicContexts: subContexts,
	}

	logging.Log.Infof("annotating module %q\n", m.Name)

	// System prompt instructing the LLM to summarize code into JSON schema.
	systemMessage := `You are a prompt engineer, structuring information about an application's code base 
so context can be provided to an LLM in the most efficient way. 
The user message contains information about a module in the application, as well as its immediate sub-modules.
A module is a folder with files, and possibly other folders within it. 

Module information includes:

- Internal context: a description of the content that lives within the module. 
This is used when an LLM prompt is constructed from a sub-module of this given module, 
and the prompt is too large to include all files within the module. 
So instead of providing all the file contents, the "Internal Context" is used as a summary. 
The summary you will write for the module you are given will only take into consideration the files you see in the user 
message, as those are the files included in the module. Do not include information about the sub-modules in the Internal Context.

- Public context: a description of content that this module exposes for other modules to use. 
This should encapsulate not only the contents of the module, but the contents of all its sub-modules.
This is used when the LLM prompt is constructed from a module outside of the hierarchy of this given module.
The "Public Context" can include snippets of interfaces, script parameters, or any useful information for the LLM to 
understand how to work with a module. If the module you are given has any sub-modules, you will have access to their Public Context. 
You will contruct a Public Context for the module you are given, and that should encapsulate not only the information 
you included in the Internal Context, but also all the Public Context information from this module's sub-modules.

Each type of context should be as descriptive as possible, using around one thousand LLM tokens, each.`

	context, err := llm.GetModuleContext(cfg, systemMessage, req)

	logging.Log.Infof("  Got response for module %q\n", m.Name)

	if err != nil {
		return fmt.Errorf("failed to call llm provider: %w", err)
	}

	if m.Annotation == nil {
		m.Annotation = &Annotation{}
	}

	if context.InternalContext != "" {
		if m.Annotation.InternalContext != "" {
			logging.Log.Infof("  Overriding field `InternalContext` of module %q.\n", m.Name)
		} else {
			logging.Log.Infof("  Creating field `InternalContext` of module %q.\n", m.Name)
		}
		m.Annotation.InternalContext = context.InternalContext
	}
	if context.PublicContext != "" {
		if m.Annotation.PublicContext != "" {
			logging.Log.Infof("  Overriding field `PublicContext` of module %q.\n", m.Name)
		} else {
			logging.Log.Infof("  Creating field `PublicContext` of module %q.\n", m.Name)
		}
		m.Annotation.PublicContext = context.PublicContext
	}
	return nil
}

// addOrUpdateExternalContext generates or updates the ExternalContext for the
// provided module *and all of its children*.
//
// Behaviour:
//  1. Build a flattened list with the module itself plus every descendant
//     module.
//  2. For every module gather its current InternalContext and PublicContext
//     (if available) – this information is provided to the LLM so it can
//     reason about how the module fits the overall hierarchy.
//  3. Call the LLM to obtain an ExternalContext string for each module.
//  4. Persist the returned ExternalContext into the Annotation of the
//     corresponding module, creating annotation objects when necessary.
//
// If the LLM call fails the error is propagated to the caller.
func addOrUpdateExternalContext(cfg *config.Config, m *Module) error {
	if m == nil {
		return nil
	}

	// ------------------------------------------------------------
	// 0. Early-exit optimisation – if EVERY module already has an
	//    ExternalContext annotation we can skip the expensive LLM call.
	// ------------------------------------------------------------
	allHaveExternal := true

	modules := collectAllModules(m)

	// ------------------------------------------------------------
	// 1. Collect modules (m + all descendants) & prepare name->ptr map.
	// ------------------------------------------------------------
	moduleMap := make(map[string]*Module, len(modules))
	for _, mod := range modules {
		if mod.Name != "." && (mod.Annotation == nil || strings.TrimSpace(mod.Annotation.ExternalContext) == "") {
			allHaveExternal = false

		}
		moduleMap[mod.Name] = mod
	}

	if allHaveExternal {
		return nil // Nothing to do – everything is already annotated.
	}

	// ------------------------------------------------------------
	// 2. Build request containing internal & public context that the
	//    LLM will use to infer external context.
	// ------------------------------------------------------------
	var modulesForRequest []payload.ModuleInfoForExternalContext
	for _, mod := range modules {
		var parentName string
		if mod.Parent != nil {
			parentName = mod.Parent.Name
		}

		var internalCtx, publicCtx string
		if mod.Annotation != nil {
			internalCtx = mod.Annotation.InternalContext
			publicCtx = mod.Annotation.PublicContext
		}

		modulesForRequest = append(modulesForRequest, payload.ModuleInfoForExternalContext{
			Name:            mod.Name,
			ParentName:      parentName,
			InternalContext: internalCtx,
			PublicContext:   publicCtx,
		})
	}
	request := &payload.ExternalContextsRequest{
		Modules: modulesForRequest,
	}

	// ------------------------------------------------------------
	// 3. Call LLM.
	// ------------------------------------------------------------
	sysPrompt := `You are a prompt engineer, structuring information about an application's code base 
so context can be provided to an LLM in the most efficient way. 
You are tasked with determining the *external context* of a module hierarchy.
For every module you receive:
  • Internal Context – a description of the files inside the module.
  • Public  Context – a description visible to other modules.
  • Parent – the name of the module's parent. If the module has no parent, it is the root module of the application.

Your job is to produce, **for each module**, an "external context" string – a
concise explanation of where the module lives in the hierarchy and what lives
*outside* of it that might be relevant to understand its role.

Return your answer as JSON following the schema you have been provided.`

	resp, err := llm.GetModuleExternalContexts(cfg, sysPrompt, request)
	if err != nil {
		return err
	}

	// ------------------------------------------------------------
	// 4. Persist results back into the module annotations.
	// ------------------------------------------------------------
	for _, ext := range resp.Modules {
		if mod, ok := moduleMap[ext.Name]; ok {
			if mod.Annotation == nil {
				mod.Annotation = &Annotation{}
			}
			mod.Annotation.ExternalContext = ext.ExternalContext
		} else {
			logging.Log.Warnf("  WARNING: module %q not found in module map\n", ext.Name)
		}
	}

	return nil
}

// collectAllModules returns a depth-first slice containing the provided module
// and all of its children.
func collectAllModules(root *Module) []*Module {
	if root == nil {
		return nil
	}
	var out []*Module
	var walk func(*Module)
	walk = func(mod *Module) {
		out = append(out, mod)
		for _, child := range mod.Modules {
			walk(child)
		}
	}
	walk(root)
	return out
}