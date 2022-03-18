package blizzard_api_call

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/blizz_oath"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
)

type BlizzardApiProvider struct {
	allowedDuringPeriod, inUse uint64
	HttpClient                 *http.Client
	clearTicks                 *time.Ticker
	stopClear                  chan bool
	TokenServer                *blizz_oath.TokenServer
	Logger                     *cpclog.CpCLog
}

func NewBlizzardApiProvider(tokenServer *blizz_oath.TokenServer, logger *cpclog.CpCLog) *BlizzardApiProvider {
	client := BlizzardApiProvider{
		allowedDuringPeriod: 0,
		inUse:               0,
		HttpClient: &http.Client{
			Timeout: 20 * time.Second,
			Transport: &http.Transport{
				DisableCompression: false,
				ForceAttemptHTTP2:  true,
				MaxConnsPerHost:    allowed_connections_per_period,
			},
		},
		clearTicks:  time.NewTicker(time.Duration(time.Second * period_reset_window)),
		stopClear:   make(chan bool),
		Logger:      logger,
		TokenServer: tokenServer,
	}
	appShutdownDetected := make(chan os.Signal, 1)
	signal.Notify(appShutdownDetected, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go client.blizzardApiFlowManager(client.stopClear, appShutdownDetected)

	return &client
}
