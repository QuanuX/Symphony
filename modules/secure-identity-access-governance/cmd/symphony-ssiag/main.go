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
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/foundationlifecycle"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/lifecycle"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/model"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/packageinstall"
	ssiagpaths "github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/paths"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/policyadmin"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/provider"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/server"
	"github.com/QuanuX/Symphony/modules/secure-identity-access-governance/internal/stavproducer"
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
	case "foundation-lifecycle":
		return runFoundationLifecycle(args[1:])
	case "package":
		return runPackage(args[1:])
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
	var audit *stavproducer.Producer
	audit, err = stavproducer.Open(scope, topsID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "symphony-ssiag: STAV audit unavailable; authorization endpoint will fail closed: %v\n", err)
		audit = nil
	}
	policyManager, err := policyadmin.New(layout.StateDir, cfg, time.Now)
	if err != nil {
		return fmt.Errorf("open protected SSIAG policy state: %w", err)
	}
	ssiagServer, err := server.NewWithPolicyAdministration(cfg, registry, audit, policyManager)
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
	auditDeferred := set.Bool("audit-deferred", false, "explicitly record bootstrap/self-impacting audit reconciliation as deferred")
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
	name := *topsName
	auditMode := "ordinary"
	if *auditDeferred {
		auditMode = "audit_deferred"
	}
	result, err := foundationlifecycle.New().DirectApply("enrollment", scope, topsID, foundationlifecycle.Intent{
		DesiredState: "enrolled", TOPSName: &name, ServiceUID: serviceUID, ServiceGID: serviceGID,
		AuditMode: auditMode, TTLSeconds: 300,
	})
	if err != nil {
		return err
	}
	layout, _ := ssiagpaths.ResolveInstance(scope, topsID)
	fmt.Printf("enrolled SSIAG tops_id=%s tops_name=%q scope=%s config=%s disposition=%s\n", topsID, *topsName, scope, layout.ConfigFile, result.Disposition)
	return nil
}

func runUnenroll(args []string) error {
	set := flag.NewFlagSet("unenroll", flag.ContinueOnError)
	scopeValue := set.String("scope", "user", "installation scope: user or system")
	topsIDValue := set.String("tops-id", "", "immutable TOPS UUID")
	purge := set.Bool("purge", false, "remove this TOPS SSIAG configuration and state")
	auditDeferred := set.Bool("audit-deferred", false, "explicitly record self-impacting audit reconciliation as deferred")
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
	if *purge {
		if err := foundationlifecycle.New().AssertNativePurgeSafe(scope, topsID); err != nil {
			return err
		}
		record, err := lifecycle.Unenroll(scope, topsID, true)
		if err != nil {
			return err
		}
		fmt.Printf("unenrolled SSIAG tops_id=%s scope=%s purge=true\n", record.TOPSID, record.Scope)
		return nil
	}
	auditMode := "ordinary"
	if *auditDeferred {
		auditMode = "audit_deferred"
	}
	result, err := foundationlifecycle.New().DirectApply("enrollment", scope, topsID, foundationlifecycle.Intent{
		DesiredState: "unenrolled_preserved", AuditMode: auditMode, TTLSeconds: 300,
	})
	if err != nil {
		return err
	}
	fmt.Printf("unenrolled SSIAG tops_id=%s scope=%s purge=false disposition=%s\n", topsID, scope, result.Disposition)
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
	auditDeferred := set.Bool("audit-deferred", false, "explicitly record bootstrap/self-impacting audit reconciliation as deferred")
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
	if *force {
		return fmt.Errorf("--force is recognized but refused by transactional SSIAG supervisor administration")
	}
	if *noStop {
		return fmt.Errorf("--no-stop is recognized but refused because it bypasses verified SSIAG shutdown")
	}
	scope, topsID, _, err := resolveInstance(*scopeValue, *topsIDValue)
	if err != nil {
		return err
	}
	if operation == "install" {
		auditMode := "ordinary"
		if *auditDeferred {
			auditMode = "audit_deferred"
		}
		desired := "native_running"
		if *noStart {
			desired = "native_installed_stopped"
		}
		result, err := foundationlifecycle.New().DirectApply("supervisor", scope, topsID, foundationlifecycle.Intent{DesiredState: desired, AuditMode: auditMode, TTLSeconds: 300})
		if err != nil {
			return err
		}
		fmt.Printf("installed SSIAG supervisor tops_id=%s scope=%s started=%t disposition=%s\n", topsID, scope, !*noStart, result.Disposition)
		return nil
	}
	auditMode := "ordinary"
	if *auditDeferred {
		auditMode = "audit_deferred"
	}
	result, err := foundationlifecycle.New().DirectApply("supervisor", scope, topsID, foundationlifecycle.Intent{DesiredState: "absent_stopped", AuditMode: auditMode, TTLSeconds: 300})
	if err != nil {
		return err
	}
	fmt.Printf("uninstalled SSIAG supervisor tops_id=%s scope=%s; configuration and state preserved disposition=%s\n", topsID, scope, result.Disposition)
	return nil
}

