package auction_history

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/util"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes/BlizzardApi"
	"github.com/jackc/pgx/v4"
)

func buildSQLWithAddins(base_sql string, addin_list []string) string {
	var construct strings.Builder
	construct.WriteString(base_sql)
	if len(addin_list) > 0 {
		construct.WriteString(" WHERE ")
		for _, addin := range addin_list {
			construct.WriteString(addin)
			construct.WriteString(" AND ")
		}
		runicConstructSql := []rune(construct.String())
		runicConstructSql = runicConstructSql[:len(runicConstructSql)-4]
		return string(runicConstructSql)
	}
	return base_sql
}

// Get all auctions filtering with parameters
func (ahs *AuctionHistoryServer) GetAuctions(ctx context.Context, item globalTypes.ItemSoftIdentity, realm globalTypes.ConnectedRealmSoftIentity, region globalTypes.RegionCode, bonuses []uint, start_dtm time.Time, end_dtm time.Time) (AuctionSummaryData, error) {
	var value_searches []any

	get_place_marker := func() string {
		return fmt.Sprintf("$%d", len(value_searches)+1)
	}

	ahs.logger.Debugf(`getAuctions(%v, %v, %s, %v, %T, %T)`, item, realm, region, bonuses, start_dtm, end_dtm)
	const (
		sql_build_price_map      string = "SELECT price, count(price) AS sales_at_price, sum(quantity) AS quantity_at_price, downloaded FROM auctions"
		sql_build_latest_dtm     string = "SELECT MAX(downloaded) AS latest_download FROM auctions"
		jsonQueryTemplate        string = `bonuses @> jsonb_build_array(%s)`

		sql_build_min_max_avg               string = "SELECT MIN(price) as min_price, MAX(price) AS max_price, SUM(price*quantity)/SUM(quantity) AS avg_price FROM auctions"
		sql_build_min_max_avg_downloaded    string = "SELECT MIN(price) as min_price, MAX(price) AS max_price, SUM(price*quantity)/SUM(quantity) AS avg_price, downloaded FROM auctions"
		sql_group_by_downloaded_addin       string = "GROUP BY downloaded"
		sql_group_by_downloaded_price_addin string = "GROUP BY downloaded,price"
	)
	var sql_addins []string

	var itemId, connectedRealmId uint

	// Get realm
	if realm.Name != "" {
		rlm, err := ahs.helper.GetConnectedRealmId(ctx, realm.Name, region)
		if err != nil {
			return AuctionSummaryData{}, err
		}
		connectedRealmId = uint(rlm)
	} else {
		connectedRealmId = uint(realm.Id)
	}

	// Get item
	if item.ItemName != "" {
		itm, err := ahs.helper.GetItemId(ctx, region, item.ItemName)
		if err != nil {
			return AuctionSummaryData{}, err
		}
		itemId = uint(itm)
	} else {
		itemId = uint(item.ItemId)
	}

	if itemId != 0 {
		sql_addins = append(sql_addins, fmt.Sprintf(`item_id = %s`, get_place_marker()))
		value_searches = append(value_searches, itemId)
	}

	if connectedRealmId != 0 {
		sql_addins = append(sql_addins, fmt.Sprintf(`connected_realm_id = %s`, get_place_marker()))
		value_searches = append(value_searches, connectedRealmId)
	}

	if len(region) > 0 {
		sql_addins = append(sql_addins, fmt.Sprintf(`region = %s`, get_place_marker()))
		value_searches = append(value_searches, strings.ToLower(string(region)))
	}

	sql_addins = append(sql_addins, fmt.Sprintf(`downloaded >= %s`, get_place_marker()))
	value_searches = append(value_searches, start_dtm)

	sql_addins = append(sql_addins, fmt.Sprintf(`downloaded <= %s`, get_place_marker()))
	value_searches = append(value_searches, end_dtm)

	if len(bonuses) > 0 {
		for _, b := range bonuses {
			if b != 0 {
				json_query := fmt.Sprintf(jsonQueryTemplate, get_place_marker())
				sql_addins = append(sql_addins, json_query)
				value_searches = append(value_searches, b)
			}
		}
	}

	var (
		min_max_avg_sql string = buildSQLWithAddins(sql_build_min_max_avg, sql_addins)
		latest_dl_sql   string = buildSQLWithAddins(sql_build_latest_dtm, sql_addins)

		downloaded_group_sql     string = buildSQLWithAddins(sql_build_min_max_avg_downloaded, sql_addins) + " " + sql_group_by_downloaded_addin
		downloaded_price_map_sql string = buildSQLWithAddins(sql_build_price_map, sql_addins) + " " + sql_group_by_downloaded_price_addin
	)

	batch := &pgx.Batch{}
	batch.Queue(min_max_avg_sql, value_searches...)
	batch.Queue(latest_dl_sql, value_searches...)
	batch.Queue(downloaded_group_sql, value_searches...)
	batch.Queue(downloaded_price_map_sql, value_searches...)

	bRes := ahs.db.SendBatch(ctx, batch)
	defer bRes.Close()

	var (
		min_value, max_value uint
		avg_value            float64
		latest_dl_value      time.Time
	)

	bRes.QueryRow().Scan(&min_value, &max_value, &avg_value)
	bRes.QueryRow().Scan(&latest_dl_value)

	price_data_by_download := make(map[time.Time]AuctionPriceSummaryRecord)

	dataRows, drError := bRes.Query()
	if drError == nil {
		for dataRows.Next() {
			var (
				downloaded time.Time
				newSummary AuctionPriceSummaryRecord
			)
			dataRows.Scan(&newSummary.MinValue, &newSummary.MaxValue, &newSummary.AvgValue, &downloaded)
			price_data_by_download[downloaded] = newSummary
		}
		dataRows.Close()
	}

	prcMapRows, prMRErr := bRes.Query()
	if prMRErr == nil {
		overallMedianMap := make(map[float64]uint64)
		for prcMapRows.Next() {
			var (
				scSum      SalesCountSummary
				downloaded time.Time
			)
			prcMapRows.Scan(&scSum.Price, &scSum.SalesAtPrice, &scSum.QuantityAtPrice, &downloaded)
			hldPDBD := price_data_by_download[downloaded]
			hldPDBD.Data = append(hldPDBD.Data, scSum)
			price_data_by_download[downloaded] = hldPDBD

			overallMedianMap[float64(scSum.Price)] += uint64(scSum.QuantityAtPrice)
		}
		prcMapRows.Close()
	}

	for key, value := range price_data_by_download {
		vHld := value
		medianMap := make(map[float64]uint64)
		for _, item := range value.Data {
			medianMap[float64(item.Price)] = uint64(item.QuantityAtPrice)
		}
		if median, medianErr := util.MedianFromMap(medianMap); medianErr == nil {
			vHld.MedianValue = median
		}
		price_data_by_download[key] = vHld
	}

	var return_value AuctionSummaryData
	return_value.Min = min_value
	return_value.Max = max_value
	return_value.Avg = avg_value
	return_value.PriceMap = price_data_by_download

	// Get spot auctions
	spotSummary, err := ahs.getSpotAuctionSummary(ctx, item, realm, region, bonuses)
	if err == nil && len(spotSummary.Data) > 0 {
		cTime := time.Now()
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

	// Calculate overall median
	overallMedianMap := make(map[float64]uint64)
	for _, summary := range return_value.PriceMap {
		for _, data := range summary.Data {
			overallMedianMap[float64(data.Price)] += uint64(data.QuantityAtPrice)
		}
	}
	if median, medianErr := util.MedianFromMap(overallMedianMap); medianErr == nil {
		return_value.Med = median
	}

	return return_value, nil
}

// Get a current auction spot summary from the internet
func (ahs *AuctionHistoryServer) getSpotAuctionSummary(ctx context.Context, item globalTypes.ItemSoftIdentity, realm globalTypes.ConnectedRealmSoftIentity, region globalTypes.RegionCode, bonuses []uint) (AuctionPriceSummaryRecord, error) {
	var realm_get uint
	if realm.Id != 0 {
		realm_get = uint(realm.Id)
	} else if realm.Name != "" {
		rlm, err := ahs.helper.GetConnectedRealmId(ctx, realm.Name, region)
		if err != nil {
			return AuctionPriceSummaryRecord{}, fmt.Errorf("no realm found with %s", realm.Name)
		}
		realm_get = uint(rlm)
	} else {
		return AuctionPriceSummaryRecord{}, fmt.Errorf("realm %v could not be found", realm)
	}

	ah, err := ahs.helper.GetAuctionHouse(ctx, globalTypes.ConnectedRealmID(realm_get), region)
	if err != nil {
		return AuctionPriceSummaryRecord{}, err
	}

	var item_id uint
	if item.ItemId != 0 {
		item_id = uint(item.ItemId)
	} else if item.ItemName != "" {
		itm, err := ahs.helper.GetItemId(ctx, region, item.ItemName)
		if err != nil {
			return AuctionPriceSummaryRecord{}, fmt.Errorf("could not find item for %v", item)
		}
		item_id = uint(itm)
	} else {
		return AuctionPriceSummaryRecord{}, fmt.Errorf("could not find item for %v", item)
	}

	var auction_set []BlizzardApi.Auction
	for _, auction := range ah.Auctions {
		if auction.Item.Id == globalTypes.ItemID(item_id) {
			if len(bonuses) == 0 || checkBonus(bonuses, auction.Item.Bonus_lists) {
				auction_set = append(auction_set, auction)
			}
		}
	}

	if len(auction_set) == 0 {
		return AuctionPriceSummaryRecord{}, nil
	}

	return_value := AuctionPriceSummaryRecord{
		MinValue: math.MaxUint,
	}

	var total_price, total_sales uint
	price_map := make(map[uint]struct {
		Quantity uint
		Sales    uint
	})

	for _, auction := range auction_set {
		var price uint
		if auction.Unit_price != 0 {
			price = auction.Unit_price
		} else if auction.Buyout != 0 {
			price = auction.Buyout
		} else {
			price = auction.Bid
		}

		if price == 0 {
			continue
		}

		if price > return_value.MaxValue {
			return_value.MaxValue = price
		}
		if price < return_value.MinValue {
			return_value.MinValue = price
		}
		total_sales += auction.Quantity
		total_price += price * auction.Quantity

		pmh := price_map[price]
		pmh.Quantity += auction.Quantity
		pmh.Sales += 1
		price_map[price] = pmh
	}

	if total_sales > 0 {
		return_value.AvgValue = float64(total_price) / float64(total_sales)
	}
	
	medianCollect := make(map[float64]uint64)
	for price, price_lu := range price_map {
		return_value.Data = append(return_value.Data, SalesCountSummary{
			Price:           price,
			QuantityAtPrice: price_lu.Quantity,
			SalesAtPrice:    price_lu.Sales,
		})
		medianCollect[float64(price)] = uint64(price_lu.Quantity)
	}

	if median, medErr := util.MedianFromMap(medianCollect); medErr == nil {
		return_value.MedianValue = median
	}

	return return_value, nil
}

// Check that the bonus list matches the target item listing
func checkBonus(bonus_list []uint, target []uint) bool {
	if len(bonus_list) == 0 {
		return true
	}
	if len(target) == 0 {
		return false
	}
	for _, list_entry := range bonus_list {
		if !slices.Contains(target, list_entry) {
			return false
		}
	}
	return true
}
