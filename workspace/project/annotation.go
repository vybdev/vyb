package project

import (
	"fmt"
	"github.com/dangazineu/vyb/llm/openai"
	"github.com/dangazineu/vyb/llm/payload"
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
func annotate(metadata *Metadata, sysfs fs.FS) error {
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
			continue
		}
		// Capture m for the goroutine.
		go func(mod *Module) {
			// Wait for all submodules to complete.
			for _, sub := range mod.Modules {
				<-dones[sub]
			}
			err := addOrUpdateSelfContainedContext(mod, sysfs)
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
	return addOrUpdateExternalContext(root)
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

// buildModuleContextRequest converts a *Module hierarchy to a *payload.ModuleSelfContainedContextRequest tree.
func buildModuleContextRequest(m *Module) *payload.ModuleSelfContainedContextRequest {
	if m == nil {
		return nil
	}

	// Collect file paths relative to this module (just the file names).
	var paths []string
	for _, f := range m.Files {
		paths = append(paths, f.Name)
	}

	// Recursively process sub-modules.
	var subs []*payload.ModuleSelfContainedContextRequest
	for _, sm := range m.Modules {
		subs = append(subs, buildModuleContextRequest(sm))
	}

	// For the root module (name == ".") we omit the ModuleSelfContainedContext so we don’t get a "# ." header.
	var ctxPtr *payload.ModuleSelfContainedContext
	//if m.Name != "." {
	//	ctxPtr = &payload.ModuleSelfContainedContext{Name: m.Name}
	//}

	return &payload.ModuleSelfContainedContextRequest{
		FilePaths:   paths,
		Directories: m.Directories,
		ModuleCtx:   ctxPtr,
		SubModules:  subs,
	}
}

// addOrUpdateSelfContainedContext calls OpenAI to construct the internal and public context of a given module.
func addOrUpdateSelfContainedContext(m *Module, sysfs fs.FS) error {
	// Build the ModuleSelfContainedContextRequest tree starting from this module.
	req := buildModuleContextRequest(m)

	fmt.Printf("annotating module %q\n", m.Name)

	// Construct user message including the files for this module.
	userMsg, err := payload.BuildModuleContextUserMessage(sysfs, req)
	if err != nil {
		return fmt.Errorf("failed to build user message: %w", err)
	}

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

	context, err := openai.GetModuleContext(systemMessage, userMsg)

	fmt.Printf("  Got response for module %q\n", m.Name)

	if err != nil {
		return fmt.Errorf("failed to call openAI: %w", err)
	}

	if m.Annotation == nil {
		m.Annotation = &Annotation{}
	}

	if context.InternalContext != "" {
		if m.Annotation.InternalContext != "" {
			fmt.Printf("  Overriding field `InternalContext` of module %q.\n", m.Name)
		} else {
			fmt.Printf("  Creating field `InternalContext` of module %q.\n", m.Name)
		}
		m.Annotation.InternalContext = context.InternalContext
	}
	if context.PublicContext != "" {
		if m.Annotation.PublicContext != "" {
			fmt.Printf("  Overriding field `PublicContext` of module %q.\n", m.Name)
		} else {
			fmt.Printf("  Creating field `PublicContext` of module %q.\n", m.Name)
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
func addOrUpdateExternalContext(m *Module) error {
	if m == nil {
		return nil
	}

	// ------------------------------------------------------------
	// 1. Collect modules (m + all descendants) & prepare name->ptr map.
	// ------------------------------------------------------------
	modules := collectAllModules(m)
	moduleMap := make(map[string]*Module, len(modules))
	for _, mod := range modules {
		moduleMap[mod.Name] = mod
	}

	// ------------------------------------------------------------
	// 2. Build user-message containing internal & public context that the
	//    LLM will use to infer external context.
	// ------------------------------------------------------------
	var sb strings.Builder
	for _, mod := range modules {
		sb.WriteString(fmt.Sprintf("## Module: %s\n", mod.Name))
		if mod.Parent != nil {
			sb.WriteString(fmt.Sprintf("### Parent: %s\n", mod.Parent.Name))
		}
		if mod.Annotation != nil {
			if mod.Annotation.InternalContext != "" {
				sb.WriteString("### Internal Context\n")
				sb.WriteString(mod.Annotation.InternalContext + "\n")
			}
			if mod.Annotation.PublicContext != "" {
				sb.WriteString("### Public Context\n")
				sb.WriteString(mod.Annotation.PublicContext + "\n")
			}
		}
	}
	userMsg := sb.String()

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

	resp, err := openai.GetModuleExternalContexts(sysPrompt, userMsg)
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
			fmt.Printf("  WARNING: module %q not found in module map\n", ext.Name)
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
