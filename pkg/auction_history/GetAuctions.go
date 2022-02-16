package auction_history

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/environment_variables"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/util"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/blizzard_api_helpers"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes/BlizzardApi"
	"github.com/jackc/pgx/v4/pgxpool"
)

// Get all auctions filtering with parameters
func GetAuctions(item globalTypes.ItemSoftIdentity, realm globalTypes.ConnectedRealmSoftIentity, region globalTypes.RegionCode, bonuses []uint, start_dtm time.Time, end_dtm time.Time) (AuctionSummaryData, error) {
	dbpool, err := pgxpool.Connect(context.Background(), environment_variables.DATABASE_CONNECTION_STRING)
	if err != nil {
		cpclog.Errorf("Unable to connect to database: %v", err)
		return AuctionSummaryData{}, err
	}
	defer dbpool.Close()

	var value_searches []interface{}

	build_sql_with_addins := func(base_sql string, addin_list []string) string {
		var construct strings.Builder
		construct_sql := base_sql
		construct.WriteString(base_sql)
		if len(addin_list) > 0 {
			construct.WriteString(" WHERE ")
			for _, addin := range addin_list {
				construct.WriteString(addin)
				construct.WriteString(" AND ")
			}
			runicConstructSql := []rune(construct.String())
			runicConstructSql = runicConstructSql[:len(runicConstructSql)-4]
			construct_sql = string(runicConstructSql)
			//construct_sql = construct_sql.slice(0, construct_sql.length-4)
		}
		return construct_sql
	}

	get_place_marker := func() string {
		return fmt.Sprintf("$%d", len(value_searches)+1)
	}

	cpclog.Debugf(`getAuctions(%v, %v, %s, %v, %T, %T)`, item, realm, region, bonuses, start_dtm, end_dtm)
	const (
		sql_archive_build        string = "SELECT downloaded, summary FROM auction_archive"
		sql_build_distinct_dtm   string = "SELECT DISTINCT downloaded FROM auctions"
		sql_build_price_map      string = "SELECT price, count(price) AS sales_at_price, sum(quantity) AS quantity_at_price FROM auctions"
		sql_group_by_price_addin string = "GROUP BY price"
		sql_build_min            string = "SELECT MIN(price) AS min_price FROM auctions"
		sql_build_max            string = "SELECT MAX(price) AS max_price FROM auctions"
		sql_build_avg            string = "SELECT SUM(price*quantity)/SUM(quantity) AS avg_price FROM auctions"
		sql_build_latest_dtm     string = "SELECT MAX(downloaded) AS latest_download FROM auctions"
		jsonQueryTemplate        string = `%s IN (SELECT json_array_elements_text(bonuses::json)::numeric)`
	)
	var sql_addins []string

	var itemId, connectedRealmId uint

	// Get realm
	if realm.Name != "" {
		rlm, err := blizzard_api_helpers.GetConnectedRealmId(realm.Name, region)
		if err != nil {
			return AuctionSummaryData{}, err
		}
		connectedRealmId = rlm
	} else {
		connectedRealmId = realm.Id
	}

	// Get item
	if item.ItemName != "" {
		itm, err := blizzard_api_helpers.GetItemId(region, item.ItemName)
		if err != nil {
			return AuctionSummaryData{}, err
		}
		itemId = itm
	} else {
		itemId = item.ItemId
	}

	if itemId != 0 {
		sql_addins = append(sql_addins, fmt.Sprintf(`item_id = %s`, get_place_marker()))
		value_searches = append(value_searches, itemId)
	} /*else {
		// All items
	}*/

	if connectedRealmId != 0 {
		// Get specific realm
		sql_addins = append(sql_addins, fmt.Sprintf(`connected_realm_id = %s`, get_place_marker()))
		value_searches = append(value_searches, connectedRealmId)
	} /*else {
		// All realms
	}*/

	if len(region) > 0 {
		// Get specific region
		sql_addins = append(sql_addins, fmt.Sprintf(`region = %s`, get_place_marker()))
		value_searches = append(value_searches, strings.ToLower(region))
	} /*else {
		// All regions
	}*/

	// Include oldest fetch date time
	sql_addins = append(sql_addins, fmt.Sprintf(`downloaded >= %s`, get_place_marker()))
	value_searches = append(value_searches, start_dtm)

	// Include newest fetch date time
	sql_addins = append(sql_addins, fmt.Sprintf(`downloaded <= %s`, get_place_marker()))
	value_searches = append(value_searches, end_dtm)

	if len(bonuses) > 0 {
		// Get only with specific bonuses
		for _, b := range bonuses {
			if b != 0 {
				cpclog.Debugf(`Add bonus %d in (select json_each.value from json_each(bonuses))`, b)
				json_query := fmt.Sprintf(jsonQueryTemplate, get_place_marker())
				sql_addins = append(sql_addins, json_query)
				value_searches = append(value_searches, b)
			}
		}
	} /*else {
		// any bonuses or none
	}*/

	var (
		min_sql               string = build_sql_with_addins(sql_build_min, sql_addins)
		max_sql               string = build_sql_with_addins(sql_build_max, sql_addins)
		avg_sql               string = build_sql_with_addins(sql_build_avg, sql_addins)
		latest_dl_sql         string = build_sql_with_addins(sql_build_latest_dtm, sql_addins)
		distinct_download_sql string = build_sql_with_addins(sql_build_distinct_dtm, sql_addins)

		min_dtm_sql     string = build_sql_with_addins(sql_build_min, append(sql_addins, fmt.Sprintf(`downloaded = %s`, get_place_marker())))
		max_dtm_sql     string = build_sql_with_addins(sql_build_max, append(sql_addins, fmt.Sprintf(`downloaded = %s`, get_place_marker())))
		avg_dtm_sql     string = build_sql_with_addins(sql_build_avg, append(sql_addins, fmt.Sprintf(`downloaded = %s`, get_place_marker())))
		price_group_sql string = build_sql_with_addins(sql_build_price_map, append(sql_addins, fmt.Sprintf(`downloaded = %s`, get_place_marker()))) + " " + sql_group_by_price_addin
	)

	var (
		min_value, max_value uint
		avg_value            float64
		latest_dl_value      time.Time
	)

	dbpool.QueryRow(context.TODO(), min_sql, value_searches...).Scan(&min_value)
	dbpool.QueryRow(context.TODO(), max_sql, value_searches...).Scan(&max_value)
	dbpool.QueryRow(context.TODO(), avg_sql, value_searches...).Scan(&avg_value)
	dbpool.QueryRow(context.TODO(), latest_dl_sql, value_searches...).Scan(&latest_dl_value)

	price_data_by_download := make(map[time.Time]AuctionPriceSummaryRecord)
	distRows, dErr := dbpool.Query(context.TODO(), distinct_download_sql, value_searches...)
	if dErr != nil {
		return AuctionSummaryData{}, dErr
	}
	defer distRows.Close()
	for distRows.Next() {
		var (
			downloaded time.Time
		)
		newSummary := AuctionPriceSummaryRecord{}
		distRows.Scan(&downloaded)
		modVS := append(value_searches, downloaded)
		dbpool.QueryRow(context.TODO(), min_dtm_sql, modVS...).Scan(&newSummary.MinValue)
		dbpool.QueryRow(context.TODO(), max_dtm_sql, modVS...).Scan(&newSummary.MaxValue)
		dbpool.QueryRow(context.TODO(), avg_dtm_sql, modVS...).Scan(&newSummary.AvgValue)

		priceGrpRows, prcGrpErr := dbpool.Query(context.TODO(), price_group_sql, modVS...)
		if prcGrpErr != nil {
			return AuctionSummaryData{}, prcGrpErr
		}
		defer priceGrpRows.Close()
		for priceGrpRows.Next() {
			newSCS := SalesCountSummary{}
			priceGrpRows.Scan(&newSCS.Price, &newSCS.SalesAtPrice, &newSCS.QuantityAtPrice)
			newSummary.Data = append(newSummary.Data, newSCS)
		}

		price_data_by_download[downloaded] = newSummary
	}

	var return_value AuctionSummaryData
	return_value.PriceMap = make(map[time.Time]AuctionPriceSummaryRecord)

	return_value.Min = min_value
	return_value.Max = max_value
	return_value.Avg = avg_value
	return_value.PriceMap = price_data_by_download

	// Get spot auctions
	spotSummary, err := getSpotAuctionSummary(item, realm, region, bonuses)
	if err != nil {
		return AuctionSummaryData{}, err
	}
	cTime := time.Now()
	if len(spotSummary.Data) > 0 {
		return_value.PriceMap[cTime] = spotSummary
		return_value.Latest = cTime

		if spotSummary.MinValue < return_value.Min {
			return_value.Min = spotSummary.MinValue
		}
		if spotSummary.MaxValue > return_value.Max {
			return_value.Max = spotSummary.MaxValue
		}
	} else {
		return_value.Latest = latest_dl_value
	}

	return_value.Archives = make([]struct {
		Timestamp time.Time           "json:\"timestamp,omitempty\""
		Data      []SalesCountSummary "json:\"data,omitempty\""
		MinValue  uint                "json:\"min_value,omitempty\""
		MaxValue  uint                "json:\"max_value,omitempty\""
		AvgValue  float64             "json:\"avg_value,omitempty\""
	}, 0)

	return return_value, nil
}

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
	cpclog.Debugf(`Spot search for item: %v and realm %v and region %s, with bonuses %v`, item, realm, region, bonuses)

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
			found_bonus = checkBonux(bonuses, auction.Item.Bonus_lists)
			cpclog.Sillyf(`Array bonus list %v returned %t for %v`, bonuses, found_bonus, auction.Item.Bonus_lists)
		}

		if found_bonus && found_item {
			auction_set = append(auction_set, auction)
		}
	}

	cpclog.Debugf(`Found %d auctions`, len(auction_set))

	return_value := AuctionPriceSummaryRecord{
		MinValue: math.MaxUint,
	}

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

func checkBonux(bonus_list []uint, target []uint) (found bool) {
	found = true

	// Take care of undefined targets
	if len(target) == 0 {
		if len(bonus_list) != 0 {
			found = false
		}
		found = true
	}

	for _, list_entry := range bonus_list {
		found = found && util.ArrayIncludes(target, list_entry)
	}

	return
}