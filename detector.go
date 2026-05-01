package bridgeevm

import (
	"bytes"
	"fmt"
	"sort"
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

// Subscription is a (contract, event-topic) pair the Detector matches against.
// Consumers building an eth_subscribeFilterLogs FilterQuery iterate
// Subscriptions and pass the deduped addresses and topics into the query.
type Subscription struct {
	BridgeAddress  common.Address
	EventSignature common.Hash
}

// Subscriptions returns every (address, topic) pair this Detector watches.
// Order is deterministic across calls, sorted by address bytes then topic
// bytes, so callers can build stable FilterQuery payloads. The returned
// slice is freshly allocated on each call; callers may mutate it.
func (d *Detector) Subscriptions() []Subscription {
	out := make([]Subscription, 0, len(d.subscriptions))
	for _, sub := range d.subscriptions {
		out = append(out, Subscription{
			BridgeAddress:  sub.address,
			EventSignature: sub.eventSignature,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if c := bytes.Compare(out[i].BridgeAddress[:], out[j].BridgeAddress[:]); c != 0 {
			return c < 0
		}
		return bytes.Compare(out[i].EventSignature[:], out[j].EventSignature[:]) < 0
	})
	return out
}

func makeLookupKey(address common.Address, topic common.Hash) lookupKey {
	var k lookupKey
	copy(k[:common.AddressLength], address[:])
	copy(k[common.AddressLength:], topic[:])
	return k
}
