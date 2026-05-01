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

func TestDetect_NilLog(t *testing.T) {
	d, err := New("ethereum")
	require.NoError(t, err)

	result, ok, err := d.Detect(nil)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, Result{}, result)
}

func TestDetect_LogWithNoTopics(t *testing.T) {
	d, err := New("ethereum")
	require.NoError(t, err)

	log := &types.Log{
		Address: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Topics:  []common.Hash{},
	}
	result, ok, err := d.Detect(log)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, Result{}, result)
}

func TestDetect_NoMatch(t *testing.T) {
	d, err := New("ethereum")
	require.NoError(t, err)

	log := &types.Log{
		Address: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Topics:  []common.Hash{common.HexToHash("0xabcd")},
	}
	result, ok, err := d.Detect(log)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, Result{}, result)
}

func TestDetect_AcrossV3Source(t *testing.T) {
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

	result, ok, err := d.Detect(log)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "across", result.BridgeName)
	assert.Equal(t, LegTypeSource, result.LegType)
	assert.Equal(t, acrossAddr, result.Contract)
	assert.Equal(t, fundsDepositedV3Topic, result.EventTopic)
	assert.NotEmpty(t, result.CorrelationID)
}

func TestDetect_CCTPSourceFromRealTx(t *testing.T) {
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

	result, ok, err := d.Detect(log)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "cctp", result.BridgeName)
	assert.Equal(t, LegTypeSource, result.LegType)
	assert.NotEmpty(t, result.CorrelationID)
}

func TestDetect_USDT0SourceFromRealTx(t *testing.T) {
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

	result, ok, err := d.Detect(log)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "usdt0", result.BridgeName)
	assert.Equal(t, LegTypeSource, result.LegType)
	assert.Equal(t, expectedGUID, result.CorrelationID)
}

// TestDetect_MatchedButMalformed exercises the documented
// (Result{}, true, err) branch on Detect: a log whose (address, topic[0])
// matches a configured bridge but whose Data is too short for correlation
// extraction to succeed. CCTP source uses an abi_bytes payload, so a log
// with empty Data forces readAbiBytesParam to error.
func TestDetect_MatchedButMalformed(t *testing.T) {
	d, err := New("ethereum")
	require.NoError(t, err)

	cctpAddr := common.HexToAddress("0x0a992d191deec32afe36203ad87d7d289a738f81")
	msgSentTopic := common.HexToHash("0x8c5261668696ce22758910d05bab8f186d6eb247ceac2af2e82c7dc17669b036")

	log := &types.Log{
		Address: cctpAddr,
		Topics:  []common.Hash{msgSentTopic},
		Data:    nil, // matched bridge, but no data — extraction will fail
	}

	result, ok, err := d.Detect(log)
	require.Error(t, err)
	assert.True(t, ok, "ok must be true: a configured bridge matched")
	assert.Equal(t, Result{}, result, "Result must be zero on extraction failure")
	assert.Contains(t, err.Error(), "bridge cctp:")
}

