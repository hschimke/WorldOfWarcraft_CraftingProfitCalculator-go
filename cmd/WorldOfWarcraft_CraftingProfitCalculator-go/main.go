package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strconv"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cache_provider"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/environment_variables"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/blizzard_api_helpers"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/wow_crafting_profits"
)

func main() {
	logger := &cpclog.CpCLog{
		LogLevel: cpclog.GetLevel(environment_variables.LOG_LEVEL),
	}
	//http.ListenAndServe("localhost:8080", nil)
	//defer profile.Start(profile.ProfilePath("."), profile.CPUProfile, profile.MemProfileHeap).Stop()
	//defer profile.Start(profile.BlockProfile).Stop()
	//defer profile.Start(profile.MemProfileHeap).Stop()
	//defer profile.Start(profile.MemProfileAllocs).Stop()
	//defer profile.Start(profile.MutexProfile).Stop()
	//allProfessions, _ := json.Marshal(globalTypes.ALL_PROFESSIONS)

	fRegion := flag.String("region", "us", "Region")
	fServer := flag.String("server", "Hyjal", "Server")
	fProfession := flag.String("profession", "[]", "Profession")
	//fProfession := flag.String("profession", "[\"Tailoring\", \"Enchanting\"]", "Profession")
	//fItem := flag.String("item", "171276", "Item")
	fItem := flag.String("item", "Grim-Veiled Bracers", "Item")
	//fItem := flag.String("item", "Crafter's Mark of the First Ones", "Item")
	//fItem := flag.String("item", "Notorious Combatant's Mail Waistguard", "Item")
	fCount := flag.Uint("count", 1, "How many of the main item to build")
	fJsonData := flag.String("json_data", "", "JSON configuration data")
	fUseJsonFlag := flag.Bool("json", false, "Use JSON to configure region, realm, and professions")
	fAllProfessionsFlag := flag.Bool("allprof", true, "Use all professions and ignore profession flag")
	flag.Parse()

	//character_config_json := globalTypes.AddonData{}
	var character_config_json globalTypes.AddonData

	err := json.Unmarshal([]byte(*fJsonData), &character_config_json)
	//err := json.Unmarshal([]byte(testJson), &character_config_json)
	if err != nil {
		fmt.Printf("JSON character input cannot be parsed: %v", err)
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

	cache := cache_provider.NewCacheProvider(context.TODO(), environment_variables.REDIS_URL)
	helper := blizzard_api_helpers.NewBlizzardApiHelper(environment_variables.CLIENT_ID, environment_variables.CLIENT_SECRET, cache, logger)

	runErr := wow_crafting_profits.CliRun(config, helper, logger)
	if runErr != nil {
		logger.Error(runErr.Error())
	}

	//fl, _ := os.Create("memprof.pprof")
	//fl2, _ := os.Create("allocs.pprof")
	//defer fl2.Close()
	//defer fl.Close()
	//pprof.WriteHeapProfile(fl)
	//pprof.Lookup("allocs").WriteTo(fl2, 0)
}
