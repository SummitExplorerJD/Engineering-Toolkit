package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/adapters"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/catalog"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/composition"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/compositionexec"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/contextsearch"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/contextview"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/doctor"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/execution"
	sqliteindex "github.com/SummitExplorerJD/Engineering-Toolkit/internal/index/sqlite"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/initfacade"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/localintelligence"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/mcpserver"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/ollama"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/plan"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/profiles"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/providersync"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/providersyncexec"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/skillexec"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/skillplan"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/skills"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/speckit"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/templates"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/workspace"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/workspaceconfig"
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
	if prefix, ok := helpPrefix(args); ok && writeScopedUsage(stdout, prefix) {
		return 0
	}

	var err error
	switch args[0] {
	case "help", "-h", "--help":
		writeUsage(stdout)
		return 0
	case "version":
		err = runVersion(args[1:], stdout, stderr)
	case "doctor":
		err = runDoctor(args[1:], stdout, stderr)
	case "init":
		err = runInit(args[1:], stdout, stderr)
	case "mcp":
		err = runMCP(args[1:], stderr)
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
	case "catalog":
		err = runCatalog(args[1:], stdout, stderr)
	case "templates":
		err = runTemplates(args[1:], stdout, stderr)
	case "profiles":
		err = runProfiles(args[1:], stdout, stderr)
	case "advise":
		err = runAdvise(args[1:], stdout, stderr)
	case "project":
		err = runProject(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		writeUsage(stderr)
		return 2
	}

	if err == nil {
		return 0
	}
	if errors.Is(err, doctor.ErrUnhealthy) {
		return 1
	}
	var usageErr *usageError
	if errors.As(err, &usageErr) {
		fmt.Fprintln(stderr, usageErr)
		return 2
	}
	fmt.Fprintf(stderr, "engkit: %v\n", err)
	return 1
}

func runMCP(args []string, stderr io.Writer) error {
	if len(args) == 0 {
		return &usageError{message: "mcp requires the serve subcommand"}
	}
	if args[0] != "serve" {
		return &usageError{message: fmt.Sprintf("unknown mcp subcommand %q", args[0])}
	}
	flags := newFlagSet("mcp serve", stderr)
	transport := flags.String("transport", "", "protocol transport; only stdio is supported")
	root := flags.String("root", "", "explicit workspace root")
	if err := flags.Parse(args[1:]); err != nil {
		return &usageError{message: err.Error()}
	}
	if flags.NArg() != 0 {
		return &usageError{message: "mcp serve does not accept positional arguments"}
	}
	if *transport != "stdio" {
		return &usageError{message: "mcp serve requires --transport stdio"}
	}
	if *root == "" {
		return &usageError{message: "mcp serve requires --root"}
	}
	server, err := mcpserver.New(*root, mcpserver.Dependencies{
		Version:       Version,
		ContextSearch: contextsearch.OptionsSearchFunc(sqliteindex.SearchWithOptions),
		ContextReader: contextview.ReaderFuncs{
			StatusFunc:  sqliteindex.Inspect,
			SourcesFunc: sqliteindex.ListSources,
			ReadFunc:    sqliteindex.ReadIndexed,
			TreeFunc:    sqliteindex.Tree,
		},
	})
	if err != nil {
		return err
	}
	return server.RunStdio(context.Background())
}

