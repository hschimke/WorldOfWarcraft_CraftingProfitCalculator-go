package blizzard_api_call

import (
	"net/http"
	"time"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/blizz_oath"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"golang.org/x/time/rate"
)

type BlizzardApiProvider struct {
	HttpClient  *http.Client
	TokenServer *blizz_oath.TokenServer
	Logger      *cpclog.CpCLog
	Limiter     *rate.Limiter
}

func NewBlizzardApiProvider(tokenServer *blizz_oath.TokenServer, logger *cpclog.CpCLog) *BlizzardApiProvider {
	// Blizzard limits: 100 requests per second, 36,000 requests per hour.
	// 36,000 requests per hour is 10 requests per second.
	// We use a rate of 10 per second with a burst of 100 to stay within both limits.
	limiter := rate.NewLimiter(rate.Limit(10), 100)

	client := BlizzardApiProvider{
		HttpClient: &http.Client{
			Timeout: 20 * time.Second,
			Transport: &http.Transport{
				DisableCompression: false,
				ForceAttemptHTTP2:  true,
				MaxConnsPerHost:    100,
			},
		},
		Logger:      logger,
		TokenServer: tokenServer,
		Limiter:     limiter,
	}

	return &client
}

// ShutdownApiManager is now a no-op as we use rate.Limiter which doesn't need a background goroutine
func (client *BlizzardApiProvider) ShutdownApiManager() {
}
