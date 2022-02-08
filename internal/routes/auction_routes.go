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
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/auction_history"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
)

type mapped struct {
	Text    string   `json:"text,omitempty"`
	Parsed  []string `json:"parsed,omitempty"`
	Reduced *string  `json:"reduced,omitempty"`
}

type SeenItemBonusesReturn struct {
	Bonuses   []uint    `json:"bonuses,omitempty"`
	Mapped    *[]mapped `json:"mapped,omitempty"`
	Collected struct {
		ILvl []struct {
			Id    uint `json:"id,omitempty"`
			Level int  `json:"level,omitempty"`
		} `json:"ilvl,omitempty"`
		Socket []struct {
			Id      uint `json:"id,omitempty"`
			Sockets *int `json:"sockets,omitempty"`
		} `json:"socket,omitempty"`
		Quality []struct {
			Id      uint `json:"id,omitempty"`
			Quality *int `json:"quality,omitempty"`
		} `json:"quality,omitempty"`
		Unknown []uint `json:"unknown,omitempty"`
		Empty   bool   `json:"empty,omitempty"`
	} `json:"collected,omitempty"`
}

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
	}

	partial := r.URL.Query().Get("partial")

	filterd_names := handleNames(names, partial)
	json.NewEncoder(w).Encode(filterd_names)
}

func handleNames(names []string, partial string) []string {
	var filteredNames []string
	if len(partial) > 0 {
		cpclog.Debugf(`Partial search for all items with "%s"`, partial)
		comparePartial := strings.ToLower(partial)
		for _, name := range names {
			if strings.Contains(strings.ToLower(name), comparePartial) {
				filteredNames = append(filteredNames, name)
			}
		}
	} else {
		cpclog.Debug("Returning all unfiltered items.")
		filteredNames = names
	}
	return filteredNames
}

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
		fmt.Fprint(w, "{ERROR:\"empty item\"}")
		return
	}

	bonuses, allBonusesErr := auction_history.GetAllBonuses(globalTypes.NewItemFromString(data.Item), data.Region)
	if allBonusesErr != nil {
		cpclog.Errorf("Issue getting bonuses %v", allBonusesErr)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "{ERROR:\"%s\"}", allBonusesErr.Error())
		return
	}

	return_value := SeenItemBonusesReturn{}

	bonus_cache_ptr, err := static_sources.GetBonuses()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	bonus_cache := *bonus_cache_ptr

	cpclog.Debugf(`Regurning bonus lists for %s`, data.Item)
	var ilvl_adjusts, socket_adjusts, quality_adjusts, unknown_adjusts uintSet
	found_empty_bonuses := false

	b_array := make([]mapped, 0)
	for _, e := range bonuses.Bonuses {
		var value strings.Builder
		cur := fmt.Sprint(e)
		v := mapped{
			Text:    fmt.Sprint(e),
			Parsed:  []string{fmt.Sprint(e)},
			Reduced: nil,
		}

		//let value = acc;
		found := false
		bonus_link := bonus_cache[cur]

		if bonus_link.Level != 0 {
			value.WriteString(fmt.Sprintf(`ilevel %d `, int(bonuses.Item.Level)+bonus_link.Level))
			found = true
			ilvl_adjusts.add(e)
		}
		if bonus_link.Socket != 0 {
			value.WriteString(`socketed `)
			found = true
			socket_adjusts.add(e)
		}
		if bonus_link.Quality != 0 {
			value.WriteString(fmt.Sprintf(`quality: %d `, bonus_cache[cur].Quality))
			found = true
			quality_adjusts.add(e)
		}
		if !found {
			unknown_adjusts.add(e)
		}

		str := value.String()
		v.Reduced = &str
		b_array = append(b_array, v)
	}

	return_value.Bonuses = bonuses.Bonuses
	return_value.Mapped = &b_array
	for _, elem := range ilvl_adjusts.toArray() {
		name := fmt.Sprint(elem)
		return_value.Collected.ILvl = append(return_value.Collected.ILvl, struct {
			Id    uint "json:\"id,omitempty\""
			Level int  "json:\"level,omitempty\""
		}{
			Id:    elem,
			Level: bonus_cache[name].Level + int(bonuses.Item.Level),
		})
	}
	for _, elem := range socket_adjusts.toArray() {
		name := fmt.Sprint(elem)
		sockets := bonus_cache[name].Socket
		return_value.Collected.Socket = append(return_value.Collected.Socket, struct {
			Id      uint "json:\"id,omitempty\""
			Sockets *int "json:\"sockets,omitempty\""
		}{
			Id:      elem,
			Sockets: &sockets,
		})
	}
	for _, elem := range quality_adjusts.toArray() {
		name := fmt.Sprint(elem)
		quality := bonus_cache[name].Quality
		return_value.Collected.Quality = append(return_value.Collected.Quality, struct {
			Id      uint "json:\"id,omitempty\""
			Quality *int "json:\"quality,omitempty\""
		}{
			Id:      elem,
			Quality: &quality,
		})
	}
	return_value.Collected.Unknown = unknown_adjusts.toArray()
	return_value.Collected.Empty = found_empty_bonuses

	json.NewEncoder(w).Encode(return_value)
}

type uintSet struct {
	set map[uint]bool
}

func (s *uintSet) add(v uint) {
	if s.set == nil {
		s.set = make(map[uint]bool)
	}
	s.set[v] = true
}

func (s *uintSet) toArray() []uint {
	var return_list []uint
	for key, pres := range s.set {
		if pres {
			return_list = append(return_list, key)
		}
	}
	return return_list
}
