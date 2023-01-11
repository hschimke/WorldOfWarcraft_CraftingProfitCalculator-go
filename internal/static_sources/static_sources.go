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
	raidbots_dl_uri                   string = "https://www.raidbots.com/static/data/live/bonuses.json"                                                                                        // Thank you raidbots
	firesong_df_crafting_source       string = "https://gist.githubusercontent.com/Firesong25/cc294b9360ab37b01d2350cc266f73e5/raw/a50661505f38d93c46757cf9122b3360a52b601b/CraftedItems.json" //https://gist.github.com/Firesong25/cc294b9360ab37b01d2350cc266f73e5
	firesong_df_crafting_fn           string = "CraftedItems.json"
)

// StaticSources allows for long cachable data to be saved. A future version may use embed
type StaticSources struct {
	bonusCache                               *BonusesCache
	rankMappingCache                         *RankMappingsCache
	shoppingRecipeExclusionList              *ShoppingRecipeExclusionList
	firesongDFCrafting                       *FireSongCraftingLinkTable
	BonusCacheFileName                       string
	RankMappingsCacheFileName                string
	ShoppingRecipeExclusionListCacheFileName string
	RootDirectory                            string
	RaidbotsURI                              string
	FireSongDfCraftingUri                    string
	FireSongDFCraftingFileName               string
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

// A mapping for Firesongs list of items and crafted ids
type FireSongCraftingLinkTable []struct {
	Id              uint
	ListOfReagents  *[]uint
	Name            string
	ProfessionId    uint
	SkillTierId     uint
	Category        *string
	IsCommodity     bool
	Reagents        *[]string
	RecipeId        uint
	CraftedQuantity uint
}

type staticSource interface {
	BonusesCache | RankMappingsCache | ShoppingRecipeExclusionList | FireSongCraftingLinkTable
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

func (s *StaticSources) fillNames() {
	if len(s.BonusCacheFileName) == 0 {
		s.BonusCacheFileName = bonuses_cache_fn
	}
	if len(s.RankMappingsCacheFileName) == 0 {
		s.RankMappingsCacheFileName = rank_mappings_cache_fn
	}
	if len(s.ShoppingRecipeExclusionListCacheFileName) == 0 {
		s.ShoppingRecipeExclusionListCacheFileName = shopping_recipe_exclusion_list_fn
	}
	if len(s.RootDirectory) == 0 {
		s.RootDirectory = static_source_dir
	}
	if len(s.RaidbotsURI) == 0 {
		s.RaidbotsURI = raidbots_dl_uri
	}
	if len(s.FireSongDfCraftingUri) == 0 {
		s.FireSongDfCraftingUri = firesong_df_crafting_source
	}
	if len(s.FireSongDFCraftingFileName) == 0 {
		s.FireSongDFCraftingFileName = firesong_df_crafting_fn
	}
}

// Fetch the bonus catch, if it cannot be found locally it will be loaded from raidbots
func (s *StaticSources) GetBonuses() (*BonusesCache, error) {
	s.fillNames()
	if s.bonusCache == nil {
		bc := BonusesCache{}
		fn := path.Join(environment_variables.STATIC_DIR_ROOT, s.RootDirectory, s.BonusCacheFileName)
		err := loadStaticResource(fn, &bc)
		if err != nil {
			// lets go get it
			fetchErr := fetchFromUri(s.RaidbotsURI, &bc)
			if fetchErr != nil {
				return nil, fetchErr
			}
		}
		s.bonusCache = &bc
	}
	return s.bonusCache, nil
}

// Fetch the rank mappings, if not available locally it will be empty
func (s *StaticSources) GetRankMappings() *RankMappingsCache {
	s.fillNames()
	if s.rankMappingCache == nil {
		rm := RankMappingsCache{}
		fn := path.Join(environment_variables.STATIC_DIR_ROOT, s.RootDirectory, s.RankMappingsCacheFileName)
		err := loadStaticResource(fn, &rm)
		if err != nil {
			s.rankMappingCache = &RankMappingsCache{}
		}
		s.rankMappingCache = &rm

	}
	return s.rankMappingCache
}

// Fetch the shopping list exclusion set, if not available locally it will be empty
func (s *StaticSources) GetShoppingRecipeExclusionList() *ShoppingRecipeExclusionList {
	s.fillNames()
	if s.shoppingRecipeExclusionList == nil {
		sre := ShoppingRecipeExclusionList{}
		fn := path.Join(environment_variables.STATIC_DIR_ROOT, s.RootDirectory, s.ShoppingRecipeExclusionListCacheFileName)
		err := loadStaticResource(fn, &sre)
		if err != nil {
			s.shoppingRecipeExclusionList = &ShoppingRecipeExclusionList{}
		}
		s.shoppingRecipeExclusionList = &sre
	}
	return s.shoppingRecipeExclusionList
}

// Fetch the crafting link table built by FireSong
// https://us.forums.blizzard.com/en/blizzard/t/dragonflight-profession-recipes-crafted-item-id/37444/7
// https://gist.github.com/Firesong25/cc294b9360ab37b01d2350cc266f73e5
func (s *StaticSources) GetFireSongsCraftingLinkTable() (*FireSongCraftingLinkTable, error) {
	s.fillNames()
	if s.firesongDFCrafting == nil {
		fdc := FireSongCraftingLinkTable{}
		fn := path.Join(environment_variables.STATIC_DIR_ROOT, s.RootDirectory, s.FireSongDFCraftingFileName)
		err := loadStaticResource(fn, &fdc)
		if err != nil {
			// lets go get it
			fetchErr := fetchFromUri(s.FireSongDfCraftingUri, &fdc)
			if fetchErr != nil {
				return nil, fetchErr
			}
		}
		s.firesongDFCrafting = &fdc
	}
	return s.firesongDFCrafting, nil
}
