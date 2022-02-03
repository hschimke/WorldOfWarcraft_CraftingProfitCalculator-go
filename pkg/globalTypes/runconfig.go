package globalTypes

type AddonData struct {
	Inventory []struct {
		Id       ItemID `json:"id,omitempty"`
		Quantity uint   `json:"quantity,omitempty"`
	} `json:"inventory,omitempty"`
	Professions []CharacterProfession `json:"professions,omitempty"`
	Realm       struct {
		Region_id   uint             `json:"region_id,omitempty"`
		Region_name string           `json:"region_name,omitempty"`
		Realm_id    ConnectedRealmID `json:"realm_id,omitempty"`
		Realm_name  RealmName        `json:"realm_name,omitempty"`
	} `json:"realm,omitempty"`
}

type RunConfiguration struct {
	internal_inventory map[ItemID]uint       `json:"internal_inventory,omitempty"`
	inventory_overlay  map[ItemID]int        `json:"inventory_overlay,omitempty"`
	Professions        []CharacterProfession `json:"professions,omitempty"`
	Realm_name         RealmName             `json:"realm_name,omitempty"`
	Realm_region       RegionCode            `json:"realm_region,omitempty"`
	Item               ItemSoftIdentity      `json:"item,omitempty"`
	Item_count         uint                  `json:"item_count,omitempty"`
}

func NewRunConfig(raw_configuration_data *AddonData, item ItemSoftIdentity, count uint) (new_conf *RunConfiguration) {
	new_conf = &RunConfiguration{}
	if raw_configuration_data != nil {
		for _, item := range raw_configuration_data.Inventory {
			new_conf.internal_inventory[item.Id] = item.Quantity
		}
		for _, prof := range raw_configuration_data.Professions {
			new_conf.Professions = append(new_conf.Professions, prof)
		}
		new_conf.Realm_name = raw_configuration_data.Realm.Realm_name
		new_conf.Realm_region = raw_configuration_data.Realm.Region_name
	}
	new_conf.Item = item
	new_conf.Item_count = count
	return
}

func (rc *RunConfiguration) ItemIsInInventory(item_id ItemID) bool {
	_, present := rc.internal_inventory[item_id]
	return present
}

func (rc *RunConfiguration) ItemCount(item_id ItemID) uint {
	available := 0
	if rc.ItemIsInInventory(item_id) {
		available += int(rc.internal_inventory[item_id])
	}
	if delta, has_overlay := rc.inventory_overlay[item_id]; has_overlay {
		available += delta
	}

	return uint(available)
}

func (rc *RunConfiguration) AdjustInventory(item_id ItemID, adjustment_delta int) {
	/*
		if (!(item_id in this.#inventory_overlay)) {
			this.#inventory_overlay[item_id] = 0;
		}*/
	rc.inventory_overlay[item_id] += adjustment_delta
}

func (rc *RunConfiguration) ResetInventoryAdjustments() {
	rc.inventory_overlay = make(map[uint]int)
}
