package globalTypes

type RegionCode = string
type ItemID = uint
type ItemName = string
type ConnectedRealmID = uint
type RealmName = string
type CharacterProfession = string
type ConnectedRealmName = string

type CraftingStatus struct {
	Recipe_ids []uint
	Craftable  bool
	Recipes    []struct {
		Recipe_id           uint
		Crafting_profession CharacterProfession
	}
}

type SkillTierCyclicLinks map[uint][]struct {
	Id    uint
	Takes float64
	Makes float64
}

type OutputFormatPrice struct {
	Sales   uint    `json:"sales"`
	High    float64 `json:"high"`
	Low     float64 `json:"low"`
	Average float64 `json:"average"`
	Median  float64 `json:"median"`
}

type ShoppingListCost struct {
	Vendor float64           `json:"vendor"`
	Ah     OutputFormatPrice `json:"ah"`
}

type ShoppingList struct {
	Quantity float64          `json:"quantity"`
	Id       ItemID           `json:"id"`
	Name     ItemName         `json:"name"`
	Cost     ShoppingListCost `json:"cost"`
}

type OutputFormatShoppingList = map[uint][]ShoppingList

type OutpoutFormatRecipeOutput struct {
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Value float64 `json:"value"`
}

type OutputFormatRecipe struct {
	Name    string                    `json:"name"`
	Rank    uint                      `json:"rank"`
	Id      uint                      `json:"id"`
	Output  OutpoutFormatRecipeOutput `json:"output"`
	Ah      OutputFormatPrice         `json:"ah"`
	High    float64                   `json:"high"`
	Low     float64                   `json:"low"`
	Average float64                   `json:"average"`
	Median  float64                   `json:"median"`
	Parts   []OutputFormatObject      `json:"parts"`
}

type OutputFormatBonusPrices struct {
	Level uint              `json:"level"`
	Ah    OutputFormatPrice `json:"ah"`
}

type OutputFormatObject struct {
	Name           string                    `json:"name"`
	Id             uint                      `json:"id"`
	Required       float64                   `json:"required"`
	Recipes        []OutputFormatRecipe      `json:"recipes"`
	Ah             OutputFormatPrice         `json:"ah"`
	Vendor         float64                   `json:"vendor"`
	Bonus_prices   []OutputFormatBonusPrices `json:"bonus_prices"`
	Shopping_lists OutputFormatShoppingList  `json:"shopping_lists"`
}

type AHItemPriceObject struct {
	Total_sales uint
	Average     float64
	Median      float64
	High        float64
	Low         float64
}

type RecipeOption struct {
	Prices []ProfitAnalysisObject
	Recipe struct {
		Recipe_id           uint
		Crafting_profession string
	}
	Rank    uint
	Rank_ah AHItemPriceObject
}

type ProfitAnalysisObject struct {
	Item_id         uint
	Item_name       string
	Ah_price        AHItemPriceObject
	Item_quantity   float64
	Vendor_price    float64
	Crafting_status CraftingStatus
	Bonus_lists     [][]uint
	Recipe_options  []RecipeOption
	Bonus_prices    []struct {
		Level uint
		Ah    AHItemPriceObject
	}
}

type RunReturn struct {
	Price        ProfitAnalysisObject `json:"price,omitempty"`
	Intermediate OutputFormatObject   `json:"intermediate,omitempty"`
	Formatted    string               `json:"formatted,omitempty"`
}

type ItemSoftIdentity struct {
	ItemName string
	ItemId   uint
}

type ConnectedRealmSoftIentity struct {
	Id   ConnectedRealmID
	Name ConnectedRealmName
}

type RunJob struct {
	JobId     string
	JobConfig struct {
		Item              ItemSoftIdentity
		Count             uint
		UseAllProfessions bool
		AddonData         AddonData
	}
}

type ReturnError struct {
	ERROR string
}

type QueuedJobReturn struct {
	JobId string `json:"job_id"`
}

var ALL_PROFESSIONS []CharacterProfession = []CharacterProfession{"Blacksmithing", "Leatherworking", "Alchemy", "Herbalism", "Cooking", "Mining", "Tailoring", "Engineering", "Enchanting", "Fishing", "Skinning", "Jewelcrafting", "Inscription", "Archaeology", "Soul Cyphering", "Abominable Stitching", "Ascension Crafting", "Stygia Crafting"}

// constants for CPC job queue needed by server and job runner
const (
	CPC_JOB_QUEUE_NAME           = "cpc-job-queue:web-jobs"
	CPC_JOB_RETURN_FORMAT_STRING = "cpc-job-queue-results:%s"
)
