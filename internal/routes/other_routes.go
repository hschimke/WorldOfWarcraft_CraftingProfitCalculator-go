package routes

import (
	"encoding/json"
	"net/http"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/static_sources"
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
