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
	High    uint
	Low     uint
	Average float64
}

type ShoppingList struct {
	Quantity uint
	Id       ItemID
	Name     ItemName
	Cost     struct {
		Vendor uint
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
	Required uint
	Recipes  []struct {
		Name    string
		Rank    uint
		Id      uint
		Output  OutpoutFormatRecipeOutput
		Ah      OutputFormatPrice
		High    uint
		Low     uint
		Average float64
		Parts   []OutputFormatObject
	}
	Ah           OutputFormatPrice
	Vendor       uint
	Bonus_prices []struct {
		Level uint
		Ah    OutputFormatPrice
	}
	Shopping_lists OutputFormatShoppingList
}

type AHItemPriceObject struct {
	Total_sales uint
	Average     float64
	High        uint
	Low         uint
}

type RecipeOption struct {
	Prices []ProfitAnalysisObject
	Recipe struct {
		Recipe_id uint
	}
	Rank    uint
	Rank_ah AHItemPriceObject
}

type ProfitAnalysisObject struct {
	Item_id         uint
	Item_name       string
	Ah_price        AHItemPriceObject
	Item_quantity   uint
	Vendor_price    uint
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
