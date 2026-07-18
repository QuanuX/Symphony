package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/client"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/config"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/lifecycle"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/model"
	ssiagpaths "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/provider"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/server"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/supervision"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/version"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "symphony-ssiag: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return errors.New("command is required")
	}
	switch args[0] {
	case "--help", "help":
		printUsage()
		return nil
	case "--version", "version":
		fmt.Printf("symphony-ssiag version %s\n", version.Version)
		return nil
	case "serve":
		return runServe(args[1:])
	case "status":
		return runStatus(args[1:])
	case "providers":
		return runProviders(args[1:])
	case "install":
		return runInstall(args[1:])
	case "uninstall":
		return runUninstall(args[1:])
	case "enroll":
		return runEnroll(args[1:])
	case "unenroll":
		return runUnenroll(args[1:])
	case "supervisor":
		return runSupervisor(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runServe(args []string) error {
	set := flag.NewFlagSet("serve", flag.ContinueOnError)
	scopeValue := set.String("scope", "user", "installation scope: user or system")
	topsIDValue := set.String("tops-id", "", "immutable TOPS UUID")
	configPath := set.String("config", "", "explicit config path")
	supervised := set.Bool("supervised", false, "assert invocation by the installed native supervisor")
	if err := set.Parse(args); err != nil {
		return err
	}
	scope, topsID, layout, err := resolveInstance(*scopeValue, *topsIDValue)
	if err != nil {
		return err
	}
	if scope == ssiagpaths.ScopeSystem && !*supervised {
		return fmt.Errorf("system-scope serve requires the installed supervisor; use --supervised only from an owner-controlled equivalent")
	}
	if scope == ssiagpaths.ScopeUser && !*supervised {
		fmt.Fprintln(os.Stderr, "symphony-ssiag: direct user-scope serve is a development/diagnostic mode; production uses supervisor install")
	}
	path := *configPath
	if path == "" {
		path = os.Getenv("SYMPHONY_SSIAG_CONFIG")
	}
	if path == "" {
		path = layout.ConfigFile
	}
	cfg, err := config.LoadTrusted(path, scope)
	if err != nil {
		return fmt.Errorf("load enrolled TOPS configuration: %w", err)
	}
	if cfg.TOPS.ID != topsID {
		return fmt.Errorf("configuration TOPS ID does not match --tops-id")
	}
	if cfg.Mode != string(scope) {
		return fmt.Errorf("configuration mode does not match --scope")
	}
	if socket := os.Getenv("SYMPHONY_SSIAG_SOCKET"); socket != "" {
		cfg.Listen.Address = socket
	} else if cfg.Listen.Address != layout.Socket {
		return fmt.Errorf("configuration socket does not match the selected TOPS layout")
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	registry, err := provider.New(cfg.Providers)
	if err != nil {
		return err
	}
	ssiagServer, err := server.New(cfg, registry)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Printf("SSIAG serving TOPS %s on local unix socket %s\n", topsID, cfg.Listen.Address)
	return ssiagServer.Run(ctx)
}

func runStatus(args []string) error {
	scope, topsID, jsonOutput, err := parseQueryFlags("status", args)
	if err != nil {
		return err
	}
	ssiagClient, err := scopedClient(scope, topsID)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	status, err := requireStatus(ctx, ssiagClient, scope, topsID)
	if err != nil {
		return err
	}
	if jsonOutput {
		return printJSON(status)
	}
	fmt.Printf("SSIAG: %s version=%s ready=%t tops_id=%s tops_name=%q mode=%s providers=%d\n", status.Name, status.Version, status.Ready, status.TOPSID, status.TOPSName, status.Mode, status.ProviderCount)
	return nil
}

func runProviders(args []string) error {
	scope, topsID, jsonOutput, err := parseQueryFlags("providers", args)
	if err != nil {
		return err
	}
	ssiagClient, err := scopedClient(scope, topsID)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if _, err := requireStatus(ctx, ssiagClient, scope, topsID); err != nil {
		return err
	}
	providers, err := ssiagClient.Providers(ctx)
	if err != nil {
		return err
	}
	if jsonOutput {
		return printJSON(providers)
	}
	if len(providers.Providers) == 0 {
		fmt.Println("SSIAG providers: none declared")
		return nil
	}
	for _, item := range providers.Providers {
		fmt.Printf("SSIAG provider: %s kind=%s status=%s\n", item.Name, item.Kind, item.Status)
	}
	return nil
}

func requireStatus(ctx context.Context, ssiagClient *client.Client, scope ssiagpaths.Scope, topsID string) (model.Status, error) {
	status, err := ssiagClient.Status(ctx)
	if err != nil {
		return model.Status{}, err
	}
	if status.TOPSID != topsID {
		return model.Status{}, fmt.Errorf("SSIAG response TOPS ID does not match requested identity")
	}
	if status.Mode != string(scope) {
		return model.Status{}, fmt.Errorf("SSIAG response mode does not match requested scope")
	}
	if !status.Ready {
		return model.Status{}, errors.New("SSIAG is not ready")
	}
	return status, nil
}

func runInstall(args []string) error {
	set := flag.NewFlagSet("install", flag.ContinueOnError)
	scopeValue := set.String("scope", "user", "installation scope: user or system")
	force := set.Bool("force", false, "replace a changed installed binary")
	if err := set.Parse(args); err != nil {
		return err
	}
	scope, err := ssiagpaths.ParseScope(*scopeValue)
	if err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve running executable: %w", err)
	}
	record, err := lifecycle.Install(executable, scope, *force)
	if err != nil {
		return err
	}
	fmt.Printf("installed symphony-ssiag scope=%s binary=%s\n", record.Scope, record.Binary)
	return nil
}

func runUninstall(args []string) error {
	set := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	scopeValue := set.String("scope", "user", "installation scope: user or system")
	force := set.Bool("force", false, "remove a binary whose digest changed")
	if err := set.Parse(args); err != nil {
		return err
	}
	scope, err := ssiagpaths.ParseScope(*scopeValue)
	if err != nil {
		return err
	}
	record, err := lifecycle.Uninstall(scope, *force)
	if err != nil {
		return err
	}
	fmt.Printf("uninstalled symphony-ssiag scope=%s binary=%s; per-TOPS state preserved\n", record.Scope, record.Binary)
	return nil
}

func runEnroll(args []string) error {
	set := flag.NewFlagSet("enroll", flag.ContinueOnError)
	scopeValue := set.String("scope", "user", "installation scope: user or system")
	topsIDValue := set.String("tops-id", "", "immutable TOPS UUID")
	topsName := set.String("tops-name", "", "mutable TOPS display name")
	serviceUIDVal := set.String("service-uid", "", "exact service UID (required for new system enrollment)")
	serviceGIDVal := set.String("service-gid", "", "exact service GID (required for new system enrollment)")
	if err := set.Parse(args); err != nil {
		return err
	}
	scope, err := ssiagpaths.ParseScope(*scopeValue)
	if err != nil {
		return err
	}
	topsID, err := requiredTOPSID(*topsIDValue)
	if err != nil {
		return err
	}
	var serviceUID, serviceGID *uint32
	if *serviceUIDVal != "" {
		parsed, err := strconv.ParseUint(*serviceUIDVal, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid service-uid value %q: must be a non-negative integer", *serviceUIDVal)
		}
		val := uint32(parsed)
		serviceUID = &val
	}
	if *serviceGIDVal != "" {
		parsed, err := strconv.ParseUint(*serviceGIDVal, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid service-gid value %q: must be a non-negative integer", *serviceGIDVal)
		}
		val := uint32(parsed)
		serviceGID = &val
	}
	record, err := lifecycle.Enroll(scope, topsID, *topsName, serviceUID, serviceGID)
	if err != nil {
		return err
	}
	fmt.Printf("enrolled SSIAG tops_id=%s tops_name=%q scope=%s config=%s\n", record.TOPSID, record.TOPSName, record.Scope, record.ConfigFile)
	return nil
}

func runUnenroll(args []string) error {
	set := flag.NewFlagSet("unenroll", flag.ContinueOnError)
	scopeValue := set.String("scope", "user", "installation scope: user or system")
	topsIDValue := set.String("tops-id", "", "immutable TOPS UUID")
	purge := set.Bool("purge", false, "remove this TOPS SSIAG configuration and state")
	if err := set.Parse(args); err != nil {
		return err
	}
	scope, err := ssiagpaths.ParseScope(*scopeValue)
	if err != nil {
		return err
	}
	topsID, err := requiredTOPSID(*topsIDValue)
	if err != nil {
		return err
	}
	record, err := lifecycle.Unenroll(scope, topsID, *purge)
	if err != nil {
		return err
	}
	fmt.Printf("unenrolled SSIAG tops_id=%s scope=%s purge=%t\n", record.TOPSID, record.Scope, *purge)
	return nil
}

func runSupervisor(args []string) error {
	if len(args) == 0 || (args[0] != "install" && args[0] != "uninstall") {
		return fmt.Errorf("supervisor requires install or uninstall")
	}
	operation := args[0]
	set := flag.NewFlagSet("supervisor "+operation, flag.ContinueOnError)
	scopeValue := set.String("scope", "user", "installation scope: user or system")
	topsIDValue := set.String("tops-id", "", "immutable TOPS UUID")
	force := set.Bool("force", false, "replace or remove a differing supervisor descriptor")
	noStart := set.Bool("no-start", false, "install the descriptor without registering or starting it")
	noStop := set.Bool("no-stop", false, "remove the descriptor without asking the native manager to stop it")
	if err := set.Parse(args[1:]); err != nil {
		return err
	}
	if set.NArg() != 0 {
		return fmt.Errorf("unexpected supervisor arguments: %v", set.Args())
	}
	if operation == "uninstall" && *noStart {
		return fmt.Errorf("--no-start is valid only for supervisor install")
	}
	if operation == "install" && *noStop {
		return fmt.Errorf("--no-stop is valid only for supervisor uninstall")
	}
	scope, topsID, layout, err := resolveInstance(*scopeValue, *topsIDValue)
	if err != nil {
		return err
	}
	cfg, err := config.LoadTrusted(layout.ConfigFile, scope)
	if err != nil {
		return fmt.Errorf("load enrolled TOPS configuration: %w", err)
	}
	install, err := ssiagpaths.ResolveInstall(scope)
	if err != nil {
		return err
	}
	if operation == "install" {
		info, statErr := os.Lstat(install.Binary)
		if statErr != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("installed SSIAG binary is required before supervisor installation")
		}
	}
	spec, err := supervision.SpecFromConfig(scope, topsID, install.Binary, cfg)
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
		fmt.Printf("installed SSIAG supervisor manager=%s name=%s tops_id=%s descriptor=%s started=%t\n", record.Manager, record.Name, topsID, record.Descriptor, !*noStart)
		return nil
	}
	record, err := supervision.Uninstall(spec, *force, !*noStop)
	if err != nil {
		return err
	}
	fmt.Printf("uninstalled SSIAG supervisor manager=%s name=%s tops_id=%s; configuration and state preserved\n", record.Manager, record.Name, topsID)
	return nil
}

