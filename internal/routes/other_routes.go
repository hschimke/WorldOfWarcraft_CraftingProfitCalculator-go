package routes

import (
	"encoding/json"
	"net/http"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/static_sources"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/blizzard_api_helpers"
)

func AddonDownload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/zip")
	http.ServeFile(w, r, "html/CraftingProfitCalculator_data.zip")
}

func Healthcheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	//	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(struct {
		Health string `json:"health"`
	}{
		Health: "ok",
	})
}

func BonusMappings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	sources, err := static_sources.GetBonuses()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		//		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(sources)
	}
}

func AllRealms(w http.ResponseWriter, r *http.Request) {
	cpclog.Debug("Getting all realms")

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	var names []string

	partial := r.URL.Query().Get("partial")
	region := r.URL.Query().Get("region")

	if len(region) > 0 {
		names = blizzard_api_helpers.GetAllRealmNames(region)
	}

	filterd_names := handleNames(names, partial)
	json.NewEncoder(w).Encode(filterd_names)
}
