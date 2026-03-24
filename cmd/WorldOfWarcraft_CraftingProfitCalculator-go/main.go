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
	"strconv"
	"syscall"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/blizz_oath"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/blizzard_api_call"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cache_provider"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/environment_variables"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/blizzard_api_helpers"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/wow_crafting_profits"
)

func main() {
	if err := environment_variables.Load(); err != nil {
		log.Fatalf("failed to load environment variables: %v", err)
	}
	logger := cpclog.NewCpCLog(cpclog.GetLevel(environment_variables.LOG_LEVEL))

	fRegion := flag.String("region", "us", "Region")
	fServer := flag.String("server", "Hyjal", "Server")
	fProfession := flag.String("profession", "[]", "Profession")
	fItem := flag.String("item", "Crafter's Mark of the First Ones", "Item")
	fCount := flag.Uint("count", 1, "How many of the main item to build")
	fJsonData := flag.String("json_data", "", "JSON configuration data")
	fUseJsonFlag := flag.Bool("json", false, "Use JSON to configure region, realm, and professions")
	fAllProfessionsFlag := flag.Bool("allprof", true, "Use all professions and ignore profession flag")
	flag.Parse()

	var character_config_json globalTypes.AddonData
	if *fJsonData != "" {
		err := json.Unmarshal([]byte(*fJsonData), &character_config_json)
		if err != nil {
			fmt.Printf("JSON character input cannot be parsed: %v", err)
		}
	}

	if !(*fUseJsonFlag) {
		err := json.Unmarshal([]byte(*fProfession), &(character_config_json.Professions))
		if err != nil {
			logger.Error(err.Error())
			character_config_json.Professions = make([]string, 0)
		}
		character_config_json.Realm.Realm_name = *fServer
		character_config_json.Realm.Region_name = *fRegion
	}

	item := globalTypes.ItemSoftIdentity{}
	if itm_id, err := strconv.ParseUint(*fItem, 0, 64); err == nil {
		item.ItemId = uint(itm_id)
	} else {
		item.ItemName = *fItem
	}

	config := globalTypes.NewRunConfig(&character_config_json, item, *fCount)
	config.UseAllProfessions = *fAllProfessionsFlag

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("Shutdown signal received, cancelling operations...")
		cancel()
	}()

	tokenServer := blizz_oath.NewTokenServer(environment_variables.CLIENT_ID, environment_variables.CLIENT_SECRET, logger)
	cache := cache_provider.NewCacheProvider(ctx, environment_variables.REDIS_URL)
	api := blizzard_api_call.NewBlizzardApiProvider(tokenServer, logger)
	helper := blizzard_api_helpers.NewBlizzardApiHelper(cache, logger, api)
	cpc := wow_crafting_profits.WoWCpCRunner{
		Helper: helper,
		Logger: logger,
	}

	runErr := cpc.CliRun(ctx, config)
	if runErr != nil {
		if errors.Is(runErr, context.Canceled) {
			logger.Info("Operation cancelled by user.")
		} else {
			logger.Errorf("Run error: %v", runErr)
		}
	}
}
