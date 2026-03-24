package blizzard_api_call

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes/BlizzardApi"
)

const (
	base_uri                    = "api.blizzard.com"
	max_retries                 = 5
	initial_retry_delay_seconds = 1
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

// getAndFill retrieves data from Blizzard API and unmarshals it into the target struct.
func getAndFill[T BlizzardApi.BlizzardApiReponse](ctx context.Context, api *BlizzardApiProvider, uri string, region globalTypes.RegionCode, data map[string]string, namespace string, target *T) error {
	token, tokenErr := api.TokenServer.GetAuthorizationToken(ctx, string(region))
	if tokenErr != nil {
		return tokenErr
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		api.Logger.Errorf("error with request: %s, err: %s", uri, err)
		return fmt.Errorf("error with request: %s, err: %s", uri, err)
	}

	req.Header.Set("User-Agent", "WorldOfWarcraft_CraftingProfitCalculator-go")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Authorization", fmt.Sprint("Bearer ", token.Access_token))
	req.Header.Set("Battlenet-Namespace", namespace)
	req.Header.Set("Accept", "application/json")

	queryParams := req.URL.Query()
	for key, value := range data {
		queryParams.Set(key, value)
	}
	req.URL.RawQuery = queryParams.Encode()

	var res *http.Response
	var lastErr error

	for attempt := 0; attempt <= max_retries; attempt++ {
		// Respect rate limits before making the call
		if err := api.Limiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limiter wait error: %w", err)
		}

		res, lastErr = api.HttpClient.Do(req)
		if lastErr != nil {
			api.Logger.Debugf("Attempt %d: Failure fetching uri %s: %v. Retrying...", attempt+1, uri, lastErr)
			if !retryWithBackoff(ctx, attempt) {
				return fmt.Errorf("max retries exceeded or context cancelled for %s: %w", uri, lastErr)
			}
			continue
		}

		// Handle 429 Too Many Requests
		if res.StatusCode == http.StatusTooManyRequests {
			retryAfter := res.Header.Get("Retry-After")
			waitDuration := time.Duration(initial_retry_delay_seconds) * time.Second
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				waitDuration = time.Duration(seconds) * time.Second
			}
			api.Logger.Warnf("Received 429 for %s, waiting %v", uri, waitDuration)
			res.Body.Close()

			select {
			case <-time.After(waitDuration):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Handle 5xx errors
		if res.StatusCode >= 500 {
			api.Logger.Warnf("Received %d for %s, retrying...", res.StatusCode, uri)
			res.Body.Close()
			if !retryWithBackoff(ctx, attempt) {
				return fmt.Errorf("max retries exceeded or context cancelled for %s: status %d", uri, res.StatusCode)
			}
			continue
		}

		// Check for other non-200 status codes
		if res.StatusCode != http.StatusOK {
			res.Body.Close()
			return fmt.Errorf("blizzard api returned status %d for %s", res.StatusCode, uri)
		}

		// Success!
		break
	}

	if lastErr != nil {
		return fmt.Errorf("failed to fetch %s after %d retries: %w", uri, max_retries, lastErr)
	}

	if res == nil {
		return fmt.Errorf("unexpected nil response for %s", uri)
	}

	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(target); err != nil {
		return fmt.Errorf("error parsing api response for %s: %w", uri, err)
	}

	return nil
}

func retryWithBackoff(ctx context.Context, attempt int) bool {
	if attempt >= max_retries {
		return false
	}
	delay := time.Duration(math.Pow(2, float64(attempt))*float64(initial_retry_delay_seconds)) * time.Second
	select {
	case <-time.After(delay):
		return true
	case <-ctx.Done():
		return false
	}
}

// GetBlizzardAPIResponse fetches a Blizzard API response given only the endpoint.
func GetBlizzardAPIResponse[T BlizzardApi.BlizzardApiReponse](ctx context.Context, api *BlizzardApiProvider, region_code globalTypes.RegionCode, data map[string]string, uri string, namespace string, target *T) error {
	built_uri := fmt.Sprintf("https://%s.%s%s", region_code, base_uri, uri)
	return getAndFill(ctx, api, built_uri, region_code, data, namespace, target)
}

// GetBlizzardRawUriResponse fetches a Blizzard API response given a fully qualified URL.
func GetBlizzardRawUriResponse[T BlizzardApi.BlizzardApiReponse](ctx context.Context, api *BlizzardApiProvider, data map[string]string, uri string, region globalTypes.RegionCode, namespace string, target *T) error {
	return getAndFill(ctx, api, uri, region, data, namespace, target)
}
