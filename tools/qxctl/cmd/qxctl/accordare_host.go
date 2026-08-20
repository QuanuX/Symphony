package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgebinding"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/maestroclient"
)

const savCurrentInputProtocol = "symphony.sav.current-resolution-input.v1"

type savSourceProjection struct {
	SourceID          string `json:"source_id"`
	OwnerVector       string `json:"owner_vector"`
	OwnerContract     string `json:"owner_contract"`
	Protocol          string `json:"protocol"`
	AuthorityRole     string `json:"authority_role"`
	CollectionState   string `json:"collection_state"`
	ContentDigest     string `json:"content_digest"`
	ObservationDigest any    `json:"observation_digest"`
	ObservedAt        any    `json:"observed_at"`
	Freshness         string `json:"freshness"`
	Payload           any    `json:"payload"`
}

func buildSAVHostCurrent(ctx context.Context, workRoot string, options accordareOptions) ([]byte, error) {
	if options.topsID == "" {
		return nil, fmt.Errorf("--tops-id is required with --host")
	}
	started := time.Now().UTC().Truncate(time.Second)
	operationID, err := randomUUID()
	if err != nil {
		return nil, err
	}
	store, err := knowledgebinding.NewStore(options.stateRoot)
	if err != nil {
		return nil, err
	}
	snapshot, err := store.Snapshot()
	if err != nil {
		return nil, fmt.Errorf("read protected engine binding evidence: %w", err)
	}
	if !snapshot.Exists {
		return nil, fmt.Errorf("knowledge engine binding registry is absent")
	}

	sources := make([]savSourceProjection, 0, len(snapshot.Registry.Bindings)+2)
	required := make([]string, 0, len(snapshot.Registry.Bindings)+2)
	scope := []string{"host.bindings", "host.installations"}
	registryPayload := canonicalJSONValue(snapshot.Registry)
	sources = append(sources, savProjection(
		"host:binding-registry", "knowledge", "knowledge/SPEC.md",
		knowledgebinding.Protocol, "protected_selection", registryPayload,
		snapshot.Registry.RegistryDigest, started))
	required = append(required, "host:binding-registry")

	for _, binding := range snapshot.Registry.Bindings {
		installed, inspectErr := knowledgeengine.InspectInstallation(binding.Role, binding.Prefix, binding.Version)
		if inspectErr != nil {
			return nil, fmt.Errorf("revalidate bound %s installation: %w", binding.Role, inspectErr)
		}
		if installed.ModuleID != binding.ModuleID || installed.EngineID != binding.EngineID ||
			installed.ReceiptDigest != binding.ReceiptDigest || installed.ExecutableDigest != binding.ExecutableDigest ||
			installed.ReceiptPath != binding.ReceiptPath || installed.ExecutablePath != binding.ExecutablePath {
			return nil, fmt.Errorf("bound %s installation changed during CURRENT assembly", binding.Role)
		}
		payload := map[string]any{
			"role": binding.Role, "module_id": binding.ModuleID, "engine_id": binding.EngineID,
			"version": binding.Version, "prefix": binding.Prefix,
			"receipt_path": binding.ReceiptPath, "receipt_digest": binding.ReceiptDigest,
			"executable_path": binding.ExecutablePath, "executable_digest": binding.ExecutableDigest,
			"state": binding.State, "default_receptor": binding.DefaultReceptor,
		}
		id := "host:installation:" + binding.Role
		contract := "modules/" + binding.ModuleID + "/INSTALL.md"
		sources = append(sources, savProjection(
			id, binding.Role, contract, installed.ReceiptProtocol,
			"installed_receipt", payload, installed.ReceiptDigest, started))
		required = append(required, id)
	}

	if options.maestroPrefix != "" {
		installation, inspectErr := knowledgeengine.InspectMaestroInstallation(options.maestroPrefix, options.maestroVersion)
		if inspectErr != nil {
			return nil, inspectErr
		}
		stateRoot, stateErr := maestroStateRoot(options.stateRoot, options.topsID)
		if stateErr != nil {
			return nil, stateErr
		}
		decision, authErr := authorizeMaestroInventory(maestroOptions{
			prefix: options.maestroPrefix, version: options.maestroVersion, repository: workRoot,
			stateRoot: options.stateRoot, topsID: options.topsID, scope: options.scope, ttl: options.ttl,
		})
		if authErr != nil {
			return nil, authErr
		}
		inventory, inventoryErr := maestroclient.Inventory(
			ctx, installation.Prefix, installation.Version, workRoot, stateRoot, options.topsID, decision)
		if inventoryErr != nil {
			return nil, inventoryErr
		}
		payload := canonicalJSONValue(inventory)
		sources = append(sources, savProjection(
			"host:maestro-inventory", "maestro", "modules/maestro/SPEC.md",
			"symphony.maestro.receptor-inventory-result.v1", "maestro_presence", payload,
			inventory.ObservationDigest, started))
		required = append(required, "host:maestro-inventory")
		scope = append(scope, "host.maestro")
	}

	sort.Slice(sources, func(i, j int) bool { return sources[i].SourceID < sources[j].SourceID })
	sort.Strings(required)
	sort.Strings(scope)
	completed := time.Now().UTC().Truncate(time.Second)
	input := map[string]any{
		"protocol": savCurrentInputProtocol, "tops_id": options.topsID, "operation_id": operationID,
		"snapshot_started_at":   started.Format(time.RFC3339),
		"snapshot_completed_at": completed.Format(time.RFC3339),
		"named_version_id":      nil, "named_version_digest": nil,
		"declared_scope": scope, "required_source_ids": required, "sources": sources,
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("encode SAV host CURRENT input: %w", err)
	}
	if err := knowledgeengine.ValidateJSONObject(encoded, 1024*1024); err != nil {
		return nil, fmt.Errorf("assembled SAV host CURRENT input violates bounded JSON: %w", err)
	}
	return encoded, nil
}

func savProjection(sourceID, ownerVector, ownerContract, protocol, authorityRole string,
	payload any, observationDigest string, observedAt time.Time,
) savSourceProjection {
	return savSourceProjection{
		SourceID: sourceID, OwnerVector: ownerVector, OwnerContract: ownerContract,
		Protocol: protocol, AuthorityRole: authorityRole, CollectionState: "available",
		ContentDigest: digestCanonicalJSON(payload), ObservationDigest: observationDigest,
		ObservedAt: observedAt.Format(time.RFC3339), Freshness: "current", Payload: payload,
	}
}

func canonicalJSONValue(value any) any {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("encode internally validated JSON evidence: %v", err))
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var result any
	if err := decoder.Decode(&result); err != nil {
		panic(fmt.Sprintf("decode internally validated JSON evidence: %v", err))
	}
	return result
}

func digestCanonicalJSON(value any) string {
	encoded, err := json.Marshal(canonicalJSONValue(value))
	if err != nil {
		panic(fmt.Sprintf("encode canonical JSON evidence: %v", err))
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}
