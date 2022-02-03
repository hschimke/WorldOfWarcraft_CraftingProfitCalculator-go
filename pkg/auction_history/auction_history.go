package auction_history

import (
	"fmt"
	"time"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/blizzard_api_helpers"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes/BlizzardApi"
)

type ScanRealmsResult struct {
	RealmNames string                       `json:"realm_names,omitempty"`
	RealmId    globalTypes.ConnectedRealmID `json:"realm_id,omitempty"`
	Region     globalTypes.RegionCode       `json:"region,omitempty"`
}

type GetAllBonusesReturn struct {
	Bonuses []map[string]string `json:"bonuses,omitempty"`
	Item    BlizzardApi.Item    `json:"item,omitempty"`
}

type AuctionPriceSummaryRecord struct {
	Data     []SalesCountSummary `json:"data,omitempty"`
	MinValue float64             `json:"min_value,omitempty"`
	MaxValue float64             `json:"max_value,omitempty"`
	AvgValue float64             `json:"avg_value,omitempty"`
}

type SalesCountSummary struct {
	SalesAtPrice    float64 `json:"sales_at_price,omitempty"`
	QuantityAtPrice float64 `json:"quantity_at_price,omitempty"`
	Price           float64 `json:"price,omitempty"`
}

type AuctionSummaryData struct {
	Min      float64                              `json:"min,omitempty"`
	Max      float64                              `json:"max,omitempty"`
	Avg      float64                              `json:"avg,omitempty"`
	Latest   float64                              `json:"latest,omitempty"`
	PriceMap map[string]AuctionPriceSummaryRecord `json:"price_map,omitempty"`
	Archives []struct {
		Timestamp string              `json:"timestamp,omitempty"`
		Data      []SalesCountSummary `json:"data,omitempty"`
		MinValue  float64             `json:"min_value,omitempty"`
		MaxValue  float64             `json:"max_value,omitempty"`
		AvgValue  float64             `json:"avg_value,omitempty"`
	} `json:"archives,omitempty"`
}

// Injest all the realms in the scan list
func ScanRealms() {}

// Get all auctions filtering with parameters
func GetAuctions(item globalTypes.ItemSoftIdentity, realm globalTypes.ConnectedRealmSoftIentity, region globalTypes.RegionCode, bonuses []uint, start_dtm time.Time, end_dtm time.Time) (AuctionSummaryData, error) {
	return AuctionSummaryData{}, fmt.Errorf("GetAuctions not implemented")
}

// Return all bonuses availble for an item
func GetAllBonuses(item globalTypes.ItemSoftIdentity, region globalTypes.RegionCode) GetAllBonusesReturn {
	return GetAllBonusesReturn{}
}

// Archive auctions, in this implementation it generally just deletes old auctions
func ArchiveAuctions() {}

// Fill in fill_count items into the database
func FillNItems(fill_count uint) {}

// Fill in fillCount names into the database
func FillNNames(fillCount uint) {}

// Get a list of all scanned realms
func GetScanRealms() ScanRealmsResult {
	return ScanRealmsResult{}
}

// Get all the names available, filtering if availble
func GetAllNames(filter string) []string {
	return make([]string, 0)
}

//async function getSpotAuctionSummary(item: ItemSoftIdentity, realm: ConnectedRealmSoftIentity, region: RegionCode, bonuses: number[] | string[] | string): Promise<AuctionPriceSummaryRecord> {
// Get a current auction spot summary from the internet
func getSpotAuctionSummary(item globalTypes.ItemSoftIdentity, realm globalTypes.ConnectedRealmSoftIentity, region globalTypes.RegionCode, bonuses []uint) (AuctionPriceSummaryRecord, error) {
	var realm_get uint
	if realm.Id != 0 {
		realm_get = realm.Id
	} else if realm.Name != "" {
		var realmGetError error
		realm_get, realmGetError = blizzard_api_helpers.GetConnectedRealmId(realm.Name, region)
		if realmGetError != nil {
			return AuctionPriceSummaryRecord{}, fmt.Errorf("no realm found with %s", realm.Name)
		}
	} else {
		return AuctionPriceSummaryRecord{}, fmt.Errorf("realm %v could not be found", realm)
	}

	ah, _ := blizzard_api_helpers.GetAuctionHouse(realm_get, region)
	cpclog.Debugf(`Spot search for item: %s and realm %v and region %s, with bonuses %v`, item, realm, region, bonuses)

	var item_id uint
	if item.ItemId != 0 {
		item_id = item.ItemId
	} else if item.ItemName != "" {
		var it_err error
		item_id, it_err = blizzard_api_helpers.GetItemId(region, item.ItemName)
		if it_err != nil {
			return AuctionPriceSummaryRecord{}, fmt.Errorf("could not find item for %v", item)
		}
	} else {
		return AuctionPriceSummaryRecord{}, fmt.Errorf("could not find item for %v", item)
	}

	auction_set := make([]BlizzardApi.Auction, 0)
	for _, auction := range ah.Auctions {
		found_item, found_bonus := false, false
		if auction.Item.Id == item_id {
			found_item = true
			cpclog.Sillyf(`Found %d`, auction.Item.Id)
		}
		if len(bonuses) == 0 {
			if len(auction.Item.Bonus_lists) > 0 {
				found_bonus = true
				cpclog.Sillyf(`Found $%d to match null bonus list`, auction.Item.Id)
			}
		} else {
			found_bonus = check_bonus(bonuses, auction.Item.Bonus_lists)
			cpclog.Sillyf(`Array bonus list %v returned %t for %v`, bonuses, found_bonus, auction.Item.Bonus_lists)
		}

		if found_bonus && found_item {
			auction_set = append(auction_set, auction)
		}
	}

	cpclog.Debugf(`Found %d auctions`, len(auction_set))

	return_value := AuctionPriceSummaryRecord{}

	total_sales, total_price := 0, 0
	price_map := make(map[uint]struct {
		Quantity uint
		Sales    uint
	})

	for _, auction := range auction_set {
		var price uint
		quantity := auction.Quantity
		if auction.Buyout != 0 {
			price = auction.Buyout
		} else {
			price = auction.Unit_price
		}

		if return_value.MaxValue < price {
			return_value.MaxValue = price
		}
		if return_value.MinValue > price {
			return_value.MinValue = price
		}
		total_sales += quantity
		total_price += price * quantity

		if _, found := price_map[price]; found {
			price_map[price] = struct {
				Quantity uint
				Sales    uint
			}{}
		}
		price_map[price].Quantity += quantity
		price_map[price].Sales += 1
	}

	return_value.AvgValue = total_price / total_sales
	for price, price_lu := range price_map {
		//const p_lookup = Number(price);
		return_value.Data = append(return_value.Data, SalesCountSummary{
			Price:           price,
			QuantityAtPrice: price_lu.Quantity,
			SalesAtPrice:    price_lu.Sales,
		})
	}

	return return_value, nil
}

func check_bonus(bonus_list []uint, target []uint) (found bool) {
	found = false
	/*
		 found := true;

		// Filter array
		const filtered : string[] | number[] = (bonus_list as any[]).filter(n=>n);
		const numbers = filtered.map(element => Number(element));
		const numbers_only = numbers.filter((number) => {
			return Number.isInteger(number);
		})

		// Take care of undefined targets
		if( target === undefined){
			if(numbers_only.length !== 0){
				return false;
			}
			return true;
		}

		for( const list_entry of numbers_only ){
			found = found && target.includes(list_entry);
		}

		return found;*/
	return
}

// Injest a realm for auction archives
func ingest(region globalTypes.RegionCode, connected_realm globalTypes.ConnectedRealmID) {}
