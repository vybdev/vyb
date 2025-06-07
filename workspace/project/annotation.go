package project

import (
	"fmt"
	"github.com/dangazineu/vyb/llm/openai"
	"github.com/dangazineu/vyb/llm/payload"
	"io/fs"
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
	return nil
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

// TODO(vyb): implement this function, along with all the other changes needed for it to work. This function is
// similar to addOrUpdateSelfContainedContext, but while that function changes the InternalContext and the PublicContext
// of a single module at a time, addOrUpdateExternalContext should add or update the ExternalContext of all modules
// within the given module (including all their child modules as well). To achieve that, this function should send the
// PublicContext and Private context of every module it finds. It should also include the name of the parent module for
// each module in the payload, so it is easier for the LLM to infer the hierarchy. The prompt should explain to the LLM
// that the ExternalContext is about where in the hierarchy a given module is located, and what is outside of it.
// The response should then be mapped back to the original modules.
// In addition to the code in this function, you will need new data structures to represent the request that is sent to
// the LLM, you may need new a data structure to represent the response (alongside a json schema file). You will also
// need a new function in the openai module to actually interact with the LLM. Please refer to the code in
// addOrUpdateSelfContainedContext, as well as all the functions it calls in order to implement the functionality.
func addOrUpdateExternalContext(m *Module, sysfs fs.FS) error {
	return nil
}
