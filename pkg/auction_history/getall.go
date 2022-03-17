package auction_history

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
)

// Get a list of all scanned realms
func (ahs *AuctionHistoryServer) GetScanRealms() ([]ScanRealmsResult, error) {
	dbpool, err := pgxpool.Connect(context.Background(), ahs.connectionString)
	if err != nil {
		ahs.logger.Errorf("Unable to connect to database: %v", err)
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
		var scr ScanRealmsResult
		realms.Scan(&scr.RealmId, &scr.Region, &scr.RealmNames)
		result = append(result, scr)
	}

	return result, nil
}

// Get all the names available, filtering if availble
func (ahs *AuctionHistoryServer) GetAllNames() []string {
	dbpool, err := pgxpool.Connect(context.Background(), ahs.connectionString)
	if err != nil {
		ahs.logger.Errorf("Unable to connect to database: %v", err)
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
