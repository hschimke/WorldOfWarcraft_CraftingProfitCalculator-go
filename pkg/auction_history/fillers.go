package auction_history

import (
	"context"
	"fmt"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/static_sources"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
)

// Fill in fillCount items into the database
func (ahs *AuctionHistoryServer) FillNItems(ctx context.Context, fillCount uint, static_source *static_sources.StaticSources) error {
	const (
		select_sql string = "SELECT item_id, region FROM items WHERE scanned = false LIMIT $1"
		update_sql string = "UPDATE items SET name = $1, craftable = $2, scanned = true WHERE item_id = $3 AND region = $4"
		delete_sql string = "DELETE FROM items WHERE item_id = $1 AND region = $2"
	)

	ahs.logger.Infof(`Filling %d items with details.`, fillCount)

	rows, rowsErr := ahs.db.Query(ctx, select_sql, fillCount)
	if rowsErr != nil {
		return fmt.Errorf("failed to query items to fill: %w", rowsErr)
	}

	type itemToProcess struct {
		item_id uint
		region  string
	}
	var items []itemToProcess

	for rows.Next() {
		var i itemToProcess
		if err := rows.Scan(&i.item_id, &i.region); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan item: %w", err)
		}
		items = append(items, i)
	}
	rows.Close()

	type processResult struct {
		item      itemToProcess
		safe      bool
		name      string
		craftable bool
	}

	var results []processResult

	for _, i := range items {
		safe := true
		fetchedItem, fetchErr := ahs.helper.GetItemDetails(ctx, globalTypes.ItemID(i.item_id), globalTypes.RegionCode(i.region))
		if fetchErr != nil {
			safe = false
		}
		isCraftable, craftErr := ahs.helper.CheckIsCrafting(ctx, globalTypes.ItemID(i.item_id), globalTypes.ALL_PROFESSIONS, globalTypes.RegionCode(i.region), static_source)
		if craftErr != nil {
			safe = false
		}

		results = append(results, processResult{
			item:      i,
			safe:      safe,
			name:      fetchedItem.Name,
			craftable: isCraftable.Craftable,
		})
	}

	if len(results) == 0 {
		return nil
	}

	transaction, tErr := ahs.db.Begin(ctx)
	if tErr != nil {
		return fmt.Errorf("failed to begin transaction: %w", tErr)
	}
	defer transaction.Rollback(ctx)

	for _, res := range results {
		if res.safe {
			_, updateErr := transaction.Exec(ctx, update_sql, res.name, res.craftable, res.item.item_id, res.item.region)
			if updateErr != nil {
				return fmt.Errorf("failed to update item %d: %w", res.item.item_id, updateErr)
			}
			ahs.logger.Debugf(`Updated item: %d:%s with name: '%s' and craftable: %t`, res.item.item_id, res.item.region, res.name, res.craftable)
		} else {
			ahs.logger.Errorf(`Issue filling %d in %s. Skipping`, res.item.item_id, res.item.region)
			if _, delErr := transaction.Exec(ctx, delete_sql, res.item.item_id, res.item.region); delErr != nil {
				return fmt.Errorf("failed to delete item %d: %w", res.item.item_id, delErr)
			}
			ahs.logger.Errorf(`DELETED %d in %s from items table.`, res.item.item_id, res.item.region)
		}
	}

	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// Fill in fillCount names into the database
func (ahs *AuctionHistoryServer) FillNNames(ctx context.Context, fillCount uint) error {
	ahs.logger.Infof(`Filling %d unnamed item names.`, fillCount)
	const (
		select_sql      string = "SELECT item_id, region FROM items WHERE name ISNULL ORDER BY item_id DESC LIMIT $1"
		update_sql      string = "UPDATE items SET name = $1 WHERE item_id = $2 AND region = $3"
		delete_item_sql string = "DELETE FROM items WHERE item_id = $1 AND region = $2"
	)

	rows, rowErr := ahs.db.Query(ctx, select_sql, fillCount)
	if rowErr != nil {
		return fmt.Errorf("failed to query unnamed items: %w", rowErr)
	}

	type itemToProcess struct {
		item_id uint
		region  string
	}
	var items []itemToProcess

	for rows.Next() {
		var i itemToProcess
		if err := rows.Scan(&i.item_id, &i.region); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan unnamed item: %w", err)
		}
		items = append(items, i)
	}
	rows.Close()

	type processResult struct {
		item itemToProcess
		safe bool
		name string
		err  error
	}

	var results []processResult

	for _, i := range items {
		fetchedItem, fetchErr := ahs.helper.GetItemDetails(ctx, globalTypes.ItemID(i.item_id), globalTypes.RegionCode(i.region))
		results = append(results, processResult{
			item: i,
			safe: fetchErr == nil,
			name: fetchedItem.Name,
			err:  fetchErr,
		})
	}

	if len(results) == 0 {
		return nil
	}

	transaction, err := ahs.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer transaction.Rollback(ctx)

	for _, res := range results {
		if !res.safe {
			ahs.logger.Errorf(`Issue filling %d in %s. Skipping: %v`, res.item.item_id, res.item.region, res.err)
			if _, delErr := transaction.Exec(ctx, delete_item_sql, res.item.item_id, res.item.region); delErr != nil {
				return fmt.Errorf("failed to delete item %d: %w", res.item.item_id, delErr)
			}
			ahs.logger.Errorf(`DELETED %d in %s from items table.`, res.item.item_id, res.item.region)
		} else {
			if _, updateErr := transaction.Exec(ctx, update_sql, res.name, res.item.item_id, res.item.region); updateErr != nil {
				return fmt.Errorf("failed to update item %d: %w", res.item.item_id, updateErr)
			}
			ahs.logger.Debugf(`Updated item: %d:%s with name: '%s'`, res.item.item_id, res.item.region, res.name)
		}
	}

	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}
