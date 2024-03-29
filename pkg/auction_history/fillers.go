package auction_history

import (
	"context"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/static_sources"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
)

// Fill in fillCount items into the database
func (ahs *AuctionHistoryServer) FillNItems(fillCount uint, static_source *static_sources.StaticSources) {
	const (
		select_sql string = "SELECT item_id, region FROM items WHERE scanned = false LIMIT $1"
		update_sql string = "UPDATE items SET name = $1, craftable = $2, scanned = true WHERE item_id = $3 AND region = $4"
		delete_sql string = "DELETE FROM items WHERE item_id = $1 AND region = $2"
	)

	ahs.logger.Infof(`Filling %d items with details.`, fillCount)

	rows, rowsErr := ahs.db.Query(context.TODO(), select_sql, fillCount)
	if rowsErr != nil {
		panic(rowsErr)
	}
	defer rows.Close()

	tranaction, tErr := ahs.db.Begin(context.TODO())
	if tErr != nil {
		panic(tErr)
	}

	for rows.Next() {
		var (
			item_id uint
			region  string
		)
		rows.Scan(&item_id, &region)

		safe := true

		fetchedItem, fetchErr := ahs.helper.GetItemDetails(item_id, region)
		if fetchErr != nil {
			safe = false
		}
		isCraftable, craftErr := ahs.helper.CheckIsCrafting(item_id, globalTypes.ALL_PROFESSIONS, region, static_source)
		if craftErr != nil {
			safe = false
		}

		if safe {
			_, updateErr := tranaction.Exec(context.TODO(), update_sql, fetchedItem.Name, isCraftable.Craftable, item_id, region)
			if updateErr != nil {
				tranaction.Rollback(context.TODO())
				panic(updateErr)
			}
			ahs.logger.Debugf(`Updated item: %d:%s with name: '%s' and craftable: %t`, item_id, region, fetchedItem.Name, isCraftable.Craftable)
		} else {
			ahs.logger.Errorf(`Issue filling %d in %s. Skipping`, item_id, region)
			tranaction.Exec(context.TODO(), delete_sql, item_id, region)
			ahs.logger.Errorf(`DELETED %d in %s from items table.`, item_id, region)
		}
	}
	tranaction.Commit(context.TODO())
}

// Fill in fillCount names into the database
func (ahs *AuctionHistoryServer) FillNNames(fillCount uint) {
	ahs.logger.Infof(`Filling %d unnamed item names.`, fillCount)
	const (
		select_sql      string = "SELECT item_id, region FROM items WHERE name ISNULL ORDER BY item_id DESC LIMIT $1"
		update_sql      string = "UPDATE items SET name = $1 WHERE item_id = $2 AND region = $3"
		delete_item_sql string = "DELETE FROM items WHERE item_id = $1 AND region = $2"
	)

	rows, rowErr := ahs.db.Query(context.TODO(), select_sql, fillCount)
	if rowErr != nil {
		panic(rowErr)
	}
	defer rows.Close()

	transaction, err := ahs.db.Begin(context.TODO())
	if err != nil {
		panic(err)
	}

	for rows.Next() {
		var (
			item_id uint
			region  string
		)
		rows.Scan(&item_id, &region)
		fetchedItem, fetchErr := ahs.helper.GetItemDetails(item_id, region)
		if fetchErr != nil {
			ahs.logger.Errorf(`Issue filling %d in %s. Skipping: %v`, item_id, region, fetchErr)
			_, delErr := transaction.Exec(context.TODO(), delete_item_sql, item_id, region)
			if delErr != nil {
				transaction.Rollback(context.TODO())
				panic(delErr)
			}
			ahs.logger.Errorf(`DELETED %d in %s from items table.`, item_id, region)
		} else {
			_, updateErr := transaction.Exec(context.TODO(), update_sql, fetchedItem.Name, item_id, region)
			if updateErr != nil {
				transaction.Rollback(context.TODO())
				panic(updateErr)
			}
			ahs.logger.Debugf(`Updated item: %d:%s with name: '%s'`, item_id, region, fetchedItem.Name)
		}
	}
	transaction.Commit(context.TODO())
}
