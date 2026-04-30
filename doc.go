// Package bridgeevm identifies cross-chain bridge events from EVM transaction logs.
//
// A Detector is built once per chain. Pass any *types.Log to Identify; if the log
// matches a known bridge event, Identify returns a Result describing the bridge,
// the leg type (source or destination), and the extracted correlation ID that
// links the leg to its counterpart on the other chain.
//
// Bridge configurations are embedded JSON, covering Across, Stargate, CCTP,
// deBridge, USDT0, and 1inch across Ethereum, Polygon, Arbitrum, Base, Optimism,
// and BSC. See the README for the full coverage matrix.
package bridgeevm