func parseQueryFlags(name string, args []string) (ssiagpaths.Scope, string, bool, error) {
	set := flag.NewFlagSet(name, flag.ContinueOnError)
	scopeValue := set.String("scope", "user", "installation scope: user or system")
	topsIDValue := set.String("tops-id", "", "immutable TOPS UUID")
	jsonOutput := set.Bool("json", false, "emit JSON")
	if err := set.Parse(args); err != nil {
		return "", "", false, err
	}
	if set.NArg() != 0 {
		return "", "", false, fmt.Errorf("unexpected %s arguments: %v", name, set.Args())
	}
	scope, err := ssiagpaths.ParseScope(*scopeValue)
	if err != nil {
		return "", "", false, err
	}
	topsID, err := requiredTOPSID(*topsIDValue)
	return scope, topsID, *jsonOutput, err
}

func resolveInstance(scopeValue, topsIDValue string) (ssiagpaths.Scope, string, ssiagpaths.InstanceLayout, error) {
	scope, err := ssiagpaths.ParseScope(scopeValue)
	if err != nil {
		return "", "", ssiagpaths.InstanceLayout{}, err
	}
	topsID, err := requiredTOPSID(topsIDValue)
	if err != nil {
		return "", "", ssiagpaths.InstanceLayout{}, err
	}
	layout, err := ssiagpaths.ResolveInstance(scope, topsID)
	return scope, topsID, layout, err
}

