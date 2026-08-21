package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	accordareclient "github.com/QuanuX/Symphony/modules/accordare-stav-producer/client"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/config"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/enrollment"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/outbox"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/packageinstall"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/paths"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/producer"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/server"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/version"
	stavclient "github.com/QuanuX/Symphony/modules/stav-append-authority/client"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "symphony-accordare-stav-producer: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return fmt.Errorf("command is required")
	}
	switch args[0] {
	case "help", "--help":
		usage()
		return nil
	case "--version":
		fmt.Printf("symphony-accordare-stav-producer version %s\n", version.Version)
		return nil
	case "enroll":
		return runEnroll(args[1:])
	case "install", "uninstall":
		return runPackage(args[0], args[1:])
	case "unenroll":
		return runUnenroll(args[1:])
	case "serve":
		return runServe(args[1:])
	case "status", "reconcile":
		return runControl(args[0], args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runPackage(operation string, args []string) error {
	set := flag.NewFlagSet(operation, flag.ContinueOnError)
	prefix := set.String("prefix", "", "absolute installation prefix")
	requestedVersion := set.String("version", version.Version, "exact immutable package version")
	if err := set.Parse(args); err != nil || set.NArg() != 0 || *prefix == "" {
		return fmt.Errorf("%s requires --prefix and no positional arguments", operation)
	}
	var result packageinstall.Result
	var err error
	if operation == "install" {
		executable, executableErr := os.Executable()
		if executableErr != nil {
			return executableErr
		}
		result, err = packageinstall.Install(executable, *prefix, *requestedVersion)
	} else {
		result, err = packageinstall.Uninstall(*prefix, *requestedVersion)
	}
	if err != nil {
		return err
	}
	fmt.Printf("%s Accordare STAV producer version=%s prefix=%s binary=%s receipt=%s digest=%s changed=%t\n", operation, result.Version, result.Prefix, result.Binary, result.Receipt, result.ReceiptDigest, result.Changed)
	return nil
}

func runEnroll(args []string) error {
	set := flag.NewFlagSet("enroll", flag.ContinueOnError)
	scopeValue := set.String("scope", "user", "installation scope: user or system")
	topsID := set.String("tops-id", "", "immutable TOPS UUID")
	stavConfig := set.String("stav-config", "", "absolute STAV append-authority configuration path")
	serviceID := set.String("service-id", "symphony.accordare.stav-producer", "SSIAG service subject identifier")
	submitterID := set.String("submitter-id", "symphony.qxctl", "SSIAG qxctl subject identifier")
	submitterKind := set.String("submitter-kind", "owner", "exact SSIAG qxctl subject kind")
	serviceUIDValue := set.String("service-uid", "", "producer effective UID")
	serviceGIDValue := set.String("service-gid", "", "producer effective GID")
	submitterUIDValue := set.String("submitter-uid", "", "qxctl effective UID")
	submitterGIDValue := set.String("submitter-gid", "", "qxctl effective GID")
	if err := set.Parse(args); err != nil {
		return err
	}
	if set.NArg() != 0 || *topsID == "" || *stavConfig == "" {
		return fmt.Errorf("enroll requires --tops-id, --stav-config, and no positional arguments")
	}
	scope, err := paths.ParseScope(*scopeValue)
	if err != nil {
		return err
	}
	serviceUID, serviceGID, err := identityNumbers(scope, *serviceUIDValue, *serviceGIDValue, "service")
	if err != nil {
		return err
	}
	submitterUID, submitterGID, err := identityNumbers(scope, *submitterUIDValue, *submitterGIDValue, "submitter")
	if err != nil {
		return err
	}
	stav, err := stavclient.LoadConfig(*stavConfig)
	if err != nil {
		return fmt.Errorf("load STAV configuration: %w", err)
	}
	if stav.TOPSID != *topsID || stav.Mode != string(scope) {
		return fmt.Errorf("STAV configuration does not match selected TOPS and scope")
	}
	layout, err := paths.Resolve(scope, *topsID)
	if err != nil {
		return err
	}
	cfg := config.Config{
		Identity: config.Identity{UID: serviceUID, GID: serviceGID, Subject: stavprotocol.SafeReference{ID: *serviceID, Kind: "symphony.identity.service"}},
		Listen:   config.Listen{Address: layout.Socket, Network: "unix"}, Mode: string(scope), Schema: config.Schema,
		STAVConfig: *stavConfig,
		Submitters: []config.Identity{{UID: submitterUID, GID: submitterGID, Subject: stavprotocol.SafeReference{ID: *submitterID, Kind: *submitterKind}}},
		TOPSID:     *topsID, VocabularyDigest: config.VocabularyDigest,
	}
	changed, err := enrollment.Enroll(layout, cfg)
	if err != nil {
		return err
	}
	fmt.Printf("enrolled Accordare STAV producer tops_id=%s scope=%s config=%s changed=%t; STAV producer grant remains installation-specific\n", *topsID, scope, layout.ConfigFile, changed)
	return nil
}

func runUnenroll(args []string) error {
	set := flag.NewFlagSet("unenroll", flag.ContinueOnError)
	scopeValue := set.String("scope", "user", "installation scope: user or system")
	topsID := set.String("tops-id", "", "immutable TOPS UUID")
	purge := set.Bool("purge", false, "remove configuration and empty outbox")
	if err := set.Parse(args); err != nil || set.NArg() != 0 || *topsID == "" {
		return fmt.Errorf("unenroll requires --tops-id and no positional arguments")
	}
	scope, err := paths.ParseScope(*scopeValue)
	if err != nil {
		return err
	}
	layout, err := paths.Resolve(scope, *topsID)
	if err != nil {
		return err
	}
	changed, err := enrollment.Unenroll(layout, *purge)
	if err != nil {
		return err
	}
	fmt.Printf("unenrolled Accordare STAV producer tops_id=%s scope=%s purge=%t changed=%t\n", *topsID, scope, *purge, changed)
	return nil
}

func runServe(args []string) error {
	set := flag.NewFlagSet("serve", flag.ContinueOnError)
	scopeValue := set.String("scope", "user", "installation scope: user or system")
	topsID := set.String("tops-id", "", "immutable TOPS UUID")
	configPath := set.String("config", "", "explicit configuration path")
	supervised := set.Bool("supervised", false, "assert invocation by the installed native supervisor")
	if err := set.Parse(args); err != nil || set.NArg() != 0 || *topsID == "" {
		return fmt.Errorf("serve requires --tops-id and no positional arguments")
	}
	scope, err := paths.ParseScope(*scopeValue)
	if err != nil {
		return err
	}
	if scope == paths.ScopeSystem && !*supervised {
		return fmt.Errorf("system-scope serve requires native supervision")
	}
	layout, err := paths.Resolve(scope, *topsID)
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
	if cfg.TOPSID != *topsID || cfg.Mode != string(scope) || (*configPath == "" && cfg.Listen.Address != layout.Socket) {
		return fmt.Errorf("configuration does not match selected TOPS and scope")
	}
	stavConfig, err := stavclient.LoadConfig(cfg.STAVConfig)
	if err != nil {
		return err
	}
	stavTransport, err := stavclient.New(stavConfig)
	if err != nil {
		return err
	}
	store, err := outbox.Open(layout.OutboxDir)
	if err != nil {
		return err
	}
	runtimeProducer, err := producer.New(cfg.TOPSID, store, stavTransport)
	if err != nil {
		return err
	}
	service, err := server.New(cfg, runtimeProducer)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Printf("serving Accordare STAV producer tops_id=%s scope=%s socket=%s\n", cfg.TOPSID, cfg.Mode, cfg.Listen.Address)
	return service.Run(ctx)
}

func runControl(operation string, args []string) error {
	set := flag.NewFlagSet(operation, flag.ContinueOnError)
	scopeValue := set.String("scope", "user", "installation scope: user or system")
	topsID := set.String("tops-id", "", "immutable TOPS UUID")
	if err := set.Parse(args); err != nil || set.NArg() != 0 || *topsID == "" {
		return fmt.Errorf("%s requires --tops-id and no positional arguments", operation)
	}
	path, err := accordareclient.ConfigPath(*scopeValue, *topsID)
	if err != nil {
		return err
	}
	client, err := accordareclient.NewFromConfig(path)
	if err != nil {
		return err
	}
	requestID, err := randomUUID()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	var result accordareclient.AuditResult
	if operation == "status" {
		result, err = client.Status(ctx, requestID, *topsID)
	} else {
		result, err = client.Reconcile(ctx, requestID, *topsID)
	}
	if err != nil {
		return err
	}
	if result.Disposition != "succeeded" {
		return fmt.Errorf("producer rejected %s: %s", operation, result.ReasonCode)
	}
	fmt.Printf("Accordare STAV producer: operation=%s tops_id=%s pending=%d\n", operation, *topsID, result.Pending)
	return nil
}

func identityNumbers(scope paths.Scope, uidValue, gidValue, label string) (uint64, uint64, error) {
	if (uidValue == "") != (gidValue == "") {
		return 0, 0, fmt.Errorf("%s UID and GID must be supplied together", label)
	}
	if scope == paths.ScopeSystem && uidValue == "" {
		return 0, 0, fmt.Errorf("system enrollment requires explicit --%s-uid and --%s-gid", label, label)
	}
	if uidValue == "" {
		return uint64(os.Geteuid()), uint64(os.Getegid()), nil
	}
	uid, err := strconv.ParseUint(uidValue, 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid %s UID", label)
	}
	gid, err := strconv.ParseUint(gidValue, 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid %s GID", label)
	}
	return uid, gid, nil
}

func randomUUID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", value[0:4], value[4:6], value[6:8], value[8:10], value[10:16]), nil
}

func usage() {
	fmt.Println("usage: symphony-accordare-stav-producer <install|uninstall|enroll|unenroll|serve|status|reconcile|--version>")
}
