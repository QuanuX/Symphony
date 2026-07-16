package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/contracts"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/inventory"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/modules"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/repository"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/ssiagclient"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/status"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/stavclient"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "--help":
		printUsage()
		os.Exit(0)
	case "--version":
		fmt.Printf("qxctl version %s\n", version.Version)
		os.Exit(0)
	case "doctor":
		if err := runDoctor(); err != nil {
			fmt.Printf("doctor failed: %v\n", err)
			os.Exit(1)
		}
	case "contracts":
		if err := runContracts(); err != nil {
			fmt.Printf("contracts failed: %v\n", err)
			os.Exit(1)
		}
	case "ssiag":
		if err := runSSIAG(os.Args[2:]); err != nil {
			fmt.Printf("ssiag failed: %v\n", err)
			os.Exit(1)
		}
	case "stav":
		if err := runSTAV(os.Args[2:]); err != nil {
			fmt.Printf("stav failed: %v\n", err)
			os.Exit(1)
		}
	case "inventory":
		if len(os.Args) == 2 {
			if err := runInventory(false); err != nil {
				fmt.Printf("inventory failed: %v\n", err)
				os.Exit(1)
			}
		} else if len(os.Args) == 3 && os.Args[2] == "--json" {
			if err := runInventory(true); err != nil {
				fmt.Printf("inventory failed: %v\n", err)
				os.Exit(1)
			}
		} else if len(os.Args) == 3 && os.Args[2] == "digest" {
			if err := runInventoryDigest(false); err != nil {
				fmt.Printf("inventory digest failed: %v\n", err)
				os.Exit(1)
			}
		} else if len(os.Args) == 4 && os.Args[2] == "digest" && os.Args[3] == "--json" {
			if err := runInventoryDigest(true); err != nil {
				fmt.Printf("inventory digest failed: %v\n", err)
				os.Exit(1)
			}
		} else {
			printUsage()
			os.Exit(1)
		}
	case "status":
		if len(os.Args) == 2 {
			if err := runStatus(false); err != nil {
				fmt.Printf("status failed: %v\n", err)
				os.Exit(1)
			}
		} else if len(os.Args) == 3 && os.Args[2] == "--json" {
			if err := runStatus(true); err != nil {
				fmt.Printf("status failed: %v\n", err)
				os.Exit(1)
			}
		} else {
			printUsage()
			os.Exit(1)
		}
	case "modules":
		if len(os.Args) == 2 {
			if err := runModules(); err != nil {
				fmt.Printf("modules failed: %v\n", err)
				os.Exit(1)
			}
		} else if len(os.Args) == 3 && os.Args[2] == "check" {
			if err := runModulesCheck(); err != nil {
				fmt.Printf("modules check failed: %v\n", err)
				os.Exit(1)
			}
		} else if len(os.Args) == 3 && os.Args[2] == "metadata" {
			if err := runModulesMetadata(false); err != nil {
				fmt.Printf("modules metadata failed: %v\n", err)
				os.Exit(1)
			}
		} else if len(os.Args) == 4 && os.Args[2] == "metadata" && os.Args[3] == "--json" {
			if err := runModulesMetadata(true); err != nil {
				fmt.Printf("modules metadata failed: %v\n", err)
				os.Exit(1)
			}
		} else {
			printUsage()
			os.Exit(1)
		}
	case "module":
		if len(os.Args) == 4 && os.Args[2] == "inspect" {
			if err := runModuleInspect(os.Args[3]); err != nil {
				fmt.Printf("module inspect failed: %v\n", err)
				os.Exit(1)
			}
		} else if len(os.Args) == 4 && os.Args[2] == "check" {
			if err := runModuleCheck(os.Args[3]); err != nil {
				fmt.Printf("module check failed: %v\n", err)
				os.Exit(1)
			}
		} else if len(os.Args) == 4 && os.Args[2] == "metadata" {
			if err := runModuleMetadata(os.Args[3], false); err != nil {
				fmt.Printf("module metadata failed: %v\n", err)
				os.Exit(1)
			}
		} else if len(os.Args) == 5 && os.Args[2] == "metadata" && os.Args[4] == "--json" {
			if err := runModuleMetadata(os.Args[3], true); err != nil {
				fmt.Printf("module metadata failed: %v\n", err)
				os.Exit(1)
			}
		} else {
			printUsage()
			os.Exit(1)
		}
	default:
		fmt.Printf("unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("qxctl - Symphony administrative spine")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  qxctl <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  --help                            Print concise usage")
	fmt.Println("  --version                         Print version")
	fmt.Println("  doctor                            Perform local repository/admin-spine checks")
	fmt.Println("  contracts                         Verify first runtime-set module contract surfaces")
	fmt.Println("  modules                           List deterministic runtime modules")
	fmt.Println("  modules check                     Verify contract shape for all modules")
	fmt.Println("  modules metadata [--json]         Extract contract metadata for all modules")
	fmt.Println("  module inspect <module-name>      Inspect a specific runtime module")
	fmt.Println("  module check <module-name>        Verify contract shape for a module")
	fmt.Println("  module metadata <module-name> [--json] Extract contract metadata for a module")
	fmt.Println("  inventory [--json]                Emit deterministic runtime inventory snapshot")
	fmt.Println("  inventory digest [--json]         Emit deterministic runtime inventory SHA-256 digest")
	fmt.Println("  status [--json]                   Report consolidated administrative status")
	fmt.Println("  ssiag status --tops-id UUID [--scope user|system] [--json] Read safe SSIAG status")
	fmt.Println("  ssiag providers --tops-id UUID [--scope user|system] [--json] List safe provider metadata")
	fmt.Println("  ssiag doctor --tops-id UUID [--scope user|system] Verify local SSIAG availability")
	fmt.Println("  stav status --tops-id UUID [--scope user|system] [--json] Read authenticated STAV status")
	fmt.Println("  stav verify --tops-id UUID [--scope user|system] [--json] Verify the STAV digest chain")
	fmt.Println("  stav query --tops-id UUID [--scope user|system] [bounded filters] [--json] Query authorized STAV projections")
	fmt.Println("  stav doctor --tops-id UUID [--scope user|system] Run authenticated STAV diagnostics")
}

