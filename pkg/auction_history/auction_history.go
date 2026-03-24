package auction_history

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes/BlizzardApi"
)

type ScanRealmsResult struct {
	RealmNames string                       `json:"realm_names,omitempty"`
	RealmId    globalTypes.ConnectedRealmID `json:"realm_id,omitempty"`
	Region     globalTypes.RegionCode       `json:"region,omitempty"`
}

type GetAllBonusesReturn struct {
	Bonuses [][]uint         `json:"bonuses,omitempty"`
	Item    BlizzardApi.Item `json:"item"`
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
	Latest   time.Time                               `json:"latest"`
	PriceMap map[time.Time]AuctionPriceSummaryRecord `json:"price_map,omitempty"`
	Archives []struct {
		Timestamp   time.Time           `json:"timestamp"`
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

// Injest all the realms in the scan list
func (ahs *AuctionHistoryServer) ScanRealms(ctx context.Context, async bool) error {
	const sql string = "SELECT connected_realm_id, region FROM realm_scan_list"

	realms, err := ahs.db.Query(ctx, sql)
	if err != nil {
		ahs.logger.Errorf("Unable to query database: %v", err)
		return err
	}
	defer realms.Close()

	for realms.Next() {
		var (
			connected_realm_id uint
			region             string
		)
		realms.Scan(&connected_realm_id, &region)
		ingestErr := ahs.ingest(ctx, globalTypes.RegionCode(region), globalTypes.ConnectedRealmID(connected_realm_id), async)
		if ingestErr != nil {
			return ingestErr
		}
	}

	return nil
}

// Add a realm for historic price data scanning
func (ahs *AuctionHistoryServer) AddScanRealm(ctx context.Context, realm globalTypes.ConnectedRealmSoftIentity, region globalTypes.RegionCode) error {
	const sql string = "INSERT INTO realm_scan_list(connected_realm_id,region,connected_realm_names) VALUES($1,$2,$3)"

	var newRealmId uint

	if realm.Id != 0 {
		newRealmId = uint(realm.Id)
	} else if realm.Name != "" {
		fetchRealmId, fetchRealmIdErr := ahs.helper.GetConnectedRealmId(ctx, realm.Name, region)
		if fetchRealmIdErr != nil {
			return fmt.Errorf("could not get realm: %w", fetchRealmIdErr)
		}
		if fetchRealmId == 0 {
			return fmt.Errorf("could not get realm ID for %s", realm.Name)
		}
		newRealmId = uint(fetchRealmId)
	} else {
		return fmt.Errorf("no realm information provided")
	}

	fetchRealm, fetchRealmErr := ahs.helper.GetBlizConnectedRealmDetail(ctx, globalTypes.ConnectedRealmID(newRealmId), region)
	if fetchRealmErr != nil {
		return fmt.Errorf("could not get realm details: %w", fetchRealmErr)
	}

	var realmNames []string
	for _, server := range fetchRealm.Realms {
		realmNames = append(realmNames, server.Name)
	}

	_, execErr := ahs.db.Exec(ctx, sql, newRealmId, strings.ToLower(string(region)), strings.Join(realmNames, ", "))
	if execErr != nil {
		ahs.logger.Errorf(`Couldn't add %v in %s to scan realms table: %v.`, realm, region, execErr)
		return execErr
	}

	return nil
}

// Remove a realm from the history scan list
func (ahs *AuctionHistoryServer) RemoveScanRealm(ctx context.Context, realm globalTypes.ConnectedRealmSoftIentity, region globalTypes.RegionCode) error {
	const sql string = "DELETE FROM realm_scan_list WHERE connected_realm_id = $1 AND region = $2"

	var newRealmId uint

	if realm.Id != 0 {
		newRealmId = uint(realm.Id)
	} else if len(realm.Name) > 0 {
		fetchRealmId, fetchRealmIdErr := ahs.helper.GetConnectedRealmId(ctx, realm.Name, region)
		if fetchRealmIdErr != nil {
			return fmt.Errorf("could not get realm: %w", fetchRealmIdErr)
		}
		newRealmId = uint(fetchRealmId)
	} else {
		return fmt.Errorf("no realm information provided")
	}

	_, execErr := ahs.db.Exec(ctx, sql, newRealmId, strings.ToLower(string(region)))
	return execErr
}

// Return all bonuses availble for an item
func (ahs *AuctionHistoryServer) GetAllBonuses(ctx context.Context, item globalTypes.ItemSoftIdentity, region globalTypes.RegionCode) (GetAllBonusesReturn, error) {
	ahs.logger.Debugf(`Fetching bonuses for %v`, item)

	const sql string = "SELECT DISTINCT bonuses FROM auctions WHERE item_id = $1"

	var searchId uint
	if item.ItemId != 0 {
		searchId = item.ItemId
	} else if item.ItemName != "" {
		itemId, idErr := ahs.helper.GetItemId(ctx, region, item.ItemName)
		if idErr != nil {
			return GetAllBonusesReturn{}, idErr
		}
		searchId = uint(itemId)
	} else {
		return GetAllBonusesReturn{}, fmt.Errorf("no item information provided")
	}

	var return_value GetAllBonusesReturn

	fetchedItem, err := ahs.helper.GetItemDetails(ctx, globalTypes.ItemID(searchId), region)
	if err != nil {
		return GetAllBonusesReturn{}, err
	}

	return_value.Item.Id = globalTypes.ItemID(searchId)
	return_value.Item.Name = item.ItemName
	return_value.Item.Level = fetchedItem.Level

	rows, rowErr := ahs.db.Query(ctx, sql, searchId)
	if rowErr != nil {
		return GetAllBonusesReturn{}, rowErr
	}
	defer rows.Close()

	for rows.Next() {
		var (
			bonusString    string
			arrayOfBonuses []uint
		)

		if err := rows.Scan(&bonusString); err != nil {
			return GetAllBonusesReturn{}, err
		}

		if err := json.Unmarshal([]byte(bonusString), &arrayOfBonuses); err != nil {
			return GetAllBonusesReturn{}, err
		}

		return_value.Bonuses = append(return_value.Bonuses, arrayOfBonuses)
	}

	ahs.logger.Debugf(`Found %d bonuses for %v`, len(return_value.Bonuses), item)

	return return_value, nil
}

// Archive auctions, in this implementation it generally just deletes old auctions
func (ahs *AuctionHistoryServer) ArchiveAuctions(ctx context.Context) {
	twoWeeksAgo := time.Now().Add(time.Hour * (-1 * 24) * 14)

	const sql string = "DELETE FROM auctions WHERE downloaded < $1"

	ahs.db.Exec(ctx, sql, twoWeeksAgo)
}
