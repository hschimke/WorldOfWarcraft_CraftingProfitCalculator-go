package BlizzardApi

import (
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
)

type ItemSearch struct {
	PageCount         uint `json:"pageCount,omitempty"`
	Page              uint `json:"page,omitempty"`
	PageSize          uint `json:"pageSize,omitempty"`
	MaxPageSize       uint `json:"maxPageSize,omitempty"`
	ResultCountCapped bool `json:"resultCountCapped,omitempty"`
	Results           []struct {
		Data struct {
			Name map[string]string  `json:"name,omitempty"`
			Id   globalTypes.ItemID `json:"id,omitempty"`
		} `json:"data"`
	} `json:"results,omitempty"`
}

type ConnectedRealmIndex struct {
	Connected_realms []struct {
		Href string `json:"href,omitempty"`
	} `json:"connected_realms,omitempty"`
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
	Purchase_price    uint                 `json:"purchase_price,omitempty"`
	Purchase_quantity uint                 `json:"purchase_quantity,omitempty"`
	Level             uint                 `json:"level,omitempty"`
	Item_class        struct {
		Name string `json:"name,omitempty"`
		Id   int    `json:"id,omitempty"`
	} `json:"item_class,omitempty"`
	Item_subclass struct {
		Name string `json:"name,omitempty"`
		Id   int    `json:"id,omitempty"`
	} `json:"item_subclass,omitempty"`
	Quality struct {
		Type string `json:"type,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"quality,omitempty"`
	Preview_item struct {
		Context int `json:"context,omitempty"`
	} `json:"preview_item,omitempty"`
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
	} `json:"skill_tiers,omitempty"`
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
	} `json:"alliance_crafted_item,omitempty"`
	Horde_crafted_item *struct {
		Id uint `json:"id,omitempty"`
	} `json:"horde_crafted_item,omitempty"`
	Crafted_item *struct {
		Id uint `json:"id,omitempty"`
	} `json:"crafted_item,omitempty"`
	Reagents []struct {
		Reagent struct {
			Id uint `json:"id,omitempty"`
		} `json:"reagent"`
		Quantity uint `json:"quantity,omitempty"`
	} `json:"reagents,omitempty"`
	Crafted_quantity struct {
		Minimum float64 `json:"minimum,omitempty"`
		Maximum float64 `json:"maximum,omitempty"`
		Value   float64 `json:"value,omitempty"`
	} `json:"crafted_quantity"`
	Modified_crafting_slots []struct {
		Slot_type struct {
			Name string `json:"name,omitempty"`
			Id   int    `json:"id,omitempty"`
		} `json:"slot_type"`
		Display_order int `json:"display_order"`
	} `json:"modified_crafting_slots,omitempty"`
}

type Auction struct {
	Id   uint64 `json:"id,omitempty"`
	Item struct {
		Id          globalTypes.ItemID `json:"id,omitempty"`
		Context     int                `json:"context,omitempty"`
		Bonus_lists []uint             `json:"bonus_lists,omitempty"`
		Modifiers   []struct {
			Type  int `json:"type,omitempty"`
			Value int `json:"value,omitempty"`
		} `json:"modifiers,omitempty"`
	} `json:"item"`
	Quantity   uint `json:"quantity,omitempty"`
	Buyout     uint `json:"buyout,omitempty"`
	Unit_price uint `json:"unit_price,omitempty"`
	Bid        uint `json:"bid,omitempty"`
	Time_left  string `json:"time_left,omitempty"`
}

type Auctions struct {
	Auctions []Auction `json:"auctions,omitempty"`
}

type Media struct {
	Assets []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"assets"`
}

type BlizzardApiReponse interface {
	Auctions | Recipe | ProfessionSkillTier | Profession | ProfessionsIndex | Item | ConnectedRealm | ConnectedRealmIndex | ItemSearch | Media
}