func runSTAV(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("STAV subcommand is required: status, verify, query, or doctor")
	}
	subcommand := args[0]
	if subcommand == "append" {
		return fmt.Errorf("qxctl stav append is prohibited; qxctl never submits arbitrary events or edits ledgers")
	}
	switch subcommand {
	case "status", "verify", "query", "doctor":
	default:
		return fmt.Errorf("unknown STAV subcommand %q", subcommand)
	}

	set := flag.NewFlagSet("stav "+subcommand, flag.ContinueOnError)
	topsID := set.String("tops-id", "", "immutable TOPS UUID")
	scope := set.String("scope", "user", "STAV scope: user or system")
	jsonOutput := false
	if subcommand != "doctor" {
		set.BoolVar(&jsonOutput, "json", false, "emit JSON")
	}
	var query stavprotocol.Query
	var throughSequence optionalUint64
	var verifyAfter uint64
	var verifyThrough optionalUint64
	if subcommand == "query" {
		query.Schema = stavprotocol.SchemaQuery
		query.EventClasses = make([]string, 0)
		query.Outcomes = make([]string, 0)
		query.Limit = 100
		set.Uint64Var(&query.AfterSequence, "after-sequence", 0, "exclusive sequence cursor")
		set.Var(&throughSequence, "through-sequence", "optional inclusive sequence ceiling")
		set.StringVar(&query.FromTime, "from-time", "", "optional inclusive UTC timestamp")
		set.StringVar(&query.ThroughTime, "through-time", "", "optional inclusive UTC timestamp")
		set.Var((*stringList)(&query.EventClasses), "event-class", "registered event class; repeat up to 16 times")
		set.Var((*stringList)(&query.Outcomes), "outcome", "generic outcome; repeat up to 5 times")
		set.StringVar(&query.CorrelationID, "correlation-id", "", "optional correlation UUID")
		set.StringVar(&query.RequestID, "request-id", "", "optional request UUID")
		set.Uint64Var(&query.Limit, "limit", 100, "page size from 1 through 1000")
	}
	if subcommand == "verify" {
		set.Uint64Var(&verifyAfter, "after-sequence", 0, "exclusive verification cursor")
		set.Var(&verifyThrough, "through-sequence", "optional inclusive verification ceiling")
	}
	if err := set.Parse(args[1:]); err != nil {
		return err
	}
	if set.NArg() != 0 {
		return fmt.Errorf("unexpected STAV arguments: %v", set.Args())
	}
	if *topsID == "" {
		return fmt.Errorf("--tops-id is required")
	}
	if _, err := stavclient.SocketForTOPS(*scope, *topsID); err != nil {
		return err
	}
	if subcommand == "query" {
		query.TOPSID = *topsID
		if throughSequence.set {
			value := throughSequence.value
			query.ThroughSequence = &value
		}
		if _, err := stavprotocol.EncodeQuery(query); err != nil {
			return fmt.Errorf("invalid bounded STAV query: %w", err)
		}
	}
	if subcommand == "verify" && verifyThrough.set && verifyThrough.value <= verifyAfter {
		return fmt.Errorf("verification through-sequence must follow after-sequence")
	}
	client, err := stavclient.NewForTOPS(*scope, *topsID)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	requestID, err := stavprotocol.GenerateUUIDv4()
	if err != nil {
		return err
	}
	request := stavprotocol.LocalRequest{
		Operation: subcommand,
		RequestID: requestID,
		Schema:    stavprotocol.SchemaLocalRequest,
		TOPSID:    *topsID,
	}
	switch subcommand {
	case "query":
		request.Query = &query
	case "verify":
		verify := stavprotocol.VerifyRequest{AfterSequence: verifyAfter}
		if verifyThrough.set {
			value := verifyThrough.value
			verify.ThroughSequence = &value
		}
		request.Verify = &verify
	case "doctor":
		request.Operation = stavprotocol.LocalOperationStatus
	}
	response, err := client.Do(ctx, request)
	if err != nil {
		return err
	}
	if response.Disposition != stavprotocol.LocalDispositionSucceeded {
		return fmt.Errorf("STAV %s rejected: %s", subcommand, response.ReasonCode)
	}
	switch subcommand {
	case "status":
		if jsonOutput {
			return printSTAVJSON(response.Status)
		}
		fmt.Printf("STAV: ready=%t tops_id=%s mode=%s events=%d ledger_bytes=%d storage=%s\n", response.Status.Ready, response.Status.TOPSID, response.Status.Mode, response.Status.Events, response.Status.LedgerBytes, response.Status.StorageState)
		return nil
	case "verify":
		if jsonOutput {
			return printSTAVJSON(response.Verification)
		}
		fmt.Printf("STAV verification: state=%s tops_id=%s after=%d through=%d checked=%d\n", response.Verification.Result.State, response.Verification.TOPSID, response.Verification.AfterSequence, response.Verification.ThroughSequence, response.Verification.EventsChecked)
		if response.Verification.Result.State != "verified" {
			return fmt.Errorf("STAV verification failed at sequence %d: %s", response.Verification.Result.AtSequence, response.Verification.Result.ReasonCode)
		}
		return nil
	case "query":
		if jsonOutput {
			return printSTAVJSON(response.Page)
		}
		for _, entry := range response.Page.Entries {
			fmt.Printf("STAV event: sequence=%d timestamp=%s class=%s operation=%s outcome=%s reason=%s request_id=%s\n", entry.Sequence, entry.Projection.Timestamp, entry.Projection.EventClass, entry.Projection.OperationID, entry.Projection.Outcome, entry.Projection.ReasonCode, entry.Projection.RequestID)
		}
		fmt.Printf("STAV query: entries=%d next=%s\n", len(response.Page.Entries), response.Page.Next.State)
		return nil
	case "doctor":
		if !response.Status.Ready {
			return fmt.Errorf("STAV append authority is not ready")
		}
		verifyID, err := stavprotocol.GenerateUUIDv4()
		if err != nil {
			return err
		}
		verificationResponse, err := client.Do(ctx, stavprotocol.LocalRequest{
			Operation: stavprotocol.LocalOperationVerify,
			RequestID: verifyID,
			Schema:    stavprotocol.SchemaLocalRequest,
			TOPSID:    *topsID,
			Verify:    &stavprotocol.VerifyRequest{AfterSequence: 0},
		})
		if err != nil {
			return err
		}
		if verificationResponse.Disposition != stavprotocol.LocalDispositionSucceeded || verificationResponse.Verification.Result.State != "verified" {
			return fmt.Errorf("STAV doctor chain verification failed")
		}
		fmt.Printf("STAV doctor: tops_id=%s ready=true events=%d storage=%s chain=verified endpoint=authenticated\n", response.Status.TOPSID, response.Status.Events, response.Status.StorageState)
		fmt.Println("STAV doctor: checks passed")
		return nil
	}
	return fmt.Errorf("unsupported STAV command")
}

