package auction_history

import (
	"context"
	"fmt"
	"math"
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
	Bonuses []uint `json:"bonuses,omitempty"`
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
	Min      uint                                `json:"min,omitempty"`
	Max      uint                                `json:"max,omitempty"`
	Avg      float64                             `json:"avg,omitempty"`
	Latest   int64                               `json:"latest,omitempty"`
	PriceMap map[int64]AuctionPriceSummaryRecord `json:"price_map,omitempty"`
	Archives []struct {
		Timestamp int64               `json:"timestamp,omitempty"`
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
	Craftable *bool                  `bson:"craftable,omitempty"`
}

// Injest all the realms in the scan list
func ScanRealms() error {
	// Connect to mongo
	mongoClient, clientError := mongo.Connect(context.TODO(), options.Client().ApplyURI(environment_variables.DATABASE_CONNECTION_STRING))
	if clientError != nil {
		//panic(clientError)
		return (clientError)
	}
	defer func() {
		if clientError = mongoClient.Disconnect(context.TODO()); clientError != nil {
			panic(clientError)
		}
	}()

	// Auctions collection
	scanRealmsCollection := mongoClient.Database("cpc").Collection("scan_realms")
	realmsToScan, scanErr := scanRealmsCollection.Find(context.TODO(), bson.D{{}})
	if scanErr != nil {
		return (scanErr)
	}

	var realms []scanRealm
	realmsToScan.All(context.TODO(), &realms)

	for _, realm := range realms {
		err := ingest(realm.Region, realm.ConnectedRealmId)
		if err != nil {
			return err
		}
	}
	return nil
}

func AddScanRealm(realm globalTypes.ConnectedRealmSoftIentity, region globalTypes.RegionCode) error {
	// Connect to mongo
	mongoClient, clientError := mongo.Connect(context.TODO(), options.Client().ApplyURI(environment_variables.DATABASE_CONNECTION_STRING))
	if clientError != nil {
		//panic(clientError)
		return (clientError)
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
			return fmt.Errorf("could not get realm %v", fetchRealmIdErr)
		}
		if fetchRealmId == 0 {
			return fmt.Errorf("could not get realm")
		}
		newRealmId = fetchRealmId
	} else {
		return fmt.Errorf("no realm")
	}

	fetchRealm, fetchRealmErr := blizzard_api_helpers.GetBlizConnectedRealmDetail(newRealmId, region)
	if fetchRealmErr != nil {
		return fmt.Errorf("could not get realm %v", fetchRealmErr)
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
	return nil
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

	var itemId, connectedRealmId uint

	// Get realm
	if realm.Id != 0 {
		connectedRealmId = realm.Id
	} else if realm.Name != "" {
		rlm, err := blizzard_api_helpers.GetConnectedRealmId(realm.Name, region)
		if err != nil {
			return AuctionSummaryData{}, err
		}
		connectedRealmId = rlm
	} else {
		return AuctionSummaryData{}, fmt.Errorf("no realm detectable for %v", realm)
	}

	// Get item
	if item.ItemId != 0 {
		itemId = item.ItemId
	} else if item.ItemName != "" {
		itm, err := blizzard_api_helpers.GetItemId(region, item.ItemName)
		if err != nil {
			return AuctionSummaryData{}, err
		}
		itemId = itm
	} else {
		return AuctionSummaryData{}, fmt.Errorf("no item detectable for %v", item)
	}

	var return_value AuctionSummaryData
	return_value.PriceMap = make(map[int64]AuctionPriceSummaryRecord)
	return_value.Min = math.MaxUint

	var filterBonuses bson.D

	filterId := bson.D{{"item.id", itemId}}
	if len(bonuses) > 0 {
		filterBonuses = bson.D{{"item.bonus_lists", bson.D{{"$all", bonuses}}}}
	} else {
		filterBonuses = bson.D{}
	}

	filterDates := bson.D{{"$and",
		bson.A{
			bson.D{{"fetched", bson.D{{"$lt", end_dtm}}}},
			bson.D{{"fetched", bson.D{{"$gt", start_dtm}}}},
		}}}
	filterConnectedRealm := bson.D{{"realm", connectedRealmId}}

	allFilters := bson.D{{"$and", bson.A{
		filterId,
		filterBonuses,
		filterDates,
		filterConnectedRealm,
	}}}

	allGroupsings := bson.D{
		{"_id", "$fetched"},
		{"total_sales", bson.D{{"$sum", "$quantity"}}},
		{"average", bson.D{{"$avg", bson.D{{"$sum", bson.A{"$unit_price", "$buyout"}}}}}},
		{"max", bson.D{{"$max", bson.D{{"$sum", bson.A{"$unit_price", "$buyout"}}}}}},
		{"min", bson.D{{"$min", bson.D{{"$sum", bson.A{"$unit_price", "$buyout"}}}}}},
	}

	//q := bson.M{}
	//jsonString, _ := json.Marshal(allFilters)
	//fmt.Printf("mgo query: %s\n", jsonString)

	// Get historical auction
	aggregationPipeline := bson.A{
		bson.D{{"$match", allFilters}},         // bonuses, item_id, dates
		bson.D{{"$group", allGroupsings}},      // group by id and date and calculate high,lo,avg,total sales
		bson.D{{"$sort", bson.D{{"_id", -1}}}}, // sort by id descending
	}

	aggregatedAuctions, err := auctionsCollection.Aggregate(context.TODO(), aggregationPipeline)
	if err != nil {
		return AuctionSummaryData{}, err
	}

	type aggregateAuctions struct {
		Id         time.Time `bson:"_id,omitempty"`
		TotalSales uint      `bson:"total_sales,omitempty"`
		Average    float64   `bson:"average,omitempty"`
		Min        uint      `bson:"min,omitempty"`
		Max        uint      `bson:"max,omitempty"`
	}

	for aggregatedAuctions.Next(context.TODO()) {
		var entry aggregateAuctions
		err := aggregatedAuctions.Decode(&entry)
		if err != nil {
			return AuctionSummaryData{}, err
		}
		return_value.PriceMap[entry.Id.Unix()] = AuctionPriceSummaryRecord{
			MinValue: entry.Min,
			MaxValue: entry.Max,
			AvgValue: entry.Average,
		}

		if entry.Min < return_value.Min {
			return_value.Min = entry.Min
		}
		if entry.Max > return_value.Max {
			return_value.Max = entry.Max
		}
	}

	// Get spot auctions
	spotSummary, err := getSpotAuctionSummary(item, realm, region, bonuses)
	if err != nil {
		return AuctionSummaryData{}, err
	}
	cTime := time.Now().Unix()
	return_value.PriceMap[cTime] = spotSummary
	return_value.Latest = cTime

	if spotSummary.MinValue < return_value.Min {
		return_value.Min = spotSummary.MinValue
	}
	if spotSummary.MaxValue > return_value.Max {
		return_value.Max = spotSummary.MaxValue
	}

	return return_value, nil
}

// Return all bonuses availble for an item
func GetAllBonuses(item globalTypes.ItemSoftIdentity, region globalTypes.RegionCode) (GetAllBonusesReturn, error) {
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
			return GetAllBonusesReturn{}, idErr
		}
		searchId = itemId
	} else {
		return GetAllBonusesReturn{}, fmt.Errorf("no item")
	}

	auctionsFilter := bson.D{
		{"$and",
			bson.A{
				bson.D{{"item.id", searchId}},
				bson.D{{"region", strings.ToLower(region)}},
				bson.D{{"item.bonus_lists", bson.D{{"$exists", true}}}},
				bson.D{{"item.bonus_lists", bson.D{{"$ne", bson.TypeNull}}}},
			},
		}}

	results, err := auctionsCollection.Distinct(context.TODO(), "item.bonus_lists", auctionsFilter)
	if err != nil {
		return GetAllBonusesReturn{}, err
	}

	var return_value GetAllBonusesReturn

	fetchedItem, err := blizzard_api_helpers.GetItemDetails(searchId, region)
	if err != nil {
		return GetAllBonusesReturn{}, err
	}

	return_value.Item.Id = searchId
	return_value.Item.Name = item.ItemName
	return_value.Item.Level = fetchedItem.Level

	for _, auction := range results {

		return_value.Bonuses = append(return_value.Bonuses, uint(auction.(int64)))
	}

	return return_value, nil
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

	filterNotScanned := bson.D{{"craftable", bson.D{{"$exists", false}}}}

	items, err := itemsCollection.Find(context.TODO(), filterNotScanned, options.Find().SetLimit(int64(fillCount)))
	if err != nil {
		panic(err)
	}

	for items.Next(context.TODO()) {
		var updateItem localItem
		if err := items.Decode(&updateItem); err == nil {
			crafting, craftCalcError := blizzard_api_helpers.CheckIsCrafting(updateItem.ItemId, globalTypes.ALL_PROFESSIONS, updateItem.Region)
			if craftCalcError != nil {
				panic(craftCalcError)
			}
			itemFilter := bson.D{
				{"$and",
					bson.A{
						bson.D{{"item_id", updateItem.ItemId}},
						bson.D{{"region", updateItem.Region}},
					},
				},
			}

			itemUpdate := bson.D{{"$set", bson.D{{"craftable", crafting.Craftable}}}}

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
		if err := items.Decode(&updateItem); err == nil {
			itemDetail, itemFetchErr := blizzard_api_helpers.GetItemDetails(updateItem.ItemId, updateItem.Region)
			if itemFetchErr != nil {
				panic(itemFetchErr)
			}
			itemFilter := bson.D{
				{"$and",
					bson.A{
						bson.D{{"item_id", updateItem.ItemId}},
						bson.D{{"region", updateItem.Region}},
					},
				},
			}

			itemUpdate := bson.D{{"$set", bson.D{{"item_name", itemDetail.Name}}}}

			itemsCollection.UpdateOne(context.TODO(), itemFilter, itemUpdate)
		}
	}
}

