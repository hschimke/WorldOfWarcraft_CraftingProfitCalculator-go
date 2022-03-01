package auction_history

import (
	"context"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/environment_variables"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/blizzard_api_helpers"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
	"github.com/jackc/pgx/v4/pgxpool"
)

// Fill in fillCount items into the database
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
