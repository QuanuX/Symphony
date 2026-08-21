package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	accordareclient "github.com/QuanuX/Symphony/modules/accordare-stav-producer/client"
	appendclient "github.com/QuanuX/Symphony/modules/stav-append-authority/client"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/commandregistry"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/ssiagclient"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/stavclient"
	"github.com/spf13/cobra"
)

const accordareProducerID = "accordare-stav-producer"

var accordarePermissions = []stavprotocol.PeerPermission{
	{EventClass: "symphony.sav.named-version.lifecycle", OperationID: "symphony.sav.named-version.alias"},
	{EventClass: "symphony.sav.named-version.lifecycle", OperationID: "symphony.sav.named-version.prepare"},
	{EventClass: "symphony.sav.named-version.lifecycle", OperationID: "symphony.sav.named-version.recover"},
	{EventClass: "symphony.sav.named-version.lifecycle", OperationID: "symphony.sav.named-version.seal"},
}

type stavGrantOptions struct {
	topsID, scope, operationID, expectedConfigDigest string
	ttl                                              time.Duration
	jsonOutput                                       bool
}

func newSTAVAccordareGrantCommand() *cobra.Command {
	family := structural("accordare-grant", fmt.Errorf("accordare-grant subcommand is required: install or remove"))
	for _, operation := range []string{"install", "remove"} {
		options := stavGrantOptions{scope: "user", ttl: 15 * time.Minute}
		leaf := &cobra.Command{
			Use: operation,
			Args: func(_ *cobra.Command, args []string) error {
				if len(args) != 0 {
					return fmt.Errorf("unexpected STAV Accordare grant arguments: %v", args)
				}
				return nil
			},
			RunE: func(_ *cobra.Command, _ []string) error { return runSTAVAccordareGrant(operation, options) },
		}
		spec := commandSpec("stav.accordare-grant."+operation, featureSTAV, "apply")
		spec.Mutability = "permission_backed_mutation"
		spec.AuthorityMode = "ssiag"
		spec.OutputProtocols = []string{"symphony.stav.grant-result.v1"}
		spec.ResultValidationProtocols = []string{"symphony.stav.grant-result.v1"}
		spec.RecoveryCommandID = stringPointer("qxcmd:symphony:stav.accordare-grant." + operation)
		commandregistry.Attach(leaf, spec)
		leaf.Flags().StringVar(&options.topsID, "tops-id", "", "immutable TOPS UUID")
		leaf.Flags().StringVar(&options.scope, "scope", "user", "STAV scope: user or system")
		leaf.Flags().StringVar(&options.operationID, "operation-id", "", "stable idempotent administration operation token")
		leaf.Flags().StringVar(&options.expectedConfigDigest, "expected-config-digest", "", "exact observed tagged STAV configuration digest")
		leaf.Flags().DurationVar(&options.ttl, "ttl", 15*time.Minute, "SSIAG capability lifetime")
		leaf.Flags().BoolVar(&options.jsonOutput, "json", false, "emit JSON")
		family.AddCommand(leaf)
	}
	return family
}

