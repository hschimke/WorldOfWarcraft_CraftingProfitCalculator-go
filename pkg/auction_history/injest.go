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
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// Injest a realm for auction archives
func ingest(region globalTypes.RegionCode, connected_realm globalTypes.ConnectedRealmID, dbpool *pgxpool.Pool, async bool) error {
	type lItm struct {
		ItemId     globalTypes.ItemID
		BonusLists []uint
		Price      uint
		Quantity   uint
	}
	items := make(map[string]map[uint]lItm)

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
				items[key][pk].ItemId, items[key][pk].Quantity, items[key][pk].Price, fetchTime, connected_realm, bonusListString, strings.ToLower(region),
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
