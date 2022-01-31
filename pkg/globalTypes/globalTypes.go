package globalTypes

type RegionCode = string
type ItemID = uint
type ItemName = string
type ConnectedRealmID = uint
type RealmName = string
type CharacterProfession = string

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
	Sales   uint
	High    float64
	Low     float64
	Average float64
}

type ShoppingList struct {
	Quantity float64
	Id       ItemID
	Name     ItemName
	Cost     struct {
		Vendor float64
		Ah     OutputFormatPrice
	}
}

type OutputFormatShoppingList = map[uint][]ShoppingList

type OutpoutFormatRecipeOutput struct {
	Min   int
	Max   int
	Value int
}

type OutputFormatObject struct {
	Name     string
	Id       uint
	Required float64
	Recipes  []struct {
		Name    string
		Rank    uint
		Id      uint
		Output  OutpoutFormatRecipeOutput
		Ah      OutputFormatPrice
		High    float64
		Low     float64
		Average float64
		Parts   []OutputFormatObject
	}
	Ah           OutputFormatPrice
	Vendor       float64
	Bonus_prices []struct {
		Level uint
		Ah    OutputFormatPrice
	}
	Shopping_lists OutputFormatShoppingList
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
