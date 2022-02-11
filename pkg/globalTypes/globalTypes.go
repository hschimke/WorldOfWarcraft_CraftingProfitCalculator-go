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
	Sales   uint    `json:"sales,omitempty"`
	High    float64 `json:"high,omitempty"`
	Low     float64 `json:"low,omitempty"`
	Average float64 `json:"average,omitempty"`
}

type ShoppingListCost struct {
	Vendor float64           `json:"vendor,omitempty"`
	Ah     OutputFormatPrice `json:"ah,omitempty"`
}

type ShoppingList struct {
	Quantity float64          `json:"quantity,omitempty"`
	Id       ItemID           `json:"id,omitempty"`
	Name     ItemName         `json:"name,omitempty"`
	Cost     ShoppingListCost `json:"cost,omitempty"`
}

type OutputFormatShoppingList = map[uint][]ShoppingList

type OutpoutFormatRecipeOutput struct {
	Min   int `json:"min,omitempty"`
	Max   int `json:"max,omitempty"`
	Value int `json:"value,omitempty"`
}

type OutputFormatRecipe struct {
	Name    string                    `json:"name,omitempty"`
	Rank    uint                      `json:"rank,omitempty"`
	Id      uint                      `json:"id,omitempty"`
	Output  OutpoutFormatRecipeOutput `json:"output,omitempty"`
	Ah      OutputFormatPrice         `json:"ah,omitempty"`
	High    float64                   `json:"high,omitempty"`
	Low     float64                   `json:"low,omitempty"`
	Average float64                   `json:"average,omitempty"`
	Parts   []OutputFormatObject      `json:"parts,omitempty"`
}

type OutputFormatBonusPrices struct {
	Level uint              `json:"level,omitempty"`
	Ah    OutputFormatPrice `json:"ah,omitempty"`
}

type OutputFormatObject struct {
	Name           string                    `json:"name,omitempty"`
	Id             uint                      `json:"id,omitempty"`
	Required       float64                   `json:"required,omitempty"`
	Recipes        []OutputFormatRecipe      `json:"recipes,omitempty"`
	Ah             OutputFormatPrice         `json:"ah,omitempty"`
	Vendor         float64                   `json:"vendor,omitempty"`
	Bonus_prices   []OutputFormatBonusPrices `json:"bonus_prices,omitempty"`
	Shopping_lists OutputFormatShoppingList  `json:"shopping_lists,omitempty"`
}

type AHItemPriceObject struct {
	Total_sales uint
	Average     float64
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
	Price        ProfitAnalysisObject
	Intermediate OutputFormatObject
	Formatted    string
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
		Item      ItemSoftIdentity
		Count     uint
		AddonData AddonData
	}
}

type ReturnError struct {
	ERROR string
}

var ALL_PROFESSIONS []CharacterProfession = []CharacterProfession{"Blacksmithing", "Leatherworking", "Alchemy", "Herbalism", "Cooking", "Mining", "Tailoring", "Engineering", "Enchanting", "Fishing", "Skinning", "Jewelcrafting", "Inscription", "Archaeology", "Soul Cyphering", "Abominable Stitching", "Ascension Crafting", "Stygia Crafting"}

const (
	CPC_JOB_QUEUE_NAME           = "cpc-job-queue:web-jobs"
	CPC_JOB_RETURN_FORMAT_STRING = "cpc-job-queue-results:%s"
)
