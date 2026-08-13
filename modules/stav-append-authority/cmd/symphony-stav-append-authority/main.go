package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/config"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/foundation"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/lifecycle"
	stavpaths "github.com/QuanuX/Symphony/modules/stav-append-authority/internal/paths"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/server"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/supervision"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/version"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "symphony-stav-append-authority: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return fmt.Errorf("command is required")
	}
	switch args[0] {
	case "--help", "help":
		printUsage()
		return nil
	case "--version":
		fmt.Printf("symphony-stav-append-authority version %s\n", version.Version)
		return nil
	case "install", "uninstall":
		return runLifecycle(args[0], args[1:])
	case "foundation-lifecycle":
		return runFoundationLifecycle(args[1:])
	case "enroll":
		return runEnroll(args[1:])
	case "unenroll":
		return runUnenroll(args[1:])
	case "serve":
		return runServe(args[1:])
	case "supervisor":
		return runSupervisor(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runEnroll(args []string) error {
	set := flag.NewFlagSet("enroll", flag.ContinueOnError)
	scopeValue := set.String("scope", "user", "installation scope: user or system")
	topsID := set.String("tops-id", "", "immutable TOPS UUID")
	authorityUIDValue := set.String("authority-uid", "", "expected append-authority effective UID (required for system scope)")
	authorityGIDValue := set.String("authority-gid", "", "expected append-authority effective GID (required for system scope)")
	auditDeferred := set.Bool("audit-deferred", false, "explicitly retain local audit evidence for later closed-producer reconciliation")
	if err := set.Parse(args); err != nil {
		return err
	}
	if set.NArg() != 0 || *topsID == "" {
		return fmt.Errorf("enroll requires --tops-id and no positional arguments")
	}
	scope, err := stavpaths.ParseScope(*scopeValue)
	if err != nil {
		return err
	}
	if (*authorityUIDValue == "") != (*authorityGIDValue == "") {
		return fmt.Errorf("authority UID and GID must be supplied together")
	}
	authorityUID, authorityGID := uint64(os.Geteuid()), uint64(os.Getegid())
	if scope == stavpaths.ScopeSystem && *authorityUIDValue == "" {
		return fmt.Errorf("new or repeated system enrollment requires explicit --authority-uid and --authority-gid")
	}
	if scope == stavpaths.ScopeUser && *authorityUIDValue != "" {
		return fmt.Errorf("user enrollment binds the authority to the enrolling effective UID/GID and does not accept an override")
	}
	if *authorityUIDValue != "" {
		authorityUID, err = strconv.ParseUint(*authorityUIDValue, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid authority UID %q", *authorityUIDValue)
		}
		authorityGID, err = strconv.ParseUint(*authorityGIDValue, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid authority GID %q", *authorityGIDValue)
		}
	}
	uid, gid := uint32(authorityUID), uint32(authorityGID)
	var uidInput, gidInput *uint32
	if scope == stavpaths.ScopeSystem {
		uidInput, gidInput = &uid, &gid
	}
	result, err := foundation.DirectApply("enrollment", scope, *topsID, "enrolled", uidInput, gidInput, *auditDeferred)
	if err != nil {
		return err
	}
	fmt.Printf("enrolled STAV append authority tops_id=%s scope=%s reconciliation_required=%t; producer and reader grants remain empty until explicitly configured\n", result.TOPSID, result.Scope, result.ReconciliationRequired)
	return nil
}

func runUnenroll(args []string) error {
	set := flag.NewFlagSet("unenroll", flag.ContinueOnError)
	scopeValue := set.String("scope", "user", "installation scope: user or system")
	topsID := set.String("tops-id", "", "immutable TOPS UUID")
	purge := set.Bool("purge", false, "delete this TOPS STAV configuration and ledger after active-listener checks")
	auditDeferred := set.Bool("audit-deferred", false, "explicitly retain local audit evidence for later closed-producer reconciliation")
	if err := set.Parse(args); err != nil {
		return err
	}
	if set.NArg() != 0 || *topsID == "" {
		return fmt.Errorf("unenroll requires --tops-id and no positional arguments")
	}
	scope, err := stavpaths.ParseScope(*scopeValue)
	if err != nil {
		return err
	}
	if *purge {
		record, err := foundation.NativePurge(scope, *topsID)
		if err != nil {
			return err
		}
		fmt.Printf("unenrolled STAV append authority tops_id=%s scope=%s purge=true\n", record.TOPSID, record.Scope)
		return nil
	}
	result, err := foundation.DirectApply("enrollment", scope, *topsID, "unenrolled_preserved", nil, nil, *auditDeferred)
	if err != nil {
		return err
	}
	fmt.Printf("unenrolled STAV append authority tops_id=%s scope=%s purge=false reconciliation_required=%t\n", result.TOPSID, result.Scope, result.ReconciliationRequired)
	return nil
}

func runServe(args []string) error {
	set := flag.NewFlagSet("serve", flag.ContinueOnError)
	scopeValue := set.String("scope", "user", "installation scope: user or system")
	topsID := set.String("tops-id", "", "immutable TOPS UUID")
	configPath := set.String("config", "", "explicit configuration path for development or supervised launch")
	supervised := set.Bool("supervised", false, "assert invocation by the installed native supervisor")
	if err := set.Parse(args); err != nil {
		return err
	}
	if set.NArg() != 0 || *topsID == "" {
		return fmt.Errorf("serve requires --tops-id and no positional arguments")
	}
	scope, err := stavpaths.ParseScope(*scopeValue)
	if err != nil {
		return err
	}
	if scope == stavpaths.ScopeSystem && !*supervised {
		return fmt.Errorf("system-scope serve requires the installed supervisor; use --supervised only from an owner-controlled equivalent")
	}
	if scope == stavpaths.ScopeUser && !*supervised {
		fmt.Fprintln(os.Stderr, "symphony-stav-append-authority: direct user-scope serve is a development/diagnostic mode; production uses supervisor install")
	}
	layout, err := stavpaths.ResolveInstance(scope, *topsID)
	if err != nil {
		return err
	}
	path := *configPath
	if path == "" {
		path = layout.ConfigFile
	}
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	if cfg.TOPSID != *topsID || cfg.Mode != string(scope) {
		return fmt.Errorf("configuration does not match selected TOPS and scope")
	}
	if *configPath == "" {
		if err := config.ValidateLayout(cfg, layout); err != nil {
			return err
		}
	}
	service, err := server.New(cfg)
	if err != nil {
		return err
	}
	defer service.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Printf("serving STAV append authority tops_id=%s scope=%s socket=%s\n", cfg.TOPSID, cfg.Mode, cfg.Listen.Address)
	return service.Run(ctx)
}

func runSupervisor(args []string) error {
	if len(args) == 0 || (args[0] != "install" && args[0] != "uninstall") {
		return fmt.Errorf("supervisor requires install or uninstall")
	}
	operation := args[0]
	set := flag.NewFlagSet("supervisor "+operation, flag.ContinueOnError)
	scopeValue := set.String("scope", "user", "installation scope: user or system")
	topsID := set.String("tops-id", "", "immutable TOPS UUID")
	force := set.Bool("force", false, "replace or remove a differing supervisor descriptor")
	noStart := set.Bool("no-start", false, "install the descriptor without registering or starting it")
	noStop := set.Bool("no-stop", false, "remove the descriptor without asking the native manager to stop it")
	auditDeferred := set.Bool("audit-deferred", false, "explicitly retain local audit evidence for later closed-producer reconciliation")
	if err := set.Parse(args[1:]); err != nil {
		return err
	}
	if set.NArg() != 0 || *topsID == "" {
		return fmt.Errorf("supervisor %s requires --tops-id and no positional arguments", operation)
	}
	if operation == "uninstall" && *noStart {
		return fmt.Errorf("--no-start is valid only for supervisor install")
	}
	if operation == "install" && *noStop {
		return fmt.Errorf("--no-stop is valid only for supervisor uninstall")
	}
	if *force {
		return fmt.Errorf("--force is preserved for grammar compatibility but unavailable through the expected-state transaction engine")
	}
	if operation == "uninstall" && *noStop {
		return fmt.Errorf("transactional supervisor uninstall requires stop; --no-stop is unavailable")
	}
	scope, err := stavpaths.ParseScope(*scopeValue)
	if err != nil {
		return err
	}
	layout, err := stavpaths.ResolveInstance(scope, *topsID)
	if err != nil {
		return err
	}
	cfg, err := config.Load(layout.ConfigFile)
	if err != nil {
		return err
	}
	if err := config.ValidateLayout(cfg, layout); err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	install, _, _, err := lifecycle.VerifyExecutable(executable, scope)
	if err != nil {
		return fmt.Errorf("verified installed STAV append-authority binary is required before supervisor administration: %w", err)
	}
	spec, err := supervision.SpecFromConfig(scope, *topsID, install.Binary, cfg)
	if err != nil {
		return err
	}
	observed, err := supervision.Observe(spec)
	if err != nil {
		return err
	}
	if observed.DescriptorState == "drifted" && !*force {
		return fmt.Errorf("STAV supervisor descriptor differs; use --force to replace or remove it")
	}
	desired := "absent_stopped"
	if operation == "install" && !*noStart {
		desired = "native_running"
	}
	if operation == "install" && *noStart {
		desired = "native_installed_stopped"
	}
	result, err := foundation.DirectApply("supervisor", scope, *topsID, desired, nil, nil, *auditDeferred)
	if err != nil {
		return err
	}
	fmt.Printf("%s STAV supervisor manager=%s name=%s tops_id=%s reconciliation_required=%t; configuration and ledgers preserved\n", map[bool]string{true: "installed", false: "uninstalled"}[operation == "install"], observed.Record.Manager, observed.Record.Name, *topsID, result.ReconciliationRequired)
	return nil
}

func runFoundationLifecycle(args []string) error {
	if len(args) == 2 && args[0] == "describe" && args[1] == "--json" {
		descriptor, err := foundation.Describe()
		if err != nil {
			return err
		}
		return foundation.EncodeBounded(os.Stdout, descriptor)
	}
	if len(args) != 0 {
		return fmt.Errorf("foundation-lifecycle accepts no flags; use describe --json for the adapter descriptor")
	}
	command, err := foundation.DecodeCommand(os.Stdin)
	if err != nil {
		return err
	}
	result, executeErr := foundation.Execute(command)
	if result.Protocol != "" {
		if err := foundation.EncodeBounded(os.Stdout, result); err != nil {
			return err
		}
	}
	return executeErr
}

func runLifecycle(command string, args []string) error {
	set := flag.NewFlagSet(command, flag.ContinueOnError)
	scopeValue := set.String("scope", "user", "installation scope: user or system")
	force := set.Bool("force", false, "replace or remove a differing installed binary")
	prefix := set.String("prefix", "", "explicit installation prefix for an immutable receipt-v2 package")
	packageVersion := set.String("version", version.Version, "exact package version; must match this compiled binary")
	if err := set.Parse(args); err != nil {
		return err
	}
	if set.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	scope, err := stavpaths.ParseScope(*scopeValue)
	if err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve current executable: %w", err)
	}

	var result lifecycle.Result
	switch command {
	case "install":
		if *prefix == "" {
			result, err = lifecycle.Install(executable, scope, *force)
		} else {
			result, err = lifecycle.InstallAt(executable, scope, *prefix, *packageVersion)
		}
	case "uninstall":
		if *prefix == "" {
			result, err = lifecycle.Uninstall(executable, scope, *force)
		} else {
			result, err = lifecycle.UninstallAt(executable, scope, *prefix, *packageVersion)
		}
	}
	if err != nil {
		return err
	}
	if result.Changed {
		fmt.Printf("%s: %s scope=%s binary=%s\n", command, lifecycleVerb(command), result.Scope, result.Binary)
	} else {
		fmt.Printf("%s: no change scope=%s binary=%s\n", command, result.Scope, result.Binary)
	}
	return nil
}

func lifecycleVerb(command string) string {
	if command == "install" {
		return "installed"
	}
	return "uninstalled"
}

func printUsage() {
	fmt.Println("symphony-stav-append-authority - durable per-TOPS STAV serialization authority")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  symphony-stav-append-authority --help")
	fmt.Println("  symphony-stav-append-authority --version")
	fmt.Println("  symphony-stav-append-authority install [--scope user|system] [--prefix PATH --version VERSION]")
	fmt.Println("  symphony-stav-append-authority uninstall [--scope user|system] [--prefix PATH --version VERSION]")
	fmt.Println("  symphony-stav-append-authority enroll --tops-id UUID [--scope user|system] [--authority-uid N --authority-gid N]")
	fmt.Println("  symphony-stav-append-authority unenroll --tops-id UUID [--scope user|system] [--purge]")
	fmt.Println("  symphony-stav-append-authority serve --tops-id UUID [--scope user|system] [--config PATH]")
	fmt.Println("  symphony-stav-append-authority supervisor install|uninstall --tops-id UUID [--scope user|system]")
	fmt.Println("  symphony-stav-append-authority foundation-lifecycle < command.json")
	fmt.Println("  symphony-stav-append-authority foundation-lifecycle describe --json")
	fmt.Println()
	fmt.Println("Enrollment creates no producer or reader grant; configure each exact UID/GID grant before use.")
}