func runInit(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return &usageError{message: "init requires a subcommand: plan or apply"}
	}
	if args[0] != "plan" && args[0] != "apply" {
		return &usageError{message: fmt.Sprintf("unknown init subcommand %q", args[0])}
	}

	operation := args[0]
	flags := newFlagSet("init "+operation, stderr)
	target := flags.String("target", "", "explicit initialization target")
	profile := flags.String("profile", "", "exact profile selection ID@VERSION")
	values := stringMapFlag{}
	flags.Var(&values, "set", "typed profile value NAME=VALUE; repeatable")
	approved := ""
	if operation == "apply" {
		flags.StringVar(&approved, "approve", "", "complete selected plan ID approved for execution")
	}
	if err := flags.Parse(args[1:]); err != nil {
		return &usageError{message: err.Error()}
	}
	if flags.NArg() != 0 {
		return &usageError{message: "init " + operation + " does not accept positional arguments"}
	}
	if *target == "" {
		return &usageError{message: "init " + operation + " requires --target"}
	}
	if *profile == "" && len(values) != 0 {
		return &usageError{message: "init " + operation + " requires --profile when --set is used"}
	}
	if operation == "apply" && approved == "" {
		return &usageError{message: "init apply requires --approve"}
	}

	profileID, profileVersion := "", ""
	var err error
	if *profile != "" {
		profileID, profileVersion, err = parseSelection(*profile)
		if err != nil {
			return &usageError{message: err.Error()}
		}
	}
	classification, err := initfacade.Classify(*target, *profile != "")
	if err != nil {
		return err
	}

	switch classification.Lifecycle {
	case initfacade.WorkspaceInitialize:
		snapshot, err := workspace.Inspect(classification.Target)
		if err != nil {
			return err
		}
		preview, err := plan.NewWorkspaceInit(snapshot)
		if err != nil {
			return err
		}
		if operation == "plan" {
			return writeJSON(stdout, preview)
		}
		result, err := execution.Initialize(preview, approved, nil)
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	case initfacade.ProjectCreate, initfacade.WorkspaceCompose:
		loaded, catalogFS, err := catalog.Builtin()
		if err != nil {
			return err
		}
		var preview composition.Plan
		if classification.Lifecycle == initfacade.ProjectCreate {
			preview, err = composition.NewCreate(classification.Target, profileID, profileVersion, values, Version, loaded, catalogFS)
		} else {
			preview, err = composition.NewCompose(classification.Target, profileID, profileVersion, values, Version, loaded, catalogFS)
		}
		if err != nil {
			return err
		}
		if operation == "plan" {
			return writeJSON(stdout, preview)
		}
		result, err := compositionexec.Execute(preview, approved, loaded, catalogFS, compositionexec.Options{ProcessID: os.Getpid()})
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	default:
		return errors.New("unified initialization selected an unsupported lifecycle")
	}
}

func runDoctor(args []string, stdout, stderr io.Writer) error {
	flags := newFlagSet("doctor", stderr)
	root := flags.String("root", ".", "workspace root to inspect")
	scope := flags.String("scope", doctor.ScopeAll, "diagnostic scope: core, workspace, integrations, or all")
	var summary bool
	flags.BoolVar(&summary, "summary", false, "render a concise human-readable summary")
	flags.BoolVar(&summary, "s", false, "render a concise human-readable summary")
	if err := flags.Parse(args); err != nil {
		return &usageError{message: err.Error()}
	}
	if flags.NArg() != 0 {
		return &usageError{message: "doctor does not accept positional arguments"}
	}
	result, err := doctor.Inspect(context.Background(), doctor.Options{Root: *root, Scope: *scope, Version: Version})
	if err != nil {
		return &usageError{message: err.Error()}
	}
	if summary {
		if err := writeDoctorSummary(stdout, result); err != nil {
			return err
		}
	} else {
		if err := writeJSON(stdout, result); err != nil {
			return err
		}
	}
	if !result.Summary.Healthy {
		return doctor.ErrUnhealthy
	}
	return nil
}

func runCatalog(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return &usageError{message: "catalog requires the inspect subcommand"}
	}
	if args[0] != "inspect" {
		return &usageError{message: fmt.Sprintf("unknown catalog subcommand %q", args[0])}
	}
	flags := newFlagSet("catalog inspect", stderr)
	if err := flags.Parse(args[1:]); err != nil {
		return &usageError{message: err.Error()}
	}
	if flags.NArg() != 0 {
		return &usageError{message: "catalog inspect does not accept positional arguments"}
	}
	loaded, _, err := catalog.Builtin()
	if err != nil {
		return err
	}
	return writeJSON(stdout, loaded)
}

