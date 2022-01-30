package BlizzardApi

import "github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"

type ItemSearch struct {
	PageCount uint
	Page      uint
	Results   []struct {
		Data struct {
			Name map[string]string
			Id   globalTypes.ItemID
		}
	}
}

type ConnectedRealmIndex struct {
	Connected_realms []struct {
		Href string
	}
}

type ConnectedRealm struct {
	Id     globalTypes.ConnectedRealmID
	Realms []struct {
		Name string
	}
}

type Item struct {
	Id                globalTypes.ItemID
	Name              globalTypes.ItemName
	Description       string
	Purchase_price    uint
	Purchase_quantity uint
	Level             uint
}

type ProfessionsIndex struct {
	Professions []struct {
		Name string
		Id   uint
	}
}

type Profession struct {
	Skill_tiers []struct {
		Name string
		Id   uint
	}
	Name string
	Id   uint
}

type Category struct {
	Recipes []struct {
		Id   uint
		Name string
	}
	Name string
}

type ProfessionSkillTier struct {
	Categories []Category
}

type Recipe struct {
	Id                    uint
	Name                  string
	Alliance_crafted_item *struct {
		Id uint
	}
	Horde_crafted_item *struct {
		Id uint
	}
	Crafted_item *struct {
		Id uint
	}
	Reagents []struct {
		Reagent struct {
			Id uint
		}
		Quantity uint
	}
	Crafted_quantity struct {
		Minimum uint
		Maximum uint
		Value   uint
	}
}

type Auctions struct {
	Auctions []struct {
		Item struct {
			Id          globalTypes.ItemID
			Bonus_lists []uint
		}
		Quantity   uint
		Buyout     uint
		Unit_price uint
	}
}

type BlizzardApiReponse interface{}