func runSTAVAccordareGrant(operation string, options stavGrantOptions) error {
	if options.topsID == "" || !validSessionToken(options.operationID) || !validTaggedDigest(options.expectedConfigDigest) {
		return fmt.Errorf("grant administration requires --tops-id, stable --operation-id, and exact --expected-config-digest")
	}
	if options.scope != "user" && options.scope != "system" {
		return fmt.Errorf("--scope must be user or system")
	}
	if options.ttl <= 0 || options.ttl > 24*time.Hour {
		return fmt.Errorf("--ttl must be greater than zero and no more than 24h")
	}
	stavConfigPath, err := stavclient.ConfigForTOPS(options.scope, options.topsID)
	if err != nil {
		return err
	}
	if override := os.Getenv("SYMPHONY_STAV_CONFIG"); override != "" {
		if !filepath.IsAbs(override) {
			return fmt.Errorf("SYMPHONY_STAV_CONFIG must be absolute")
		}
		stavConfigPath = filepath.Clean(override)
	}
	raw, err := readSafeConfig(stavConfigPath)
	if err != nil {
		return err
	}
	actualDigest := taggedSHA256Bytes(raw)
	journalPath := stavConfigPath + ".accordare-grant-attempt.json"
	if err := recoverGrantAttempt(journalPath, options.topsID, actualDigest); err != nil {
		return err
	}
	if actualDigest != options.expectedConfigDigest {
		return fmt.Errorf("STAV configuration changed: actual=%s", actualDigest)
	}
	cfg, err := appendclient.LoadConfig(stavConfigPath)
	if err != nil {
		return err
	}
	if cfg.TOPSID != options.topsID || cfg.Mode != options.scope {
		return fmt.Errorf("STAV configuration does not match selected TOPS and scope")
	}
	if info, statErr := os.Lstat(cfg.Listen.Address); statErr == nil && info.Mode()&os.ModeSocket != 0 {
		return fmt.Errorf("STAV append authority must be stopped before grant mutation")
	} else if statErr != nil && !os.IsNotExist(statErr) {
		return statErr
	}
	producerConfigPath, err := accordareclient.ConfigPath(options.scope, options.topsID)
	if err != nil {
		return err
	}
	producerConfig, err := accordareclient.LoadConfig(producerConfigPath)
	if err != nil {
		return fmt.Errorf("exact Accordare producer enrollment is required: %w", err)
	}
	producerSubject := stavprotocol.SafeReference{ID: accordareProducerID, Kind: "symphony.stav.producer"}
	desired := stavprotocol.ProducerGrant{GID: producerConfig.Identity.GID, UID: producerConfig.Identity.UID, Producer: producerSubject, Subject: producerConfig.Identity.Subject, Permissions: append([]stavprotocol.PeerPermission(nil), accordarePermissions...)}
	resource := stavGrantResource(operation, options, actualDigest, desired)
	if _, err := authorizeSTAVGrant(options, operation, resource); err != nil {
		return err
	}
	updated, changed, err := setAccordareGrant(cfg, desired, operation == "install")
	if err != nil {
		return err
	}
	if !changed {
		return printSTAVGrantResult(options.jsonOutput, operation, options.topsID, false, actualDigest, actualDigest)
	}
	encoded, err := stavprotocol.EncodeAppendAuthorityConfig(updated)
	if err != nil {
		return err
	}
	newDigest := taggedSHA256Bytes(encoded)
	journal := map[string]any{"schema": "symphony.stav.grant-attempt.v1", "operation": operation, "operation_id": options.operationID, "tops_id": options.topsID, "previous_config_digest": actualDigest, "new_config_digest": newDigest}
	journalBytes, _ := json.Marshal(journal)
	if err := writeAtomicMode(journalPath, append(journalBytes, '\n'), 0o600); err != nil {
		return fmt.Errorf("persist grant recovery attempt: %w", err)
	}
	if err := writeAtomicMode(stavConfigPath, encoded, 0o600); err != nil {
		return err
	}
	verified, err := readSafeConfig(stavConfigPath)
	if err != nil || taggedSHA256Bytes(verified) != newDigest {
		return fmt.Errorf("verify committed STAV grant configuration")
	}
	if err := os.Remove(journalPath); err != nil {
		return fmt.Errorf("grant committed but recovery marker cleanup failed: %w", err)
	}
	if err := syncDir(filepath.Dir(journalPath)); err != nil {
		return err
	}
	return printSTAVGrantResult(options.jsonOutput, operation, options.topsID, true, actualDigest, newDigest)
}

type grantAttempt struct {
	Schema               string `json:"schema"`
	Operation            string `json:"operation"`
	OperationID          string `json:"operation_id"`
	TOPSID               string `json:"tops_id"`
	PreviousConfigDigest string `json:"previous_config_digest"`
	NewConfigDigest      string `json:"new_config_digest"`
}

func recoverGrantAttempt(path, topsID, actualDigest string) error {
	data, err := readSafeConfig(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		// readSafeConfig intentionally collapses stat errors, so distinguish a
		// genuinely absent marker without accepting an unsafe object.
		if _, statErr := os.Lstat(path); os.IsNotExist(statErr) {
			return nil
		}
		return fmt.Errorf("STAV grant recovery marker is unsafe: %w", err)
	}
	var attempt grantAttempt
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&attempt); err != nil {
		return fmt.Errorf("STAV grant recovery marker is invalid")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("STAV grant recovery marker is invalid")
	}
	if attempt.Schema != "symphony.stav.grant-attempt.v1" || attempt.TOPSID != topsID ||
		(attempt.Operation != "install" && attempt.Operation != "remove") || !validSessionToken(attempt.OperationID) ||
		!validTaggedDigest(attempt.PreviousConfigDigest) || !validTaggedDigest(attempt.NewConfigDigest) ||
		(actualDigest != attempt.PreviousConfigDigest && actualDigest != attempt.NewConfigDigest) {
		return fmt.Errorf("STAV grant recovery cannot reconcile current configuration")
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	return syncDir(filepath.Dir(path))
}

