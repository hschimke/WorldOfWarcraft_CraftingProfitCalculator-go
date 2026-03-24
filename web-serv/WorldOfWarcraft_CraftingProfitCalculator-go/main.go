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
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/middleware"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/routes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/blizzard_api_helpers"
)

func main() {
	if err := environment_variables.Load(); err != nil {
		log.Fatalf("failed to load environment variables: %v", err)
	}
	ctx := context.Background()
	cache := cache_provider.NewCacheProvider(ctx, environment_variables.REDIS_URL)
	logger := cpclog.NewCpCLog(cpclog.GetLevel(environment_variables.LOG_LEVEL))
	tokenServer := blizz_oath.NewTokenServer(environment_variables.CLIENT_ID, environment_variables.CLIENT_SECRET, logger)
	api := blizzard_api_call.NewBlizzardApiProvider(tokenServer, logger)
	apiHelper := blizzard_api_helpers.NewBlizzardApiHelper(cache, logger, api)
	cpcRoutes := routes.NewCPCRoutes(ctx, environment_variables.DATABASE_CONNECTION_STRING, environment_variables.REDIS_URL, apiHelper, cache, logger)
	defer cpcRoutes.Shutdown()
	router := http.NewServeMux()

	// Create rate limiter: 10 requests per second, burst of 20
	rateLimiter := middleware.NewIPRateLimiter(10, 20)

	/*
		var frontend fs.FS = os.DirFS("html/build")
		httpFS := http.FS(frontend)
		fileServer := http.FileServer(httpFS)
		serveIndex := serveFileContents("index.html", httpFS)

		router.Handle("/", intercept404(fileServer, serveIndex))
	*/
	spa := spaHandler{staticPath: "html/build", indexPath: "index.html"}

	// Static files - no rate limiting
	router.Handle("/", spa)

	// API endpoints - apply rate limiting
	router.Handle("/json_output_QUEUED", rateLimiter.Middleware(http.HandlerFunc(cpcRoutes.JsonOutputQueue)))
	router.Handle("/json_output_CHECK", rateLimiter.Middleware(http.HandlerFunc(cpcRoutes.JsonOutputCheck)))

	if !environment_variables.DISABLE_AUCTION_HISTORY {
		router.Handle("/all_items", rateLimiter.Middleware(http.HandlerFunc(cpcRoutes.AllItems)))
		router.Handle("/scanned_realms", rateLimiter.Middleware(http.HandlerFunc(cpcRoutes.ScannedRealms)))
		router.Handle("/auction_history", rateLimiter.Middleware(http.HandlerFunc(cpcRoutes.AuctionHistory)))
		router.Handle("/seen_item_bonuses", rateLimiter.Middleware(http.HandlerFunc(cpcRoutes.SeenItemBonuses)))
	}

	router.Handle("/bonus_mappings", rateLimiter.Middleware(http.HandlerFunc(cpcRoutes.BonusMappings)))
	router.Handle("/addon-download", rateLimiter.Middleware(http.HandlerFunc(cpcRoutes.AddonDownload)))
	router.Handle("/all_realm_names", rateLimiter.Middleware(http.HandlerFunc(cpcRoutes.AllRealms)))

	// Healthcheck - no rate limiting
	router.HandleFunc("/healthcheck", cpcRoutes.Healthcheck)

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
