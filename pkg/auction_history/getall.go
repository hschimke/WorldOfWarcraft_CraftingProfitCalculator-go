package auction_history

import (
	"context"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/environment_variables"
	"github.com/jackc/pgx/v4/pgxpool"
)

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
