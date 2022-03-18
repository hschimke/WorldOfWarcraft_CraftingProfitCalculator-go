package blizzard_api_call

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes/BlizzardApi"
)

const (
	allowed_connections_per_period = 50
	period_reset_window            = 1
	base_uri                       = "api.blizzard.com"
	max_retries                    = 15
	sleep_seconds_between_tries    = 3
)

// Locale constants for API useage
const (
	ENGLISH_US          string = "en_US" // US English
	SPANISH_MEXICO      string = "es_MX" // Spanish - Mexico
	PORTUGUESE          string = "pt_BR" // Portuguese
	GERMAN              string = "de_DE" // German
	ENGLISH_GB          string = "en_GB" // English - Great Britain
	SPANISH_SPAIN       string = "es_ES" // Spanish - Spain
	FRENCH              string = "fr_FR" // French
	ITALIAN             string = "it_IT" // Italian
	RUSSIAN             string = "ru_RU" // Russian
	KOREAN              string = "ko_KR" // Korean
	CHINESE_TRADITIONAL string = "zh_TW" // Chinese (Traditional)
	CHINESE_SIMPLIFIED  string = "zh_CN" // Chinese (Simplified)
)

// Control reset windows and connection flow for Blizzard API connections
func (client *BlizzardApiProvider) blizzardApiFlowManager(stopper chan bool, appShutdownSignal chan os.Signal) {
	client.Logger.Info("Starting API Flow Manager")
	for {
		select {
		case <-client.clearTicks.C:
			client.Logger.Silly("Reset window ", atomic.LoadUint64(&client.allowedDuringPeriod))
			atomic.StoreUint64(&client.allowedDuringPeriod, atomic.LoadUint64(&client.inUse))
		case <-stopper:
			client.Logger.Info("Stopping API Flow Manager")
			client.clearTicks.Stop()
			return
		case <-appShutdownSignal:
			client.Logger.Info("App Shutdown Detected: API Flow Manager shutting down")
			client.clearTicks.Stop()
			return
		}
	}
}

// Stop the flow manager
func (client *BlizzardApiProvider) ShutdownApiManager() {
	client.stopClear <- true
}

// Get a response from Blizzard API and fill a struct with the results
func getAndFill[T BlizzardApi.BlizzardApiReponse](api *BlizzardApiProvider, uri string, region globalTypes.RegionCode, data map[string]string, namespace string, target *T) error {
	token, tokenErr := api.TokenServer.GetAuthorizationToken(region)
	if tokenErr != nil {
		return tokenErr
	}

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		api.Logger.Errorf("error with request: %s, err: %s", uri, err)
		return fmt.Errorf("error with request: %s, err: %s", uri, err)
	}
	req.Header.Set("User-Agent", "WorldOfWarcraft_CraftingProfitCalculator-go")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Authorization", fmt.Sprint("Bearer ", token.Access_token))
	req.Header.Set("Battlenet-Namespace", namespace)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Keep-Alive", "timeout=3600")

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
		res, getErr = api.HttpClient.Do(req)
		if getErr != nil {
			api.Logger.Debugf("Failure fetching uri, will retry %d more times. (%v)", max_retries-attempt, getErr)
			time.Sleep(time.Second * sleep_seconds_between_tries)
		} else {
			break
		}
	}

	if getErr != nil {
		api.Logger.Error("An error was encountered while retrieving a uri(", uri, "): ", getErr)
		return fmt.Errorf("error fetching uri: %s, err: %s", uri, getErr)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	parseErr := json.NewDecoder(res.Body).Decode(&target)
	if parseErr != nil {
		api.Logger.Error("An error was encountered while parsing response: ", parseErr)
		return fmt.Errorf("error parsing api response for: %s, err: %s", uri, parseErr)
	}
	return nil
}

func getBlizzardAPIResponse[T BlizzardApi.BlizzardApiReponse](api *BlizzardApiProvider, data map[string]string, uri string, region globalTypes.RegionCode, namespace string, target *T) (int, error) {
	var proceed bool = false
	var wait_count uint = 0
	for !proceed {
		if atomic.LoadUint64(&api.allowedDuringPeriod) >= allowed_connections_per_period {
			wait_count++
			time.Sleep(time.Duration(time.Second * 1))
		} else {
			proceed = true
			atomic.AddUint64(&api.allowedDuringPeriod, 1)
		}
	}
	if wait_count > 10 {
		api.Logger.Debugf("Waited %v seconds for an available API window.", wait_count)
	} else if wait_count > 0 && wait_count <= 10 {
		api.Logger.Sillyf("Waited %v seconds for an available API window.", wait_count)
	}
	atomic.AddUint64(&api.inUse, 1)
	getAndFillerr := getAndFill(api, uri, region, data, namespace, target)
	if getAndFillerr != nil {
		atomic.AddUint64(&api.inUse, ^uint64(0))
		api.Logger.Errorf("issue fetching blizzard data: (%s)", uri)
		return -1, fmt.Errorf("issue fetching blizzard data: (%s)", uri)
	}

	atomic.AddUint64(&api.inUse, ^uint64(0))

	return int(wait_count), nil
}

// Fetch a Blizzard API response given only the endpoint
func GetBlizzardAPIResponse[T BlizzardApi.BlizzardApiReponse](api *BlizzardApiProvider, region_code globalTypes.RegionCode, data map[string]string, uri string, namespace string, target *T) (int, error) {
	built_uri := fmt.Sprintf("https://%s.%s%s", region_code, base_uri, uri)
	return getBlizzardAPIResponse(api, data, built_uri, region_code, namespace, target)
}

// Fetch a Blizzard API response given a fully qualified URL
func GetBlizzardRawUriResponse[T BlizzardApi.BlizzardApiReponse](api *BlizzardApiProvider, data map[string]string, uri string, region globalTypes.RegionCode, namespace string, target *T) (int, error) {
	return getBlizzardAPIResponse(api, data, uri, region, namespace, target)
}
