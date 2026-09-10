package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/knowledgeengine"
	"github.com/QuanuX/Symphony/tools/qxctl/internal/scvtransport"
)

func runSCVAcquire(options scvOptions) error {
	input, err := scvInput(options)
	if err != nil {
		return err
	}
	if len(input) != 2 || input["source"] == nil || input["locator_id"] == nil {
		return fmt.Errorf("acquire requires exactly source and locator_id")
	}
	locator, ok := input["locator_id"].(string)
	if !ok {
		return fmt.Errorf("locator_id must be text")
	}
	source, err := knowledgeengine.SCVCanonical(input["source"])
	if err != nil {
		return err
	}
	// Reject invalid/tampered source and installation before starting any network
	// request. The owner validates every capture again after transport finishes.
	if _, err := invokeSCV(options, "source_status", map[string]any{"source": json.RawMessage(source)}); err != nil {
		return err
	}
	capture, err := scvtransport.Acquire(context.Background(), scvtransport.Input{Source: source, LocatorID: locator})
	if err != nil {
		return err
	}
	result, err := invokeSCV(options, "capture_import", capture)
	if err != nil {
		return err
	}
	return outputSCV(options, result)
}
