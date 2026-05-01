---
name: Bug report
about: Report a defect in detection or correlation extraction
title: "fix: "
labels: bug
---

**Describe the bug**

A clear description of what is wrong.

**To reproduce**

Minimal Go snippet, ideally with the offending log:

```go
log := &types.Log{
    Address: common.HexToAddress("0x..."),
    Topics:  []common.Hash{common.HexToHash("0x...")},
    Data:    common.FromHex("0x..."),
}

result, ok, err := bridgeevm.New("ethereum").Detect(log)
```

**Expected behaviour**

What you expected `Detect` to return.

**Actual behaviour**

What it actually returned (Result fields, ok, err).

**On-chain reference**

If applicable: tx hash, block number, chain.

**Environment**

- bridge-detect-evm version (commit SHA or tag):
- Go version:
- OS / arch:
