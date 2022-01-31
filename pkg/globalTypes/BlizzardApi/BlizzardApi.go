package BlizzardApi

import "github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"

type ItemSearch struct {
	PageCount uint `json:"page_count,omitempty"`
	Page      uint `json:"page,omitempty"`
	Results   []struct {
		Data struct {
			Name map[string]string  `json:"name,omitempty"`
			Id   globalTypes.ItemID `json:"id,omitempty"`
		} `json:"data,omitempty"`
	} `json:"results,omitempty"`
}

type ConnectedRealmIndex struct {
	Connected_realms []struct {
		Href string `json:"href,omitempty"`
	} `json:"connected___realms,omitempty"`
}

type ConnectedRealm struct {
	Id     globalTypes.ConnectedRealmID `json:"id,omitempty"`
	Realms []struct {
		Name string `json:"name,omitempty"`
	} `json:"realms,omitempty"`
}

type Item struct {
	Id                globalTypes.ItemID   `json:"id,omitempty"`
	Name              globalTypes.ItemName `json:"name,omitempty"`
	Description       string               `json:"description,omitempty"`
	Purchase_price    uint                 `json:"purchase___price,omitempty"`
	Purchase_quantity uint                 `json:"purchase___quantity,omitempty"`
	Level             uint                 `json:"level,omitempty"`
}

type ProfessionsIndex struct {
	Professions []struct {
		Name string `json:"name,omitempty"`
		Id   uint   `json:"id,omitempty"`
	} `json:"professions,omitempty"`
}

type Profession struct {
	Skill_tiers []struct {
		Name string `json:"name,omitempty"`
		Id   uint   `json:"id,omitempty"`
	} `json:"skill___tiers,omitempty"`
	Name string `json:"name,omitempty"`
	Id   uint   `json:"id,omitempty"`
}

type Category struct {
	Recipes []struct {
		Id   uint   `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"recipes,omitempty"`
	Name string `json:"name,omitempty"`
}

type ProfessionSkillTier struct {
	Categories []Category `json:"categories,omitempty"`
}

type Recipe struct {
	Id                    uint   `json:"id,omitempty"`
	Name                  string `json:"name,omitempty"`
	Alliance_crafted_item *struct {
		Id uint `json:"id,omitempty"`
	} `json:"alliance___crafted___item,omitempty"`
	Horde_crafted_item *struct {
		Id uint `json:"id,omitempty"`
	} `json:"horde___crafted___item,omitempty"`
	Crafted_item *struct {
		Id uint `json:"id,omitempty"`
	} `json:"crafted___item,omitempty"`
	Reagents []struct {
		Reagent struct {
			Id uint `json:"id,omitempty"`
		} `json:"reagent,omitempty"`
		Quantity uint `json:"quantity,omitempty"`
	} `json:"reagents,omitempty"`
	Crafted_quantity struct {
		Minimum uint `json:"minimum,omitempty"`
		Maximum uint `json:"maximum,omitempty"`
		Value   uint `json:"value,omitempty"`
	} `json:"crafted___quantity,omitempty"`
}

type Auctions struct {
	Auctions []struct {
		Item struct {
			Id          globalTypes.ItemID `json:"id,omitempty"`
			Bonus_lists []uint             `json:"bonus___lists,omitempty"`
		} `json:"item,omitempty"`
		Quantity   uint `json:"quantity,omitempty"`
		Buyout     uint `json:"buyout,omitempty"`
		Unit_price uint `json:"unit___price,omitempty"`
	} `json:"auctions,omitempty"`
}

type BlizzardApiReponse interface{}
