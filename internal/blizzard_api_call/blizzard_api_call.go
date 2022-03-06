package blizzard_api_call

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/blizz_oath"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/environment_variables"
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

var (
	allowed_during_period uint64 = 0
	in_use                uint64 = 0
	httpClient            *http.Client
	clearTicks            *time.Ticker
	stopClear             chan bool
)

// Control reset windows and connection flow for Blizzard API connections
func blizzardApiFlowManager(stopper chan bool, appShutdownSignal chan os.Signal) {
	cpclog.Info("Starting API Flow Manager")
	for {
		select {
		case <-clearTicks.C:
			cpclog.Silly("Reset window ", atomic.LoadUint64(&allowed_during_period))
			atomic.StoreUint64(&allowed_during_period, atomic.LoadUint64(&in_use))
		case <-stopper:
			cpclog.Info("Stopping API Flow Manager")
			clearTicks.Stop()
			return
		case <-appShutdownSignal:
			cpclog.Info("App Shutdown Detected: API Flow Manager shutting down")
			clearTicks.Stop()
			return
		}
	}
}

// Stop the flow manager
func ShutdownApiManager() {
	stopClear <- true
}

func init() {
	httpClient = &http.Client{
		Timeout: 20 * time.Second,
		Transport: &http.Transport{
			DisableCompression: false,
			ForceAttemptHTTP2:  true,
			MaxConnsPerHost:    allowed_connections_per_period,
		},
	}
	clearTicks = time.NewTicker(time.Duration(time.Second * period_reset_window))
	stopClear = make(chan bool)
	appShutdownDetected := make(chan os.Signal, 1)
	signal.Notify(appShutdownDetected, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go blizzardApiFlowManager(stopClear, appShutdownDetected)
}

// Get a response from Blizzard API and fill a struct with the results
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
			cpclog.Errorf("Failure fetching uri, will retry %d more times. (%v)", max_retries-attempt, getErr)
			time.Sleep(time.Second * sleep_seconds_between_tries)
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
		//fmt.Println(io.ReadAll(res.Body))
		cpclog.Error("An error was encountered while parsing response: ", parseErr)
		return fmt.Errorf("error parsing api response for: %s, err: %s", uri, parseErr)
	}
	return nil
}

func getBlizzardAPIResponse(data map[string]string, uri string, region globalTypes.RegionCode, target BlizzardApi.BlizzardApiReponse) (int, error) {
	var proceed bool = false
	var wait_count uint = 0
	for !proceed {
		if atomic.LoadUint64(&allowed_during_period) >= allowed_connections_per_period {
			wait_count++
			time.Sleep(time.Duration(time.Second * 1))
		} else {
			proceed = true
			atomic.AddUint64(&allowed_during_period, 1)
		}
	}
	if wait_count > 10 {
		cpclog.Debugf("Waited %v seconds for an available API window.", wait_count)
	} else if wait_count > 0 && wait_count <= 10 {
		cpclog.Sillyf("Waited %v seconds for an available API window.", wait_count)
	}
	atomic.AddUint64(&in_use, 1)
	getAndFillerr := getAndFill(uri, region, data, target)
	if getAndFillerr != nil {
		atomic.AddUint64(&in_use, ^uint64(0))
		cpclog.Errorf("issue fetching blizzard data: (%s)", uri)
		return -1, fmt.Errorf("issue fetching blizzard data: (%s)", uri)
	}

	atomic.AddUint64(&in_use, ^uint64(0))

	return int(wait_count), nil
}

// Fetch a Blizzard API response given only the endpoint
func GetBlizzardAPIResponse(region_code globalTypes.RegionCode, data map[string]string, uri string, target BlizzardApi.BlizzardApiReponse) (int, error) {
	built_uri := fmt.Sprintf("https://%s.%s%s", region_code, base_uri, uri)
	return getBlizzardAPIResponse(data, built_uri, region_code, target)
}

// Fetch a Blizzard API response given a fully qualified URL
func GetBlizzardRawUriResponse(data map[string]string, uri string, region globalTypes.RegionCode, target BlizzardApi.BlizzardApiReponse) (int, error) {
	return getBlizzardAPIResponse(data, uri, region, target)
}