func runTemplates(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return &usageError{message: "templates requires a subcommand: list, inspect, or scaffold"}
	}
	switch args[0] {
	case "list":
		flags := newFlagSet("templates list", stderr)
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "templates list does not accept positional arguments"}
		}
		loaded, _, err := catalog.Builtin()
		if err != nil {
			return err
		}
		items := make([]templates.Reference, 0, len(loaded.Templates))
		for _, entry := range loaded.Templates {
			items = append(items, templates.Reference{ID: entry.ID, Version: entry.Version, SHA256: entry.SHA256})
		}
		return writeJSON(stdout, templates.ListResult{SchemaVersion: templates.ListSchema, CatalogID: loaded.ID, CatalogVersion: loaded.Version, Templates: templates.SortedReferences(items)})
	case "inspect":
		flags := newFlagSet("templates inspect", stderr)
		id := flags.String("id", "", "template identifier")
		version := flags.String("version", "", "exact template version")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "templates inspect does not accept positional arguments"}
		}
		if *id == "" || *version == "" {
			return &usageError{message: "templates inspect requires --id and --version"}
		}
		loaded, catalogFS, err := catalog.Builtin()
		if err != nil {
			return err
		}
		entry, ok := catalog.Find(loaded.Templates, *id, *version)
		if !ok {
			return fmt.Errorf("template %s@%s is not in the packaged catalog", *id, *version)
		}
		template, err := templates.Load(catalogFS, path.Join(entry.Path, "template.yaml"))
		if err != nil {
			return err
		}
		if template.ID != entry.ID || template.Version != entry.Version {
			return fmt.Errorf("template identity does not match its catalog entry")
		}
		return writeJSON(stdout, templates.Inspection{SchemaVersion: templates.InspectionSchema, CatalogID: loaded.ID, CatalogVersion: loaded.Version, Template: template, DirectorySHA256: entry.SHA256})
	case "scaffold":
		flags := newFlagSet("templates scaffold", stderr)
		output := flags.String("output", "", "explicit absent output directory")
		id := flags.String("id", "", "template identifier")
		version := flags.String("version", "", "exact template version")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "templates scaffold does not accept positional arguments"}
		}
		if *output == "" || *id == "" || *version == "" {
			return &usageError{message: "templates scaffold requires --output, --id, and --version"}
		}
		result, err := templates.Scaffold(*output, *id, *version)
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	default:
		return &usageError{message: fmt.Sprintf("unknown templates subcommand %q", args[0])}
	}
}

func runProfiles(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return &usageError{message: "profiles requires a subcommand: list, inspect, or recommend"}
	}
	switch args[0] {
	case "list":
		flags := newFlagSet("profiles list", stderr)
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "profiles list does not accept positional arguments"}
		}
		loaded, _, err := catalog.Builtin()
		if err != nil {
			return err
		}
		items := make([]profiles.Reference, 0, len(loaded.Profiles))
		for _, entry := range loaded.Profiles {
			items = append(items, profiles.Reference{ID: entry.ID, Version: entry.Version, SHA256: entry.SHA256})
		}
		return writeJSON(stdout, profiles.ListResult{SchemaVersion: profiles.ListSchema, CatalogID: loaded.ID, CatalogVersion: loaded.Version, Profiles: profiles.SortedReferences(items)})
	case "inspect":
		flags := newFlagSet("profiles inspect", stderr)
		id := flags.String("id", "", "profile identifier")
		version := flags.String("version", "", "exact profile version")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "profiles inspect does not accept positional arguments"}
		}
		if *id == "" || *version == "" {
			return &usageError{message: "profiles inspect requires --id and --version"}
		}
		loaded, catalogFS, err := catalog.Builtin()
		if err != nil {
			return err
		}
		entry, ok := catalog.Find(loaded.Profiles, *id, *version)
		if !ok {
			return fmt.Errorf("profile %s@%s is not in the packaged catalog", *id, *version)
		}
		profile, err := profiles.Load(catalogFS, path.Join(entry.Path, "profile.yaml"))
		if err != nil {
			return err
		}
		if profile.ID != entry.ID || profile.Version != entry.Version {
			return fmt.Errorf("profile identity does not match its catalog entry")
		}
		return writeJSON(stdout, profiles.Inspection{SchemaVersion: profiles.InspectionSchema, CatalogID: loaded.ID, CatalogVersion: loaded.Version, Profile: profile, DirectorySHA256: entry.SHA256})
	case "recommend":
		flags := newFlagSet("profiles recommend", stderr)
		root := flags.String("root", ".", "workspace root to inspect")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "profiles recommend does not accept positional arguments"}
		}
		loaded, catalogFS, err := catalog.Builtin()
		if err != nil {
			return err
		}
		result, err := profiles.Recommend(*root, loaded, catalogFS)
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	default:
		return &usageError{message: fmt.Sprintf("unknown profiles subcommand %q", args[0])}
	}
}