// Get a list of all scanned realms
func GetScanRealms() ([]ScanRealmsResult, error) {
	// Connect to mongo
	mongoClient, clientError := mongo.Connect(context.TODO(), options.Client().ApplyURI(environment_variables.DATABASE_CONNECTION_STRING))
	if clientError != nil {
		//panic(clientError)
		return []ScanRealmsResult{}, clientError
	}
	defer func() {
		if clientError = mongoClient.Disconnect(context.TODO()); clientError != nil {
			panic(clientError)
		}
	}()

	// Auctions collection
	scanRealmsCollection := mongoClient.Database("cpc").Collection("scan_realms")
	realmsToScan, scanErr := scanRealmsCollection.Find(context.TODO(), bson.D{{}})
	if scanErr != nil {
		return []ScanRealmsResult{}, scanErr
	}

	var realms []scanRealm
	err := realmsToScan.All(context.TODO(), &realms)
	if err != nil {
		return []ScanRealmsResult{}, err
	}

	var result []ScanRealmsResult
	for _, realm := range realms {
		result = append(result, ScanRealmsResult{
			RealmNames: strings.Join(realm.Names, ","),
			RealmId:    realm.ConnectedRealmId,
			Region:     realm.Region,
		})
	}
	return result, nil

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
		if len(name.(string)) > 0 {
			return_value = append(return_value, name.(string))
		}
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
	cpclog.Infof("start ingest for %v - %v", region, connected_realm)
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
		auction.ConnectedRealmId = connected_realm
		auctionInsert = append(auctionInsert, auction)
		itemsToChurn = append(itemsToChurn, localItem{
			ItemId: auction.Item.Id,
			Region: region,
		})
	}

	churnAuctionItemsOnInjest(itemsToChurn)

	_, insertErr := auctionsCollection.InsertMany(context.TODO(), auctionInsert)
	if insertErr != nil {
		return insertErr
	}

	cpclog.Infof("finished ingest for %v - %v", region, connected_realm)
	return nil
}

