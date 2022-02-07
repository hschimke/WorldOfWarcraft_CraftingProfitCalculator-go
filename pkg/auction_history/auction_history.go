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
				bson.D{{"region", region}},
				bson.D{{"item.bonus_lists", bson.D{{"$exists", true}}}},
				bson.D{{"item.bonus_lists", bson.D{{"$ne", bson.TypeNull}}}},
			},
		}}

	results, err := auctionsCollection.Distinct(context.TODO(), "item.bonus_lists", auctionsFilter)
	if err != nil {
		return GetAllBonusesReturn{}, err
	}

	var return_value GetAllBonusesReturn

	return_value.Item.Id = searchId
	return_value.Item.Name = item.ItemName

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
