package auction_history

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/environment_variables"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/blizzard_api_helpers"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes/BlizzardApi"

	"github.com/jackc/pgx/v4/pgxpool"
)

type ScanRealmsResult struct {
	RealmNames string                       `json:"realm_names,omitempty"`
	RealmId    globalTypes.ConnectedRealmID `json:"realm_id,omitempty"`
	Region     globalTypes.RegionCode       `json:"region,omitempty"`
}

type GetAllBonusesReturn struct {
	Bonuses [][]uint         `json:"bonuses,omitempty"`
	Item    BlizzardApi.Item `json:"item,omitempty"`
}

type AuctionPriceSummaryRecord struct {
	Data        []SalesCountSummary `json:"data,omitempty"`
	MinValue    uint                `json:"min_value,omitempty"`
	MaxValue    uint                `json:"max_value,omitempty"`
	AvgValue    float64             `json:"avg_value,omitempty"`
	MedianValue float64             `json:"med_value,omitempty"`
}

type SalesCountSummary struct {
	SalesAtPrice    uint `json:"sales_at_price,omitempty"`
	QuantityAtPrice uint `json:"quantity_at_price,omitempty"`
	Price           uint `json:"price,omitempty"`
}

type AuctionSummaryData struct {
	Min      uint                                    `json:"min,omitempty"`
	Max      uint                                    `json:"max,omitempty"`
	Avg      float64                                 `json:"avg,omitempty"`
	Med      float64                                 `json:"med,omitempty"`
	Latest   time.Time                               `json:"latest,omitempty"`
	PriceMap map[time.Time]AuctionPriceSummaryRecord `json:"price_map,omitempty"`
	Archives []struct {
		Timestamp   time.Time           `json:"timestamp,omitempty"`
		Data        []SalesCountSummary `json:"data,omitempty"`
		MinValue    uint                `json:"min_value,omitempty"`
		MaxValue    uint                `json:"max_value,omitempty"`
		AvgValue    float64             `json:"avg_value,omitempty"`
		MedianValue float64             `json:"median_value,omitempty"`
	} `json:"archives"`
}

type localItem struct {
	ItemName  string
	ItemId    uint
	Region    globalTypes.RegionCode
	Craftable *bool
}

func init() {
	const (
		sql_create_item_table            string = "CREATE TABLE IF NOT EXISTS auctions (item_id NUMERIC, bonuses TEXT, quantity NUMERIC, price NUMERIC, downloaded TIMESTAMP WITH TIME ZONE, connected_realm_id NUMERIC, region TEXT)"
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

// Add a realm for historic price data scanning
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

// Remove a realm from the history scan list
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

	dbpool.Exec(context.TODO(), sql, twoWeeksAgo)
}
