package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/blizz_oath"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/blizzard_api_call"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cache_provider"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/environment_variables"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/routes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/blizzard_api_helpers"
)

func main() {
	cache := cache_provider.NewCacheProvider(context.TODO(), environment_variables.REDIS_URL)
	logger := &cpclog.CpCLog{
		LogLevel: cpclog.GetLevel(environment_variables.LOG_LEVEL),
	}
	tokenServer := blizz_oath.NewTokenServer(environment_variables.CLIENT_ID, environment_variables.CLIENT_SECRET, logger)
	api := blizzard_api_call.NewBlizzardApiProvider(tokenServer, logger)
	apiHelper := blizzard_api_helpers.NewBlizzardApiHelper(cache, logger, api)
	cpcRoutes := routes.NewCPCRoutes(environment_variables.DATABASE_CONNECTION_STRING, environment_variables.REDIS_URL, apiHelper, cache, logger)
	router := http.NewServeMux()
	/*
		var frontend fs.FS = os.DirFS("html/build")
		httpFS := http.FS(frontend)
		fileServer := http.FileServer(httpFS)
		serveIndex := serveFileContents("index.html", httpFS)

		router.Handle("/", intercept404(fileServer, serveIndex))
	*/
	spa := spaHandler{staticPath: "html/build", indexPath: "index.html"}

	router.Handle("/", spa)

	router.HandleFunc("/json_output_QUEUED", cpcRoutes.JsonOutputQueue)
	router.HandleFunc("/json_output_CHECK", cpcRoutes.JsonOutputCheck)
	//http.HandleFunc("/json_output", routes.JsonOutput)

	if !environment_variables.DISABLE_AUCTION_HISTORY {
		router.HandleFunc("/all_items", cpcRoutes.AllItems)
		router.HandleFunc("/scanned_realms", cpcRoutes.ScannedRealms)
		router.HandleFunc("/auction_history", cpcRoutes.AuctionHistory)
		router.HandleFunc("/seen_item_bonuses", cpcRoutes.SeenItemBonuses)
	}

	router.HandleFunc("/bonus_mappings", cpcRoutes.BonusMappings)
	router.HandleFunc("/addon-download", cpcRoutes.AddonDownload)
	router.HandleFunc("/healthcheck", cpcRoutes.Healthcheck)
	router.HandleFunc("/all_realm_names", cpcRoutes.AllRealms)

	address := fmt.Sprintf(":%d", environment_variables.SERVER_PORT)

	server := &http.Server{
		Handler:      router,
		Addr:         address,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
	}

	fmt.Println("Starting CPC client and api server")

	go func() {
		log.Fatal(server.ListenAndServe())
	}()

	closeRequested := make(chan os.Signal, 1)
	signal.Notify(closeRequested, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-closeRequested
	logger.Info("Shutting down")
	server.Shutdown(context.Background())
}