func runAdvise(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] != "profiles" {
		return &usageError{message: "advise requires the profiles subcommand"}
	}
	flags := newFlagSet("advise profiles", stderr)
	root := flags.String("root", "", "explicit initialized workspace root")
	goal := flags.String("goal", "", "bounded recommendation goal")
	modelOverride := flags.String("model", "", "explicit alternate allowlisted model tag")
	if err := flags.Parse(args[1:]); err != nil {
		return &usageError{message: err.Error()}
	}
	if flags.NArg() != 0 {
		return &usageError{message: "advise profiles does not accept positional arguments"}
	}
	if *root == "" || strings.TrimSpace(*goal) == "" {
		return &usageError{message: "advise profiles requires --root and --goal"}
	}
	loaded, catalogFS, err := catalog.Builtin()
	if err != nil {
		return err
	}
	recommendations, err := profiles.Recommend(*root, loaded, catalogFS)
	if err != nil {
		return err
	}
	config, err := workspaceconfig.Load(filepath.FromSlash(recommendations.Root))
	if err != nil {
		return err
	}
	model := ""
	enabled := config.Intelligence != nil && config.Intelligence.Enabled
	if enabled {
		model = config.Intelligence.Model
	}
	if *modelOverride != "" {
		if !enabled {
			return errors.New("--model requires local intelligence to be enabled in .engkit/config.yaml")
		}
		if !workspaceconfig.AllowedModel(*modelOverride) {
			return errors.New("--model must name an allowlisted local model tag")
		}
		model = *modelOverride
	}
	candidates := make([]localintelligence.Candidate, 0, len(recommendations.Recommendations))
	for _, recommendation := range recommendations.Recommendations {
		if !recommendation.Compatible {
			continue
		}
		candidates = append(candidates, localintelligence.Candidate{
			ID:                recommendation.ID + "@" + recommendation.Version,
			DeterministicRank: len(candidates) + 1,
			Evidence:          append([]string(nil), recommendation.Reasons...),
		})
	}
	result, err := localintelligence.Advise(context.Background(), localintelligence.Options{
		Root: recommendations.Root, Goal: *goal, Enabled: enabled, Model: model, Candidates: candidates,
		Context: localintelligence.ContextFunc(sqliteindex.AdviceExcerpts), Adviser: ollama.New(),
	})
	if err != nil {
		return err
	}
	return writeJSON(stdout, result)
}

func runProject(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return &usageError{message: "project requires a subcommand: inspect, plan-create, plan-compose, create, compose, or recover"}
	}
	switch args[0] {
	case "inspect":
		flags := newFlagSet("project inspect", stderr)
		root := flags.String("root", ".", "workspace root to inspect")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "project inspect does not accept positional arguments"}
		}
		loaded, catalogFS, err := catalog.Builtin()
		if err != nil {
			return err
		}
		result, err := composition.Inspect(*root, loaded, catalogFS)
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	case "plan-create":
		flags := newFlagSet("project plan-create", stderr)
		destination := flags.String("destination", "", "absent or empty project destination")
		profile := flags.String("profile", "", "exact profile selection ID@VERSION")
		values := stringMapFlag{}
		flags.Var(&values, "set", "typed profile value NAME=VALUE; repeatable")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "project plan-create does not accept positional arguments"}
		}
		if *destination == "" || *profile == "" {
			return &usageError{message: "project plan-create requires --destination and --profile"}
		}
		id, exactVersion, err := parseSelection(*profile)
		if err != nil {
			return &usageError{message: err.Error()}
		}
		loaded, catalogFS, err := catalog.Builtin()
		if err != nil {
			return err
		}
		result, err := composition.NewCreate(*destination, id, exactVersion, values, Version, loaded, catalogFS)
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	case "plan-compose":
		flags := newFlagSet("project plan-compose", stderr)
		root := flags.String("root", ".", "existing workspace root")
		profile := flags.String("profile", "", "exact profile selection ID@VERSION")
		values := stringMapFlag{}
		flags.Var(&values, "set", "typed profile value NAME=VALUE; repeatable")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "project plan-compose does not accept positional arguments"}
		}
		if *profile == "" {
			return &usageError{message: "project plan-compose requires --profile"}
		}
		id, exactVersion, err := parseSelection(*profile)
		if err != nil {
			return &usageError{message: err.Error()}
		}
		loaded, catalogFS, err := catalog.Builtin()
		if err != nil {
			return err
		}
		result, err := composition.NewCompose(*root, id, exactVersion, values, Version, loaded, catalogFS)
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	case "create":
		flags := newFlagSet("project create", stderr)
		destination := flags.String("destination", "", "absent or empty project destination")
		profile := flags.String("profile", "", "exact profile selection ID@VERSION")
		approved := flags.String("approve", "", "complete plan ID approved for execution")
		values := stringMapFlag{}
		flags.Var(&values, "set", "typed profile value NAME=VALUE; repeatable")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "project create does not accept positional arguments"}
		}
		if *destination == "" || *profile == "" {
			return &usageError{message: "project create requires --destination and --profile"}
		}
		id, exactVersion, err := parseSelection(*profile)
		if err != nil {
			return &usageError{message: err.Error()}
		}
		loaded, catalogFS, err := catalog.Builtin()
		if err != nil {
			return err
		}
		preview, err := composition.NewCreate(*destination, id, exactVersion, values, Version, loaded, catalogFS)
		if err != nil {
			return err
		}
		result, err := compositionexec.Execute(preview, *approved, loaded, catalogFS, compositionexec.Options{ProcessID: os.Getpid()})
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	case "compose":
		flags := newFlagSet("project compose", stderr)
		root := flags.String("root", ".", "initialized workspace root")
		profile := flags.String("profile", "", "exact profile selection ID@VERSION")
		approved := flags.String("approve", "", "complete plan ID approved for execution")
		values := stringMapFlag{}
		flags.Var(&values, "set", "typed profile value NAME=VALUE; repeatable")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "project compose does not accept positional arguments"}
		}
		if *profile == "" {
			return &usageError{message: "project compose requires --profile"}
		}
		id, exactVersion, err := parseSelection(*profile)
		if err != nil {
			return &usageError{message: err.Error()}
		}
		loaded, catalogFS, err := catalog.Builtin()
		if err != nil {
			return err
		}
		preview, err := composition.NewCompose(*root, id, exactVersion, values, Version, loaded, catalogFS)
		if err != nil {
			return err
		}
		result, err := compositionexec.Execute(preview, *approved, loaded, catalogFS, compositionexec.Options{ProcessID: os.Getpid()})
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	case "recover":
		flags := newFlagSet("project recover", stderr)
		root := flags.String("root", ".", "intended project destination or initialized workspace root")
		approved := flags.String("approve", "", "complete interrupted plan ID")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "project recover does not accept positional arguments"}
		}
		result, err := compositionexec.Recover(*root, *approved, compositionexec.Options{ProcessID: os.Getpid()})
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	default:
		return &usageError{message: fmt.Sprintf("unknown project subcommand %q", args[0])}
	}
}

