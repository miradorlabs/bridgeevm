package bridgeevm

import (
	"github.com/ethereum/go-ethereum/common"
)

type LegType string

const (
	LegTypeSource      LegType = "source"
	LegTypeDestination LegType = "destination"
)

type Result struct {
	BridgeName        string
	BridgeDescription string
	LegType           LegType
	CorrelationID     string
	Contract          common.Address
	EventTopic        common.Hash
	EventName         string
}
