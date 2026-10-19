package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/adapters"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/execution"
	sqliteindex "github.com/SummitExplorerJD/Engineering-Toolkit/internal/index/sqlite"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/plan"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/skills"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/speckit"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/workspace"
)

const versionSchema = "engkit.version/v1alpha1"

// Version is replaced through linker flags for release builds.
var Version = "dev"

// Run executes the CLI and returns a process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		writeUsage(stderr)
		return 2
	}

	var err error
	switch args[0] {
	case "help", "-h", "--help":
		writeUsage(stdout)
		return 0
	case "version":
		err = runVersion(args[1:], stdout, stderr)
	case "workspace":
		err = runWorkspace(args[1:], stdout, stderr)
	case "context":
		err = runContext(args[1:], stdout, stderr)
	case "adapters":
		err = runAdapters(args[1:], stdout, stderr)
	case "speckit":
		err = runSpecKit(args[1:], stdout, stderr)
	case "skills":
		err = runSkills(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		writeUsage(stderr)
		return 2
	}

	if err == nil {
		return 0
	}
	var usageErr *usageError
	if errors.As(err, &usageErr) {
		fmt.Fprintln(stderr, usageErr)
		return 2
	}
	fmt.Fprintf(stderr, "engkit: %v\n", err)
	return 1
}

func runSkills(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return &usageError{message: "skills requires a subcommand: list, inspect, or validate"}
	}
	switch args[0] {
	case "list":
		flags := newFlagSet("skills list", stderr)
		root := flags.String("root", ".", "workspace root to inspect")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "skills list does not accept positional arguments"}
		}
		result, err := skills.List(*root)
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	case "inspect":
		flags := newFlagSet("skills inspect", stderr)
		root := flags.String("root", ".", "workspace root to inspect")
		id := flags.String("id", "", "normalized skill identifier")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "skills inspect does not accept positional arguments"}
		}
		if *id == "" {
			return &usageError{message: "skills inspect requires --id"}
		}
		result, err := skills.Inspect(*root, *id)
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	case "validate":
		flags := newFlagSet("skills validate", stderr)
		root := flags.String("root", ".", "workspace root to validate")
		id := flags.String("id", "", "optional normalized skill identifier")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "skills validate does not accept positional arguments"}
		}
		result, err := skills.Validate(*root, *id)
		if err != nil {
			return err
		}
		if err := writeJSON(stdout, result); err != nil {
			return err
		}
		if !result.Valid {
			return skills.ErrValidationFailed
		}
		return nil
	default:
		return &usageError{message: fmt.Sprintf("unknown skills subcommand %q", args[0])}
	}
}

func runAdapters(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return &usageError{message: "adapters requires a subcommand: list or inspect"}
	}
	switch args[0] {
	case "list":
		flags := newFlagSet("adapters list", stderr)
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "adapters list does not accept positional arguments"}
		}
		return writeJSON(stdout, adapters.List())
	case "inspect":
		flags := newFlagSet("adapters inspect", stderr)
		root := flags.String("root", ".", "workspace root to inspect")
		provider := flags.String("provider", "", "provider: codex, claude, or opencode")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "adapters inspect does not accept positional arguments"}
		}
		result, err := adapters.Inspect(*root, *provider)
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	default:
		return &usageError{message: fmt.Sprintf("unknown adapters subcommand %q", args[0])}
	}
}

func runSpecKit(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return &usageError{message: "speckit requires the plan subcommand"}
	}
	if args[0] != "plan" {
		return &usageError{message: fmt.Sprintf("unknown speckit subcommand %q", args[0])}
	}
	flags := newFlagSet("speckit plan", stderr)
	root := flags.String("root", ".", "workspace root to inspect")
	provider := flags.String("integration", "", "integration: codex, claude, or opencode")
	version := flags.String("version", "", "pinned official Spec Kit release tag")
	if err := flags.Parse(args[1:]); err != nil {
		return &usageError{message: err.Error()}
	}
	if flags.NArg() != 0 {
		return &usageError{message: "speckit plan does not accept positional arguments"}
	}
	result, err := speckit.NewPlan(*root, *provider, *version)
	if err != nil {
		return err
	}
	return writeJSON(stdout, result)
}

func runContext(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return &usageError{message: "context requires a subcommand: rebuild, status, or search"}
	}
	switch args[0] {
	case "rebuild":
		flags := newFlagSet("context rebuild", stderr)
		root := flags.String("root", ".", "initialized workspace root")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "context rebuild does not accept positional arguments"}
		}
		result, err := sqliteindex.Rebuild(context.Background(), *root, nil)
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	case "status":
		flags := newFlagSet("context status", stderr)
		root := flags.String("root", ".", "initialized workspace root")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "context status does not accept positional arguments"}
		}
		result, err := sqliteindex.Inspect(context.Background(), *root)
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	case "search":
		flags := newFlagSet("context search", stderr)
		root := flags.String("root", ".", "initialized workspace root")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 1 {
			return &usageError{message: "context search requires exactly one query argument"}
		}
		result, err := sqliteindex.Search(context.Background(), *root, flags.Arg(0))
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	default:
		return &usageError{message: fmt.Sprintf("unknown context subcommand %q", args[0])}
	}
}

type usageError struct {
	message string
}

func (e *usageError) Error() string { return e.message }

func runVersion(args []string, stdout, stderr io.Writer) error {
	flags := newFlagSet("version", stderr)
	asJSON := flags.Bool("json", false, "write version information as JSON")
	if err := flags.Parse(args); err != nil {
		return &usageError{message: err.Error()}
	}
	if flags.NArg() != 0 {
		return &usageError{message: "version does not accept positional arguments"}
	}
	if *asJSON {
		return writeJSON(stdout, struct {
			SchemaVersion string `json:"schema_version"`
			Version       string `json:"version"`
		}{SchemaVersion: versionSchema, Version: Version})
	}
	_, err := fmt.Fprintf(stdout, "engkit %s\n", Version)
	return err
}

func runWorkspace(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return &usageError{message: "workspace requires a subcommand: inspect, plan-init, or init"}
	}

	switch args[0] {
	case "inspect":
		return runWorkspaceInspect(args[1:], stdout, stderr)
	case "plan-init":
		return runWorkspacePlanInit(args[1:], stdout, stderr)
	case "init":
		return runWorkspaceInit(args[1:], stdout, stderr)
	default:
		return &usageError{message: fmt.Sprintf("unknown workspace subcommand %q", args[0])}
	}
}

func runWorkspaceInit(args []string, stdout, stderr io.Writer) error {
	flags := newFlagSet("workspace init", stderr)
	root := flags.String("root", ".", "workspace root to initialize")
	approvedPlanID := flags.String("approve", "", "complete plan ID approved for execution")
	if err := flags.Parse(args); err != nil {
		return &usageError{message: err.Error()}
	}
	if flags.NArg() != 0 {
		return &usageError{message: "workspace init does not accept positional arguments"}
	}

	snapshot, err := workspace.Inspect(*root)
	if err != nil {
		return err
	}
	preview, err := plan.NewWorkspaceInit(snapshot)
	if err != nil {
		return err
	}
	result, err := execution.Initialize(preview, *approvedPlanID, nil)
	if err != nil {
		return err
	}
	return writeJSON(stdout, result)
}

func runWorkspaceInspect(args []string, stdout, stderr io.Writer) error {
	flags := newFlagSet("workspace inspect", stderr)
	root := flags.String("root", ".", "workspace root to inspect")
	if err := flags.Parse(args); err != nil {
		return &usageError{message: err.Error()}
	}
	if flags.NArg() != 0 {
		return &usageError{message: "workspace inspect does not accept positional arguments"}
	}

	snapshot, err := workspace.Inspect(*root)
	if err != nil {
		return err
	}
	return writeJSON(stdout, snapshot)
}

func runWorkspacePlanInit(args []string, stdout, stderr io.Writer) error {
	flags := newFlagSet("workspace plan-init", stderr)
	root := flags.String("root", ".", "workspace root to plan")
	if err := flags.Parse(args); err != nil {
		return &usageError{message: err.Error()}
	}
	if flags.NArg() != 0 {
		return &usageError{message: "workspace plan-init does not accept positional arguments"}
	}

	snapshot, err := workspace.Inspect(*root)
	if err != nil {
		return err
	}
	preview, err := plan.NewWorkspaceInit(snapshot)
	if err != nil {
		return err
	}
	return writeJSON(stdout, preview)
}

func newFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(stderr)
	return flags
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("write JSON output: %w", err)
	}
	return nil
}

func writeUsage(writer io.Writer) {
	fmt.Fprintln(writer, `Engineering Toolkit

Usage:
  engkit version [--json]
  engkit workspace inspect [--root PATH]
  engkit workspace plan-init [--root PATH]
  engkit workspace init [--root PATH] --approve PLAN_ID
  engkit context rebuild [--root PATH]
  engkit context status [--root PATH]
  engkit context search [--root PATH] QUERY
	engkit adapters list
	engkit adapters inspect [--root PATH] --provider PROVIDER
	engkit speckit plan [--root PATH] --integration PROVIDER --version RELEASE_TAG
	engkit skills list [--root PATH]
	engkit skills inspect [--root PATH] --id SKILL_ID
	engkit skills validate [--root PATH] [--id SKILL_ID]

inspect and plan-init are read-only. init revalidates the workspace and only
publishes .engkit when --approve exactly matches the current plan ID. Adapter
inspection and Spec Kit planning are read-only; no Spec Kit execution command
is provided in this milestone. Skill listing, inspection, and validation are
read-only and never execute package resources.`)
}
