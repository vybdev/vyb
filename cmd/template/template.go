package template

import (
	"fmt"
	"github.com/vybdev/vyb/config"
	"os"
	"path/filepath"
	"strings"

	"github.com/cbroglie/mustache"
	"github.com/spf13/cobra"
	"github.com/vybdev/vyb/llm"
	"github.com/vybdev/vyb/llm/payload"
	"github.com/vybdev/vyb/workspace/context"
	"github.com/vybdev/vyb/workspace/matcher"
	"github.com/vybdev/vyb/workspace/project"
	"github.com/vybdev/vyb/workspace/selector"
)

var systemExclusionPatterns = []string{
	".git/",
	".gitignore",
	".vyb/",
	// the following files are excluded here just temporarily until a .vybignore logic is implemented
	"LICENSE",
	"go.sum",
}

type Model struct {
	Family config.ModelFamily `yaml:"family"`
	Size   config.ModelSize   `yaml:"size"`
}

type Definition struct {
	Name  string `yaml:"name"`
	Model Model  `yaml:"model"`

	// ArgExclusionPatterns specifies patterns for files that should be excluded as command arguments.
	ArgExclusionPatterns []string `yaml:"argExclusionPatterns"`

	// ArgInclusionPatterns specifies a list of matching patterns for files that can be used as command arguments.
	ArgInclusionPatterns []string `yaml:"argInclusionPatterns"`

	// RequestExclusionPatterns specifies patterns for files that should be excluded from the request payload.
	RequestExclusionPatterns []string `yaml:"requestExclusionPatterns"`
	// RequestInclusionPatterns specifies patterns for files that should be included in the request payload.
	RequestInclusionPatterns []string `yaml:"requestInclusionPatterns"`

	// ModificationExclusionPatterns specifies patterns for files that should never be modified when executing this command.
	ModificationExclusionPatterns []string `yaml:"modificationExclusionPatterns"`
	// ModificationInclusionPatterns specifies patterns for files that could be modified when executing this command.
	ModificationInclusionPatterns []string `yaml:"modificationInclusionPatterns"`

	// Prompt specifies the command-specific user prompt that should be included in the LLM request
	Prompt string `yaml:"prompt"`
	// TargetSpecificPrompt specifies additional instructions to be included in the user prompt, if a target is provided.
	TargetSpecificPrompt string `yaml:"targetSpecificPrompt"`
	// ShortDescription is a developer-provided description for the command.
	ShortDescription string `yaml:"shortDescription"`
	// LongDescription is a developer-provided description for the command.
	LongDescription string `yaml:"longDescription"`
}

// prepareExecutionContext builds and validates an ExecutionContext based on
// the current working directory and an optional *target* argument.
func prepareExecutionContext(target *string) (*context.ExecutionContext, error) {
	absWorkingDir, err := filepath.Abs(".")
	if err != nil {
		return nil, fmt.Errorf("failed to determine absolute working dir: %w", err)
	}

	// Locate project root using existing helper.
	distToRoot, err := project.FindDistanceToRoot(absWorkingDir)
	if err != nil {
		return nil, fmt.Errorf("unable to determine project root: %w", err)
	}

	absRoot, err := filepath.Abs(distToRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to determine absolute project root: %w", err)
	}

	// Resolve absolute target (if any).
	var absTarget *string
	if target != nil {
		at, err := filepath.Abs(*target)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve target %s: %w", *target, err)
		}
		absTarget = &at
	}

	// Let ExecutionContext enforce invariants.
	ec, err := context.NewExecutionContext(absRoot, absWorkingDir, absTarget)
	if err != nil {
		return nil, err
	}
	return ec, nil
}

