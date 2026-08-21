package producer

import (
	"context"
	"fmt"

	stavprotocol "github.com/QuanuX/Symphony/libraries/stav-protocol-go"
	"github.com/QuanuX/Symphony/modules/accordare-stav-producer/internal/outbox"
)

type Transport interface {
	Do(context.Context, stavprotocol.LocalRequest) (stavprotocol.LocalResponse, error)
}

type Producer struct {
	topsID    string
	outbox    *outbox.Store
	transport Transport
}

func New(topsID string, store *outbox.Store, transport Transport) (*Producer, error) {
	if err := stavprotocol.ValidateTOPSID(topsID); err != nil {
		return nil, err
	}
	if store == nil || transport == nil {
		return nil, fmt.Errorf("producer requires an outbox and STAV transport")
	}
	return &Producer{topsID: topsID, outbox: store, transport: transport}, nil
}

func (producer *Producer) Submit(ctx context.Context, candidate stavprotocol.Candidate, digest string) (stavprotocol.Receipt, bool, error) {
	if candidate.Topology.TOPSID != producer.topsID {
		return stavprotocol.Receipt{}, false, fmt.Errorf("candidate TOPS mismatch")
	}
	if err := producer.outbox.Put(candidate, digest); err != nil {
		return stavprotocol.Receipt{}, false, err
	}
	receipt, err := producer.append(ctx, candidate, digest)
	if err != nil {
		return stavprotocol.Receipt{}, true, err
	}
	if err := producer.outbox.Remove(candidate.Correlation.RequestID); err != nil {
		// The committed STAV receipt is authoritative. A retained identical
		// outbox entry is safe: reconciliation replays idempotently.
		return receipt, true, fmt.Errorf("receipt committed but outbox cleanup requires reconciliation: %w", err)
	}
	return receipt, false, nil
}

func (producer *Producer) Reconcile(ctx context.Context) (uint64, uint64, error) {
	pending, err := producer.outbox.List()
	if err != nil {
		return 0, 0, err
	}
	var committed uint64
	for _, item := range pending {
		if _, err := producer.append(ctx, item.Candidate, item.CandidateDigest); err != nil {
			continue
		}
		if err := producer.outbox.Remove(item.Candidate.Correlation.RequestID); err != nil {
			return committed, uint64(len(pending)) - committed, err
		}
		committed++
	}
	remaining, err := producer.outbox.Count()
	return committed, remaining, err
}

func (producer *Producer) Pending() (uint64, error) { return producer.outbox.Count() }

func (producer *Producer) append(ctx context.Context, candidate stavprotocol.Candidate, digest string) (stavprotocol.Receipt, error) {
	response, err := producer.transport.Do(ctx, stavprotocol.LocalRequest{
		Candidate: &candidate, Operation: stavprotocol.LocalOperationAppend,
		RequestID: candidate.Correlation.RequestID, Schema: stavprotocol.SchemaLocalRequest, TOPSID: producer.topsID,
	})
	if err != nil {
		return stavprotocol.Receipt{}, err
	}
	if err := response.Validate(); err != nil || response.Disposition != stavprotocol.LocalDispositionSucceeded || response.Receipt == nil {
		return stavprotocol.Receipt{}, fmt.Errorf("STAV returned an invalid append response")
	}
	receipt := *response.Receipt
	if receipt.Disposition != "committed" || receipt.CandidateDigest != digest || receipt.RequestID != candidate.Correlation.RequestID {
		return stavprotocol.Receipt{}, fmt.Errorf("STAV did not commit the exact candidate")
	}
	return receipt, nil
}
