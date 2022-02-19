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

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/environment_variables"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/routes"
)

func main() {
	cpclog.LogLevel = cpclog.GetLevel("debug")
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

	router.HandleFunc("/json_output_QUEUED", routes.JsonOutputQueue)
	router.HandleFunc("/json_output_CHECK", routes.JsonOutputCheck)
	//http.HandleFunc("/json_output", routes.JsonOutput)

	if !environment_variables.DISABLE_AUCTION_HISTORY {
		router.HandleFunc("/all_items", routes.AllItems)
		router.HandleFunc("/scanned_realms", routes.ScannedRealms)
		router.HandleFunc("/auction_history", routes.AuctionHistory)
		router.HandleFunc("/seen_item_bonuses", routes.SeenItemBonuses)
	}

	router.HandleFunc("/bonus_mappings", routes.BonusMappings)
	router.HandleFunc("/addon-download", routes.AddonDownload)
	router.HandleFunc("/healthcheck", routes.Healthcheck)
	router.HandleFunc("/all_realm_names", routes.AllRealms)

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
	cpclog.Info("Shutting down")
	server.Shutdown(context.Background())
}
