package auction_history

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/environment_variables"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/blizzard_api_helpers"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes/BlizzardApi"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
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
	Names            []globalTypes.RealmName
	ConnectedRealmId globalTypes.ConnectedRealmID
	Region           globalTypes.RegionCode
}

type localItem struct {
	ItemName  string
	ItemId    uint
	Region    globalTypes.RegionCode
	Craftable *bool
}

func init() {
	const (
		sql_create_item_table            string = "CREATE TABLE IF NOT EXISTS auctions (item_id NUMERIC, bonuses TEXT, quantity NUMERIC, price NUMERIC, downloaded NUMERIC, connected_realm_id NUMERIC, region TEXT)"
		sql_create_items_table           string = "CREATE TABLE IF NOT EXISTS items (item_id NUMERIC, region TEXT, name TEXT, craftable BOOLEAN, scanned BOOLEAN, PRIMARY KEY (item_id,region))"
		sql_create_realm_scan_table      string = "CREATE TABLE IF NOT EXISTS realm_scan_list (connected_realm_id NUMERIC, connected_realm_names TEXT, region TEXT, PRIMARY KEY (connected_realm_id,region))"
		sql_create_archive_table         string = "CREATE TABLE IF NOT EXISTS auction_archive (item_id NUMERIC, bonuses TEXT, quantity NUMERIC, summary JSON, downloaded NUMERIC, connected_realm_id NUMERIC, region TEXT)"
		sql_create_auction_archive_index string = "CREATE INDEX IF NOT EXISTS auction_archive_index ON auction_archive (item_id, bonuses, downloaded, connected_realm_id, region)"
		sql_create_auctions_index        string = "CREATE INDEX IF NOT EXISTS auctions_index ON auctions (item_id, bonuses, quantity, price, downloaded, connected_realm_id, region)"
		sql_create_items_name_ind        string = "CREATE INDEX IF NOT EXISTS items_name_index on items (name)"
	)

	dbpool, err := pgxpool.Connect(context.Background(), environment_variables.DATABASE_CONNECTION_STRING)
	if err != nil {
		cpclog.Errorf("Unable to connect to database: %v", err)
		panic(err)
	}
	defer dbpool.Close()

	dbpool.Exec(context.TODO(), sql_create_item_table)
	dbpool.Exec(context.TODO(), sql_create_items_table)
	dbpool.Exec(context.TODO(), sql_create_realm_scan_table)
	dbpool.Exec(context.TODO(), sql_create_archive_table)
	dbpool.Exec(context.TODO(), sql_create_auction_archive_index)
	dbpool.Exec(context.TODO(), sql_create_auctions_index)
	dbpool.Exec(context.TODO(), sql_create_items_name_ind)
}

// Injest all the realms in the scan list
func ScanRealms(async bool) error {
	const sql string = "SELECT connected_realm_id, region FROM realm_scan_list"

	dbpool, err := pgxpool.Connect(context.Background(), environment_variables.DATABASE_CONNECTION_STRING)
	if err != nil {
		cpclog.Errorf("Unable to connect to database: %v", err)
		return err
	}
	defer dbpool.Close()

	realms, err := dbpool.Query(context.TODO(), sql)
	if err != nil {
		cpclog.Errorf("Unable to query database: %v", err)
		return err
	}
	defer realms.Close()

	for realms.Next() {
		var (
			connected_realm_id uint
			region             string
		)
		realms.Scan(&connected_realm_id, &region)
		ingestErr := ingest(region, connected_realm_id, dbpool, async)
		if ingestErr != nil {
			return ingestErr
		}
	}

	return nil
}

