package bridgeevm

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type Detector struct {
	chainName     string
	subscriptions map[lookupKey]*subscription
}

type lookupKey [common.AddressLength + common.HashLength]byte

type subscription struct {
	address        common.Address
	eventSignature common.Hash
	bridgeName     string
	bridgeDesc     string
	eventName      string
	correlation    []correlationField
	legType        LegType
}

// New loads the embedded bridge configs for the given chain (case-insensitive)
// and returns a Detector with an O(1) address+topic lookup. Pass one of the
// Chain* constants for the supported set; any other string returns a usable
// Detector with no subscriptions, and Detect will always return ok=false.
func New(chainName string) (*Detector, error) {
	cfgs, err := configsForChain(chainName)
	if err != nil {
		return nil, fmt.Errorf("load bridge configs for %s: %w", chainName, err)
	}

	subs := make(map[lookupKey]*subscription, len(cfgs))
	for _, cfg := range cfgs {
		address := common.HexToAddress(cfg.BridgeContract.Address)
		topic := common.HexToHash(cfg.BridgeTopic.Hash)

		legType := LegType(strings.ToLower(cfg.BridgeTopic.Type))
		subs[makeLookupKey(address, topic)] = &subscription{
			address:        address,
			eventSignature: topic,
			bridgeName:     cfg.BridgeName,
			bridgeDesc:     cfg.BridgeDescription,
			eventName:      cfg.BridgeTopic.Name,
			correlation:    cfg.BridgeTopic.Correlation,
			legType:        legType,
		}
	}

	return &Detector{
		chainName:     strings.ToLower(chainName),
		subscriptions: subs,
	}, nil
}

// Detect returns the bridge details for log if it matches a known bridge.
//
// The boolean indicates whether a configured bridge matched the log's
// (address, topic[0]) pair. The error is non-nil only when a bridge matched but
// correlation extraction failed (malformed log data); in that case the boolean
// is true and Result is the zero value. Callers that just want to know
// "is this a bridge?" can ignore the error.
func (d *Detector) Detect(log *types.Log) (Result, bool, error) {
	if log == nil || len(log.Topics) == 0 {
		return Result{}, false, nil
	}

	sub, ok := d.subscriptions[makeLookupKey(log.Address, log.Topics[0])]
	if !ok {
		return Result{}, false, nil
	}

	correlationID, err := extractCorrelationFields(log, sub.correlation)
	if err != nil {
		return Result{}, true, fmt.Errorf("bridge %s: %w", sub.bridgeName, err)
	}

	return Result{
		BridgeName:        sub.bridgeName,
		BridgeDescription: sub.bridgeDesc,
		LegType:           sub.legType,
		CorrelationID:     correlationID,
		Contract:          sub.address,
		EventTopic:        sub.eventSignature,
		EventName:         sub.eventName,
	}, true, nil
}

// ChainName returns the chain this detector was built for, lowercased.
func (d *Detector) ChainName() string { return d.chainName }

// Len returns the number of (address, topic) subscriptions configured for this
// chain. A Detector with Len() == 0 will always return ok=false from Detect;
// callers can use this to detect chains that have no embedded coverage.
func (d *Detector) Len() int { return len(d.subscriptions) }

func makeLookupKey(address common.Address, topic common.Hash) lookupKey {
	var k lookupKey
	copy(k[:common.AddressLength], address[:])
	copy(k[common.AddressLength:], topic[:])
	return k
}
