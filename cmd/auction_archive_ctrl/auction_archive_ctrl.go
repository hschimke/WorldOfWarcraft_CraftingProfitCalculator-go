package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/blizzard_api_call"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cache_provider"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/environment_variables"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/auction_history"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/blizzard_api_helpers"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
)

func main() {
	fmt.Println("Auction Archive Control Program")

	logger := &cpclog.CpCLog{
		LogLevel: cpclog.GetLevel(environment_variables.LOG_LEVEL),
	}

	fAddScanRealm := flag.Bool("add_scan_realm", false, "Add a scanned realm")                     // (X)
	fArchiveAuctions := flag.Bool("archive_auctions", false, "Perform an auction archive")         // (-)
	fFillNItems := flag.Bool("fill_n_items", false, "Fill items with crafting data")               // (X)
	fFillNNames := flag.Bool("fill_n_names", false, "Fill items with names")                       // (X)
	fGetAllBonuses := flag.Bool("get_all_bonuses", false, "Return all bonuses for item")           // (X)
	fGetAllNames := flag.Bool("get_all_names", false, "Return all names in the system")            // (X)
	fGetAuctions := flag.Bool("get_auctions", false, "Perform an auction search")                  // (X)
	fGetScanRealms := flag.Bool("get_scan_realms", false, "Return a list of all scanned realms")   // (X)
	fRemoveScanRealm := flag.Bool("remove_scan_realm", false, "Remove a realm from the scan list") // (X)
	fScanRealms := flag.Bool("scan_realms", false, "Perform a scan on all scan realms")            // (X)
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

	cache := cache_provider.NewCacheProvider(context.TODO(), environment_variables.REDIS_URL)
	api := blizzard_api_call.NewBlizzardApiProvider(environment_variables.CLIENT_ID, environment_variables.CLIENT_SECRET, logger)
	helper := blizzard_api_helpers.NewBlizzardApiHelper(cache, logger, api)
	auctionHouseDataServer := auction_history.NewAuctionHistoryServer(environment_variables.DATABASE_CONNECTION_STRING, helper, logger)

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
	err := json.Unmarshal([]byte(*fBonuses), &bonuses)
	if err != nil {
		panic(fmt.Sprintf("bad bonuses: %v", *fBonuses))
	}

	if *fAddScanRealm {
		fmt.Println("AddScanRealm selected for ", realm, " ", *fRegion)
		err := auctionHouseDataServer.AddScanRealm(realm, *fRegion)
		if err != nil {
			fmt.Println("Error adding realm")
			fmt.Println(err)
		}
	}

	if *fArchiveAuctions {
		fmt.Println("ArchiveAuctions selected")
		auctionHouseDataServer.ArchiveAuctions()
	}

	if *fFillNItems {
		fmt.Println("FillNItems selected with N=", *fCount)
		auctionHouseDataServer.FillNItems(*fCount)
	}

	if *fFillNNames {
		fmt.Println("FillNNames selected with N=", *fCount)
		auctionHouseDataServer.FillNNames(*fCount)
	}

	if *fGetAllBonuses {
		fmt.Println("GetAllBonuses selected with item: ", item, " and region: ", *fRegion)
		all_bonuses, err := auctionHouseDataServer.GetAllBonuses(item, *fRegion)
		if err != nil {
			fmt.Println("Error getting bonuses")
			fmt.Println(err)
		}
		fmt.Println(all_bonuses)
	}

	if *fGetAllNames {
		fmt.Println("GetAllNames selected")
		all_names := auctionHouseDataServer.GetAllNames()
		fmt.Println(all_names)
	}

	if *fGetAuctions {
		fmt.Println("GetAuctions selected: ", item, " ", realm, " ", *fRegion, " ", start_dtm, "->", end_dtm)
		auctions, err := auctionHouseDataServer.GetAuctions(item, realm, *fRegion, bonuses, start_dtm, end_dtm)
		if err != nil {
			fmt.Println("Error selecting auctions")
			fmt.Println(err)
		}
		fmt.Println(auctions)
	}

	if *fGetScanRealms {
		fmt.Println("GetScanRealms selected")
		scan_realms, err := auctionHouseDataServer.GetScanRealms()
		if err != nil {
			fmt.Println("Error getting all scan realms")
			fmt.Println(err)
		}
		fmt.Println(scan_realms)
	}

	if *fRemoveScanRealm {
		fmt.Println("RemoveScanRealm selected for ", realm, " ", *fRegion)
		auctionHouseDataServer.RemoveScanRealm(realm, *fRegion)
	}

	if *fScanRealms {
		fmt.Println("ScanRealms selected")
		err := auctionHouseDataServer.ScanRealms(false)
		if err != nil {
			fmt.Println("Error scanning realms")
			fmt.Println(err)
		}
	}
}
