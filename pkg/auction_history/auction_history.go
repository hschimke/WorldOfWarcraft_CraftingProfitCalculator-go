package auction_history

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/environment_variables"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/blizzard_api_helpers"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes/BlizzardApi"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ScanRealmsResult struct {
	RealmNames string                       `json:"realm_names,omitempty"`
	RealmId    globalTypes.ConnectedRealmID `json:"realm_id,omitempty"`
	Region     globalTypes.RegionCode       `json:"region,omitempty"`
}

type GetAllBonusesReturn struct {
	Bonuses [][]uint `json:"bonuses,omitempty"`
	//Bonuses []map[string]string `json:"bonuses,omitempty"`
	Item BlizzardApi.Item `json:"item,omitempty"`
}

type AuctionPriceSummaryRecord struct {
	Data     []SalesCountSummary `json:"data,omitempty"`
	MinValue uint                `json:"min_value,omitempty"`
	MaxValue uint                `json:"max_value,omitempty"`
	AvgValue float64             `json:"avg_value,omitempty"`
}

type SalesCountSummary struct {
	SalesAtPrice    uint `json:"sales_at_price,omitempty"`
	QuantityAtPrice uint `json:"quantity_at_price,omitempty"`
	Price           uint `json:"price,omitempty"`
}

type AuctionSummaryData struct {
	Min      uint                                 `json:"min,omitempty"`
	Max      uint                                 `json:"max,omitempty"`
	Avg      float64                              `json:"avg,omitempty"`
	Latest   uint                                 `json:"latest,omitempty"`
	PriceMap map[string]AuctionPriceSummaryRecord `json:"price_map,omitempty"`
	Archives []struct {
		Timestamp string              `json:"timestamp,omitempty"`
		Data      []SalesCountSummary `json:"data,omitempty"`
		MinValue  uint                `json:"min_value,omitempty"`
		MaxValue  uint                `json:"max_value,omitempty"`
		AvgValue  float64             `json:"avg_value,omitempty"`
	} `json:"archives,omitempty"`
}

type scanRealm struct {
	Names            []globalTypes.RealmName      `bson:"names"`
	ConnectedRealmId globalTypes.ConnectedRealmID `bson:"connected_realm_id"`
	Region           globalTypes.RegionCode       `bson:"region"`
}

type localItem struct {
	ItemName  string                 `bson:"item_name"`
	ItemId    uint                   `bson:"item_id"`
	Region    globalTypes.RegionCode `bson:"region"`
	Craftable bool                   `bson:"craftable,omitempty"`
}

// Injest all the realms in the scan list
func ScanRealms() {
	// Connect to mongo
	mongoClient, clientError := mongo.Connect(context.TODO(), options.Client().ApplyURI(environment_variables.DATABASE_CONNECTION_STRING))
	if clientError != nil {
		//panic(clientError)
		panic(clientError)
	}
	defer func() {
		if clientError = mongoClient.Disconnect(context.TODO()); clientError != nil {
			panic(clientError)
		}
	}()

	// Auctions collection
	scanRealmsCollection := mongoClient.Database("cpc").Collection("scan_realms")
	realmsToScan, scanErr := scanRealmsCollection.Find(context.TODO(), nil)
	if scanErr != nil {
		panic(scanErr)
	}

	var realms []scanRealm
	realmsToScan.Decode(&realms)

	for _, realm := range realms {
		err := ingest(realm.Region, realm.ConnectedRealmId)
		if err != nil {
			continue
		}
	}
}

func AddScanRealm(realm globalTypes.ConnectedRealmSoftIentity, region globalTypes.RegionCode) {
	// Connect to mongo
	mongoClient, clientError := mongo.Connect(context.TODO(), options.Client().ApplyURI(environment_variables.DATABASE_CONNECTION_STRING))
	if clientError != nil {
		//panic(clientError)
		panic(clientError)
	}
	defer func() {
		if clientError = mongoClient.Disconnect(context.TODO()); clientError != nil {
			panic(clientError)
		}
	}()

	// Auctions collection
	scanRealmsCollection := mongoClient.Database("cpc").Collection("scan_realms")

	var (
		newRealmId         uint
		realmNameComposite []string
	)

	// Id passed in is cononical, if name is passed in get ID from that, otherwise panic
	if realm.Id != 0 {
		newRealmId = realm.Id
	} else if realm.Name != "" {
		fetchRealmId, fetchRealmIdErr := blizzard_api_helpers.GetConnectedRealmId(realm.Name, region)
		if fetchRealmIdErr != nil {
			panic("could not get realm")
		}
		newRealmId = fetchRealmId
	} else {
		panic("no realm")
	}

	fetchRealm, fetchRealmErr := blizzard_api_helpers.GetBlizConnectedRealmDetail(newRealmId, region)
	if fetchRealmErr != nil {
		panic("could not get realm")
	}

	for _, server := range fetchRealm.Realms {
		realmNameComposite = append(realmNameComposite, server.Name)
	}

	searchFilter := bson.D{
		{"$and",
			bson.A{
				bson.D{{"connected_realm_id", newRealmId}},
				bson.D{{"region", region}},
			}},
	}

	newRealm := scanRealm{
		Names:            realmNameComposite,
		ConnectedRealmId: newRealmId,
		Region:           region,
	}

	update := bson.D{{"$setOnInsert", newRealm}}

	scanRealmsCollection.UpdateOne(context.TODO(), searchFilter, update, options.Update().SetUpsert(true))
}

