package bridgeevm

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_Ethereum(t *testing.T) {
	d, err := New("ethereum")
	require.NoError(t, err)
	assert.Equal(t, "ethereum", d.ChainName())
	assert.Greater(t, d.Len(), 0)
}

func TestNew_CaseInsensitive(t *testing.T) {
	d, err := New("Ethereum")
	require.NoError(t, err)
	assert.Equal(t, "ethereum", d.ChainName())
	assert.Greater(t, d.Len(), 0)
}

func TestNew_UnknownChain(t *testing.T) {
	d, err := New("unknownchain")
	require.NoError(t, err)
	assert.Equal(t, 0, d.Len())
}

func TestIdentify_NilLog(t *testing.T) {
	d, err := New("ethereum")
	require.NoError(t, err)

	result, ok, err := d.Identify(nil)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, Result{}, result)
}

func TestIdentify_LogWithNoTopics(t *testing.T) {
	d, err := New("ethereum")
	require.NoError(t, err)

	log := &types.Log{
		Address: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Topics:  []common.Hash{},
	}
	result, ok, err := d.Identify(log)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, Result{}, result)
}

func TestIdentify_NoMatch(t *testing.T) {
	d, err := New("ethereum")
	require.NoError(t, err)

	log := &types.Log{
		Address: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Topics:  []common.Hash{common.HexToHash("0xabcd")},
	}
	result, ok, err := d.Identify(log)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, Result{}, result)
}

func TestIdentify_AcrossV3Source(t *testing.T) {
	d, err := New("arbitrum")
	require.NoError(t, err)

	acrossAddr := common.HexToAddress("0xe35e9842fceaCA96570B734083f4a58e8F7C5f2A")
	fundsDepositedV3Topic := common.HexToHash("0x32ed1a409ef04c7b0227189c3a103dc5ac10e775a15b785dcc510201f7c25ad3")

	log := &types.Log{
		Address: acrossAddr,
		Topics: []common.Hash{
			fundsDepositedV3Topic,
			common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000002105"),
			common.HexToHash("0x6f1b280c20fb309b3653566de6f876cd100c6d26bc9722649865147ce22480e7"),
			common.HexToHash("0x0000000000000000000000006892ca799bbb10cb13e3cc2a7b587365ba2f3597"),
		},
	}

	result, ok, err := d.Identify(log)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "across", result.BridgeName)
	assert.Equal(t, LegTypeSource, result.LegType)
	assert.Equal(t, acrossAddr, result.Contract)
	assert.Equal(t, fundsDepositedV3Topic, result.EventTopic)
	assert.NotEmpty(t, result.CorrelationID)
}

func TestIdentify_CCTPSourceFromRealTx(t *testing.T) {
	d, err := New("ethereum")
	require.NoError(t, err)

	cctpAddr := common.HexToAddress("0x0a992d191deec32afe36203ad87d7d289a738f81")
	msgSentTopic := common.HexToHash("0x8c5261668696ce22758910d05bab8f186d6eb247ceac2af2e82c7dc17669b036")

	// Real MessageSent from Ethereum tx 0xa64f215dc8ff01a07b59381c887dcd4485eac7db67820d115e7fb6fa674e4c25
	// Encodes nonce 0x67c4e (425550) destined for chain 6 (Base).
	data := common.FromHex("0x000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000f80000000000000000000000060000000000067c4e000000000000000000000000bd3fa81b58ba92a82136038b25adec7066af31550000000000000000000000001682ae6375c4e4a97e4b583bc394c861a46d8962000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb480000000000000000000000002917956eff0b5eaf030abdb4ef4296df775009ca0000000000000000000000000000000000000000000000000000000f8b0229bf0000000000000000000000001601843c5e9bc251a3272907010afa41fa18347e0000000000000000")

	log := &types.Log{
		Address: cctpAddr,
		Topics:  []common.Hash{msgSentTopic},
		Data:    data,
	}

	result, ok, err := d.Identify(log)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "cctp", result.BridgeName)
	assert.Equal(t, LegTypeSource, result.LegType)
	assert.NotEmpty(t, result.CorrelationID)
}

func TestIdentify_USDT0SourceFromRealTx(t *testing.T) {
	d, err := New("ethereum")
	require.NoError(t, err)

	usdt0Addr := common.HexToAddress("0x6C96dE32CEa08842dcc4058c14d3aaAD7Fa41dee")
	oftSentTopic := common.HexToHash("0x85496b760a4b7f8d66384b9df21b381f5d1b1e79f229a47aaf4c232edc2fe59a")
	expectedGUID := "0x49d3cc164481b1d3f2f2dc596813827c2388f2ef6d4202128701635383d55116"

	log := &types.Log{
		Address: usdt0Addr,
		Topics: []common.Hash{
			oftSentTopic,
			common.HexToHash(expectedGUID),
			common.HexToHash("0x000000000000000000000000c186fa914353c44b2e33ebe05f21846f1048beda"),
		},
		Data: common.FromHex("0x000000000000000000000000000000000000000000000000000000000000759e0000000000000000000000000000000000000000000000000000000005c7a8900000000000000000000000000000000000000000000000000000000005c7a890"),
	}

	result, ok, err := d.Identify(log)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "usdt0", result.BridgeName)
	assert.Equal(t, LegTypeSource, result.LegType)
	assert.Equal(t, expectedGUID, result.CorrelationID)
}

// TestIdentify_MatchedButMalformed exercises the documented
// (Result{}, true, err) branch on Identify: a log whose (address, topic[0])
// matches a configured bridge but whose Data is too short for correlation
// extraction to succeed. CCTP source uses an abi_bytes payload, so a log
// with empty Data forces readAbiBytesParam to error.
func TestIdentify_MatchedButMalformed(t *testing.T) {
	d, err := New("ethereum")
	require.NoError(t, err)

	cctpAddr := common.HexToAddress("0x0a992d191deec32afe36203ad87d7d289a738f81")
	msgSentTopic := common.HexToHash("0x8c5261668696ce22758910d05bab8f186d6eb247ceac2af2e82c7dc17669b036")

	log := &types.Log{
		Address: cctpAddr,
		Topics:  []common.Hash{msgSentTopic},
		Data:    nil, // matched bridge, but no data — extraction will fail
	}

	result, ok, err := d.Identify(log)
	require.Error(t, err)
	assert.True(t, ok, "ok must be true: a configured bridge matched")
	assert.Equal(t, Result{}, result, "Result must be zero on extraction failure")
	assert.Contains(t, err.Error(), "bridge cctp:")
}

func TestIdentify_SubscriptionsCoverBothLegTypes(t *testing.T) {
	d, err := New("ethereum")
	require.NoError(t, err)

	hasSource, hasDestination := false, false
	for _, sub := range d.subscriptions {
		switch sub.legType {
		case LegTypeSource:
			hasSource = true
		case LegTypeDestination:
			hasDestination = true
		}
	}
	assert.True(t, hasSource, "expected at least one source subscription for ethereum")
	assert.True(t, hasDestination, "expected at least one destination subscription for ethereum")
}
