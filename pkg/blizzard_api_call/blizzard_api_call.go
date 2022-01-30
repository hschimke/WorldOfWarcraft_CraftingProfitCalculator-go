package blizzard_api_call

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/blizz_oath"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/environment_variables"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes/BlizzardApi"
)

const (
	allowed_connections_per_period = 100
	period_reset_window            = 5
	base_uri                       = "api.blizzard.com"
)

var (
	allowed_during_period uint = 0
	in_use                uint = 0
	//	run                   bool = false
	httpClient *http.Client
	clearTicks time.Ticker
	stopClear  chan bool
)

// Maybe redo with: https://go.dev/tour/concurrency/5
func blizzardApiFlowManager(stopper chan bool) {
	for {

		select {
		case <-clearTicks.C:
			allowed_during_period = 0 + in_use
		case <-stopper:
			clearTicks.Stop()
			return
		}

	}
}

func ShutdownApiManager() {
	stopClear <- true
}

func manageBlizzardTimeout() {
	go blizzardApiFlowManager(stopClear)
}

func init() {
	httpClient = &http.Client{
		Timeout: 10 * time.Second,
	}
	time.NewTicker(time.Duration(time.Second * period_reset_window))
	stopClear = make(chan bool)
	manageBlizzardTimeout()
}

func getAndFill(uri string, region globalTypes.RegionCode, data interface{}, target BlizzardApi.BlizzardApiReponse) error {
	token, tokenErr := blizz_oath.GetAuthorizationToken(environment_variables.CLIENT_ID, environment_variables.CLIENT_SECRET, region)
	if tokenErr != nil {
		return tokenErr
	}

	encoded_data, encodeErr := json.Marshal(data)
	if encodeErr != nil {
		return fmt.Errorf("error with request: %s, err: %s", uri, encodeErr)
	}
	req, err := http.NewRequest(http.MethodPost, uri, bytes.NewBuffer(encoded_data))
	if err != nil {
		cpclog.Errorf("error with request: %s, err: %s", uri, err)
		return fmt.Errorf("error with request: %s, err: %s", uri, err)
	}
	req.Header.Set("User-Agent", "WorldOfWarcraft_CraftingProfitCalculator-go")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.Access_token))
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	res, getErr := httpClient.Do(req)
	if getErr != nil {
		cpclog.Errorf("An error was encountered while retrieving an authorization token: ", getErr)
		return fmt.Errorf("error fetching uri: %s, err: %s", uri, getErr)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	parseErr := json.NewDecoder(res.Body).Decode(&target)
	if parseErr != nil {
		//log.Print(io.ReadAll(res.Body))
		cpclog.Errorf("An error was encountered while retrieving an authorization token: ", parseErr)
		return fmt.Errorf("error parsing api response for: %s, err: %s", uri, parseErr)
	}
	return nil
}

func GetBlizzardAPIResponse(region_code globalTypes.RegionCode, data interface{}, uri string, target BlizzardApi.BlizzardApiReponse) (int, error) {
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

func GetBlizzardRawUriResponse(data interface{}, uri string, region globalTypes.RegionCode, target BlizzardApi.BlizzardApiReponse) (int, error) {
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
