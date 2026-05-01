package bridgeevm

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validJSON is a known-good single-bridge config used as the baseline that
// each subtest mutates to exercise one rejection path.
const validJSON = `[{
  "chainName": "ethereum",
  "bridgeName": "test",
  "bridgeDescription": "valid baseline",
  "bridgeContract": {"address": "0x1234567890123456789012345678901234567890"},
  "bridgeTopic": {
    "type": "source",
    "hash": "0x1111111111111111111111111111111111111111111111111111111111111111",
    "name": "Test",
    "correlation": [{"index": 1, "type": "uint256", "field": "id", "source": "topics"}]
  }
}]`

func TestLoadBridgeConfigs_Valid(t *testing.T) {
	fsys := fstest.MapFS{
		"config/ethereum/test.json": {Data: []byte(validJSON)},
	}
	out, err := loadBridgeConfigs(fsys)
	require.NoError(t, err)
	require.Contains(t, out, "ethereum")
	assert.Len(t, out["ethereum"], 1)
}

func TestLoadBridgeConfigs_Rejections(t *testing.T) {
	cases := []struct {
		name      string
		path      string
		body      string
		errSubstr string
	}{
		{
			name:      "unparseable JSON",
			path:      "config/ethereum/bad.json",
			body:      `{not json`,
			errSubstr: "decode",
		},
		{
			name:      "nil entry in array",
			path:      "config/ethereum/bad.json",
			body:      `[null]`,
			errSubstr: "nil bridge config",
		},
		{
			name:      "missing chainName",
			path:      "config/ethereum/bad.json",
			body:      `[{"bridgeName":"x","bridgeDescription":"d","bridgeContract":{"address":"0x1234567890123456789012345678901234567890"},"bridgeTopic":{"type":"source","hash":"0x1111111111111111111111111111111111111111111111111111111111111111","correlation":[{"index":0,"type":"uint256","source":"topics"}]}}]`,
			errSubstr: "missing chainName",
		},
		{
			name:      "missing bridgeName",
			path:      "config/ethereum/bad.json",
			body:      `[{"chainName":"ethereum","bridgeDescription":"d","bridgeContract":{"address":"0x1234567890123456789012345678901234567890"},"bridgeTopic":{"type":"source","hash":"0x1111111111111111111111111111111111111111111111111111111111111111","correlation":[{"index":0,"type":"uint256","source":"topics"}]}}]`,
			errSubstr: "missing bridgeName",
		},
		{
			name:      "missing bridgeDescription",
			path:      "config/ethereum/bad.json",
			body:      `[{"chainName":"ethereum","bridgeName":"x","bridgeContract":{"address":"0x1234567890123456789012345678901234567890"},"bridgeTopic":{"type":"source","hash":"0x1111111111111111111111111111111111111111111111111111111111111111","correlation":[{"index":0,"type":"uint256","source":"topics"}]}}]`,
			errSubstr: "missing bridgeDescription",
		},
		{
			name:      "invalid contract address",
			path:      "config/ethereum/bad.json",
			body:      `[{"chainName":"ethereum","bridgeName":"x","bridgeDescription":"d","bridgeContract":{"address":"not-an-address"},"bridgeTopic":{"type":"source","hash":"0x1111111111111111111111111111111111111111111111111111111111111111","correlation":[{"index":0,"type":"uint256","source":"topics"}]}}]`,
			errSubstr: "invalid contract address",
		},
		{
			name:      "topic hash missing 0x prefix",
			path:      "config/ethereum/bad.json",
			body:      `[{"chainName":"ethereum","bridgeName":"x","bridgeDescription":"d","bridgeContract":{"address":"0x1234567890123456789012345678901234567890"},"bridgeTopic":{"type":"source","hash":"1111111111111111111111111111111111111111111111111111111111111111","correlation":[{"index":0,"type":"uint256","source":"topics"}]}}]`,
			errSubstr: "missing 0x prefix",
		},
		{
			name:      "topic hash wrong length",
			path:      "config/ethereum/bad.json",
			body:      `[{"chainName":"ethereum","bridgeName":"x","bridgeDescription":"d","bridgeContract":{"address":"0x1234567890123456789012345678901234567890"},"bridgeTopic":{"type":"source","hash":"0x1234","correlation":[{"index":0,"type":"uint256","source":"topics"}]}}]`,
			errSubstr: "is 2 bytes; expected 32",
		},
		{
			name:      "topic hash non-hex",
			path:      "config/ethereum/bad.json",
			body:      `[{"chainName":"ethereum","bridgeName":"x","bridgeDescription":"d","bridgeContract":{"address":"0x1234567890123456789012345678901234567890"},"bridgeTopic":{"type":"source","hash":"0xZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ","correlation":[{"index":0,"type":"uint256","source":"topics"}]}}]`,
			errSubstr: "is not hex",
		},
		{
			name:      "chainName mismatches directory",
			path:      "config/ethereum/bad.json",
			body:      `[{"chainName":"polygon","bridgeName":"x","bridgeDescription":"d","bridgeContract":{"address":"0x1234567890123456789012345678901234567890"},"bridgeTopic":{"type":"source","hash":"0x1111111111111111111111111111111111111111111111111111111111111111","correlation":[{"index":0,"type":"uint256","source":"topics"}]}}]`,
			errSubstr: `does not match directory "ethereum"`,
		},
		{
			name:      "missing bridgeTopic.type",
			path:      "config/ethereum/bad.json",
			body:      `[{"chainName":"ethereum","bridgeName":"x","bridgeDescription":"d","bridgeContract":{"address":"0x1234567890123456789012345678901234567890"},"bridgeTopic":{"hash":"0x1111111111111111111111111111111111111111111111111111111111111111","correlation":[{"index":0,"type":"uint256","source":"topics"}]}}]`,
			errSubstr: "missing bridgeTopic.type",
		},
		{
			name:      "invalid bridgeTopic.type",
			path:      "config/ethereum/bad.json",
			body:      `[{"chainName":"ethereum","bridgeName":"x","bridgeDescription":"d","bridgeContract":{"address":"0x1234567890123456789012345678901234567890"},"bridgeTopic":{"type":"sink","hash":"0x1111111111111111111111111111111111111111111111111111111111111111","correlation":[{"index":0,"type":"uint256","source":"topics"}]}}]`,
			errSubstr: "invalid bridgeTopic.type",
		},
		{
			name:      "missing correlation",
			path:      "config/ethereum/bad.json",
			body:      `[{"chainName":"ethereum","bridgeName":"x","bridgeDescription":"d","bridgeContract":{"address":"0x1234567890123456789012345678901234567890"},"bridgeTopic":{"type":"source","hash":"0x1111111111111111111111111111111111111111111111111111111111111111","correlation":[]}}]`,
			errSubstr: "missing correlation fields",
		},
		{
			name:      "negative index on topics source",
			path:      "config/ethereum/bad.json",
			body:      `[{"chainName":"ethereum","bridgeName":"x","bridgeDescription":"d","bridgeContract":{"address":"0x1234567890123456789012345678901234567890"},"bridgeTopic":{"type":"source","hash":"0x1111111111111111111111111111111111111111111111111111111111111111","correlation":[{"index":-1,"type":"uint256","source":"topics"}]}}]`,
			errSubstr: "index must be >= 0",
		},
		{
			name:      "packed source missing size",
			path:      "config/ethereum/bad.json",
			body:      `[{"chainName":"ethereum","bridgeName":"x","bridgeDescription":"d","bridgeContract":{"address":"0x1234567890123456789012345678901234567890"},"bridgeTopic":{"type":"source","hash":"0x1111111111111111111111111111111111111111111111111111111111111111","correlation":[{"offset":0,"size":0,"type":"uint64","source":"packed"}]}}]`,
			errSubstr: "size must be > 0",
		},
		{
			name:      "abi_bytes negative bytesIndex",
			path:      "config/ethereum/bad.json",
			body:      `[{"chainName":"ethereum","bridgeName":"x","bridgeDescription":"d","bridgeContract":{"address":"0x1234567890123456789012345678901234567890"},"bridgeTopic":{"type":"source","hash":"0x1111111111111111111111111111111111111111111111111111111111111111","correlation":[{"bytesIndex":-1,"offset":0,"size":8,"type":"uint64","source":"abi_bytes"}]}}]`,
			errSubstr: "bytesIndex must be >= 0",
		},
		{
			name:      "abi_bytes_hash negative offset",
			path:      "config/ethereum/bad.json",
			body:      `[{"chainName":"ethereum","bridgeName":"x","bridgeDescription":"d","bridgeContract":{"address":"0x1234567890123456789012345678901234567890"},"bridgeTopic":{"type":"source","hash":"0x1111111111111111111111111111111111111111111111111111111111111111","correlation":[{"bytesIndex":0,"offset":-1,"type":"bytes32","source":"abi_bytes_hash"}]}}]`,
			errSubstr: "offset must be >= 0",
		},
		{
			name:      "invalid source",
			path:      "config/ethereum/bad.json",
			body:      `[{"chainName":"ethereum","bridgeName":"x","bridgeDescription":"d","bridgeContract":{"address":"0x1234567890123456789012345678901234567890"},"bridgeTopic":{"type":"source","hash":"0x1111111111111111111111111111111111111111111111111111111111111111","correlation":[{"index":0,"type":"uint256","source":"banana"}]}}]`,
			errSubstr: "invalid source",
		},
		{
			name:      "missing correlation type",
			path:      "config/ethereum/bad.json",
			body:      `[{"chainName":"ethereum","bridgeName":"x","bridgeDescription":"d","bridgeContract":{"address":"0x1234567890123456789012345678901234567890"},"bridgeTopic":{"type":"source","hash":"0x1111111111111111111111111111111111111111111111111111111111111111","correlation":[{"index":0,"source":"topics"}]}}]`,
			errSubstr: "missing type",
		},
		{
			name:      "unsupported correlation type",
			path:      "config/ethereum/bad.json",
			body:      `[{"chainName":"ethereum","bridgeName":"x","bridgeDescription":"d","bridgeContract":{"address":"0x1234567890123456789012345678901234567890"},"bridgeTopic":{"type":"source","hash":"0x1111111111111111111111111111111111111111111111111111111111111111","correlation":[{"index":0,"type":"hyperloop","source":"topics"}]}}]`,
			errSubstr: "unsupported correlation type",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fsys := fstest.MapFS{tc.path: {Data: []byte(tc.body)}}
			_, err := loadBridgeConfigs(fsys)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errSubstr)
		})
	}
}

