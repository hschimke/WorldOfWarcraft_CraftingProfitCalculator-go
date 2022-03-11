package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cache_provider"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/static_sources"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/util"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/auction_history"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
)

type mapped struct {
	Text    string  `json:"text,omitempty"`
	Parsed  []uint  `json:"parsed,omitempty"`
	Reduced *string `json:"reduced,omitempty"`
}

type SeenItemBonusesReturn struct {
	Bonuses   []map[string]string `json:"bonuses,omitempty"`
	Mapped    *[]mapped           `json:"mapped,omitempty"`
	Collected struct {
		ILvl []struct {
			Id    string `json:"id,omitempty"`
			Level int    `json:"level,omitempty"`
		} `json:"ilvl"`
		Socket []struct {
			Id      string `json:"id,omitempty"`
			Sockets *int   `json:"sockets,omitempty"`
		} `json:"socket"`
		Quality []struct {
			Id      string `json:"id,omitempty"`
			Quality *int   `json:"quality,omitempty"`
		} `json:"quality"`
		Unknown []string `json:"unknown"`
		Empty   bool     `json:"empty,omitempty"`
	} `json:"collected,omitempty"`
}

// Return a list of all realms being actively scanned for history
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

// Return a unique list of all items in the items table
func AllItems(w http.ResponseWriter, r *http.Request) {
	const (
		cacheNS  string = "AH-FUNCTIONS"
		cacheKey string = "ALL_ITEMS_NAMES"
	)

	cpclog.Debug("Getting all items")

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	found, err := cache_provider.CacheCheck(cacheNS, cacheKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "[]")
	}
	var names []string
	if found {
		cpclog.Debug("Cached all items found.")
		err := cache_provider.CacheGet(cacheNS, cacheKey, &names)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("X-CPC-ERROR", err.Error())
			fmt.Fprint(w, "[]")
		}
	} else {
		cpclog.Debug("Getting fresh all items.")
		names = auction_history.GetAllNames()
		cache_provider.CacheSet(cacheNS, cacheKey, names, time.Hour)
	}

	partial := r.URL.Query().Get("partial")

	filterd_names := util.FilterStringArray(names, partial, "items")
	json.NewEncoder(w).Encode(filterd_names)
}

