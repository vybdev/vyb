package template

import (
    "fmt"
    "io/fs"
    "path/filepath"
    "strings"

    "github.com/dangazineu/vyb/llm/payload"
    "github.com/dangazineu/vyb/workspace/context"
    "github.com/dangazineu/vyb/workspace/project"
)

// buildExtendedUserMessage composes the user-message payload that will be
// sent to the LLM. It prepends module context information — as dictated
// by the specification — before the raw file contents. When metadata is
// nil or when any contextual information is missing the function falls
// back gracefully, emitting only what is available.
func buildExtendedUserMessage(rootFS fs.FS, meta *project.Metadata, ec *context.ExecutionContext, filePaths []string) (string, error) {
    // If metadata is missing we revert to the original behaviour – emit
    // just the files.
    if meta == nil || meta.Modules == nil {
        return payload.BuildUserMessage(rootFS, filePaths)
    }

    // Helper to clean/normalise relative paths.
    rel := func(abs string) string {
        if abs == "" {
            return ""
        }
        r, _ := filepath.Rel(ec.ProjectRoot, abs)
        return filepath.ToSlash(r)
    }

    workingRel := rel(ec.WorkingDir)
    targetRel := rel(ec.TargetDir)

    workingMod := project.FindModule(meta.Modules, workingRel)
    targetMod := project.FindModule(meta.Modules, targetRel)

    if workingMod == nil || targetMod == nil {
        return "", fmt.Errorf("failed to locate working and target modules")
    }

    var sb strings.Builder

    // ------------------------------------------------------------
    // 1. External context of working module.
    // ------------------------------------------------------------
    if ann := workingMod.Annotation; ann != nil && ann.ExternalContext != "" {
        sb.WriteString(fmt.Sprintf("# Module: `%s`\n", workingMod.Name))
        sb.WriteString("## External Context\n")
        sb.WriteString(ann.ExternalContext + "\n")
    }

    // ------------------------------------------------------------
    // 2. Internal context of modules between working and target.
    // ------------------------------------------------------------
    for m := targetMod.Parent; m != nil && m != workingMod; m = m.Parent {
        if ann := m.Annotation; ann != nil && ann.InternalContext != "" {
            sb.WriteString(fmt.Sprintf("# Module: `%s`\n", m.Name))
            sb.WriteString("## Internal Context\n")
            sb.WriteString(ann.InternalContext + "\n")
        }
    }

    // ------------------------------------------------------------
    // 3. Public context of sibling modules along the path from the
    //    parent of the target module up to (and including) the working
    //    module. This replaces the previous logic that only considered
    //    direct children of the working module.
    // ------------------------------------------------------------

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
                sb.WriteString(fmt.Sprintf("# Module: `%s`\n", child.Name))
                sb.WriteString("## Public Context\n")
                sb.WriteString(ann.PublicContext + "\n")
            }
        }
        if ancestor == workingMod {
            break
        }
    }

    // ------------------------------------------------------------
    // 4. Public context of immediate sub-modules of target module.
    // ------------------------------------------------------------
    for _, child := range targetMod.Modules {
        if ann := child.Annotation; ann != nil && ann.PublicContext != "" {
            sb.WriteString(fmt.Sprintf("# Module: `%s`\n", child.Name))
            sb.WriteString("## Public Context\n")
            sb.WriteString(ann.PublicContext + "\n")
        }
    }

    // ------------------------------------------------------------
    // 5. Append file contents (only files from target module were
    //    selected by selector.Select).
    // ------------------------------------------------------------
    filesMsg, err := payload.BuildUserMessage(rootFS, filePaths)
    if err != nil {
        return "", err
    }
    sb.WriteString(filesMsg)

    return sb.String(), nil
}
