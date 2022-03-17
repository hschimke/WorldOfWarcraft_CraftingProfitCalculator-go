package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/util"
)

// Download the WOW addon
func (routes *CPCRoutes) AddonDownload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/zip")
	http.ServeFile(w, r, "html/CraftingProfitCalculator_data.zip")
}

// Internal healthcheck
func (routes *CPCRoutes) Healthcheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	//	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(struct {
		Health string `json:"health"`
	}{
		Health: "ok",
	})
}

// Get a list of all bonus mappings for a given bonus
func (routes *CPCRoutes) BonusMappings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	bonus := r.URL.Query().Get("bonus")

	if len(bonus) <= 0 {
		http.Error(w, "bonus not found", http.StatusBadRequest)
		return
	}

	sources, err := routes.staticSources.GetBonuses()
	if err != nil {
		http.Error(w, "Could not load bonuses", http.StatusInternalServerError)
	} else {
		if bns, fnd := (*sources)[bonus]; fnd {
			json.NewEncoder(w).Encode(bns)
		} else {
			http.Error(w, fmt.Sprintf("bonus '%s' not found", bonus), http.StatusBadRequest)
		}
	}
}

// Return a list of all realms availble
func (routes *CPCRoutes) AllRealms(w http.ResponseWriter, r *http.Request) {
	routes.logger.Debug("Getting all realms")

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	var names []string

	partial := r.URL.Query().Get("partial")
	region := r.URL.Query().Get("region")

	if len(region) > 0 {
		names = routes.helper.GetAllRealmNames(region)
	}

	filterd_names := util.FilterStringArray(names, partial, "realms")
	json.NewEncoder(w).Encode(filterd_names)
}
