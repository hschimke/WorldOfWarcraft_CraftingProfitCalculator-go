package routes

import (
	"encoding/json"
	"net/http"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/auction_history"
)

func ScannedRealms(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	gsc, err := auction_history.GetScanRealms()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(gsc)
	}
}

func AllItems(w http.ResponseWriter, r *http.Request) {}

func AuctionHistory(w http.ResponseWriter, r *http.Request) {}

func SeenItemBonuses(w http.ResponseWriter, r *http.Request) {}
