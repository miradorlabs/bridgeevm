# bridgeevm

[![CI](https://github.com/miradorlabs/bridgeevm/actions/workflows/ci.yml/badge.svg)](https://github.com/miradorlabs/bridgeevm/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/miradorlabs/bridgeevm.svg)](https://pkg.go.dev/github.com/miradorlabs/bridgeevm)
[![Go Report Card](https://goreportcard.com/badge/github.com/miradorlabs/bridgeevm)](https://goreportcard.com/report/github.com/miradorlabs/bridgeevm)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Detect cross-chain bridge events from EVM transaction logs.

> **Status:** pre-1.0. The public API may change in any `0.x.0` release; patch
> releases (`0.x.y`) will not break callers. See [CHANGELOG.md](CHANGELOG.md).

Pass any `*types.Log` to `Detector.Detect`; if the log matches a known bridge event, you get back the bridge name, the leg type (source or destination), and a correlation ID that links the leg to its counterpart on the other chain.

```go
import (
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/miradorlabs/bridgeevm"
)

detector, err := bridgeevm.New(bridgeevm.ChainEthereum)
if err != nil {
    log.Fatal(err)
}

result, ok, err := detector.Detect(log)
if err != nil {
    // log matched a bridge but the data was malformed
}
if !ok {
    // not a bridge log
    return
}

fmt.Printf("%s leg of %s (correlation %s)\n",
    result.LegType, result.BridgeName, result.CorrelationID)
```

## Coverage

| Bridge       | ethereum | polygon | arbitrum | base | optimism | bsc |
|--------------|:-:|:-:|:-:|:-:|:-:|:-:|
| Across V2 src | ✓ | ✓ | ✓ | ✓ | ✓ | – |
| Across V2 dst | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Across V3 src | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Across V3 dst | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Stargate V1 src | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Stargate V2 src | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Stargate V2 dst | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| CCTP src/dst    | ✓ | ✓ | ✓ | ✓ | ✓ | – |
| CCTP V2 src/dst | ✓ | ✓ | ✓ | ✓ | ✓ | – |
| deBridge        | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| USDT0           | ✓ | ✓ | ✓ | ✓ | – | – |
| 1inch Fusion+   | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |

## API

```go
type Detector struct{ /* ... */ }

func New(chainName string) (*Detector, error)
func (d *Detector) Detect(log *types.Log) (Result, bool, error)
func (d *Detector) ChainName() string
func (d *Detector) Len() int

type Result struct {
    BridgeName        string
    BridgeDescription string
    LegType           LegType         // "source" or "destination"
    CorrelationID     string
    Contract          common.Address
    EventTopic        common.Hash
    EventName         string
}

type LegType string
const (
    LegTypeSource      LegType = "source"
    LegTypeDestination LegType = "destination"
)
```

`Detect` returns `(_, false, nil)` for logs that don't match any known bridge — this is the common case, not an error. The `error` return is non-nil only when a bridge event matched but its data was malformed.

A `Detector` is read-only and safe to share across goroutines.

## How it works

Bridge configurations are embedded JSON, one file per protocol per chain. Each config declares the contract address, event topic hash, and how to extract the correlation ID from `topics`, `data`, packed bytes, ABI-encoded dynamic bytes, or a keccak hash thereof.

`New` builds an `O(1)` `(address, topic[0]) → subscription` map; `Detect` is a single map lookup followed by correlation extraction. No network calls and no ABI decoding beyond the configured field. The lookup itself is allocation-free (verified by `BenchmarkDetect_Miss` and `BenchmarkLookupKey`); the hit path allocates only the resulting correlation ID string.

## Receipt-level helpers

This module deliberately operates on a single log at a time. If you need per-receipt counts or cross-log dedup (e.g. Across V3 SpokePool emits `FundsDeposited` twice in the same tx with the same `depositId`), wrap `Detect` in a small loop in your application.

## License

MIT. See [LICENSE](LICENSE).
