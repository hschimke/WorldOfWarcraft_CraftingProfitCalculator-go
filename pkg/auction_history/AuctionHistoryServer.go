package auction_history

import (
	"context"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/blizzard_api_helpers"
	"github.com/jackc/pgx/v4/pgxpool"
)

type AuctionHistoryServer struct {
	helper           *blizzard_api_helpers.BlizzardApiHelper
	connectionString string
	logger           *cpclog.CpCLog
	ctx              context.Context
	db               *pgxpool.Pool
}

func NewAuctionHistoryServer(ctx context.Context, connectionString string, helper *blizzard_api_helpers.BlizzardApiHelper, logger *cpclog.CpCLog) *AuctionHistoryServer {
	ahs := AuctionHistoryServer{
		helper:           helper,
		connectionString: connectionString,
		logger:           logger,
		ctx:              ctx,
	}
	var dbErr error
	ahs.db, dbErr = pgxpool.Connect(ahs.ctx, ahs.connectionString)
	if dbErr != nil {
		panic(dbErr.Error())
	}
	ahs.dbSetup()
	return &ahs
}

func (ahs *AuctionHistoryServer) Shutdown() {
	ahs.db.Close()
}

func (ahs *AuctionHistoryServer) dbSetup() {
	const (
		sql_create_item_table            string = "CREATE TABLE IF NOT EXISTS auctions (item_id BIGINT NOT NULL, bonuses JSONB, quantity BIGINT NOT NULL, price BIGINT NOT NULL, downloaded TIMESTAMP WITH TIME ZONE NOT NULL, connected_realm_id BIGINT NOT NULL, region TEXT NOT NULL)"
		sql_create_items_table           string = "CREATE TABLE IF NOT EXISTS items (item_id BIGINT NOT NULL, region TEXT NOT NULL, name TEXT, craftable BOOLEAN, scanned BOOLEAN, PRIMARY KEY (item_id,region))"
		sql_create_realm_scan_table      string = "CREATE TABLE IF NOT EXISTS realm_scan_list (connected_realm_id BIGINT NOT NULL, connected_realm_names TEXT, region TEXT NOT NULL, PRIMARY KEY (connected_realm_id,region))"
		sql_create_archive_table         string = "CREATE TABLE IF NOT EXISTS auction_archive (item_id BIGINT NOT NULL, bonuses JSONB, quantity BIGINT, summary JSONB, downloaded BIGINT, connected_realm_id BIGINT, region TEXT)"
		sql_create_auction_archive_index string = "CREATE INDEX IF NOT EXISTS auction_archive_index ON auction_archive (item_id, downloaded, connected_realm_id, region)"
		sql_create_auctions_index        string = "CREATE INDEX IF NOT EXISTS auctions_index ON auctions (item_id, quantity, price, downloaded, connected_realm_id, region)"
		sql_create_auctions_bonuses_idx  string = "CREATE INDEX IF NOT EXISTS auctions_bonuses_idx ON auctions USING GIN (bonuses)"
		sql_create_items_name_ind        string = "CREATE INDEX IF NOT EXISTS items_name_index on items (name)"
	)

	dbpool, err := ahs.db.Acquire(ahs.ctx) //pgxpool.Connect(context.Background(), ahs.connectionString)
	if err != nil {
		ahs.logger.Errorf("Unable to connect to database: %v", err)
		panic(err)
	}
	defer dbpool.Release()

	dbpool.Exec(context.TODO(), sql_create_item_table)
	dbpool.Exec(context.TODO(), sql_create_items_table)
	dbpool.Exec(context.TODO(), sql_create_realm_scan_table)
	dbpool.Exec(context.TODO(), sql_create_archive_table)
	dbpool.Exec(context.TODO(), sql_create_auction_archive_index)
	dbpool.Exec(context.TODO(), sql_create_auctions_index)
	dbpool.Exec(context.TODO(), sql_create_auctions_bonuses_idx)
	dbpool.Exec(context.TODO(), sql_create_items_name_ind)
}
