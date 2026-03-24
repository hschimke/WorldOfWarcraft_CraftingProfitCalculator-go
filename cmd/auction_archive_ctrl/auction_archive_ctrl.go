package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
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
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
)

func main() {
	fmt.Println("Auction Archive Control Program")

	if err := environment_variables.Load(); err != nil {
		log.Fatalf("failed to load environment variables: %v", err)
	}

	logger := cpclog.NewCpCLog(cpclog.GetLevel(environment_variables.LOG_LEVEL))

	fAddScanRealm := flag.Bool("add_scan_realm", false, "Add a scanned realm")
	fArchiveAuctions := flag.Bool("archive_auctions", false, "Perform an auction archive")
	fFillNItems := flag.Bool("fill_n_items", false, "Fill items with crafting data")
	fFillNNames := flag.Bool("fill_n_names", false, "Fill items with names")
	fGetAllBonuses := flag.Bool("get_all_bonuses", false, "Return all bonuses for item")
	fGetAllNames := flag.Bool("get_all_names", false, "Return all names in the system")
	fGetAuctions := flag.Bool("get_auctions", false, "Perform an auction search")
	fGetScanRealms := flag.Bool("get_scan_realms", false, "Return a list of all scanned realms")
	fRemoveScanRealm := flag.Bool("remove_scan_realm", false, "Remove a realm from the scan list")
	fScanRealms := flag.Bool("scan_realms", false, "Perform a scan on all scan realms")
	fLogLevel := flag.String("log_level", "info", "Loglevel to output")

	fRealmName := flag.String("realm_name", "", "A name of a realm")
	fRealmId := flag.Uint("realm_id", 0, "A connected realm ID")
	fRegion := flag.String("region", "us", "The region to work within")
	fCount := flag.Uint("count", 0, "Used for any thing with a count")
	fItemName := flag.String("item_name", "", "Name of an item")
	fItemId := flag.Uint("item_id", 0, "An item id number")
	fStartDtm := flag.String("start_dtm", "", "Start date")
	fEndDtm := flag.String("end_dtm", "", "End date")
	fBonuses := flag.String("bonuses", "[]", "json formatted array of bonuses")

	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	cache := cache_provider.NewCacheProvider(ctx, environment_variables.REDIS_URL)
	tokenServer := blizz_oath.NewTokenServer(environment_variables.CLIENT_ID, environment_variables.CLIENT_SECRET, logger)
	api := blizzard_api_call.NewBlizzardApiProvider(tokenServer, logger)
	helper := blizzard_api_helpers.NewBlizzardApiHelper(cache, logger, api)
	auctionHouseDataServer := auction_history.NewAuctionHistoryServer(ctx, environment_variables.DATABASE_CONNECTION_STRING, helper, logger)
	defer auctionHouseDataServer.Shutdown()

	logger.LogLevel = cpclog.GetLevel(*fLogLevel)

	realm := globalTypes.ConnectedRealmSoftIentity{
		Id:   *fRealmId,
		Name: *fRealmName,
	}

	item := globalTypes.ItemSoftIdentity{
		ItemName: *fItemName,
		ItemId:   *fItemId,
	}

	var start_dtm, end_dtm time.Time

	if *fStartDtm == "" {
		start_dtm = time.Now().AddDate(-1, 0, 0)
	} else {
		var err error
		start_dtm, err = time.Parse(time.ANSIC, *fStartDtm)
		if err != nil {
			panic(fmt.Sprintf("bad time: %s", *fStartDtm))
		}
	}

	if *fEndDtm == "" {
		end_dtm = time.Now()
	} else {
		var err error
		end_dtm, err = time.Parse(time.ANSIC, *fEndDtm)
		if err != nil {
			panic(fmt.Sprintf("bad time: %s", *fEndDtm))
		}
	}

	var bonuses []uint
	if err := json.Unmarshal([]byte(*fBonuses), &bonuses); err != nil {
		panic(fmt.Sprintf("bad bonuses: %v", *fBonuses))
	}

	if *fAddScanRealm {
		if err := auctionHouseDataServer.AddScanRealm(ctx, realm, *fRegion); err != nil {
			fmt.Printf("Error adding realm: %v\n", err)
		}
	}

	if *fArchiveAuctions {
		auctionHouseDataServer.ArchiveAuctions(ctx)
	}

	if *fFillNItems {
		if err := auctionHouseDataServer.FillNItems(ctx, *fCount, &static_sources.StaticSources{}); err != nil {
			fmt.Printf("Error filling items: %v\n", err)
		}
	}

	if *fFillNNames {
		if err := auctionHouseDataServer.FillNNames(ctx, *fCount); err != nil {
			fmt.Printf("Error filling names: %v\n", err)
		}
	}

	if *fGetAllBonuses {
		all_bonuses, err := auctionHouseDataServer.GetAllBonuses(ctx, item, *fRegion)
		if err != nil {
			fmt.Printf("Error getting bonuses: %v\n", err)
		} else {
			fmt.Println(all_bonuses)
		}
	}

	if *fGetAllNames {
		all_names := auctionHouseDataServer.GetAllNames(ctx)
		fmt.Println(all_names)
	}

	if *fGetAuctions {
		auctions, err := auctionHouseDataServer.GetAuctions(ctx, item, realm, *fRegion, bonuses, start_dtm, end_dtm)
		if err != nil {
			fmt.Printf("Error selecting auctions: %v\n", err)
		} else {
			fmt.Println(auctions)
		}
	}

	if *fGetScanRealms {
		scan_realms, err := auctionHouseDataServer.GetScanRealms(ctx)
		if err != nil {
			fmt.Printf("Error getting all scan realms: %v\n", err)
		} else {
			fmt.Println(scan_realms)
		}
	}

	if *fRemoveScanRealm {
		if err := auctionHouseDataServer.RemoveScanRealm(ctx, realm, *fRegion); err != nil {
			fmt.Printf("Error removing realm: %v\n", err)
		}
	}

	if *fScanRealms {
		if err := auctionHouseDataServer.ScanRealms(ctx, false); err != nil {
			fmt.Printf("Error scanning realms: %v\n", err)
		}
	}

	if errors.Is(ctx.Err(), context.Canceled) {
		fmt.Println("Operations cancelled.")
	}
}
