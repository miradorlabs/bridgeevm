# Changelog

## [0.5.0](https://github.com/miradorlabs/bridgeevm/compare/v0.4.0...v0.5.0) (2026-05-26)


### Features

* **config:** add Hyperliquid HyperEVM with CCTP V2, Across, deBridge ([eaf18d8](https://github.com/miradorlabs/bridgeevm/commit/eaf18d80ff985eaa7e12b049bdd4eba728f3d9f7))
* **config:** add Hyperliquid HyperEVM with CCTP V2, Across, deBridge, USDT0 ([c0cc487](https://github.com/miradorlabs/bridgeevm/commit/c0cc487fb80a0f19c10ec062aa7fe7280a5293ad))
* **config:** add WHYPE LayerZero OFT for hyperevm, ethereum, optimism ([6ddc57e](https://github.com/miradorlabs/bridgeevm/commit/6ddc57eedae496224f928f53e9b1a0d1d7d7d823))
* **config:** add XAUt0 LayerZero OFT across 5 chains ([59d6c4f](https://github.com/miradorlabs/bridgeevm/commit/59d6c4fde58463e8367339d7a5b3db4304bc5c82))


### Bug Fixes

* **config:** update deBridge FulfilledOrder topic to current contract event ([a746e69](https://github.com/miradorlabs/bridgeevm/commit/a746e69e2d07c5bb6e1c5e26e72f236d985d1422))


### Dependencies

* **deps:** Bump github.com/ethereum/go-ethereum from 1.17.2 to 1.17.3 ([#22](https://github.com/miradorlabs/bridgeevm/issues/22)) ([1979034](https://github.com/miradorlabs/bridgeevm/commit/1979034d5d32168d1af22eddaf383b5a7972d5c8))

## [0.4.0](https://github.com/miradorlabs/bridgeevm/compare/v0.3.0...v0.4.0) (2026-05-14)


### ⚠ BREAKING CHANGES

* remove Relay bridge detection ([#20](https://github.com/miradorlabs/bridgeevm/issues/20))

### Features

* remove Relay bridge detection ([#20](https://github.com/miradorlabs/bridgeevm/issues/20)) ([bfba6c3](https://github.com/miradorlabs/bridgeevm/commit/bfba6c3496142da49764fedb84e51abec33ebe08))

## [0.3.0](https://github.com/miradorlabs/bridgeevm/compare/v0.2.1...v0.3.0) (2026-05-13)


### Features

* add Relay bridge detection across all EVM chains ([#16](https://github.com/miradorlabs/bridgeevm/issues/16)) ([66528b1](https://github.com/miradorlabs/bridgeevm/commit/66528b1e364c5f1ab22348cec4dafccd2939cb5a))


### Documentation

* bump CONTRIBUTING prerequisite to Go 1.26 ([#17](https://github.com/miradorlabs/bridgeevm/issues/17)) ([29441d8](https://github.com/miradorlabs/bridgeevm/commit/29441d88e492971d3ae4dfe88148cbec4f4e68ef))

## [0.2.1](https://github.com/miradorlabs/bridgeevm/compare/v0.2.0...v0.2.1) (2026-05-01)


### Code Refactoring

* add Addresses and Topics accessors for FilterQuery consumers ([#14](https://github.com/miradorlabs/bridgeevm/issues/14)) ([4e13bf2](https://github.com/miradorlabs/bridgeevm/commit/4e13bf2f7caff637bb6d9785987126effccb8dba))

## [0.2.0](https://github.com/miradorlabs/bridgeevm/compare/v0.1.0...v0.2.0) (2026-05-01)


### Features

* expose Subscriptions accessor for filter-query consumers ([f15c1ef](https://github.com/miradorlabs/bridgeevm/commit/f15c1efcd7a22b8c584f885991833bc0b5227c64))
* expose Subscriptions accessor for filter-query consumers ([59cd77d](https://github.com/miradorlabs/bridgeevm/commit/59cd77d4ac7f6218fa83d53916cc5af2c436bb57))

## 0.1.0 (2026-05-01)


### Features

* bootstrap release-please at v0.1.0 ([#9](https://github.com/miradorlabs/bridgeevm/issues/9)) ([a10c8a1](https://github.com/miradorlabs/bridgeevm/commit/a10c8a1bcb92607e85623f7166732d17597667a1))

## Changelog

All notable changes to this project are managed by
[release-please](https://github.com/googleapis/release-please) and generated
from conventional-commit messages on `main`.

While the major version is `0`, minor releases (`0.x.0`) may include
breaking API changes; patch releases (`0.x.y`) will not.