type stringMapFlag map[string]string

func (values *stringMapFlag) String() string { return "" }

func (values *stringMapFlag) Set(raw string) error {
	name, value, found := strings.Cut(raw, "=")
	if !found || name == "" {
		return fmt.Errorf("--set requires NAME=VALUE")
	}
	if *values == nil {
		*values = stringMapFlag{}
	}
	if _, exists := (*values)[name]; exists {
		return fmt.Errorf("--set variable %q is duplicated", name)
	}
	(*values)[name] = value
	return nil
}

func parseSelection(value string) (string, string, error) {
	id, version, found := strings.Cut(value, "@")
	if !found || id == "" || version == "" || strings.Contains(version, "@") {
		return "", "", fmt.Errorf("profile must use exact ID@VERSION syntax")
	}
	return id, version, nil
}

func runSkills(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return &usageError{message: "skills requires a subcommand: list, inspect, validate, plan-create, plan-update, plan-remove, plan-sync, plan-unsync, create, update, remove, recover, sync, unsync, or recover-sync"}
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
	case "plan-create":
		flags := newFlagSet("skills plan-create", stderr)
		root := flags.String("root", ".", "workspace root to inspect")
		source := flags.String("source", "", "complete canonical source package")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "skills plan-create does not accept positional arguments"}
		}
		if *source == "" {
			return &usageError{message: "skills plan-create requires --source"}
		}
		result, err := skillplan.NewCreate(*root, *source)
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	case "plan-update":
		flags := newFlagSet("skills plan-update", stderr)
		root := flags.String("root", ".", "workspace root to inspect")
		id := flags.String("id", "", "normalized target skill identifier")
		source := flags.String("source", "", "complete canonical source package")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "skills plan-update does not accept positional arguments"}
		}
		if *id == "" || *source == "" {
			return &usageError{message: "skills plan-update requires --id and --source"}
		}
		result, err := skillplan.NewUpdate(*root, *id, *source)
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	case "plan-remove":
		flags := newFlagSet("skills plan-remove", stderr)
		root := flags.String("root", ".", "workspace root to inspect")
		id := flags.String("id", "", "normalized target skill identifier")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "skills plan-remove does not accept positional arguments"}
		}
		if *id == "" {
			return &usageError{message: "skills plan-remove requires --id"}
		}
		result, err := skillplan.NewRemove(*root, *id)
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	case "plan-sync":
		flags := newFlagSet("skills plan-sync", stderr)
		root := flags.String("root", ".", "workspace root to inspect")
		provider := flags.String("provider", "", "provider: codex, claude, or opencode")
		id := flags.String("id", "", "optional normalized canonical skill identifier")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "skills plan-sync does not accept positional arguments"}
		}
		if *provider == "" {
			return &usageError{message: "skills plan-sync requires --provider"}
		}
		result, err := providersync.NewPlan(*root, *provider, *id)
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	case "plan-unsync":
		flags := newFlagSet("skills plan-unsync", stderr)
		root := flags.String("root", ".", "workspace root to inspect")
		provider := flags.String("provider", "", "provider: codex, claude, or opencode")
		id := flags.String("id", "", "normalized canonical skill identifier")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "skills plan-unsync does not accept positional arguments"}
		}
		if *provider == "" || *id == "" {
			return &usageError{message: "skills plan-unsync requires --provider and --id"}
		}
		result, err := providersync.NewUnsyncPlan(*root, *provider, *id)
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	case "sync":
		flags := newFlagSet("skills sync", stderr)
		root := flags.String("root", ".", "initialized workspace root")
		provider := flags.String("provider", "", "provider: codex, claude, or opencode")
		id := flags.String("id", "", "normalized canonical skill identifier")
		approved := flags.String("approve", "", "complete synchronization plan ID")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "skills sync does not accept positional arguments"}
		}
		if *provider == "" || *id == "" {
			return &usageError{message: "skills sync requires --provider and --id"}
		}
		preview, err := providersync.NewPlan(*root, *provider, *id)
		if err != nil {
			return err
		}
		result, err := providersyncexec.Execute(preview, *approved, providersyncexec.Options{ProcessID: os.Getpid()})
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	case "unsync":
		flags := newFlagSet("skills unsync", stderr)
		root := flags.String("root", ".", "initialized workspace root")
		provider := flags.String("provider", "", "provider: codex, claude, or opencode")
		id := flags.String("id", "", "normalized canonical skill identifier")
		approved := flags.String("approve", "", "complete unsynchronization plan ID")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "skills unsync does not accept positional arguments"}
		}
		if *provider == "" || *id == "" {
			return &usageError{message: "skills unsync requires --provider and --id"}
		}
		preview, err := providersync.NewUnsyncPlan(*root, *provider, *id)
		if err != nil {
			return err
		}
		result, err := providersyncexec.Execute(preview, *approved, providersyncexec.Options{ProcessID: os.Getpid()})
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	case "recover-sync":
		flags := newFlagSet("skills recover-sync", stderr)
		root := flags.String("root", ".", "initialized workspace root")
		provider := flags.String("provider", "", "provider: codex, claude, or opencode")
		id := flags.String("id", "", "normalized canonical skill identifier")
		approved := flags.String("approve", "", "complete interrupted provider plan ID")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "skills recover-sync does not accept positional arguments"}
		}
		if *provider == "" || *id == "" {
			return &usageError{message: "skills recover-sync requires --provider and --id"}
		}
		result, err := providersyncexec.Recover(*root, *provider, *id, *approved, providersyncexec.Options{ProcessID: os.Getpid()})
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	case "create":
		flags := newFlagSet("skills create", stderr)
		root := flags.String("root", ".", "initialized workspace root")
		source := flags.String("source", "", "complete canonical source package")
		approved := flags.String("approve", "", "complete plan ID approved for execution")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "skills create does not accept positional arguments"}
		}
		if *source == "" {
			return &usageError{message: "skills create requires --source"}
		}
		preview, err := skillplan.NewCreate(*root, *source)
		if err != nil {
			return err
		}
		result, err := skillexec.Execute(preview, *approved, skillexec.Options{ProcessID: os.Getpid()})
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	case "update":
		flags := newFlagSet("skills update", stderr)
		root := flags.String("root", ".", "initialized workspace root")
		id := flags.String("id", "", "normalized target skill identifier")
		source := flags.String("source", "", "complete canonical source package")
		approved := flags.String("approve", "", "complete plan ID approved for execution")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "skills update does not accept positional arguments"}
		}
		if *id == "" || *source == "" {
			return &usageError{message: "skills update requires --id and --source"}
		}
		preview, err := skillplan.NewUpdate(*root, *id, *source)
		if err != nil {
			return err
		}
		result, err := skillexec.Execute(preview, *approved, skillexec.Options{ProcessID: os.Getpid()})
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	case "remove":
		flags := newFlagSet("skills remove", stderr)
		root := flags.String("root", ".", "initialized workspace root")
		id := flags.String("id", "", "normalized target skill identifier")
		approved := flags.String("approve", "", "complete plan ID approved for execution")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "skills remove does not accept positional arguments"}
		}
		if *id == "" {
			return &usageError{message: "skills remove requires --id"}
		}
		preview, err := skillplan.NewRemove(*root, *id)
		if err != nil {
			return err
		}
		result, err := skillexec.Execute(preview, *approved, skillexec.Options{ProcessID: os.Getpid()})
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	case "recover":
		flags := newFlagSet("skills recover", stderr)
		root := flags.String("root", ".", "initialized workspace root")
		id := flags.String("id", "", "normalized target skill identifier")
		approved := flags.String("approve", "", "complete interrupted plan ID")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "skills recover does not accept positional arguments"}
		}
		if *id == "" {
			return &usageError{message: "skills recover requires --id"}
		}
		result, err := skillexec.Recover(*root, *id, *approved, skillexec.Options{ProcessID: os.Getpid()})
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
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
		return &usageError{message: "context requires a subcommand: rebuild, status, sources, search, or read"}
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
		pathPrefix := flags.String("path-prefix", "", "normalized indexed path prefix")
		kind := flags.String("kind", "", "exact artifact kind")
		language := flags.String("language", "", "exact source language")
		limit := flags.Int("limit", 0, "result count: 1-20; default 5")
		maxContentBytes := flags.Int("max-content-bytes", 0, "combined snippet bytes: 1-16384; default 4096")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 1 {
			return &usageError{message: "context search requires exactly one query argument"}
		}
		result, err := sqliteindex.SearchWithOptions(context.Background(), *root, flags.Arg(0), contextsearch.Options{PathPrefix: *pathPrefix, Kind: *kind, Language: *language, Limit: *limit, MaxContentBytes: *maxContentBytes})
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	case "sources":
		flags := newFlagSet("context sources", stderr)
		root := flags.String("root", ".", "initialized workspace root")
		pathPrefix := flags.String("path-prefix", "", "normalized indexed path prefix")
		kind := flags.String("kind", "", "exact artifact kind")
		language := flags.String("language", "", "exact source language")
		limit := flags.Int("limit", 0, "source count: 1-200; default 50")
		after := flags.String("after", "", "exclusive normalized source path")
		projection := flags.String("projection", "", "projection fingerprint required with --after")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "context sources does not accept positional arguments"}
		}
		result, err := sqliteindex.ListSources(context.Background(), *root, contextview.SourceOptions{PathPrefix: *pathPrefix, Kind: *kind, Language: *language, Limit: *limit, After: *after, ProjectionFingerprint: *projection})
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	case "read":
		flags := newFlagSet("context read", stderr)
		root := flags.String("root", ".", "initialized workspace root")
		indexedPath := flags.String("path", "", "exact normalized indexed path")
		chunk := flags.String("chunk", "", "exact indexed chunk identity")
		startLine := flags.Int("start-line", 0, "inclusive source start line")
		endLine := flags.Int("end-line", 0, "inclusive source end line")
		maxBytes := flags.Int("max-bytes", 0, "response content bytes: 1-32768; default 8192")
		if err := flags.Parse(args[1:]); err != nil {
			return &usageError{message: err.Error()}
		}
		if flags.NArg() != 0 {
			return &usageError{message: "context read does not accept positional arguments"}
		}
		result, err := sqliteindex.ReadIndexed(context.Background(), *root, contextview.ReadRequest{Path: *indexedPath, ChunkID: *chunk, StartLine: *startLine, EndLine: *endLine, MaxBytes: *maxBytes})
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

