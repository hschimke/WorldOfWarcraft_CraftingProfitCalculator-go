package main

import (
	"log"
	"net/http"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/environment_variables"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/routes"
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	panic("doesn't work")
}

// https://stackoverflow.com/questions/31622052/how-to-serve-up-a-json-response-using-go
// https://stackoverflow.com/questions/49338485/how-to-post-a-json-request-and-recieve-json-response-to-go-server-go-language
// https://stackoverflow.com/questions/41854398/in-golang-how-do-i-get-the-handlefunc-function-to-parse-a-json-into-variables
// https://stackoverflow.com/questions/15672556/handling-json-post-request-in-go
// https://go.dev/doc/articles/wiki/

func main() {
	http.HandleFunc("/", homeHandler)

	http.HandleFunc("/json_output_QUEUED", routes.JsonOutputQueue)
	http.HandleFunc("/json_output_CHECK", routes.JsonOutputCheck)
	http.HandleFunc("/json_output", routes.JsonOutput)

	if !environment_variables.DISABLE_AUCTION_HISTORY {
		http.HandleFunc("/all_items", routes.AllItems)
		http.HandleFunc("/scanned_realms", routes.ScannedRealms)
		http.HandleFunc("/auction_history", routes.AuctionHistory)
		http.HandleFunc("/seen_item_bonuses", routes.SeenItemBonuses)
	}

	http.HandleFunc("/bonus_mappings", routes.BonusMappings)
	http.HandleFunc("/addon-download", routes.AddonDownload)
	http.HandleFunc("/healthcheck", routes.Healthcheck)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