func runFoundationLifecycle(args []string) error {
	engine := foundationlifecycle.New()
	if len(args) == 2 && args[0] == "describe" && args[1] == "--json" {
		scope, err := engine.InstalledScope()
		if err != nil {
			return err
		}
		descriptor, err := engine.Descriptor(scope)
		if err != nil {
			return err
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetEscapeHTML(false)
		return encoder.Encode(descriptor)
	}
	if len(args) != 0 {
		return fmt.Errorf("foundation-lifecycle accepts a command on stdin or describe --json")
	}
	command, err := foundationlifecycle.DecodeCommand(os.Stdin)
	if err != nil {
		return err
	}
	scope, err := ssiagpaths.ParseScope(command.Scope)
	if err != nil {
		return err
	}
	if err := engine.VerifyInvocation(scope); err != nil {
		return err
	}
	result, err := engine.Execute(command)
	if err != nil {
		return err
	}
	return foundationlifecycle.EncodeResult(os.Stdout, result)
}

func runPackage(args []string) error {
	if len(args) == 0 || (args[0] != "install" && args[0] != "uninstall") {
		return fmt.Errorf("package requires install or uninstall")
	}
	operation := args[0]
	set := flag.NewFlagSet("package "+operation, flag.ContinueOnError)
	prefix := set.String("prefix", "", "exact absolute receipt-v2 installation prefix")
	packageVersion := set.String("version", version.Version, "exact side-by-side package version")
	if err := set.Parse(args[1:]); err != nil {
		return err
	}
	if set.NArg() != 0 || *prefix == "" {
		return fmt.Errorf("package %s requires --prefix and --version", operation)
	}
	if operation == "install" {
		executable, err := os.Executable()
		if err != nil {
			return err
		}
		result, err := packageinstall.Install(executable, *prefix, *packageVersion)
		if err != nil {
			return err
		}
		fmt.Printf("installed SSIAG receipt-v2 package version=%s prefix=%s binary=%s receipt=%s digest=%s changed=%t\n", result.Version, result.Prefix, result.Binary, result.Receipt, result.ReceiptDigest, result.Changed)
		return nil
	}
	result, err := packageinstall.Uninstall(*prefix, *packageVersion)
	if err != nil {
		return err
	}
	fmt.Printf("uninstalled SSIAG receipt-v2 package version=%s prefix=%s receipt=%s changed=%t\n", result.Version, result.Prefix, result.Receipt, result.Changed)
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
	fmt.Println("  foundation-lifecycle  Run the bounded module-owned machine lifecycle adapter")
	fmt.Println("  package     Install/uninstall one immutable receipt-v2 adapter package")
	fmt.Println("  serve       Run the local safe-metadata and authorization SSIAG API for one TOPS")
	fmt.Println("  status      Read safe SSIAG status for one TOPS")
	fmt.Println("  providers   List safe provider descriptors for one TOPS")
	fmt.Println("  version     Print version")
}
