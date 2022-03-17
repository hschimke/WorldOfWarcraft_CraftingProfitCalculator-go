package static_sources

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/environment_variables"
)

const (
	bonuses_cache_fn                  string = "bonuses.json"
	rank_mappings_cache_fn            string = "rank-mappings.json"
	shopping_recipe_exclusion_list_fn string = "shopping-recipe-exclusion-list.json"
	static_source_dir                 string = "./static_files"
	raidbots_dl_uri                   string = "https://www.raidbots.com/static/data/live/bonuses.json" // Thank you raidbots
)

type StaticSources struct {
	bonus_cache                    *BonusesCache
	rank_mapping_cache             *RankMappingsCache
	shopping_recipe_exclusion_list *ShoppingRecipeExclusionList
}

// A simplified version of the data availble for bonus mappings from raidbots
type BonusesCache map[string]struct {
	Id      int `json:"id,omitempty"`
	Level   int `json:"level,omitempty"`
	Quality int `json:"quality,omitempty"`
	Socket  int `json:"socket,omitempty"`
}

// Rank mappings between rank and level
type RankMappingsCache struct {
	Available_levels []uint
	Rank_mapping     []uint
}

// A list of recipes to exclude from shopping searches
type ShoppingRecipeExclusionList struct {
	Exclusions []uint
}

type staticSource interface {
	BonusesCache | RankMappingsCache | ShoppingRecipeExclusionList
}

// load a static resource from the filesystem
func loadStaticResource[T staticSource](fn string, target *T) error {
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

// load a static resource from a URI
func fetchFromUri[T staticSource](uri string, target *T) error {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		//logger.Fatal(err)
		//level.Error(logger).Log(err)
		return fmt.Errorf("error with request: %s, err: %s", uri, err)
	}
	req.Header.Set("User-Agent", "WorldOfWarcraft_CraftingProfitCalculator-go")

	res, getErr := httpClient.Do(req)
	if getErr != nil {
		//level.Error(logger).Log("An error was encountered while retrieving an authorization token: ", getErr)
		return fmt.Errorf("error fetching uri: %s, err: %v", uri, getErr)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	parseErr := json.NewDecoder(res.Body).Decode(&target)
	if parseErr != nil {
		//log.Print(io.ReadAll(res.Body))
		//level.Error(logger).Log("An error was encountered while retrieving an authorization token: ", parseErr)
		return fmt.Errorf("error parsing api response for: %s, err: %v", uri, parseErr)
	}
	return nil
}

// Fetch the bonus catch, if it cannot be found locally it will be loaded from raidbots
func (s *StaticSources) GetBonuses() (*BonusesCache, error) {
	if s.bonus_cache == nil {
		bc := BonusesCache{}
		fn := path.Join(environment_variables.STATIC_DIR_ROOT, static_source_dir, bonuses_cache_fn)
		err := loadStaticResource(fn, &bc)
		if err != nil {
			// lets go get it
			fetchErr := fetchFromUri(raidbots_dl_uri, &bc)
			if fetchErr != nil {
				return nil, fetchErr
			}
		}
		s.bonus_cache = &bc
	}
	return s.bonus_cache, nil
}

// Fetch the rank mappings, if not available locally it will be empty
func (s *StaticSources) GetRankMappings() *RankMappingsCache {
	if s.rank_mapping_cache == nil {
		rm := RankMappingsCache{}
		fn := path.Join(environment_variables.STATIC_DIR_ROOT, static_source_dir, rank_mappings_cache_fn)
		err := loadStaticResource(fn, &rm)
		if err != nil {
			s.rank_mapping_cache = &RankMappingsCache{}
		}
		s.rank_mapping_cache = &rm

	}
	return s.rank_mapping_cache
}

// Fetch the shopping list exclusion set, if not available locally it will be empty
func (s *StaticSources) GetShoppingRecipeExclusionList() *ShoppingRecipeExclusionList {
	if s.shopping_recipe_exclusion_list == nil {
		sre := ShoppingRecipeExclusionList{}
		fn := path.Join(environment_variables.STATIC_DIR_ROOT, static_source_dir, shopping_recipe_exclusion_list_fn)
		err := loadStaticResource(fn, &sre)
		if err != nil {
			s.shopping_recipe_exclusion_list = &ShoppingRecipeExclusionList{}
		}
		s.shopping_recipe_exclusion_list = &sre
	}
	return s.shopping_recipe_exclusion_list
}