func TestLoadBridgeConfigs_DuplicateSubscription(t *testing.T) {
	dup := `[
		{"chainName":"ethereum","bridgeName":"a","bridgeDescription":"first","bridgeContract":{"address":"0x1234567890123456789012345678901234567890"},"bridgeTopic":{"type":"source","hash":"0x1111111111111111111111111111111111111111111111111111111111111111","correlation":[{"index":0,"type":"uint256","source":"topics"}]}},
		{"chainName":"ethereum","bridgeName":"b","bridgeDescription":"colliding","bridgeContract":{"address":"0x1234567890123456789012345678901234567890"},"bridgeTopic":{"type":"source","hash":"0x1111111111111111111111111111111111111111111111111111111111111111","correlation":[{"index":0,"type":"uint256","source":"topics"}]}}
	]`
	fsys := fstest.MapFS{
		"config/ethereum/dup.json": {Data: []byte(dup)},
	}
	_, err := loadBridgeConfigs(fsys)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate subscription on ethereum")
	assert.Contains(t, err.Error(), "config/ethereum/dup.json[0]")
	assert.Contains(t, err.Error(), "config/ethereum/dup.json[1]")
}

func TestLoadBridgeConfigs_NoConfigDir(t *testing.T) {
	// fstest.MapFS without a "config" entry — fs.ReadDir will error.
	fsys := fstest.MapFS{"unrelated.txt": {Data: []byte("hello")}}
	_, err := loadBridgeConfigs(fsys)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read embedded bridge config root")
}

func TestLoadBridgeConfigs_IgnoresNonJSONAndSubdirs(t *testing.T) {
	fsys := fstest.MapFS{
		"config/ethereum/test.json":         {Data: []byte(validJSON)},
		"config/ethereum/README.md":         {Data: []byte("# ignored")},
		"config/ethereum/subdir/inner.json": {Data: []byte(`[]`)}, // skipped because parent is a dir
		"config/.hidden":                    {Data: []byte("")},   // skipped: not a directory
	}
	out, err := loadBridgeConfigs(fsys)
	require.NoError(t, err)
	require.Len(t, out["ethereum"], 1)
}