var usageLines = []string{
	"version [--json]",
	"doctor [--root PATH] [--scope core|workspace|integrations|all] [--summary|-s]",
	"init plan --target PATH [--profile PROFILE_ID@VERSION] [--set NAME=VALUE]",
	"init apply --target PATH [--profile PROFILE_ID@VERSION] [--set NAME=VALUE] --approve PLAN_ID",
	"mcp serve --transport stdio --root PATH",
	"workspace inspect [--root PATH]",
	"workspace plan-init [--root PATH]",
	"workspace init [--root PATH] --approve PLAN_ID",
	"context rebuild [--root PATH]",
	"context status [--root PATH]",
	"context sources [--root PATH] [--path-prefix PATH] [--kind KIND] [--language LANGUAGE]",
	"context search [--root PATH] [--path-prefix PATH] [--kind KIND] [--language LANGUAGE] QUERY",
	"context read [--root PATH] --path FILE (--chunk SHA256 | --start-line N --end-line N)",
	"adapters list",
	"adapters inspect [--root PATH] --provider PROVIDER",
	"speckit plan [--root PATH] --integration PROVIDER --version RELEASE_TAG",
	"skills list [--root PATH]",
	"skills inspect [--root PATH] --id SKILL_ID",
	"skills validate [--root PATH] [--id SKILL_ID]",
	"skills plan-create [--root PATH] --source PACKAGE_PATH",
	"skills plan-update [--root PATH] --id SKILL_ID --source PACKAGE_PATH",
	"skills plan-remove [--root PATH] --id SKILL_ID",
	"skills plan-sync [--root PATH] --provider PROVIDER [--id SKILL_ID]",
	"skills plan-unsync [--root PATH] --provider PROVIDER --id SKILL_ID",
	"skills create [--root PATH] --source PACKAGE_PATH --approve PLAN_ID",
	"skills update [--root PATH] --id SKILL_ID --source PACKAGE_PATH --approve PLAN_ID",
	"skills remove [--root PATH] --id SKILL_ID --approve PLAN_ID",
	"skills recover [--root PATH] --id SKILL_ID --approve PLAN_ID",
	"skills sync [--root PATH] --provider PROVIDER --id SKILL_ID --approve PLAN_ID",
	"skills unsync [--root PATH] --provider PROVIDER --id SKILL_ID --approve PLAN_ID",
	"skills recover-sync [--root PATH] --provider PROVIDER --id SKILL_ID --approve PLAN_ID",
	"catalog inspect",
	"templates list",
	"templates inspect --id TEMPLATE_ID --version VERSION",
	"templates scaffold --output PATH --id TEMPLATE_ID --version VERSION",
	"profiles list",
	"profiles inspect --id PROFILE_ID --version VERSION",
	"profiles recommend [--root PATH]",
	"advise profiles --root PATH --goal TEXT [--model MODEL_TAG]",
	"project inspect [--root PATH]",
	"project plan-create --destination PATH --profile PROFILE_ID@VERSION [--set NAME=VALUE]",
	"project plan-compose [--root PATH] --profile PROFILE_ID@VERSION [--set NAME=VALUE]",
	"project create --destination PATH --profile PROFILE_ID@VERSION [--set NAME=VALUE] --approve PLAN_ID",
	"project compose [--root PATH] --profile PROFILE_ID@VERSION [--set NAME=VALUE] --approve PLAN_ID",
	"project recover [--root PATH] --approve PLAN_ID",
}