func TestDetect_SubscriptionsCoverBothLegTypes(t *testing.T) {
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

func TestSubscriptions_Ethereum(t *testing.T) {
	d, err := New("ethereum")
	require.NoError(t, err)

	subs := d.Subscriptions()
	assert.Equal(t, d.Len(), len(subs))
	require.NotEmpty(t, subs)

	var zeroAddr common.Address
	var zeroTopic common.Hash
	for _, s := range subs {
		assert.NotEqual(t, zeroAddr, s.BridgeAddress, "subscription has zero address")
		assert.NotEqual(t, zeroTopic, s.EventSignature, "subscription has zero topic")
	}
}

func TestSubscriptions_UnknownChain(t *testing.T) {
	d, err := New("unknownchain")
	require.NoError(t, err)

	subs := d.Subscriptions()
	assert.Equal(t, 0, len(subs))
}

func TestSubscriptions_Deterministic(t *testing.T) {
	d, err := New("ethereum")
	require.NoError(t, err)

	first := d.Subscriptions()
	second := d.Subscriptions()
	assert.Equal(t, first, second, "Subscriptions must return identical slices across calls")
}

func TestSubscriptions_RoundTripsThroughDetect(t *testing.T) {
	for _, chain := range []string{ChainArbitrum, ChainBase, ChainBSC, ChainEthereum, ChainOptimism, ChainPolygon} {
		t.Run(chain, func(t *testing.T) {
			d, err := New(chain)
			require.NoError(t, err)

			subs := d.Subscriptions()
			require.NotEmpty(t, subs, "expected at least one subscription for %s", chain)

			for _, s := range subs {
				log := &types.Log{
					Address: s.BridgeAddress,
					Topics:  []common.Hash{s.EventSignature},
				}
				_, ok, _ := d.Detect(log)
				assert.True(t, ok, "Detect must match every Subscription pair (chain=%s addr=%s topic=%s)", chain, s.BridgeAddress.Hex(), s.EventSignature.Hex())
			}
		})
	}
}

func TestAddresses_Ethereum(t *testing.T) {
	d, err := New("ethereum")
	require.NoError(t, err)

	addrs := d.Addresses()
	require.NotEmpty(t, addrs)

	var zero common.Address
	seen := make(map[common.Address]struct{}, len(addrs))
	for _, a := range addrs {
		assert.NotEqual(t, zero, a, "address must not be zero value")
		_, dup := seen[a]
		assert.False(t, dup, "address %s appears twice", a.Hex())
		seen[a] = struct{}{}
	}
}

func TestTopics_Ethereum(t *testing.T) {
	d, err := New("ethereum")
	require.NoError(t, err)

	topics := d.Topics()
	require.NotEmpty(t, topics)

	var zero common.Hash
	seen := make(map[common.Hash]struct{}, len(topics))
	for _, h := range topics {
		assert.NotEqual(t, zero, h, "topic must not be zero value")
		_, dup := seen[h]
		assert.False(t, dup, "topic %s appears twice", h.Hex())
		seen[h] = struct{}{}
	}
}

func TestAddresses_UnknownChain(t *testing.T) {
	d, err := New("unknownchain")
	require.NoError(t, err)
	assert.Equal(t, 0, len(d.Addresses()))
}

func TestTopics_UnknownChain(t *testing.T) {
	d, err := New("unknownchain")
	require.NoError(t, err)
	assert.Equal(t, 0, len(d.Topics()))
}

func TestAddresses_Deterministic(t *testing.T) {
	d, err := New("ethereum")
	require.NoError(t, err)
	assert.Equal(t, d.Addresses(), d.Addresses())
}

func TestTopics_Deterministic(t *testing.T) {
	d, err := New("ethereum")
	require.NoError(t, err)
	assert.Equal(t, d.Topics(), d.Topics())
}

func TestAddressesAndTopics_MatchSubscriptions(t *testing.T) {
	for _, chain := range []string{ChainArbitrum, ChainBase, ChainBSC, ChainEthereum, ChainOptimism, ChainPolygon} {
		t.Run(chain, func(t *testing.T) {
			d, err := New(chain)
			require.NoError(t, err)

			wantAddrs := make(map[common.Address]struct{})
			wantTopics := make(map[common.Hash]struct{})
			for _, s := range d.Subscriptions() {
				wantAddrs[s.BridgeAddress] = struct{}{}
				wantTopics[s.EventSignature] = struct{}{}
			}

			gotAddrs := d.Addresses()
			gotTopics := d.Topics()

			assert.Equal(t, len(wantAddrs), len(gotAddrs))
			for _, a := range gotAddrs {
				_, ok := wantAddrs[a]
				assert.True(t, ok, "Addresses returned %s not present in Subscriptions", a.Hex())
			}

			assert.Equal(t, len(wantTopics), len(gotTopics))
			for _, h := range gotTopics {
				_, ok := wantTopics[h]
				assert.True(t, ok, "Topics returned %s not present in Subscriptions", h.Hex())
			}
		})
	}
}