func printSTAVGrantResult(jsonOutput bool, operation, topsID string, changed bool, previous, current string) error {
	result := map[string]any{"protocol": "symphony.stav.grant-result.v1", "operation": operation, "tops_id": topsID, "changed": changed, "previous_config_digest": previous, "current_config_digest": current, "authority_restart_required": changed}
	if jsonOutput {
		return printIndentedJSON(result)
	}
	fmt.Printf("STAV Accordare grant: operation=%s tops_id=%s changed=%t previous=%s current=%s restart_required=%t\n", operation, topsID, changed, previous, current, changed)
	return nil
}

func setAccordareGrant(cfg stavprotocol.AppendAuthorityConfig, desired stavprotocol.ProducerGrant, install bool) (stavprotocol.AppendAuthorityConfig, bool, error) {
	index := -1
	for candidate := range cfg.Authentication.Producers {
		if cfg.Authentication.Producers[candidate].Producer == desired.Producer {
			index = candidate
			break
		}
	}
	if install {
		if index >= 0 {
			if grantsEqual(cfg.Authentication.Producers[index], desired) {
				return cfg, false, nil
			}
			return stavprotocol.AppendAuthorityConfig{}, false, fmt.Errorf("existing Accordare grant differs; remove it with exact expected state before replacement")
		}
		cfg.Authentication.Producers = append(cfg.Authentication.Producers, desired)
		sort.Slice(cfg.Authentication.Producers, func(i, j int) bool {
			return cfg.Authentication.Producers[i].Producer.ID < cfg.Authentication.Producers[j].Producer.ID
		})
	} else {
		if index < 0 {
			return cfg, false, nil
		}
		cfg.Authentication.Producers = append(cfg.Authentication.Producers[:index], cfg.Authentication.Producers[index+1:]...)
	}
	if err := cfg.Validate(); err != nil {
		return stavprotocol.AppendAuthorityConfig{}, false, err
	}
	return cfg, true, nil
}

func grantsEqual(left, right stavprotocol.ProducerGrant) bool {
	leftJSON, _ := json.Marshal(left)
	rightJSON, _ := json.Marshal(right)
	return bytes.Equal(leftJSON, rightJSON)
}

func stavGrantResource(operation string, options stavGrantOptions, digest string, grant stavprotocol.ProducerGrant) string {
	value := map[string]any{"operation": operation, "operation_id": options.operationID, "tops_id": options.topsID, "scope": options.scope, "expected_config_digest": digest, "grant": grant}
	encoded, _ := json.Marshal(value)
	return "symphony.stav.accordare-grant:" + hex.EncodeToString(sha256Sum(encoded))
}

func authorizeSTAVGrant(options stavGrantOptions, operation, resource string) (ssiagclient.AuthorizationDecision, error) {
	client, err := ssiagclient.NewForTOPS(options.scope, options.topsID, 4*time.Second)
	if err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if _, err := requireSSIAGStatus(ctx, client, options.topsID, options.scope); err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	requestID, err := randomUUID()
	if err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	correlationID, err := randomUUID()
	if err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	now := time.Now().UTC().Truncate(time.Second)
	request := ssiagclient.AuthorizationRequest{Schema: "symphony.ssiag.authorization-request.v1", RequestID: requestID, CorrelationID: correlationID, Operation: "symphony.stav.accordare-grant." + operation, Resource: resource, Audience: "qxctl", Scope: "tops:" + options.topsID, RequestedAt: now, RequestedExpiresAt: now.Add(options.ttl).UTC().Truncate(time.Second)}
	decision, err := client.Authorize(ctx, request)
	if err != nil {
		return ssiagclient.AuthorizationDecision{}, err
	}
	if err := validateSessionAuthorization(decision, request, options.topsID); err != nil {
		return ssiagclient.AuthorizationDecision{}, fmt.Errorf("SSIAG STAV grant authorization rejected: %w", err)
	}
	return decision, nil
}

func readSafeConfig(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o022 != 0 || info.Size() <= 0 || info.Size() > 1<<20 {
		return nil, fmt.Errorf("STAV configuration is unavailable or unsafe")
	}
	return os.ReadFile(path)
}

func writeAtomicMode(path string, data []byte, mode os.FileMode) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".stav-grant-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err := temporary.Chmod(mode); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	return syncDir(filepath.Dir(path))
}

func syncDir(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func taggedSHA256Bytes(data []byte) string { return "sha256:" + hex.EncodeToString(sha256Sum(data)) }

func sha256Sum(data []byte) []byte {
	digest := sha256.Sum256(data)
	return digest[:]
}