func helpPrefix(args []string) ([]string, bool) {
	for index, argument := range args {
		if argument == "help" || argument == "-h" || argument == "--help" {
			return args[:index], true
		}
	}
	return nil, false
}

func writeScopedUsage(writer io.Writer, prefix []string) bool {
	if len(prefix) == 0 {
		writeUsage(writer)
		return true
	}
	wanted := strings.Join(prefix, " ")
	matched := make([]string, 0)
	for _, line := range usageLines {
		command := strings.Fields(line)
		if len(command) < len(prefix) {
			continue
		}
		if strings.Join(command[:len(prefix)], " ") == wanted {
			matched = append(matched, line)
		}
	}
	if len(matched) == 0 {
		return false
	}
	fmt.Fprintln(writer, "Usage:")
	for _, line := range matched {
		fmt.Fprintf(writer, "  engkit %s\n", line)
	}
	if strings.HasPrefix(wanted, "init") {
		fmt.Fprintln(writer, `
Selection:
  absent or empty target + exact profile  -> new-project composition
  existing uninitialized target           -> workspace initialization
  initialized target + exact profile      -> existing-workspace composition
  compound, incomplete, or ambiguous state -> refused

Unified init emits the selected specialized plan unchanged. Recovery remains
available through the specialized workspace and project commands.`)
	}
	return true
}

func writeUsage(writer io.Writer) {
	fmt.Fprintln(writer, "Engineering Toolkit\n\nUsage:")
	for _, line := range usageLines {
		fmt.Fprintf(writer, "  engkit %s\n", line)
	}
	fmt.Fprintln(writer, `
inspect and plan-init are read-only. init revalidates the workspace and only
publishes .engkit when --approve exactly matches the current plan ID. Adapter
inspection and Spec Kit planning are read-only; no Spec Kit execution command
is provided in this milestone. Skill listing, inspection, validation, and
mutation planning are read-only. Skill mutations require an initialized
workspace and exact plan approval; they never execute package resources.
Catalog, template, and profile inspection are offline and read-only. Template
scaffolding writes only to an explicit absent output directory. Project
inspection, recommendation, and composition planning are read-only. Project
creation and composition rebuild the exact approved plan, publish only additive
outputs, and retain durable recovery evidence.`)
}