func RemoveScanRealm(realm globalTypes.ConnectedRealmSoftIentity, region globalTypes.RegionCode) {
	// Connect to mongo
	mongoClient, clientError := mongo.Connect(context.TODO(), options.Client().ApplyURI(environment_variables.DATABASE_CONNECTION_STRING))
	if clientError != nil {
		//panic(clientError)
		panic(clientError)
	}
	defer func() {
		if clientError = mongoClient.Disconnect(context.TODO()); clientError != nil {
			panic(clientError)
		}
	}()

	// Auctions collection
	scanRealmsCollection := mongoClient.Database("cpc").Collection("scan_realms")

	var (
		newRealmId uint
	)

	// Id passed in is cononical, if name is passed in get ID from that, otherwise panic
	if realm.Id != 0 {
		newRealmId = realm.Id
	} else if realm.Name != "" {
		fetchRealmId, fetchRealmIdErr := blizzard_api_helpers.GetConnectedRealmId(realm.Name, region)
		if fetchRealmIdErr != nil {
			panic("could not get realm")
		}
		newRealmId = fetchRealmId
	} else {
		panic("no realm")
	}

	searchFilter := bson.D{
		{"$and",
			bson.A{
				bson.D{{"connected_realm_id", newRealmId}},
				bson.D{{"region", region}},
			}},
	}

	//scanRealmsCollection.UpdateOne(context.TODO(), searchFilter, update, options.Update().SetUpsert(true))
	scanRealmsCollection.DeleteOne(context.TODO(), searchFilter)
}

// Get all auctions filtering with parameters
func GetAuctions(item globalTypes.ItemSoftIdentity, realm globalTypes.ConnectedRealmSoftIentity, region globalTypes.RegionCode, bonuses []uint, start_dtm time.Time, end_dtm time.Time) (AuctionSummaryData, error) {

}

// Return all bonuses availble for an item
func GetAllBonuses(item globalTypes.ItemSoftIdentity, region globalTypes.RegionCode) GetAllBonusesReturn {
	// Connect to mongo
	mongoClient, clientError := mongo.Connect(context.TODO(), options.Client().ApplyURI(environment_variables.DATABASE_CONNECTION_STRING))
	if clientError != nil {
		//panic(clientError)
		panic(clientError)
	}
	defer func() {
		if clientError = mongoClient.Disconnect(context.TODO()); clientError != nil {
			panic(clientError)
		}
	}()

	// Auctions collection
	auctionsCollection := mongoClient.Database("cpc").Collection("auctions")

	var searchId uint
	if item.ItemId != 0 {
		searchId = item.ItemId
	} else if item.ItemName != "" {
		itemId, idErr := blizzard_api_helpers.GetItemId(region, item.ItemName)
		if idErr != nil {
			return GetAllBonusesReturn{}
		}
		searchId = itemId
	} else {
		return GetAllBonusesReturn{}
	}

	auctionsFilter := bson.D{
		{"$and",
			bson.A{
				bson.D{{"item.id", searchId}},
				bson.D{{"region", region}},
				bson.D{{"$exists", bson.D{{"item.bonus_lists", true}}}},
			},
		}}

	results, err := auctionsCollection.Distinct(context.TODO(), "item.bonus_lists", auctionsFilter)
	if err != nil {
		return GetAllBonusesReturn{}
	}

	var return_value GetAllBonusesReturn

	return_value.Item.Id = searchId
	return_value.Item.Name = item.ItemName

	for _, auction := range results {
		return_value.Bonuses = append(return_value.Bonuses, auction.([]uint))
	}

	return return_value
}