func requiredTOPSID(value string) (string, error) {
	if value == "" {
		value = os.Getenv("SYMPHONY_SSIAG_TOPS_ID")
	}
	if value == "" {
		return "", fmt.Errorf("--tops-id or SYMPHONY_SSIAG_TOPS_ID is required")
	}
	if err := ssiagpaths.ValidateTOPSID(value); err != nil {
		return "", err
	}
	return value, nil
}

func scopedClient(scope ssiagpaths.Scope, topsID string) (*client.Client, error) {
	return client.NewForTOPS(scope, topsID, 4*time.Second)
}

func printJSON(value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func printUsage() {
	fmt.Println("symphony-ssiag - Symphony Secure Identity and Access Governance")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  symphony-ssiag <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  install     Install the shared host binary")
	fmt.Println("  uninstall   Remove the host binary; preserve all TOPS state")
	fmt.Println("  enroll      Create or update one TOPS enrollment")
	fmt.Println("  unenroll    Remove one TOPS enrollment; preserve data unless --purge")
	fmt.Println("  supervisor  Install/uninstall one TOPS native liveness service")
	fmt.Println("  serve       Run the local metadata-only SSIAG API for one TOPS")
	fmt.Println("  status      Read safe SSIAG status for one TOPS")
	fmt.Println("  providers   List safe provider descriptors for one TOPS")
	fmt.Println("  version     Print version")
}