func printSTAVJSON(value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

type stringList []string

func (s *stringList) String() string { return fmt.Sprint([]string(*s)) }

func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type optionalUint64 struct {
	set   bool
	value uint64
}

func (v *optionalUint64) String() string {
	if !v.set {
		return ""
	}
	return strconv.FormatUint(v.value, 10)
}

func (v *optionalUint64) Set(value string) error {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid non-negative integer: %w", err)
	}
	v.set = true
	v.value = parsed
	return nil
}

func runSSIAG(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("SSIAG subcommand is required: status, providers, or doctor")
	}
	set := flag.NewFlagSet("ssiag "+args[0], flag.ContinueOnError)
	jsonOutput := set.Bool("json", false, "emit JSON")
	scope := set.String("scope", "user", "SSIAG scope: user or system")
	topsID := set.String("tops-id", "", "immutable TOPS UUID")
	if err := set.Parse(args[1:]); err != nil {
		return err
	}
	if set.NArg() != 0 {
		return fmt.Errorf("unexpected SSIAG arguments: %v", set.Args())
	}
	if *topsID == "" {
		*topsID = os.Getenv("SYMPHONY_SSIAG_TOPS_ID")
	}
	if *topsID == "" {
		return fmt.Errorf("--tops-id or SYMPHONY_SSIAG_TOPS_ID is required")
	}
	socket, err := ssiagclient.SocketForTOPS(*scope, *topsID)
	if err != nil {
		return err
	}
	client := ssiagclient.New(socket, 4*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	switch args[0] {
	case "status":
		status, err := requireSSIAGStatus(ctx, client, *topsID, *scope)
		if err != nil {
			return err
		}
		if *jsonOutput {
			return printSSIAGJSON(status)
		}
		fmt.Printf("SSIAG: %s version=%s ready=%t tops_id=%s tops_name=%q mode=%s providers=%d\n", status.Name, status.Version, status.Ready, status.TOPSID, status.TOPSName, status.Mode, status.ProviderCount)
		return nil
	case "providers":
		if _, err := requireSSIAGStatus(ctx, client, *topsID, *scope); err != nil {
			return err
		}
		providers, err := client.Providers(ctx)
		if err != nil {
			return err
		}
		if *jsonOutput {
			return printSSIAGJSON(providers)
		}
		if len(providers.Providers) == 0 {
			fmt.Println("SSIAG providers: none declared")
			return nil
		}
		for _, provider := range providers.Providers {
			fmt.Printf("SSIAG provider: %s kind=%s status=%s\n", provider.Name, provider.Kind, provider.Status)
		}
		return nil
	case "doctor":
		if *jsonOutput {
			return fmt.Errorf("SSIAG doctor does not support --json")
		}
		status, err := requireSSIAGStatus(ctx, client, *topsID, *scope)
		if err != nil {
			return err
		}
		providers, err := client.Providers(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("SSIAG doctor: schema=%s tops_id=%s ready=true providers=%d\n", status.Schema, status.TOPSID, len(providers.Providers))
		fmt.Println("SSIAG doctor: checks passed")
		return nil
	default:
		return fmt.Errorf("unknown SSIAG subcommand %q", args[0])
	}
}

func requireSSIAGStatus(ctx context.Context, client *ssiagclient.Client, topsID, scope string) (ssiagclient.Status, error) {
	status, err := client.Status(ctx)
	if err != nil {
		return ssiagclient.Status{}, err
	}
	if status.TOPSID != topsID {
		return ssiagclient.Status{}, fmt.Errorf("SSIAG response TOPS ID does not match requested identity")
	}
	if status.Mode != scope {
		return ssiagclient.Status{}, fmt.Errorf("SSIAG response mode does not match requested scope")
	}
	if !status.Ready {
		return ssiagclient.Status{}, fmt.Errorf("SSIAG is not ready")
	}
	return status, nil
}

func printSSIAGJSON(value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func runDoctor() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("could not find Symphony repository root: %w", err)
	}
	fmt.Printf("found repository root: %s\n", repoRoot)

	expectedModules := []string{
		"node-troll",
		"bus-troll",
		"hotpath-runtime",
	}

	for _, mod := range expectedModules {
		modPath := filepath.Join(repoRoot, "modules", mod)
		if !repository.IsDir(modPath) {
			return fmt.Errorf("missing expected module directory: modules/%s", mod)
		}
		fmt.Printf("verified module exists: modules/%s\n", mod)
	}

	validatorPath := filepath.Join(repoRoot, "tools", "symphony-validator")
	if !repository.IsDir(validatorPath) {
		return fmt.Errorf("missing validator directory: tools/symphony-validator")
	}
	fmt.Println("verified validator exists: tools/symphony-validator")

	fmt.Println("doctor checks passed")
	return nil
}

