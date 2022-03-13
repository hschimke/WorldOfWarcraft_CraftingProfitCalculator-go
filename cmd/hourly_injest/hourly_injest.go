package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/environment_variables"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/auction_history"
)

func job(ctx context.Context, async bool) {
	cpclog.Info("Starting hourly injest job.")

	auction_history.ScanRealms(async)
	auction_history.FillNItems(20)
	if time.Now().Hour() == 4 {
		cpclog.Info("Performing daily archive.")
		auction_history.ArchiveAuctions()
	}
	cpclog.Info("Finished hourly injest job.")
}

func fillNames(ctx context.Context) {
	auction_history.FillNNames(100)
}

func main() {
	var (
		server_mode             = environment_variables.STANDALONE_CONTAINER
		include_auction_history = !environment_variables.DISABLE_AUCTION_HISTORY
	)

	cpclog.LogLevel = cpclog.GetLevel(environment_variables.LOG_LEVEL)

	if include_auction_history {
		switch server_mode {
		case "hourly":

			cpclog.Info("Started in default mode. Running job and exiting.")

			job(context.Background(), false)
			fillNames(context.Background())

		case "worker":

			cpclog.Info("Started as a worker thread, actions will be as if standalone but no server is running elsewhere.")
			fallthrough

		case "standalone":

			cpclog.Info("Started in standalone container mode. Scheduling hourly job.")
			cpclog.Info("Started in standalone container mode. Scheduling name fetch job.")

			nameFetchTick := time.NewTicker(time.Minute * 5)
			injestFetchTick := time.NewTicker(time.Hour * 1)

			ctx, cancel := context.WithCancel(context.Background())

			go func() {
				for range nameFetchTick.C {
					fillNames(ctx)
				}
			}()

			go func() {
				for range injestFetchTick.C {
					if time.Now().Hour()%3 == 0 {
						job(ctx, true)
					}
				}
			}()

			closeRequested := make(chan os.Signal, 1)
			signal.Notify(closeRequested, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

			<-closeRequested
			cancel()
			nameFetchTick.Stop()
			injestFetchTick.Stop()
			cpclog.Info("Shutting down")

		case "normal":
			fallthrough
		default:
			cpclog.Info("Started in normal mode taking no action.")
		}
	} else {
		cpclog.Info("Started without auction history enabled, why do I exist? Exiting.")
	}
}
