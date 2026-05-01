# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

While the major version is `0`, minor releases (`0.x.0`) may include
breaking API changes; patch releases (`0.x.y`) will not.

## [Unreleased]

### Added
- `Detector.Len()` returns the number of configured `(address, topic)`
  subscriptions for the chain.
- Runnable godoc examples for `New` and `Detector.Detect`.
- Benchmarks (`BenchmarkDetect_Hit`, `BenchmarkDetect_Miss`,
  `BenchmarkLookupKey`) — `Miss` and the lookup key are 0 allocs/op.
- `doc.go` carrying the package-level godoc.
- Load-time validation rejects bridge configs with an invalid
  `bridgeTopic.type` (must be `source` or `destination`), an unsupported
  correlation type, or a duplicate `(address, topic)` subscription on the
  same chain. These previously failed at first matching log, or silently
  overwrote each other.
- Targeted unit tests for the extraction error paths
  (`extraction_unit_test.go`) and load-time validation rejection paths
  (`config_test.go`, driven via `testing/fstest.MapFS`). Library coverage
  is 94.5% of statements.
- `make tools` target installs a pinned `golangci-lint` into `./bin/`
  for contributors who don't want a system-wide install.

### Changed
- Module path renamed from `github.com/miradorlabs/bridge-detect-evm`
  to `github.com/miradorlabs/bridgeevm` so that the last path component
  matches the package name. Consumers must update their imports.
- Map lookup key in `Detector.Detect` is now a fixed-size byte array
  instead of a concatenated string, eliminating per-call allocations on
  the hot path.
- `correlationField.Source` and `correlationField.Type` are normalized
  (lowercased, trimmed) once at config-load time rather than on every log.
- `readAbiBytesParam` now caps offset/length at `math.MaxInt32` *before*
  narrowing the big.Int to `uint64`, so values above MaxUint64 cannot
  silently truncate past the cap. The bounds check moved into a small
  `readBoundedWord` helper and is exercised by dedicated unit tests.
- `go.mod` minimum Go version dropped to `1.24.0` (matches the actual
  dependency floor) with `toolchain go1.26.1` as the development pin.
- Relicensed from Apache 2.0 to MIT to align with the rest of the
  Mirador open-source SDK family. The `NOTICE` file is no longer needed
  under MIT and has been removed.

### Removed
- `config/anvil/` is no longer embedded into the library binary.
  The fixture moved to `testdata/configs/anvil/` for repo-internal use.
- Removed duplicate Stargate V2 pool entries from `arbitrum`, `base`,
  `bsc`, and `optimism` configs. Each chain previously declared two pools
  (e.g. USDC and ETH) at the same contract address — a copy-paste error
  caught by the new duplicate-subscription validation. Only the first,
  verified pool per chain is kept; correct addresses for the other pools
  should be added back as a follow-up.

### Fixed
- CCTP V2 source configs across all chains had `"type": "keccak256"`,
  which was never a recognized type — `extractFromAbiBytesHash` ignores
  the type field entirely and always returns a 32-byte hash. The configs
  now declare `"type": "bytes32"` to match the actual output shape, and
  the new load-time type validation ensures this can't drift again.
- README claim that the hot path is allocation-free is now accurate
  (verified by `BenchmarkDetect_Miss` and `BenchmarkLookupKey`).