func runModules() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("could not find Symphony repository root: %w", err)
	}

	output, err := modules.List(repoRoot)
	for _, line := range output {
		fmt.Println(line)
	}
	if err != nil {
		fmt.Println("modules: checks failed")
		return err
	}
	return nil
}

func runModulesCheck() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("could not find Symphony repository root: %w", err)
	}

	output, err := modules.CheckAll(repoRoot)
	for _, line := range output {
		fmt.Println(line)
	}
	if err != nil {
		fmt.Println("modules check: checks failed")
		return err
	}
	return nil
}

func runModuleInspect(moduleName string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("could not find Symphony repository root: %w", err)
	}

	output, err := modules.Inspect(repoRoot, moduleName)
	for _, line := range output {
		fmt.Println(line)
	}
	if err != nil {
		fmt.Println("inspection: checks failed")
		return err
	}
	return nil
}

func runModuleCheck(moduleName string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("could not find Symphony repository root: %w", err)
	}

	output, err := modules.Check(repoRoot, moduleName)
	for _, line := range output {
		fmt.Println(line)
	}
	if err != nil {
		fmt.Println("module check: checks failed")
		return err
	}
	return nil
}

func runContracts() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("could not find Symphony repository root: %w", err)
	}

	output, err := contracts.Verify(repoRoot)
	for _, line := range output {
		fmt.Println(line)
	}
	if err != nil {
		fmt.Println("contracts: checks failed")
		return err
	}
	return nil
}

