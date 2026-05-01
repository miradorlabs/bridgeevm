package bridgeevm

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// BenchmarkDetect_Hit measures the hot path on a log that matches a known
// bridge. The lookup itself is allocation-free (fixed-size array key);
// allocations come from correlation ID extraction (string assembly).
func BenchmarkDetect_Hit(b *testing.B) {
	d, err := New("arbitrum")
	if err != nil {
		b.Fatal(err)
	}
	log := &types.Log{
		Address: common.HexToAddress("0xe35e9842fceaCA96570B734083f4a58e8F7C5f2A"),
		Topics: []common.Hash{
			common.HexToHash("0x32ed1a409ef04c7b0227189c3a103dc5ac10e775a15b785dcc510201f7c25ad3"),
			common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000002105"),
			common.HexToHash("0x6f1b280c20fb309b3653566de6f876cd100c6d26bc9722649865147ce22480e7"),
			common.HexToHash("0x0000000000000000000000006892ca799bbb10cb13e3cc2a7b587365ba2f3597"),
		},
	}

	b.ReportAllocs()
	for b.Loop() {
		_, _, _ = d.Detect(log)
	}
}

// BenchmarkDetect_Miss measures the common case: an arbitrary log that
// does not match any known bridge. This path should be zero-alloc.
func BenchmarkDetect_Miss(b *testing.B) {
	d, err := New("ethereum")
	if err != nil {
		b.Fatal(err)
	}
	log := &types.Log{
		Address: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Topics:  []common.Hash{common.HexToHash("0xdeadbeef")},
	}

	b.ReportAllocs()
	for b.Loop() {
		_, _, _ = d.Detect(log)
	}
}

// BenchmarkLookupKey isolates the map-key construction to verify it is
// zero-alloc (was 3 allocs/op when keyed on string concatenation of Hex()).
func BenchmarkLookupKey(b *testing.B) {
	addr := common.HexToAddress("0xe35e9842fceaCA96570B734083f4a58e8F7C5f2A")
	topic := common.HexToHash("0x32ed1a409ef04c7b0227189c3a103dc5ac10e775a15b785dcc510201f7c25ad3")

	b.ReportAllocs()
	for b.Loop() {
		_ = makeLookupKey(addr, topic)
	}
}
