package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/config"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/lifecycle"
	stavpaths "github.com/QuanuX/Symphony/modules/stav-append-authority/internal/paths"
	"github.com/QuanuX/Symphony/modules/stav-append-authority/internal/server"
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
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runEnroll(args []string) error {
	set := flag.NewFlagSet("enroll", flag.ContinueOnError)
	scopeValue := set.String("scope", "user", "installation scope: user or system")
	topsID := set.String("tops-id", "", "immutable TOPS UUID")
	authorityUID := set.Uint64("authority-uid", uint64(os.Geteuid()), "expected append-authority effective UID")
	authorityGID := set.Uint64("authority-gid", uint64(os.Getegid()), "expected append-authority effective GID")
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
	record, err := lifecycle.Enroll(scope, *topsID, *authorityUID, *authorityGID)
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
	fmt.Println()
	fmt.Println("Enrollment creates no producer or reader grant; configure each exact UID/GID grant before use.")
}