func churnAuctionItemsOnInjest(items []localItem) {
	cpclog.Infof("start item churn for %d items", len(items))
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
		//new_option, err := bson.Marshal(item)
		//if err != nil {
		//	continue
		//}
		update := bson.D{{"$setOnInsert", item}}
		_, updateErr := itemsCollection.UpdateOne(context.TODO(), filter, update, options.Update().SetUpsert(true))
		if updateErr != nil {
			panic(updateErr)
		}
	}
	cpclog.Info("finished item churn")
}

/*
async function ingest(region: RegionCode, connected_realm: ConnectedRealmID): Promise<void> {
        logger.info(`Injest job started for ${region}:${connected_realm}`);
        // Get auction house
        const auction_house = await getAuctionHouse(connected_realm, region);
        const downloaded = Date.now();
        // Loop over each auction and add it.
        const items: Record<string, Record<string | number, {
            item_id: ItemID,
            bonus_lists: Array<number>,
            price: number,
            quantity: number
        }>> = {};
        auction_house.auctions.forEach((auction) => {
            const item_id_key = auction.item.id + (('bonus_lists' in auction.item) ? JSON.stringify(auction.item.bonus_lists) : '');
            if (!(item_id_key in items)) {
                items[item_id_key] = {};
            }
            let price = 0;
            const quantity = auction.quantity;
            if (auction.buyout !== undefined) {
                price = auction.buyout;
            } else {
                price = auction.unit_price;
            }
            if (!(price in items[item_id_key])) {
                items[item_id_key][price] = {
                    item_id: auction.item.id,
                    bonus_lists: auction.item.bonus_lists,
                    price: price,
                    quantity: 0,
                };
            }
            items[item_id_key][price].quantity += quantity;
        });

        const item_set: Set<number> = new Set();
        const insert_values_array = [];

        for (const key of Object.keys(items)) {
            for (const pk of Object.keys(items[key])) {
                item_set.add(items[key][pk].item_id);
                //item_id, quantity, price, downloaded, connected_realm_id, bonuses
                insert_values_array.push([items[key][pk].item_id, items[key][pk].quantity, items[key][pk].price, downloaded, connected_realm, JSON.stringify(items[key][pk].bonus_lists), region.toLocaleLowerCase()]);
            }
        }

        const client = await db.getClient();

        type HowMany = { how_many: number };

        await client.query('BEGIN TRANSACTION');

        for (const item of item_set) {
            try {
                await client.query(sql_insert_item, [item, region, null, false, false]);
            } catch (err) {
                logger.error(`Could not save ${item} in region ${region}.`, err);
            }
        }

        await Promise.all(insert_values_array.map((values) => {
            return client.query(sql_insert_auction, values);
        }));

        await client.query('COMMIT TRANSACTION');
        await client.release();

        logger.info(`Injest job finished for ${region}:${connected_realm}`);
    }

    async function getAllBonuses(item: ItemSoftIdentity, region: RegionCode): Promise<GetAllBonusesReturn> {
        logger.debug(`Fetching bonuses for ${item}`);
        const sql = 'SELECT DISTINCT bonuses FROM auctions WHERE item_id = $1';

        let item_id = 0;
        if (typeof item === 'number') {
            item_id = item;
        } else if (Number.isFinite(Number(item))) {
            item_id = Number(item);
        } else {
            item_id = await getItemId(region, item);
            if (item_id < 0) {
                logger.error(`No itemId could be found for ${item}`);
                throw (new Error(`No itemId could be found for ${item}`));
            }
            logger.info(`Found ${item_id} for ${item}`);
        }

        const bonuses: Record<string, string>[] = await db.all(sql, [item_id]);

        logger.debug(`Found ${bonuses.length} bonuses for ${item}`);

        const item_details = await getItemDetails(item_id, region);

        return {
            bonuses: bonuses,
            item: item_details,
        };
    }

    async function fillNItems(fill_count: number = 5): Promise<void> {
        logger.info(`Filling ${fill_count} items with details.`);
        const select_sql = 'SELECT item_id, region FROM items WHERE scanned = false LIMIT $1';
        const update_sql = 'UPDATE items SET name = $1, craftable = $2, scanned = true WHERE item_id = $3 AND region = $4';
        const client = await db.getClient();
        type ItemRow = { item_id: number, region: RegionCode }
        const rows = (await client.query<ItemRow>(select_sql, [fill_count])).rows;
        await client.query('BEGIN TRANSACTION');
        for (const item of rows) {
            try {
                const fetched_item = await getItemDetails(item.item_id, item.region);
                const is_craftable = await checkIsCrafting(item.item_id, ALL_PROFESSIONS, item.region);
                await client.query(update_sql, [fetched_item.name, is_craftable.craftable, item.item_id, item.region]);
                logger.debug(`Updated item: ${item.item_id}:${item.region} with name: '${fetched_item.name}' and craftable: ${is_craftable.craftable}`);
            } catch (e) {
                logger.error(`Issue filling ${item.item_id} in ${item.region}. Skipping`, e);
                await client.query('DELETE FROM items WHERE item_id = $1 AND region = $2', [item.item_id, item.region]);
                logger.error(`DELETED ${item.item_id} in ${item.region} from items table.`);
            }
        }
        await client.query('COMMIT TRANSACTION');
        client.release();
    }

    async function fillNNames(fillCount: number = 5): Promise<void> {
        logger.info(`Filling ${fillCount} unnamed item names.`);
        const select_sql = 'SELECT item_id, region FROM items WHERE name ISNULL ORDER BY item_id DESC LIMIT $1';
        const update_sql = 'UPDATE items SET name = $1 WHERE item_id = $2 AND region = $3';
        const client = await db.getClient();
        type ItemRow = { item_id: number, region: RegionCode }
        const rows = (await client.query<ItemRow>(select_sql, [fillCount])).rows;
        await client.query('BEGIN TRANSACTION');
        for (const item of rows) {
            try {
                const fetched_item = await getItemDetails(item.item_id, item.region);
                await client.query(update_sql, [fetched_item.name, item.item_id, item.region]);
                logger.debug(`Updated item: ${item.item_id}:${item.region} with name: '${fetched_item.name}'`);
            } catch (e) {
                logger.error(`Issue filling ${item.item_id} in ${item.region}. Skipping`, e);
                await client.query('DELETE FROM items WHERE item_id = $1 AND region = $2', [item.item_id, item.region]);
                logger.error(`DELETED ${item.item_id} in ${item.region} from items table.`);
            }
        }
        await client.query('COMMIT TRANSACTION');
        client.release();
    }

    async function getAuctions(item: ItemSoftIdentity, realm: ConnectedRealmSoftIentity, region: RegionCode, bonuses: number[] | string[] | string, start_dtm: number | string | undefined, end_dtm: number | string | undefined): Promise<AuctionSummaryData> {
        logger.debug(`getAuctions(${item}, ${realm}, ${region}, ${bonuses}, ${start_dtm}, ${end_dtm})`);
        //const sql_build = 'SELECT * FROM auctions';
        const sql_archive_build = 'SELECT downloaded, summary FROM auction_archive';
        const sql_build_distinct_dtm = 'SELECT DISTINCT downloaded FROM auctions';
        const sql_build_price_map = 'SELECT price, count(price) AS sales_at_price, sum(quantity) AS quantity_at_price FROM auctions';
        const sql_group_by_price_addin = 'GROUP BY price';
        const sql_build_min = 'SELECT MIN(price) AS min_price FROM auctions';
        const sql_build_max = 'SELECT MAX(price) AS max_price FROM auctions';
        const sql_build_avg = 'SELECT SUM(price*quantity)/SUM(quantity) AS avg_price FROM auctions';
        const sql_build_latest_dtm = 'SELECT MAX(downloaded) AS latest_download FROM auctions';
        const sql_addins = [];
        const value_searches = [];
        if (item !== undefined) {
            // Get specific items
            let item_id = 0;
            if (typeof item === 'number') {
                item_id = item;
            } else if (Number.isFinite(Number(item))) {
                item_id = Number(item);
            } else {
                item_id = await getItemId(region, item);
                if (item_id < 0) {
                    logger.error(`No itemId could be found for ${item}`);
                    throw (new Error(`No itemId could be found for ${item}`));
                }
                logger.info(`Found ${item_id} for ${item}`);
            }
            sql_addins.push(`item_id = ${get_place_marker()}`);
            value_searches.push(item_id);
        } else {
            // All items
        }
        if (realm !== undefined) {
            let server_id = 0;
            if (typeof realm === 'number') {
                server_id = realm;
            } else if (Number.isFinite(Number(realm))) {
                server_id = Number(realm);
            } else {
                server_id = await getConnectedRealmId(realm, region);
                if (server_id < 0) {
                    logger.error(`No connected realm id could be found for ${realm}`);
                    throw (new Error(`No connected realm id could be found for ${realm}`));
                }
                logger.info(`Found ${server_id} for ${realm}`);
            }
            // Get specific realm
            sql_addins.push(`connected_realm_id = ${get_place_marker()}`);
            value_searches.push(server_id);
        } else {
            // All realms
        }
        if (region !== undefined) {
            // Get specific region
            sql_addins.push(`region = ${get_place_marker()}`);
            value_searches.push(region.toLocaleLowerCase());
        } else {
            // All regions
        }
        if (bonuses !== undefined) {
            // Get only with specific bonuses
            if (bonuses === null) {
                sql_addins.push('bonuses IS NULL');
            }
            else if (typeof bonuses === 'string') {
                sql_addins.push(`bonuses = ${get_place_marker()}`);
                value_searches.push(bonuses);
            } else {
                bonuses.forEach((b: string | number) => {
                    if (b !== null && b !== undefined && b !== '') {
                        logger.debug(`Add bonus ${b} in (select json_each.value from json_each(bonuses))`);
                        const json_query = db_type === 'sqlite3' ? `${get_place_marker()} IN (SELECT json_each.value FROM json_each(bonuses))` : `${get_place_marker()} IN (SELECT json_array_elements_text(bonuses::json)::numeric)`
                        sql_addins.push(json_query);
                        value_searches.push(Number(b));
                    }
                });
            }
        } else {
            // any bonuses or none
        }
        if (start_dtm !== undefined) {
            // Include oldest fetch date time
            sql_addins.push(`downloaded >= ${get_place_marker()}`);
            value_searches.push(start_dtm);
        } else {
            // No start fetched date time limit
        }
        if (end_dtm !== undefined) {
            // Include newest fetch date time
            sql_addins.push(`downloaded <= ${get_place_marker()}`);
            value_searches.push(end_dtm);
        } else {
            // No latest fetched date time
        }

        //const run_sql = build_sql_with_addins(sql_build, sql_addins);
        const min_sql = build_sql_with_addins(sql_build_min, sql_addins);
        const max_sql = build_sql_with_addins(sql_build_max, sql_addins);
        const avg_sql = build_sql_with_addins(sql_build_avg, sql_addins);
        const latest_dl_sql = build_sql_with_addins(sql_build_latest_dtm, sql_addins);
        const distinct_download_sql = build_sql_with_addins(sql_build_distinct_dtm, sql_addins);

        const min_dtm_sql = build_sql_with_addins(sql_build_min, [...sql_addins, `downloaded = ${get_place_marker()}`]);
        const max_dtm_sql = build_sql_with_addins(sql_build_max, [...sql_addins, `downloaded = ${get_place_marker()}`]);
        const avg_dtm_sql = build_sql_with_addins(sql_build_avg, [...sql_addins, `downloaded = ${get_place_marker()}`]);
        const price_group_sql = build_sql_with_addins(sql_build_price_map, [...sql_addins, `downloaded = ${get_place_marker()}`]) + ' ' + sql_group_by_price_addin;

        //const client = await db.getClient();
        type MinPrice = { min_price: number };
        type MaxPrice = { max_price: number };
        type AvgPrice = { avg_price: number };
        type LatestDownload = { latest_download: number };
        type Downloaded = { downloaded: number };
        type Summary = { downloaded: number, summary: AuctionPriceSummaryRecord | string };

        const min_value = (await db.get<MinPrice>(min_sql, value_searches)).min_price;
        const max_value = (await db.get<MaxPrice>(max_sql, value_searches)).max_price;
        const avg_value = (await db.get<AvgPrice>(avg_sql, value_searches)).avg_price;
        const latest_dl_value = (await db.get<LatestDownload>(latest_dl_sql, value_searches)).latest_download;

        const price_data_by_download: Record<number, AuctionPriceSummaryRecord> = {};
        for (const row of (await db.all<Downloaded>(distinct_download_sql, value_searches))) {
            price_data_by_download[row.downloaded] = {
                data: await db.all(price_group_sql, [...value_searches, row.downloaded]),
                min_value: (await db.get<MinPrice>(min_dtm_sql, [...value_searches, row.downloaded])).min_price,
                max_value: (await db.get<MaxPrice>(max_dtm_sql, [...value_searches, row.downloaded])).max_price,
                avg_value: (await db.get<AvgPrice>(avg_dtm_sql, [...value_searches, row.downloaded])).avg_price
            };
        }

        // Get archives if they exist
        const archive_fetch_sql = build_sql_with_addins(sql_archive_build, sql_addins);
        const archives = await db.all<Summary>(archive_fetch_sql, value_searches);

        const archived_results: Record<string, Array<AuctionPriceSummaryRecord>> = {};
        logger.debug(`Found ${archives.length} archive rows.`);
        for (const archive of archives) {
            if (!(archive.downloaded in archived_results)) {
                archived_results[archive.downloaded] = [];
            }
            archived_results[archive.downloaded].push((db_type === 'pg' ? archive.summary : JSON.parse(<string>archive.summary)));
        }

        const archive_build = [];

        for (const key of Object.keys(archived_results)) {
            const arch = archived_results[key];

            const arch_build = {
                timestamp: key,
                data: [] as Array<SalesCountSummaryPrice>,
                min_value: Number.MAX_SAFE_INTEGER,
                max_value: Number.MIN_SAFE_INTEGER,
                avg_value: 0,
            };

            const price_link: Record<string, SalesCountSummary> = {};

            for (const a of arch) {

                if (arch_build.min_value > a.min_value) {
                    arch_build.min_value = a.min_value;
                }
                if (arch_build.max_value < a.max_value) {
                    arch_build.max_value = a.max_value;
                }
                arch_build.avg_value += a.avg_value;
                if (a.data !== undefined) {
                    for (const p of a.data) {
                        if (!(p.price in price_link)) {
                            price_link[p.price] = {
                                sales_at_price: 0,
                                quantity_at_price: 0,
                            }
                        }
                        price_link[p.price].sales_at_price += p.sales_at_price;
                        price_link[p.price].quantity_at_price += p.quantity_at_price;
                    }
                }
            }
            arch_build.avg_value = arch_build.avg_value / arch.length;

            Object.keys(price_link).forEach((key) => {
                arch_build.data.push({
                    price: Number(key),
                    sales_at_price: price_link[key].sales_at_price,
                    quantity_at_price: price_link[key].quantity_at_price,
                })
            });

            archive_build.push(arch_build);
        }

        const now_moment = Date.now();
        const spot_summary = await getSpotAuctionSummary(item, realm, region, bonuses);
        if (spot_summary.avg_value !== 0) {
            price_data_by_download[now_moment] = spot_summary;
        }
        const final_latest_value = spot_summary.avg_value === 0 ? latest_dl_value : now_moment;

        logger.debug(`Found max: ${max_value}, min: ${min_value}, avg: ${avg_value}`);

        return {
            min: min_value,
            max: max_value,
            avg: avg_value,
            //latest: latest_dl_value,
            latest: final_latest_value,
            price_map: price_data_by_download,
            archives: archive_build,
        };

        function build_sql_with_addins(base_sql: string, addin_list: Array<string>): string {
            let construct_sql = base_sql;
            if (addin_list.length > 0) {
                construct_sql += ' WHERE ';
                for (const addin of addin_list) {
                    construct_sql += addin;
                    construct_sql += ' AND ';
                }
                construct_sql = construct_sql.slice(0, construct_sql.length - 4);
            }
            return construct_sql;
        }

        function get_place_marker(): string {
            return `$${value_searches.length + 1}`;
        }
    }

    async function archiveAuctions(): Promise<void> {
        const backstep_time_diff = (6.048e+8); // One Week
        //const backstep_time_diff = 1.21e+9; // Two weeks
        const delete_diff = 1.21e+9; // two weeks
        const day_diff = 8.64e+7;
        const backstep_time = Date.now() - backstep_time_diff;

        const sql_get_downloaded_oldest = 'SELECT MIN(downloaded) AS oldest FROM auctions';
        const sql_get_distinct_rows_from_downloaded = 'SELECT DISTINCT item_id, bonuses, connected_realm_id, region FROM auctions WHERE downloaded BETWEEN $1 AND $2';
        const sql_delete_archived_auctions = 'DELETE FROM auctions WHERE downloaded BETWEEN $1 AND $2';

        const sql_price_map = 'SELECT price, count(price) AS sales_at_price, sum(quantity) AS quantity_at_price FROM auctions WHERE item_id=$1 AND bonuses=$2 AND connected_realm_id=$3 AND region=$6 AND downloaded BETWEEN $4 AND $5 GROUP BY price';
        const sql_min = 'SELECT MIN(price) AS min_price FROM auctions WHERE item_id=$1 AND bonuses=$2 AND connected_realm_id=$3 AND region=$6 AND downloaded BETWEEN $4 AND $5';
        const sql_max = 'SELECT MAX(price) AS max_price FROM auctions WHERE item_id=$1 AND bonuses=$2 AND connected_realm_id=$3 AND region=$6 AND downloaded BETWEEN $4 AND $5';
        const sql_avg = 'SELECT SUM(price*quantity)/SUM(quantity) AS avg_price FROM auctions WHERE item_id=$1 AND bonuses=$2 AND connected_realm_id=$3 AND region=$6 AND downloaded BETWEEN $4 AND $5';

        const client = await db.getClient();

        let count = 0;

        type Oldest = { oldest: number };
        type DistinctRows = { item_id: number, bonuses: string, connected_realm_id: number, region: string };
        type Min = { min_price: number };
        type Max = { max_price: number };
        type Average = { avg_price: number };

        await client.query('BEGIN TRANSACTION', []);

        let running = true;
        while (running) {
            // Get oldest downloaded
            const current_oldest = Number((await client.query<Oldest>(sql_get_downloaded_oldest, [])).rows[0].oldest);
            //console.log((await client.query(sql_get_downloaded_oldest, [])).rows[0].oldest);
            logger.debug(`Current oldest is ${(new Date(current_oldest)).toLocaleString()}`);
            // Check if oldest fits our criteria
            if (current_oldest < backstep_time) {
                // Pick the whole day
                const start_ticks = current_oldest;
                const end_ticks = current_oldest + day_diff;
                logger.info(`Scan between ${(new Date(start_ticks)).toLocaleString()} and ${(new Date(end_ticks)).toLocaleString()}`);
                // Run for that day
                // Get a list of all distinct item/server combinations
                const items = (await client.query<DistinctRows>(sql_get_distinct_rows_from_downloaded, [start_ticks, end_ticks])).rows;
                for (const item of items) {
                    const vals = [item.item_id, item.bonuses, item.connected_realm_id, start_ticks, end_ticks, item.region];

                    // Run the getAuctions command for the combo
                    const summary: AuctionPriceSummaryRecord = {
                        data: (await client.query<SalesCountSummaryPrice>(sql_price_map, vals)).rows,
                        min_value: (await client.query<Min>(sql_min, vals)).rows[0].min_price,
                        max_value: (await client.query<Max>(sql_max, vals)).rows[0].max_price,
                        avg_value: (await client.query<Average>(sql_avg, vals)).rows[0].avg_price
                    }

                    let quantity = 0;
                    if (summary.data !== undefined) {
                        quantity = summary.data.reduce((acc, cur) => {
                            return acc + cur.quantity_at_price;
                        }, 0);
                    }

                    // Add the archive
                    await client.query(sql_insert_auction_archive, [item.item_id, quantity, JSON.stringify(summary), start_ticks, item.connected_realm_id, item.bonuses, item.region]);
                    count++;
                }
                // Delete the archived data
                await client.query(sql_delete_archived_auctions, [start_ticks, end_ticks]);
                // Done
            } else {
                running = false;
                logger.info(`Finished archive task. Archived ${count} records`);
            }
        }

        const delete_backstep = Date.now() - delete_diff;
        const delete_auctions_older = 'DELETE FROM auctions WHERE downloaded < $1';
        const delete_archive_older = 'DELETE FROM auction_archive WHERE downloaded < $1';

        client.query(delete_auctions_older, [delete_backstep]);
        client.query(delete_archive_older, [delete_backstep]);

        await client.query('COMMIT TRANSACTION', []);
        client.release();
    }

    async function getSpotAuctionSummary(item: ItemSoftIdentity, realm: ConnectedRealmSoftIentity, region: RegionCode, bonuses: number[] | string[] | string): Promise<AuctionPriceSummaryRecord> {
        let realm_get = realm;
        if (typeof (realm) === 'string') {
            realm_get = await getConnectedRealmId(realm, region);
        } else if (typeof (realm) === 'number') {
            realm_get = realm;
        } else {
            throw new Error('Realm not a string or number');
        }
        const ah = await getAuctionHouse(realm_get, region);
        logger.debug(`Spot search for item: ${item} and realm ${realm} and region ${region}, with bonuses ${JSON.stringify(bonuses)}`);

        let item_id = item;
        if (typeof (item) === 'string') {
            item_id = await getItemId(region, item);
        }

        const auction_set = ah.auctions.filter((auction) => {
            let found_item = false;
            let found_bonus = false;
            if (auction.item.id == item_id) {
                found_item = true;
                logger.silly(`Found ${auction.item.id}`);
            }

            if (bonuses === null) {
                if (auction.item.bonus_lists === undefined || auction.item.bonus_lists.length === 0) {
                    found_bonus = true;
                    logger.silly(`Found ${auction.item.id} to match null bonus list`);
                }
            } else if (typeof (bonuses) === 'string') {
                const bonus_parse = JSON.parse(bonuses);
                if (Array.isArray(bonus_parse)) {
                    found_bonus = check_bonus(bonus_parse, auction.item.bonus_lists);
                    logger.silly(`String bonus list ${bonuses} returned ${found_bonus} for ${JSON.stringify(auction.item.bonus_lists)}`);
                }
            } else {
                found_bonus = check_bonus(bonuses, auction.item.bonus_lists);
                logger.silly(`Array bonus list ${JSON.stringify(bonuses)} returned ${found_bonus} for ${JSON.stringify(auction.item.bonus_lists)}`);
            }

            return found_bonus && found_item;
        });
        logger.debug(`Found ${auction_set.length} auctions`);

        const return_value: AuctionPriceSummaryRecord = {
            min_value: Number.MAX_SAFE_INTEGER,
            max_value: Number.MIN_SAFE_INTEGER,
            avg_value: 0,
            data: []
        };

        let total_sales = 0;
        let total_price = 0;
        const price_map: Record<number, { quantity: number, sales: number }> = {};

        for (const auction of auction_set) {
            let price = 0;
            const quantity = auction.quantity;
            if (auction.buyout !== undefined) {
                price = auction.buyout;
            } else {
                price = auction.unit_price;
            }

            if (return_value.max_value < price) {
                return_value.max_value = price;
            }
            if (return_value.min_value > price) {
                return_value.min_value = price;
            }
            total_sales += quantity;
            total_price += price * quantity;

            if (price_map[price] === undefined) {
                price_map[price] = {
                    quantity: 0,
                    sales: 0
                }
            }
            price_map[price].quantity += quantity;
            price_map[price].sales += 1
        }

        return_value.avg_value = total_price / total_sales;
        for (const price of Object.keys(price_map)) {
            const p_lookup = Number(price);
            return_value.data?.push(
                {
                    price: p_lookup,
                    quantity_at_price: price_map[p_lookup].quantity,
                    sales_at_price: price_map[p_lookup].sales
                }
            );
        }

        return return_value;

        function check_bonus(bonus_list: number[] | string[], target?: number[]) {
            let found = true;

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

            return found;
        };
    }

    async function addRealmToScanList(realm_name: RealmName, realm_region: RegionCode): Promise<void> {
        const sql = 'INSERT INTO realm_scan_list(connected_realm_id,region) VALUES($1,$2)';
        try {
            await db.run(sql, [await getConnectedRealmId(realm_name, realm_region), realm_region.toUpperCase()]);
        } catch (err) {
            logger.error(`Couldn't add ${realm_name} in ${realm_region} to scan realms table.`, err);
        }
    }

    async function removeRealmFromScanList(realm_name: RealmName, realm_region: RegionCode): Promise<void> {
        const sql = 'DELETE FROM realm_scan_list WHERE connected_realm_id = $1 AND region = $2';
        await db.run(sql, [await getConnectedRealmId(realm_name, realm_region), realm_region.toUpperCase()]);
    }

    async function getScanRealms(): Promise<{ realm_names: string, realm_id: ConnectedRealmID, region: RegionCode }[]> {
        const query = 'SELECT connected_realm_id, region FROM realm_scan_list';
        const data = await db.all<{ connected_realm_id: number, region: string }>(query);
        const ret_val: { realm_names: string, realm_id: ConnectedRealmID, region: RegionCode }[] = [];
        for (const row of data) {
            ret_val.push({
                realm_id: row.connected_realm_id,
                region: getRegionCode(row.region),
                realm_names: (await getBlizConnectedRealmDetail(row.connected_realm_id, getRegionCode(row.region))).realms.reduce((prev, rlm) => { return `${prev}, ${rlm.name}` }, '')
            });
        }
        return ret_val;
    }

    async function scanRealms(): Promise<void> {
        type RealmScanListEntry = { region: RegionCode, connected_realm_id: number };
        const getScannableRealms = 'SELECT connected_realm_id, region FROM realm_scan_list';
        const realm_scan_list = await db.all<RealmScanListEntry>(getScannableRealms, []);
        await Promise.all(realm_scan_list.map((realm: RealmScanListEntry) => {
            return ingest(realm.region, realm.connected_realm_id);
        }));
    }

    async function getAllNames(): Promise<string[]> {
        const name_list = await db.all<{ name: string }>('SELECT DISTINCT name FROM items WHERE name NOTNULL');
        return name_list.reduce((prev: string[], curr) => {
            return [...prev, curr.name];
        }, []);
    }
*/

/*
 */
