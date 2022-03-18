package auction_history

import (
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/blizzard_api_helpers"
)

type AuctionHistoryServer struct {
	helper           *blizzard_api_helpers.BlizzardApiHelper
	connectionString string
	logger           *cpclog.CpCLog
}

func NewAuctionHistoryServer(connectionString string, helper *blizzard_api_helpers.BlizzardApiHelper, logger *cpclog.CpCLog) *AuctionHistoryServer {
	ahs := AuctionHistoryServer{
		helper:           helper,
		connectionString: connectionString,
		logger:           logger,
	}
	ahs.dbSetup()
	return &ahs
}