// Archive auctions, in this implementation it generally just deletes old auctions
func ArchiveAuctions() {
	// Connect to mongo
	mongoClient, clientError := mongo.Connect(context.TODO(), options.Client().ApplyURI(environment_variables.DATABASE_CONNECTION_STRING))
	if clientError != nil {
		//panic(clientError)
		panic(clientError)
	}
	defer func() {
		if clientError = mongoClient.Disconnect(context.TODO()); clientError != nil {
			panic(clientError)
		}
	}()

	// Auctions collection
	auctionsCollection := mongoClient.Database("cpc").Collection("auctions")

	twoWeeksAgo := time.Now().Add(time.Hour * (-1 * 24) * 14)

	deleteFilter := bson.D{{"fetched", bson.D{{"$lt", twoWeeksAgo}}}}

	auctionsCollection.DeleteMany(context.TODO(), deleteFilter)
}

// Fill in fill_count items into the database
func FillNItems(fillCount uint) {
	// Connect to mongo
	mongoClient, clientError := mongo.Connect(context.TODO(), options.Client().ApplyURI(environment_variables.DATABASE_CONNECTION_STRING))
	if clientError != nil {
		//panic(clientError)
		panic(clientError)
	}
	defer func() {
		if clientError = mongoClient.Disconnect(context.TODO()); clientError != nil {
			panic(clientError)
		}
	}()

	// Items collection
	itemsCollection := mongoClient.Database("cpc").Collection("items")

	filterNotScanned := bson.D{{"$exists", bson.D{{"craftable", false}}}}

	items, err := itemsCollection.Find(context.TODO(), filterNotScanned, options.Find().SetLimit(int64(fillCount)))
	if err != nil {
		panic(err)
	}

	for items.Next(context.TODO()) {
		var updateItem localItem
		if err := items.Decode(&updateItem); err != nil {
			crafting, craftCalcError := blizzard_api_helpers.CheckIsCrafting(updateItem.ItemId, globalTypes.ALL_PROFESSIONS, updateItem.Region)
			if craftCalcError != nil {
				continue
			}
			itemFilter := bson.D{
				{"$and",
					bson.A{
						bson.D{{"item_id", updateItem.ItemId}},
						bson.D{{"region", updateItem.Region}},
					},
				},
			}

			itemUpdate := bson.D{{"craftable", crafting.Craftable}}

			itemsCollection.UpdateOne(context.TODO(), itemFilter, itemUpdate)
		}
	}
}

// Fill in fillCount names into the database
func FillNNames(fillCount uint) {
	// Connect to mongo
	mongoClient, clientError := mongo.Connect(context.TODO(), options.Client().ApplyURI(environment_variables.DATABASE_CONNECTION_STRING))
	if clientError != nil {
		//panic(clientError)
		panic(clientError)
	}
	defer func() {
		if clientError = mongoClient.Disconnect(context.TODO()); clientError != nil {
			panic(clientError)
		}
	}()

	// Items collection
	itemsCollection := mongoClient.Database("cpc").Collection("items")

	filterNoName := bson.D{
		{"$or",
			bson.A{
				bson.D{{"item_name", bson.D{{"$exists", false}}}},
				bson.D{{"item_name", bson.D{{"$eq", ""}}}},
			},
		},
	}

	items, err := itemsCollection.Find(context.TODO(), filterNoName, options.Find().SetLimit(int64(fillCount)))
	if err != nil {
		panic(err)
	}

	for items.Next(context.TODO()) {
		var updateItem localItem
		if err := items.Decode(&updateItem); err != nil {
			itemDetail, itemFetchErr := blizzard_api_helpers.GetItemDetails(updateItem.ItemId, updateItem.Region)
			if itemFetchErr != nil {
				continue
			}
			itemFilter := bson.D{
				{"$and",
					bson.A{
						bson.D{{"item_id", updateItem.ItemId}},
						bson.D{{"region", updateItem.Region}},
					},
				},
			}

			itemUpdate := bson.D{{"item_name", itemDetail.Name}}

			itemsCollection.UpdateOne(context.TODO(), itemFilter, itemUpdate)
		}
	}
}

// Get a list of all scanned realms
func GetScanRealms() []ScanRealmsResult {
	// Connect to mongo
	mongoClient, clientError := mongo.Connect(context.TODO(), options.Client().ApplyURI(environment_variables.DATABASE_CONNECTION_STRING))
	if clientError != nil {
		//panic(clientError)
		panic(clientError)
	}
	defer func() {
		if clientError = mongoClient.Disconnect(context.TODO()); clientError != nil {
			panic(clientError)
		}
	}()

	// Auctions collection
	scanRealmsCollection := mongoClient.Database("cpc").Collection("scan_realms")
	realmsToScan, scanErr := scanRealmsCollection.Find(context.TODO(), nil)
	if scanErr != nil {
		panic(scanErr)
	}

	var realms []scanRealm
	realmsToScan.Decode(&realms)

	var result []ScanRealmsResult
	for _, realm := range realms {
		result = append(result, ScanRealmsResult{
			RealmNames: strings.Join(realm.Names, ","),
			RealmId:    realm.ConnectedRealmId,
			Region:     realm.Region,
		})
	}
	return result

}

