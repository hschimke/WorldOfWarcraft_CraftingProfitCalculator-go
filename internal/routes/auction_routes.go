package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/auction_history"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
)

func ScannedRealms(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	gsc, err := auction_history.GetScanRealms()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		//		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(gsc)
	}
}

func AllItems(w http.ResponseWriter, r *http.Request) {}

func AuctionHistory(w http.ResponseWriter, r *http.Request) {
	type expectedBody struct {
		Item     string `json:"item"`
		Realm    string `json:"realm"`
		Region   string `json:"region"`
		Bonuses  []uint `json:"bonuses"`
		StartDtm string `json:"start_dtm"`
		EndDtm   string `json:"end_dtm"`
	}

	if r.Body == nil {
		http.Error(w, "request body required", http.StatusBadRequest)
		return
	}
	var data expectedBody
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cpclog.Infof(`AuctionHistory request for item: %s, realm: %s, region: %s, bonuses: %v, start_dtm: %s, end_dtm: %s`, data.Item, data.Realm, data.Region, data.Bonuses, data.StartDtm, data.EndDtm)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	item := globalTypes.NewItemFromString(data.Item)
	realm := globalTypes.NewRealmFromString(data.Realm)

	startTime, err := time.Parse(time.UnixDate, data.StartDtm)
	if err != nil {
		startTime = time.Now().AddDate(-2, 0, 0)
	}
	endTime, err := time.Parse(time.UnixDate, data.EndDtm)
	if err != nil {
		endTime = time.Now()
	}

	auctionData, auctionDataError := auction_history.GetAuctions(item, realm, data.Region, data.Bonuses, startTime, endTime)
	if auctionDataError != nil {
		cpclog.Error("Issue getting auctions ", auctionDataError)
		fmt.Fprintf(w, "{ ERROR: %v }", auctionDataError)
		return
	}

	cpclog.Debug("returned auction data")
	json.NewEncoder(w).Encode(auctionData)
}

func SeenItemBonuses(w http.ResponseWriter, r *http.Request) {}
