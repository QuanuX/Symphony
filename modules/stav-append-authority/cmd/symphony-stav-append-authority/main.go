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
	record, err := lifecycle.Enroll(scope, *topsID, authorityUID, authorityGID)
	if err != nil {
		return err
	}
	fmt.Printf("enrolled STAV append authority tops_id=%s scope=%s config=%s; producer and reader grants remain empty until explicitly configured\n", record.TOPSID, record.Scope, record.ConfigFile)
	return nil
}

func runUnenroll(args []string) error {
	set := flag.NewFlagSet("unenroll", flag.ContinueOnError)
	scopeValue := set.String("scope", "user", "installation scope: user or system")
	topsID := set.String("tops-id", "", "immutable TOPS UUID")
	purge := set.Bool("purge", false, "delete this TOPS STAV configuration and ledger after active-listener checks")
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
	record, err := lifecycle.Unenroll(scope, *topsID, *purge)
	if err != nil {
		return err
	}
	fmt.Printf("unenrolled STAV append authority tops_id=%s scope=%s purge=%t\n", record.TOPSID, record.Scope, *purge)
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
	install, err := stavpaths.ResolveInstall(scope)
	if err != nil {
		return err
	}
	if operation == "install" {
		info, statErr := os.Lstat(install.Binary)
		if statErr != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("installed STAV append-authority binary is required before supervisor installation")
		}
	}
	spec, err := supervision.SpecFromConfig(scope, *topsID, install.Binary, cfg)
	if err != nil {
		return err
	}
	if operation == "install" {
		record, err := supervision.Install(spec, *force)
		if err != nil {
			return err
		}
		if !*noStart {
			if err := supervision.Start(record); err != nil {
				return fmt.Errorf("descriptor installed at %s but activation failed: %w", record.Descriptor, err)
			}
		}
		fmt.Printf("installed STAV supervisor manager=%s name=%s tops_id=%s descriptor=%s started=%t\n", record.Manager, record.Name, *topsID, record.Descriptor, !*noStart)
		return nil
	}
	record, err := supervision.Uninstall(spec, *force, !*noStop)
	if err != nil {
		return err
	}
	fmt.Printf("uninstalled STAV supervisor manager=%s name=%s tops_id=%s; configuration and ledgers preserved\n", record.Manager, record.Name, *topsID)
	return nil
}

func runLifecycle(command string, args []string) error {
	set := flag.NewFlagSet(command, flag.ContinueOnError)
	scopeValue := set.String("scope", "user", "installation scope: user or system")
	force := set.Bool("force", false, "replace or remove a differing installed binary")
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
		result, err = lifecycle.Install(executable, scope, *force)
	case "uninstall":
		result, err = lifecycle.Uninstall(executable, scope, *force)
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
	fmt.Println("  symphony-stav-append-authority install [--scope user|system] [--force]")
	fmt.Println("  symphony-stav-append-authority uninstall [--scope user|system] [--force]")
	fmt.Println("  symphony-stav-append-authority enroll --tops-id UUID [--scope user|system] [--authority-uid N --authority-gid N]")
	fmt.Println("  symphony-stav-append-authority unenroll --tops-id UUID [--scope user|system] [--purge]")
	fmt.Println("  symphony-stav-append-authority serve --tops-id UUID [--scope user|system] [--config PATH]")
	fmt.Println("  symphony-stav-append-authority supervisor install|uninstall --tops-id UUID [--scope user|system]")
	fmt.Println()
	fmt.Println("Enrollment creates no producer or reader grant; configure each exact UID/GID grant before use.")
}