func AddScanRealm(realm globalTypes.ConnectedRealmSoftIentity, region globalTypes.RegionCode) error {
	dbpool, err := pgxpool.Connect(context.Background(), environment_variables.DATABASE_CONNECTION_STRING)
	if err != nil {
		cpclog.Errorf("Unable to connect to database: %v", err)
		return err
	}
	defer dbpool.Close()

	const sql string = "INSERT INTO realm_scan_list(connected_realm_id,region,connected_realm_names) VALUES($1,$2,$3)"

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

	_, execErr := dbpool.Exec(context.TODO(), sql, newRealmId, strings.ToLower(region), strings.Join(realmNameComposite, ", "))
	if execErr != nil {
		cpclog.Errorf(`Couldn't add %v in %s to scan realms table: %v.`, realm, region, err)
		return execErr
	}

	return nil
}

func RemoveScanRealm(realm globalTypes.ConnectedRealmSoftIentity, region globalTypes.RegionCode) error {
	dbpool, err := pgxpool.Connect(context.Background(), environment_variables.DATABASE_CONNECTION_STRING)
	if err != nil {
		cpclog.Errorf("Unable to connect to database: %v", err)
		return err
	}
	defer dbpool.Close()

	const sql string = "DELETE FROM realm_scan_list WHERE connected_realm_id = $1 AND region = $2"

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

	_, execErr := dbpool.Exec(context.TODO(), sql, newRealmId, strings.ToLower(region))
	if execErr != nil {
		return execErr
	}

	return nil
}

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
	} else {
		// All items
	}

	if connectedRealmId != 0 {
		// Get specific realm
		sql_addins = append(sql_addins, fmt.Sprintf(`connected_realm_id = %s`, get_place_marker()))
		value_searches = append(value_searches, connectedRealmId)
	} else {
		// All realms
	}

	if len(region) > 0 {
		// Get specific region
		sql_addins = append(sql_addins, fmt.Sprintf(`region = %s`, get_place_marker()))
		value_searches = append(value_searches, strings.ToLower(region))
	} else {
		// All regions
	}

	// Include oldest fetch date time
	sql_addins = append(sql_addins, fmt.Sprintf(`downloaded >= %s`, get_place_marker()))
	value_searches = append(value_searches, start_dtm.Unix())

	// Include newest fetch date time
	sql_addins = append(sql_addins, fmt.Sprintf(`downloaded <= %s`, get_place_marker()))
	value_searches = append(value_searches, end_dtm.Unix())

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
	} else {
		// any bonuses or none
	}

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
		min_value, max_value, latest_dl_value uint
		avg_value                             float64
	)

	dbpool.QueryRow(context.TODO(), min_sql, value_searches...).Scan(&min_value)
	dbpool.QueryRow(context.TODO(), max_sql, value_searches...).Scan(&max_value)
	dbpool.QueryRow(context.TODO(), avg_sql, value_searches...).Scan(&avg_value)
	dbpool.QueryRow(context.TODO(), latest_dl_sql, value_searches...).Scan(&latest_dl_value)

	price_data_by_download := make(map[int64]AuctionPriceSummaryRecord)
	distRows, dErr := dbpool.Query(context.TODO(), distinct_download_sql, value_searches...)
	if dErr != nil {
		return AuctionSummaryData{}, dErr
	}
	defer distRows.Close()
	for distRows.Next() {
		var (
			downloaded int64
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
	return_value.PriceMap = make(map[int64]AuctionPriceSummaryRecord)

	return_value.Min = min_value
	return_value.Max = max_value
	return_value.Avg = avg_value
	return_value.PriceMap = price_data_by_download

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
	dbpool, err := pgxpool.Connect(context.Background(), environment_variables.DATABASE_CONNECTION_STRING)
	if err != nil {
		cpclog.Errorf("Unable to connect to database: %v", err)
		return GetAllBonusesReturn{}, err
	}
	defer dbpool.Close()

	cpclog.Debugf(`Fetching bonuses for %v`, item)

	const sql string = "SELECT DISTINCT bonuses FROM auctions WHERE item_id = $1"

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

	var return_value GetAllBonusesReturn

	fetchedItem, err := blizzard_api_helpers.GetItemDetails(searchId, region)
	if err != nil {
		return GetAllBonusesReturn{}, err
	}

	return_value.Item.Id = searchId
	return_value.Item.Name = item.ItemName
	return_value.Item.Level = fetchedItem.Level

	rows, rowErr := dbpool.Query(context.TODO(), sql, searchId)
	if rowErr != nil {
		return GetAllBonusesReturn{}, rowErr
	}
	defer rows.Close()

	for rows.Next() {
		var (
			bonusString    string
			arrayOfBonuses []uint
		)

		rows.Scan(&bonusString)

		jsonErr := json.Unmarshal([]byte(bonusString), &arrayOfBonuses)
		if jsonErr != nil {
			return GetAllBonusesReturn{}, jsonErr
		}

		return_value.Bonuses = append(return_value.Bonuses, arrayOfBonuses)
	}

	cpclog.Debugf(`Found %d bonuses for %v`, len(return_value.Bonuses), item)

	return return_value, nil
}

// Archive auctions, in this implementation it generally just deletes old auctions
func ArchiveAuctions() {
	dbpool, err := pgxpool.Connect(context.Background(), environment_variables.DATABASE_CONNECTION_STRING)
	if err != nil {
		cpclog.Errorf("Unable to connect to database: %v", err)
		panic(err)
	}
	defer dbpool.Close()

	twoWeeksAgo := time.Now().Add(time.Hour * (-1 * 24) * 14)

	const sql string = "DELETE FROM auctions WHERE downloaded < $1"

	dbpool.Exec(context.TODO(), sql, twoWeeksAgo.Unix())
}

// Fill in fill_count items into the database
func FillNItems(fillCount uint) {
	dbpool, err := pgxpool.Connect(context.Background(), environment_variables.DATABASE_CONNECTION_STRING)
	if err != nil {
		cpclog.Errorf("Unable to connect to database: %v", err)
		panic(err)
	}
	defer dbpool.Close()

	const (
		select_sql string = "SELECT item_id, region FROM items WHERE scanned = false LIMIT $1"
		update_sql string = "UPDATE items SET name = $1, craftable = $2, scanned = true WHERE item_id = $3 AND region = $4"
		delete_sql string = "DELETE FROM items WHERE item_id = $1 AND region = $2"
	)

	cpclog.Infof(`Filling %d items with details.`, fillCount)

	rows, rowsErr := dbpool.Query(context.TODO(), select_sql, fillCount)
	if rowsErr != nil {
		panic(rowsErr)
	}
	defer rows.Close()

	tranaction, tErr := dbpool.Begin(context.TODO())
	if tErr != nil {
		panic(tErr)
	}
	defer tranaction.Commit(context.TODO())

	for rows.Next() {
		var (
			item_id uint
			region  string
		)
		rows.Scan(&item_id, &region)

		safe := true

		fetchedItem, fetchErr := blizzard_api_helpers.GetItemDetails(item_id, region)
		if fetchErr != nil {
			safe = false
		}
		isCraftable, craftErr := blizzard_api_helpers.CheckIsCrafting(item_id, globalTypes.ALL_PROFESSIONS, region)
		if craftErr != nil {
			safe = false
		}

		if safe {
			_, updateErr := tranaction.Exec(context.TODO(), update_sql, fetchedItem.Name, isCraftable.Craftable, item_id, region)
			if updateErr != nil {
				tranaction.Rollback(context.TODO())
				panic(updateErr)
			}
			cpclog.Debugf(`Updated item: %d:%s with name: '%s' and craftable: %t`, item_id, region, fetchedItem.Name, isCraftable.Craftable)
		} else {
			cpclog.Errorf(`Issue filling %d in %s. Skipping`, item_id, region)
			tranaction.Exec(context.TODO(), delete_sql, item_id, region)
			cpclog.Errorf(`DELETED %d in %s from items table.`, item_id, region)
		}
	}
}

// Fill in fillCount names into the database
func FillNNames(fillCount uint) {
	dbpool, err := pgxpool.Connect(context.Background(), environment_variables.DATABASE_CONNECTION_STRING)
	if err != nil {
		cpclog.Errorf("Unable to connect to database: %v", err)
		panic(err)
	}
	defer dbpool.Close()

	cpclog.Infof(`Filling %d unnamed item names.`, fillCount)
	const (
		select_sql      string = "SELECT item_id, region FROM items WHERE name ISNULL ORDER BY item_id DESC LIMIT $1"
		update_sql      string = "UPDATE items SET name = $1 WHERE item_id = $2 AND region = $3"
		delete_item_sql string = "DELETE FROM items WHERE item_id = $1 AND region = $2"
	)

	rows, rowErr := dbpool.Query(context.TODO(), select_sql, fillCount)
	if rowErr != nil {
		panic(rowErr)
	}
	defer rows.Close()

	transaction, err := dbpool.Begin(context.TODO())
	if err != nil {
		panic(err)
	}
	defer transaction.Commit(context.TODO())

	for rows.Next() {
		var (
			item_id uint
			region  string
		)
		rows.Scan(&item_id, &region)
		fetchedItem, fetchErr := blizzard_api_helpers.GetItemDetails(item_id, region)
		if fetchErr != nil {
			cpclog.Errorf(`Issue filling %d in %s. Skipping: %v`, item_id, region, fetchErr)
			_, delErr := transaction.Exec(context.TODO(), delete_item_sql, item_id, region)
			if delErr != nil {
				transaction.Rollback(context.TODO())
				panic(delErr)
			}
			cpclog.Errorf(`DELETED %d in %s from items table.`, item_id, region)
		} else {
			_, updateErr := transaction.Exec(context.TODO(), update_sql, fetchedItem.Name, item_id, region)
			if updateErr != nil {
				transaction.Rollback(context.TODO())
				panic(updateErr)
			}
			cpclog.Debugf(`Updated item: %d:%s with name: '%s'`, item_id, region, fetchedItem.Name)
		}
	}
}

// Get a list of all scanned realms
func GetScanRealms() ([]ScanRealmsResult, error) {
	dbpool, err := pgxpool.Connect(context.Background(), environment_variables.DATABASE_CONNECTION_STRING)
	if err != nil {
		cpclog.Errorf("Unable to connect to database: %v", err)
		return []ScanRealmsResult{}, err
	}
	defer dbpool.Close()

	const sql string = "SELECT connected_realm_id, region, connected_realm_names FROM realm_scan_list"

	realms, realmErr := dbpool.Query(context.TODO(), sql)
	if realmErr != nil {
		return []ScanRealmsResult{}, realmErr
	}
	defer realms.Close()

	var result []ScanRealmsResult
	for realms.Next() {
		var (
			connected_realm_id            uint
			region, connected_realm_names string
		)
		realms.Scan(&connected_realm_id, &region, &connected_realm_names)
		result = append(result, ScanRealmsResult{
			RealmNames: connected_realm_names,
			RealmId:    connected_realm_id,
			Region:     region,
		})
	}

	return result, nil
}

// Get all the names available, filtering if availble
func GetAllNames() []string {
	dbpool, err := pgxpool.Connect(context.Background(), environment_variables.DATABASE_CONNECTION_STRING)
	if err != nil {
		cpclog.Errorf("Unable to connect to database: %v", err)
		panic(err)
	}
	defer dbpool.Close()

	const sql string = "SELECT DISTINCT name FROM items WHERE name NOTNULL"

	names, nameErr := dbpool.Query(context.TODO(), sql)
	if nameErr != nil {
		panic(nameErr)
	}
	defer names.Close()

	var return_value []string
	for names.Next() {
		var name string
		names.Scan(&name)
		if len(name) > 0 {
			return_value = append(return_value, name)
		}
	}

	return return_value
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
func ingest(region globalTypes.RegionCode, connected_realm globalTypes.ConnectedRealmID, dbpool *pgxpool.Pool, async bool) error {
	type lItm struct {
		ItemId     globalTypes.ItemID
		BonusLists []uint
		Price      uint
		Quantity   uint
	}
	var items map[string]map[uint]lItm
	items = make(map[string]map[uint]lItm)

	cpclog.Infof("start ingest for %v - %v", region, connected_realm)

	// Get Auctions
	auctions, auctionError := blizzard_api_helpers.GetAuctionHouse(connected_realm, region)
	if auctionError != nil {
		return auctionError
	}

	fetchTime := time.Now()

	for _, auction := range auctions.Auctions {
		item_id_key := fmt.Sprint(auction.Item.Id)
		if len(auction.Item.Bonus_lists) > 0 {
			if blstr, err := json.Marshal(auction.Item.Bonus_lists); err == nil {
				item_id_key += string(blstr)
			}
		}
		if _, present := items[item_id_key]; !present {
			items[item_id_key] = make(map[uint]lItm)
		}

		var price uint
		quantity := auction.Quantity

		if auction.Buyout != 0 {
			price = auction.Buyout
		} else {
			price = auction.Unit_price
		}

		if _, prcPres := items[item_id_key][price]; !prcPres {
			items[item_id_key][price] = lItm{
				ItemId:     auction.Item.Id,
				BonusLists: auction.Item.Bonus_lists,
				Price:      price,
				Quantity:   0,
			}
		}
		hld := items[item_id_key][price]
		hld.Quantity += quantity
		items[item_id_key][price] = hld
	}

	//const item_set: Set<number> = new Set();
	var insert_values_array [][]interface{}
	var item_set []localItem

	for key, itm := range items {
		for pk, r := range itm {
			item_set = append(item_set, localItem{
				ItemId: r.ItemId,
				Region: region,
			})
			//item_id, quantity, price, downloaded, connected_realm_id, bonuses
			var bonusListString string
			if bstr, jsonErr := json.Marshal(items[key][pk].BonusLists); jsonErr == nil {
				bonusListString = string(bstr)
			} else {
				bonusListString = "[]"
			}
			insert_values_array = append(insert_values_array, []interface{}{
				items[key][pk].ItemId, items[key][pk].Quantity, items[key][pk].Price, fetchTime.Unix(), connected_realm, bonusListString, strings.ToLower(region),
			})
		}
	}

	if async {
		go churnAuctionItemsOnInjest(item_set)
	} else {
		churnAuctionItemsOnInjest(item_set)
	}

	copyCount, copyErr := dbpool.CopyFrom(context.TODO(),
		pgx.Identifier{"auctions"},
		[]string{"item_id", "quantity", "price", "downloaded", "connected_realm_id", "bonuses", "region"},
		pgx.CopyFromRows(insert_values_array),
	)
	if copyErr != nil {
		return copyErr
	}

	cpclog.Infof("finished ingest of %d auctions for %v - %v", copyCount, region, connected_realm)
	return nil
}

func churnAuctionItemsOnInjest(items []localItem) {
	dbpool, err := pgxpool.Connect(context.Background(), environment_variables.DATABASE_CONNECTION_STRING)
	if err != nil {
		cpclog.Errorf("Unable to connect to database: %v", err)
		panic(err)
	}
	defer dbpool.Close()
	cpclog.Infof("start item churn for %d items", len(items))

	insertBatch := &pgx.Batch{}

	const sql_insert_item = "INSERT INTO items(item_id, region, name, craftable, scanned) VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING"

	// Churn Items
	for _, item := range items {
		insertBatch.Queue(sql_insert_item, item.ItemId, item.Region, nil, false, false)
	}

	batchRes := dbpool.SendBatch(context.TODO(), insertBatch)
	batchRes.Close()

	cpclog.Info("finished item churn")
}
