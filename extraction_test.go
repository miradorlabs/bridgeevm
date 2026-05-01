package bridgeevm

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/across.json
var acrossTestDataJSON []byte

//go:embed testdata/stargate.json
var stargateTestDataJSON []byte

//go:embed testdata/cctp.json
var cctpTestDataJSON []byte

//go:embed testdata/cctp_v2.json
var cctpV2TestDataJSON []byte

//go:embed testdata/usdt0.json
var usdt0TestDataJSON []byte

//go:embed testdata/1inch.json
var oneInchTestDataJSON []byte

type bridgeTestCase struct {
	Description           string        `json:"description"`
	Bridge                string        `json:"bridge"`
	Source                bridgeTestLeg `json:"source"`
	Destination           bridgeTestLeg `json:"destination"`
	ExpectedCorrelationID string        `json:"expectedCorrelationId"`
}

type bridgeTestLeg struct {
	Chain      string   `json:"chain"`
	TxHash     string   `json:"txHash"`
	LogAddress string   `json:"logAddress"`
	LogTopic   string   `json:"logTopic"`
	LogTopics  []string `json:"logTopics"`
	LogData    string   `json:"logData"`
}

// TestCorrelation_AllProtocols loads real source/destination tx pairs from
// embedded test data and verifies that Detector.Detect produces the same
// correlation ID on both legs and matches the expected value.
func TestCorrelation_AllProtocols(t *testing.T) {
	suites := []struct {
		name string
		data []byte
	}{
		{"Across", acrossTestDataJSON},
		{"Stargate", stargateTestDataJSON},
		{"CCTP", cctpTestDataJSON},
		{"CCTP-V2", cctpV2TestDataJSON},
		{"USDT0", usdt0TestDataJSON},
		{"1inch", oneInchTestDataJSON},
	}

	for _, suite := range suites {
		t.Run(suite.name, func(t *testing.T) {
			var cases []bridgeTestCase
			require.NoError(t, json.Unmarshal(suite.data, &cases))
			require.NotEmpty(t, cases)

			for _, tc := range cases {
				t.Run(tc.Description, func(t *testing.T) {
					sourceResult := detectOrFatal(t, tc.Source)
					destResult := detectOrFatal(t, tc.Destination)

					assert.Equal(t, tc.Bridge, sourceResult.BridgeName)
					assert.Equal(t, LegTypeSource, sourceResult.LegType)
					assert.Equal(t, tc.Bridge, destResult.BridgeName)
					assert.Equal(t, LegTypeDestination, destResult.LegType)

					assert.Equal(t, tc.ExpectedCorrelationID, sourceResult.CorrelationID,
						"source correlation ID mismatch")
					assert.Equal(t, tc.ExpectedCorrelationID, destResult.CorrelationID,
						"destination correlation ID mismatch")
					assert.Equal(t, sourceResult.CorrelationID, destResult.CorrelationID,
						"source and destination correlation IDs must match")
				})
			}
		})
	}
}

func detectOrFatal(t *testing.T, leg bridgeTestLeg) Result {
	t.Helper()

	d, err := New(leg.Chain)
	require.NoError(t, err)

	result, ok, err := d.Detect(buildLogFromLeg(leg))
	require.NoError(t, err)
	require.True(t, ok, "expected log on chain %s to be detected as a bridge", leg.Chain)
	return result
}

func buildLogFromLeg(leg bridgeTestLeg) *types.Log {
	var topics []common.Hash
	switch {
	case len(leg.LogTopics) > 0:
		topics = make([]common.Hash, len(leg.LogTopics))
		for i, t := range leg.LogTopics {
			topics[i] = common.HexToHash(t)
		}
	case leg.LogTopic != "":
		topics = []common.Hash{common.HexToHash(leg.LogTopic)}
	}

	return &types.Log{
		Address: common.HexToAddress(leg.LogAddress),
		Topics:  topics,
		Data:    common.FromHex(leg.LogData),
	}
}
