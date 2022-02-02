package blizzard_api_call

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/blizz_oath"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/environment_variables"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes/BlizzardApi"
)

const (
	allowed_connections_per_period = 100
	period_reset_window            = 5
	base_uri                       = "api.blizzard.com"
	max_retries                    = 5
)

var (
	allowed_during_period uint = 0
	in_use                uint = 0
	//	run                   bool = false
	httpClient *http.Client
	clearTicks *time.Ticker
	stopClear  chan bool
)

// Maybe redo with: https://go.dev/tour/concurrency/5
func blizzardApiFlowManager(stopper chan bool) {
	cpclog.Info("Starting API Flow Manager")
	for {
		select {
		case <-clearTicks.C:
			cpclog.Silly("Reset window ", allowed_during_period, " ")
			allowed_during_period = 0 + in_use
		case <-stopper:
			cpclog.Info("Stopping API Flow Manager")
			clearTicks.Stop()
			return
		}
	}
}

func ShutdownApiManager() {
	stopClear <- true
}

func init() {
	httpClient = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			ForceAttemptHTTP2:  true,
			DisableCompression: false,
		},
	}
	clearTicks = time.NewTicker(time.Duration(time.Second * period_reset_window))
	stopClear = make(chan bool)
	go blizzardApiFlowManager(stopClear)
}

func getAndFill(uri string, region globalTypes.RegionCode, data map[string]string, target BlizzardApi.BlizzardApiReponse) error {
	token, tokenErr := blizz_oath.GetAuthorizationToken(environment_variables.CLIENT_ID, environment_variables.CLIENT_SECRET, region)
	if tokenErr != nil {
		return tokenErr
	}

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		cpclog.Errorf("error with request: %s, err: %s", uri, err)
		return fmt.Errorf("error with request: %s, err: %s", uri, err)
	}
	req.Header.Set("User-Agent", "WorldOfWarcraft_CraftingProfitCalculator-go")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Authorization", fmt.Sprint("Bearer ", token.Access_token))

	queryParams := req.URL.Query()
	for key, value := range data {
		queryParams.Set(key, value)
	}
	req.URL.RawQuery = queryParams.Encode()

	var (
		res    *http.Response
		getErr error
	)

	for attempt := 0; attempt < max_retries; attempt++ {
		res, getErr = httpClient.Do(req)
		if getErr != nil {
			//cpclog.Error("An error was encountered while retrieving a uri(", uri, "): ", getErr)
			//return fmt.Errorf("error fetching uri: %s, err: %s", uri, getErr)
			time.Sleep(time.Second)
		} else {
			break
		}
	}

	if getErr != nil {
		cpclog.Error("An error was encountered while retrieving a uri(", uri, "): ", getErr)
		return fmt.Errorf("error fetching uri: %s, err: %s", uri, getErr)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	parseErr := json.NewDecoder(res.Body).Decode(&target)
	if parseErr != nil {
		fmt.Println(io.ReadAll(res.Body))
		cpclog.Error("An error was encountered while parsing response: ", parseErr)
		return fmt.Errorf("error parsing api response for: %s, err: %s", uri, parseErr)
	}
	return nil
}

func GetBlizzardAPIResponse(region_code globalTypes.RegionCode, data map[string]string, uri string, target BlizzardApi.BlizzardApiReponse) (int, error) {
	var proceed bool = false
	var wait_count uint = 0
	for !proceed {
		if allowed_during_period > allowed_connections_per_period {
			wait_count++
			time.Sleep(time.Duration(time.Second * 1))
		} else {
			proceed = true
			allowed_during_period++
		}
	}
	if wait_count > 0 {
		cpclog.Debugf("Waited %v seconds for an available API window.", wait_count)
	}
	in_use++
	built_uri := fmt.Sprintf("https://%s.%s%s", region_code, base_uri, uri)
	getAndFillerr := getAndFill(built_uri, region_code, data, target)
	if getAndFillerr != nil {
		return -1, fmt.Errorf("issue fetching blizzard data: (https://%s.%s%s", region_code, base_uri, uri)
	}
	in_use--
	return int(wait_count), nil
}

func GetBlizzardRawUriResponse(data map[string]string, uri string, region globalTypes.RegionCode, target BlizzardApi.BlizzardApiReponse) (int, error) {
	var proceed bool = false
	var wait_count uint = 0
	for !proceed {
		if allowed_during_period > allowed_connections_per_period {
			wait_count++
			time.Sleep(time.Duration(time.Second * 1))
		} else {
			proceed = true
			allowed_during_period++
		}
	}
	if wait_count > 0 {
		cpclog.Debugf("Waited %v seconds for an available API window.", wait_count)
	}
	in_use++
	//built_uri := fmt.Sprintf("https://%s.%s%s", region_code, base_uri, uri)
	getAndFillerr := getAndFill(uri, region, data, target)
	if getAndFillerr != nil {
		return -1, fmt.Errorf("issue fetching blizzard data: (%s", uri)
	}
	in_use--
	return int(wait_count), nil
}
