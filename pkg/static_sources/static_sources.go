package static_sources

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"time"
)

const (
	bonuses_cache_fn                  string = "bonuses.json"
	rank_mappings_cache_fn            string = "rank-mappings.json"
	shopping_recipe_exclusion_list_fn string = "shopping-recipe-exclusion-list.json"
	static_source_dir                 string = "./static_files"
	raidbots_dl_uri                   string = "https://www.raidbots.com/static/data/live/bonuses.json"
)

var (
	bonus_cache                    *BonusesCache
	rank_mapping_cache             *RankMappingsCache
	shopping_recipe_exclusion_list *ShoppingRecipeExclusionList
)

type BonusesCache map[uint]struct {
	Id      uint
	Level   uint
	Quality uint
	Socket  uint
}

type RankMappingsCache struct {
	Available_levels []uint
	Rank_mapping     []uint
}

type ShoppingRecipeExclusionList struct {
	Exclusions []uint
}

func loadStaticResource(fn string, target interface{}) error {
	file, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer file.Close()

	parseError := json.NewDecoder(file).Decode(&target)
	if parseError != nil {
		return parseError
	}
	return nil
}

func fetchFromUri(uri string, target interface{}) error {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest(http.MethodPost, uri, nil)
	if err != nil {
		//logger.Fatal(err)
		//level.Error(logger).Log(err)
		return fmt.Errorf("error with request: %s, err: %s", uri, err)
	}
	req.Header.Set("User-Agent", "WorldOfWarcraft_CraftingProfitCalculator-go")

	res, getErr := httpClient.Do(req)
	if getErr != nil {
		//level.Error(logger).Log("An error was encountered while retrieving an authorization token: ", getErr)
		return fmt.Errorf("error fetching uri: %s, err: %s", uri, getErr)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	parseErr := json.NewDecoder(res.Body).Decode(&target)
	if parseErr != nil {
		//log.Print(io.ReadAll(res.Body))
		//level.Error(logger).Log("An error was encountered while retrieving an authorization token: ", parseErr)
		return fmt.Errorf("error parsing api response for: %s, err: %s", uri, parseErr)
	}
	return nil
}

func GetBonuses() (*BonusesCache, error) {
	if bonus_cache == nil {
		fn := path.Join(static_source_dir, bonuses_cache_fn)
		err := loadStaticResource(fn, bonus_cache)
		if err != nil {
			// lets go get it
			fetchErr := fetchFromUri(raidbots_dl_uri, bonus_cache)
			if fetchErr != nil {
				return nil, fetchErr
			}
		}
	}
	return bonus_cache, nil
}

func GetRankMappings() *RankMappingsCache {
	if rank_mapping_cache == nil {
		fn := path.Join(static_source_dir, rank_mappings_cache_fn)
		err := loadStaticResource(fn, rank_mapping_cache)
		if err != nil {
			rank_mapping_cache = &RankMappingsCache{}
		}
	}
	return rank_mapping_cache
}

func GetShoppingRecipeExclusionList() *ShoppingRecipeExclusionList {
	if shopping_recipe_exclusion_list == nil {
		fn := path.Join(static_source_dir, shopping_recipe_exclusion_list_fn)
		err := loadStaticResource(fn, shopping_recipe_exclusion_list)
		if err != nil {
			shopping_recipe_exclusion_list = &ShoppingRecipeExclusionList{}
		}
	}
	return shopping_recipe_exclusion_list
}

/*
let bonuses_cache: BonusesCache;
let rank_mappings_cache: RankMappingsCache;
let shopping_recipe_exclusion_list: ShoppingRecipeExclusionList;
*/
