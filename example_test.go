package bridgeevm_test

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/miradorlabs/bridgeevm"
)

// ExampleNew shows how to construct a Detector for a specific chain.
// New is cheap and safe to call once at process start.
func ExampleNew() {
	d, err := bridgeevm.New("ethereum")
	if err != nil {
		panic(err)
	}
	fmt.Println(d.ChainName(), d.Len() > 0)
	// Output: ethereum true
}

// ExampleDetector_Detect shows the common path: hand any *types.Log to
// Detect and read the bridge name, leg, and correlation ID off the result.
func ExampleDetector_Detect() {
	d, _ := bridgeevm.New("arbitrum")

	log := &types.Log{
		Address: common.HexToAddress("0xe35e9842fceaCA96570B734083f4a58e8F7C5f2A"),
		Topics: []common.Hash{
			common.HexToHash("0x32ed1a409ef04c7b0227189c3a103dc5ac10e775a15b785dcc510201f7c25ad3"),
			common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000002105"),
			common.HexToHash("0x6f1b280c20fb309b3653566de6f876cd100c6d26bc9722649865147ce22480e7"),
			common.HexToHash("0x0000000000000000000000006892ca799bbb10cb13e3cc2a7b587365ba2f3597"),
		},
	}

	result, ok, err := d.Detect(log)
	if err != nil {
		// A bridge matched but its data was malformed; the boolean is true.
		panic(err)
	}
	if !ok {
		fmt.Println("not a bridge log")
		return
	}
	fmt.Printf("%s %s leg, correlation %s\n",
		result.BridgeName, result.LegType, result.CorrelationID)
	// Output: across source leg, correlation 50254707460338143966371593114785553611174598192372265612265721490268521529575
}

// ExampleDetector_Detect_unknownLog shows that Detect returns ok=false
// (with no error) for the common case of a log that does not match any
// configured bridge.
func ExampleDetector_Detect_unknownLog() {
	d, _ := bridgeevm.New("ethereum")

	log := &types.Log{
		Address: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Topics:  []common.Hash{common.HexToHash("0xdeadbeef")},
	}

	_, ok, err := d.Detect(log)
	fmt.Println(ok, errors.Is(err, nil))
	// Output: false true
}
