package bridgeevm

// Supported chain identifiers for use with New.
//
// New accepts any string and returns a usable Detector with zero
// subscriptions for unrecognized chains; passing one of these constants is
// the recommended way to avoid typos and to discover the supported set from
// godoc.
const (
	ChainArbitrum = "arbitrum"
	ChainBase     = "base"
	ChainBSC      = "bsc"
	ChainEthereum = "ethereum"
	ChainOptimism = "optimism"
	ChainPolygon  = "polygon"
)