func execute(cmd *cobra.Command, args []string, def *Definition) error {
	if len(def.ArgInclusionPatterns) == 0 && len(args) > 0 {
		return fmt.Errorf("command \"%s\" expects no arguments, but got %v", cmd.Use, args)
	}

	// ---------------------------
	// Retrieve --all flag value.
	// ---------------------------
	includeAll, _ := cmd.Flags().GetBool("all")

	var target *string
	if len(args) > 0 {
		target = &args[0]
	}

	ec, err := prepareExecutionContext(target)
	if err != nil {
		return err
	}

	absRoot := ec.ProjectRoot

	// relTarget is the *file* provided by the user (if any), relative to root.
	var relTarget *string
	if target != nil {
		absTarget, _ := filepath.Abs(*target)
		rt, _ := filepath.Rel(absRoot, absTarget)
		relTarget = &rt
	}

	rootFS := os.DirFS(absRoot)

	cfg, err := config.Load(absRoot)
	if err != nil {
		return err
	}

	if relTarget != nil {
		if !matcher.IsIncluded(rootFS, *relTarget, append(systemExclusionPatterns, def.ArgExclusionPatterns...), def.ArgInclusionPatterns) {
			return fmt.Errorf("command \"%s\" does not support given target %s", cmd.Use, *relTarget)
		}
	}

	files, err := selector.Select(rootFS, ec, append(systemExclusionPatterns, def.ArgExclusionPatterns...), def.ArgInclusionPatterns)
	if err != nil {
		return err
	}

	// ------------------------------------------------------------
	// Load stored metadata (with annotations) and merge with a fresh
	// snapshot produced from the current filesystem state. This
	// guarantees we operate with up-to-date file information while
	// keeping previously generated annotations intact.
	// ------------------------------------------------------------
	storedMeta, err := project.LoadMetadata(absRoot)
	if err != nil {
		return err
	}
	freshMeta, err := project.BuildMetadataFS(rootFS)
	if err != nil {
		return err
	}

	// Validate that the module name sets are identical.
	if !equalModuleNameSets(storedMeta.Modules, freshMeta.Modules) {
		return fmt.Errorf("module hierarchy mismatch between stored metadata and filesystem snapshot – please run 'vyb update' first")
	}

	// Merge – keep annotations from storedMeta, replace structure from freshMeta.
	storedMeta.Patch(freshMeta)
	meta := storedMeta

	// ------------------------------------------------------------
	// Unless --all is provided, filter out files that belong to
	// descendant modules of the target module (i.e. keep only files
	// whose module == targetModule).
	// ------------------------------------------------------------
	if !includeAll && meta.Modules != nil {
		relTargetDir, _ := filepath.Rel(absRoot, ec.TargetDir)
		relTargetDir = filepath.ToSlash(relTargetDir)
		targetModule := project.FindModule(meta.Modules, relTargetDir)
		if targetModule != nil {
			var filtered []string
			for _, f := range files {
				if project.FindModule(meta.Modules, f) == targetModule {
					filtered = append(filtered, f)
				}
			}
			files = filtered
		}
	}

	fmt.Printf("The following files will be included in the request:\n")
	for _, file := range files {
		if relTarget != nil && file == *relTarget {
			fmt.Printf("  %s <-- TARGET\n", file)
		} else {
			fmt.Printf("  %s\n", file)
		}
	}

	userMsg, err := buildExtendedUserMessage(rootFS, meta, ec, files)
	if err != nil {
		return err
	}

	promptGeneralInstructions, _ := embedded.ReadFile("embedded/prompts/instructions.md.mustache")
	tmpl, err := mustache.ParseString(string(promptGeneralInstructions))
	if err != nil {
		return err
	}

	rendered, err := tmpl.Render(def)
	if err != nil {
		return err
	}

	systemMessage := rendered

	proposal, err := llm.GetWorkspaceChangeProposals(cfg, def.Model.Family, def.Model.Size, systemMessage, userMsg)
	if err != nil {
		return err
	}

	// --------------------------------------------------------
	// Validate that every file in the proposal is allowed to be modified.
	// --------------------------------------------------------
	invalidFiles := []string{}

	// helper closure to assert path containment using absolute paths.
	isWithinDir := func(dir, candidate string) bool {
		dir = filepath.Clean(dir)
		candidate = filepath.Clean(candidate)
		if dir == candidate {
			return true
		}
		return strings.HasPrefix(candidate, dir+string(os.PathSeparator))
	}

	for _, prop := range proposal.Proposals {
		// 1. Pattern based validation (existing behaviour).
		if !matcher.IsIncluded(rootFS, prop.FileName, append(systemExclusionPatterns, def.ModificationExclusionPatterns...), def.ModificationInclusionPatterns) {
			invalidFiles = append(invalidFiles, prop.FileName)
			continue
		}
		// 2. Must reside within the working_dir using absolute paths.
		absProp := filepath.Join(absRoot, prop.FileName)
		if !isWithinDir(ec.WorkingDir, absProp) {
			invalidFiles = append(invalidFiles, prop.FileName+" (outside working_dir)")
		}
	}

	if len(invalidFiles) > 0 {
		return fmt.Errorf("change proposal contains modifications to unallowed files: %v", invalidFiles)
	}

	if err := applyProposals(absRoot, proposal.Proposals); err != nil {
		return err
	}

	fmt.Printf("Change summary: %s\n\n", proposal.Summary)
	fmt.Printf("Change description: %s\n\n", proposal.Description)
	fmt.Printf("Changed files: \n")
	for _, file := range proposal.Proposals {
		fmt.Printf("  %s -- delete? %v\n", file.FileName, file.Delete)
	}

	return nil
}

// applyProposals applies all file modifications as proposed by the LLM.
func applyProposals(absRoot string, proposals []payload.FileChangeProposal) error {
	for _, prop := range proposals {
		absPath := filepath.Join(absRoot, prop.FileName)
		if prop.Delete {
			if err := os.Remove(absPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to delete file %s: %w", absPath, err)
			}
			fmt.Printf("Deleted file: %s\n", prop.FileName)
		} else {
			dir := filepath.Dir(absPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
			if err := os.WriteFile(absPath, []byte(prop.Content), 0644); err != nil {
				return fmt.Errorf("failed to write to file %s: %w", absPath, err)
			}
			fmt.Printf("Modified file: %s\n", prop.FileName)
		}
	}
	return nil
}

func Register(rootCmd *cobra.Command) error {
	// Register subcommands.
	defs := load()
	for _, def := range defs {
		cmd := &cobra.Command{
			Use:   def.Name,
			Long:  def.LongDescription,
			Short: def.ShortDescription,
			RunE: func(cmd *cobra.Command, args []string) error {
				return execute(cmd, args, def)
			},
		}
		cmd.Flags().BoolP("all", "a", false, "include all files, even those in descendant modules")
		rootCmd.AddCommand(cmd)
	}
	return nil
}

// collectModuleNames flattens a module tree into a set of names.
func collectModuleNames(m *project.Module, set map[string]struct{}) {
	if m == nil {
		return
	}
	set[m.Name] = struct{}{}
	for _, c := range m.Modules {
		collectModuleNames(c, set)
	}
}

// equalModuleNameSets returns true when both module trees enumerate exactly
// the same set of module names.
func equalModuleNameSets(a, b *project.Module) bool {
	sa, sb := map[string]struct{}{}, map[string]struct{}{}
	collectModuleNames(a, sa)
	collectModuleNames(b, sb)
	if len(sa) != len(sb) {
		return false
	}
	for k := range sa {
		if _, ok := sb[k]; !ok {
			return false
		}
	}
	return true
}