func runModuleMetadata(moduleName string, jsonOutput bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("could not find Symphony repository root: %w", err)
	}

	if jsonOutput {
		outputBytes, err := modules.MetadataJSON(repoRoot, moduleName)
		if err != nil {
			fmt.Printf("module metadata failed: %v\n", err)
			return err
		}
		fmt.Println(string(outputBytes))
		return nil
	}

	output, err := modules.Metadata(repoRoot, moduleName)
	for _, line := range output {
		fmt.Println(line)
	}
	if err != nil {
		fmt.Println("module metadata: checks failed")
		return err
	}
	return nil
}

func runModulesMetadata(jsonOutput bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("could not find Symphony repository root: %w", err)
	}

	if jsonOutput {
		outputBytes, err := modules.MetadataAllJSON(repoRoot)
		if err != nil {
			fmt.Printf("modules metadata failed: %v\n", err)
			return err
		}
		fmt.Println(string(outputBytes))
		return nil
	}

	output, err := modules.MetadataAll(repoRoot)
	for _, line := range output {
		fmt.Println(line)
	}
	if err != nil {
		fmt.Println("modules metadata: checks failed")
		return err
	}
	return nil
}

func runInventory(jsonOutput bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("could not find Symphony repository root: %w", err)
	}

	if jsonOutput {
		outputBytes, err := inventory.SnapshotJSON(repoRoot)
		if err != nil {
			return err
		}
		fmt.Println(string(outputBytes))
		return nil
	}

	output, err := inventory.Snapshot(repoRoot)
	if err != nil {
		return err
	}
	for _, line := range output {
		fmt.Println(line)
	}
	return nil
}

func runInventoryDigest(jsonOutput bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("could not find Symphony repository root: %w", err)
	}

	if jsonOutput {
		outputBytes, err := inventory.DigestJSON(repoRoot)
		if err != nil {
			return err
		}
		fmt.Println(string(outputBytes))
		return nil
	}

	output, err := inventory.Digest(repoRoot)
	if err != nil {
		return err
	}
	for _, line := range output {
		fmt.Println(line)
	}
	return nil
}

func runStatus(jsonOutput bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	repoRoot, err := repository.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("could not find Symphony repository root: %w", err)
	}

	if jsonOutput {
		outputBytes, err := status.ReportJSON(repoRoot)
		if err != nil {
			return err
		}
		fmt.Println(string(outputBytes))
		return nil
	}

	output, err := status.Report(repoRoot)
	if err != nil {
		return err
	}
	for _, line := range output {
		fmt.Println(line)
	}
	return nil
}
