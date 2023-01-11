package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/blizz_oath"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/blizzard_api_call"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cache_provider"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/environment_variables"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/static_sources"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/auction_history"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/blizzard_api_helpers"
)

func job(ctx context.Context, auctionHouse *auction_history.AuctionHistoryServer, logger *cpclog.CpCLog, async bool) {
	logger.Info("Starting hourly injest job.")

	auctionHouse.ScanRealms(async)
	auctionHouse.FillNItems(20, &static_sources.StaticSources{})
	logger.Info("Performing daily archive.")
	auctionHouse.ArchiveAuctions()
	logger.Info("Finished hourly injest job.")
}

func fillNames(ctx context.Context, auctionHouse *auction_history.AuctionHistoryServer) {
	auctionHouse.FillNNames(100)
}

func main() {
	var (
		logger = &cpclog.CpCLog{
			LogLevel: cpclog.GetLevel(environment_variables.LOG_LEVEL),
		}
		ctx, cancel             = context.WithCancel(context.Background())
		server_mode             = environment_variables.STANDALONE_CONTAINER
		include_auction_history = !environment_variables.DISABLE_AUCTION_HISTORY
		cache                   = cache_provider.NewCacheProvider(ctx, environment_variables.REDIS_URL)
		tokenServer             = blizz_oath.NewTokenServer(environment_variables.CLIENT_ID, environment_variables.CLIENT_SECRET, logger)
		api                     = blizzard_api_call.NewBlizzardApiProvider(tokenServer, logger)
		helper                  = blizzard_api_helpers.NewBlizzardApiHelper(cache, logger, api)
		auctionHouseServer      = auction_history.NewAuctionHistoryServer(ctx, environment_variables.DATABASE_CONNECTION_STRING, helper, logger)
	)
	defer auctionHouseServer.Shutdown()

	if include_auction_history {
		switch server_mode {
		case "hourly":

			logger.Info("Started in default mode. Running job and exiting.")

			job(ctx, auctionHouseServer, logger, false)
			fillNames(ctx, auctionHouseServer)

		case "worker":

			logger.Info("Started as a worker thread, actions will be as if standalone but no server is running elsewhere.")
			fallthrough

		case "standalone":

			logger.Info("Started in standalone container mode. Scheduling hourly job.")
			logger.Info("Started in standalone container mode. Scheduling name fetch job.")

			nameFetchTick := time.NewTicker(time.Minute * 5)
			injestFetchTick := time.NewTicker(time.Hour * 1)

			go func() {
				for range nameFetchTick.C {
					fillNames(ctx, auctionHouseServer)
				}
			}()

			go func() {
				for range injestFetchTick.C {
					if time.Now().Hour()%3 == 0 {
						job(ctx, auctionHouseServer, logger, true)
					}
				}
			}()

			closeRequested := make(chan os.Signal, 1)
			signal.Notify(closeRequested, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

			<-closeRequested
			cancel()
			nameFetchTick.Stop()
			injestFetchTick.Stop()
			logger.Info("Shutting down")

		case "normal":
			fallthrough
		default:
			logger.Info("Started in normal mode taking no action.")
		}
	} else {
		logger.Info("Started without auction history enabled, why do I exist? Exiting.")
	}
}