// Perform a search for auction history data
func AuctionHistory(w http.ResponseWriter, r *http.Request) {
	type expectedBody struct {
		Item     string   `json:"item"`
		Realm    string   `json:"realm"`
		Region   string   `json:"region"`
		Bonuses  []string `json:"bonuses"`
		StartDtm string   `json:"start_dtm"`
		EndDtm   string   `json:"end_dtm"`
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

	auctionData, auctionDataError := auction_history.GetAuctions(item, realm, data.Region, util.ParseStringArrayToUint(data.Bonuses), startTime, endTime)
	if auctionDataError != nil {
		cpclog.Error("Issue getting auctions ", auctionDataError)
		json.NewEncoder(w).Encode(globalTypes.ReturnError{ERROR: auctionDataError.Error()})
		return
	}

	cpclog.Debug("returned auction data")
	json.NewEncoder(w).Encode(auctionData)
}

// Return a list of all the bonuses seen for an item
func SeenItemBonuses(w http.ResponseWriter, r *http.Request) {
	type seenItemBonusesData struct {
		Item   string `json:"item,omitempty"`
		Region string `json:"region,omitempty"`
	}

	if r.Body == nil {
		http.Error(w, "request body required", http.StatusBadRequest)
		return
	}
	var data seenItemBonusesData
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	cpclog.Debugf(`Getting seen bonus lists for %s in %s`, data.Item, data.Region)

	if data.Item == "" {
		json.NewEncoder(w).Encode(globalTypes.ReturnError{ERROR: "empty item"})
		return
	}

	bonuses, allBonusesErr := auction_history.GetAllBonuses(globalTypes.NewItemFromString(data.Item), data.Region)
	if allBonusesErr != nil {
		cpclog.Errorf("Issue getting bonuses %v", allBonusesErr)
		json.NewEncoder(w).Encode(globalTypes.ReturnError{ERROR: allBonusesErr.Error()})
		w.WriteHeader(http.StatusInternalServerError)
		//w.WriteHeader(http.StatusInternalServerError)
		//fmt.Fprintf(w, "{ERROR:\"%s\"}", allBonusesErr.Error())
		return
	}

	return_value := SeenItemBonusesReturn{}

	bonus_cache_ptr, err := static_sources.GetBonuses()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	bonuses_cache := *bonus_cache_ptr

	cpclog.Debugf(`Regurning bonus lists for %s`, data.Item)
	var ilvl_adjusts, socket_adjusts, quality_adjusts, unknown_adjusts util.Set[string]
	found_empty_bonuses := false

	b_array := make([]mapped, 0)

	for _, e := range bonuses.Bonuses {
		var v mapped
		var sb strings.Builder
		v.Parsed = e
		if len(v.Parsed) != 0 {
			for _, curU := range v.Parsed {
				cur := fmt.Sprint(curU)
				if bonus_link, blPres := bonuses_cache[cur]; blPres {
					found := false
					if bonus_link.Level != 0 {
						sb.WriteString(fmt.Sprintf(`ilevel %d `, int(bonuses.Item.Level)+bonus_link.Level))
						found = true
						ilvl_adjusts.Add(cur)
					}
					if bonus_link.Socket != 0 {
						sb.WriteString(`socketed `)
						found = true
						socket_adjusts.Add(cur)
					}
					if bonus_link.Quality != 0 {
						sb.WriteString(fmt.Sprintf(`quality: %d `, bonuses_cache[cur].Quality))
						found = true
						quality_adjusts.Add(cur)
					}
					if !found {
						unknown_adjusts.Add(cur)
					}
				}
				//return value;
			}
			sb_hld := sb.String()
			v.Reduced = &sb_hld
		} else {
			found_empty_bonuses = true
		}
		b_array = append(b_array, v)
		strVal, _ := json.Marshal(e)
		return_value.Bonuses = append(return_value.Bonuses, map[string]string{"bonuses": string(strVal)})
	}

	return_value.Mapped = &b_array

	return_value.Collected.ILvl = make([]struct {
		Id    string "json:\"id,omitempty\""
		Level int    "json:\"level,omitempty\""
	}, 0)
	return_value.Collected.Socket = make([]struct {
		Id      string "json:\"id,omitempty\""
		Sockets *int   "json:\"sockets,omitempty\""
	}, 0)
	return_value.Collected.Quality = make([]struct {
		Id      string "json:\"id,omitempty\""
		Quality *int   "json:\"quality,omitempty\""
	}, 0)

	for _, elem := range ilvl_adjusts.ToArray() {
		name := fmt.Sprint(elem)
		return_value.Collected.ILvl = append(return_value.Collected.ILvl, struct {
			Id    string "json:\"id,omitempty\""
			Level int    "json:\"level,omitempty\""
		}{
			Id:    elem,
			Level: bonuses_cache[name].Level + int(bonuses.Item.Level),
		})
	}
	for _, elem := range socket_adjusts.ToArray() {
		name := fmt.Sprint(elem)
		sockets := bonuses_cache[name].Socket
		return_value.Collected.Socket = append(return_value.Collected.Socket, struct {
			Id      string "json:\"id,omitempty\""
			Sockets *int   "json:\"sockets,omitempty\""
		}{
			Id:      elem,
			Sockets: &sockets,
		})
	}
	for _, elem := range quality_adjusts.ToArray() {
		name := fmt.Sprint(elem)
		quality := bonuses_cache[name].Quality
		return_value.Collected.Quality = append(return_value.Collected.Quality, struct {
			Id      string "json:\"id,omitempty\""
			Quality *int   "json:\"quality,omitempty\""
		}{
			Id:      elem,
			Quality: &quality,
		})
	}
	return_value.Collected.Unknown = unknown_adjusts.ToArray()
	return_value.Collected.Empty = found_empty_bonuses

	json.NewEncoder(w).Encode(return_value)
}