// Get all the names available, filtering if availble
func GetAllNames() []string {
	// Connect to mongo
	mongoClient, clientError := mongo.Connect(context.TODO(), options.Client().ApplyURI(environment_variables.DATABASE_CONNECTION_STRING))
	if clientError != nil {
		//panic(clientError)
		return []string{}
	}
	defer func() {
		if clientError = mongoClient.Disconnect(context.TODO()); clientError != nil {
			panic(clientError)
		}
	}()

	// Items collection
	itemsCollection := mongoClient.Database("cpc").Collection("items")

	filter_name_exists := bson.D{
		{"$or",
			bson.A{
				bson.D{{"item_name", bson.D{{"$exists", true}}}},
				bson.D{{"item_name", bson.D{{"$ne", ""}}}},
			},
		},
	}

	results, err := itemsCollection.Distinct(context.TODO(), "item_name", filter_name_exists)
	if err != nil {
		return []string{}
	}

	var return_value []string
	for _, name := range results {
		return_value = append(return_value, name.(string))
	}
	return return_value
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

	//total_sales, total_price := 0, 0
	var total_price, total_sales uint
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

		if _, found := price_map[price]; !found {
			price_map[price] = struct {
				Quantity uint
				Sales    uint
			}{}
		}
		pmh := price_map[price]
		pmh.Quantity += quantity
		pmh.Sales += 1
		price_map[price] = pmh
	}

	return_value.AvgValue = float64(total_price) / float64(total_sales)
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

func arrayIncludes(array []uint, search uint) bool {
	for _, num := range array {
		if num == search {
			return true
		}
	}
	return false
}

func check_bonus(bonus_list []uint, target []uint) (found bool) {
	found = true

	// Take care of undefined targets
	if len(target) == 0 {
		if len(bonus_list) != 0 {
			found = false
		}
		found = true
	}

	for _, list_entry := range bonus_list {
		found = found && arrayIncludes(target, list_entry)
	}

	return
}

// Injest a realm for auction archives
func ingest(region globalTypes.RegionCode, connected_realm globalTypes.ConnectedRealmID) error {
	// Connect to mongo
	mongoClient, clientError := mongo.Connect(context.TODO(), options.Client().ApplyURI(environment_variables.DATABASE_CONNECTION_STRING))
	if clientError != nil {
		//panic(clientError)
		return clientError
	}
	defer func() {
		if clientError = mongoClient.Disconnect(context.TODO()); clientError != nil {
			panic(clientError)
		}
	}()

	// Auctions collection
	auctionsCollection := mongoClient.Database("cpc").Collection("auctions")

	// Get Auctions
	auctions, auctionError := blizzard_api_helpers.GetAuctionHouse(connected_realm, region)
	if auctionError != nil {
		return auctionError
	}

	fetchTime := time.Now()
	auctionInsert := make([]interface{}, 0)
	var itemsToChurn []localItem
	for _, auction := range auctions.Auctions {
		auction.Fetched = fetchTime
		auction.Region = region
		auctionInsert = append(auctionInsert, auction)
		itemsToChurn = append(itemsToChurn, localItem{
			ItemId: auction.Item.Id,
			Region: region,
		})
	}

	go churnAuctionItemsOnInjest(itemsToChurn)

	_, insertErr := auctionsCollection.InsertMany(context.TODO(), auctionInsert)
	if insertErr != nil {
		return insertErr
	}

	return nil
}

func churnAuctionItemsOnInjest(items []localItem) {
	// Connect to mongo
	mongoClient, clientError := mongo.Connect(context.TODO(), options.Client().ApplyURI(environment_variables.DATABASE_CONNECTION_STRING))
	if clientError != nil {
		//panic(clientError)
		panic(clientError)
	}
	defer func() {
		if clientError = mongoClient.Disconnect(context.TODO()); clientError != nil {
			panic(clientError)
		}
	}()

	// Items collection
	itemsCollection := mongoClient.Database("cpc").Collection("items")

	// Churn Items
	for _, item := range items {
		// Upsert Item
		filter := bson.D{{"item_id", item.ItemId}}
		new_option, err := bson.Marshal(item)
		if err != nil {
			continue
		}
		update := bson.D{{"$setOnInsert", new_option}}
		itemsCollection.UpdateOne(context.TODO(), filter, update, options.Update().SetUpsert(true))
	}
}
