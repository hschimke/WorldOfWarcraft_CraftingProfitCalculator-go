package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strconv"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/wow_crafting_profits"
)

func main() {

	allProfessions, _ := json.Marshal(globalTypes.ALL_PROFESSIONS)

	fRegion := flag.String("region", "us", "Region")
	fServer := flag.String("server", "Hyjal", "Server")
	fProfession := flag.String("profession", string(allProfessions), "Profession")
	//fProfession := flag.String("profession", "[\"Tailoring\", \"Enchanting\"]", "Profession")
	//fItem := flag.String("item", "171276", "Item")
	//fItem := flag.String("item", "Grim-Veiled Bracers", "Item")
	//fItem := flag.String("item", "Grim-Veiled Bracers", "Item")
	fItem := flag.String("item", "Crafter's Mark of the First Ones", "Item")
	fCount := flag.Uint("count", 1, "How many of the main item to build")
	fJsonData := flag.String("json_data", "", "JSON configuration data")
	fUseJsonFlag := flag.Bool("json", false, "Use JSON to configure region, realm, and professions")
	fAllProfessionsFlag := flag.Bool("allprof", true, "Use all professions and ignore profession flag")
	flag.Parse()

	character_config_json := globalTypes.AddonData{}

	err := json.Unmarshal([]byte(*fJsonData), &character_config_json)
	if err != nil {
		fmt.Print("JSON character input cannot be parsed.")
	}

	if !(*fUseJsonFlag) {
		character_config_json.Inventory = make([]struct {
			Id       uint "json:\"id,omitempty\""
			Quantity uint "json:\"quantity,omitempty\""
		}, 0)
		err := json.Unmarshal([]byte(*fProfession), &(character_config_json.Professions))
		if err != nil {
			cpclog.Error(err.Error())
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

	runErr := wow_crafting_profits.CliRun(config)
	if runErr != nil {
		cpclog.Error(runErr.Error())
	}
}
