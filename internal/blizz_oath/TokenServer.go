package blizz_oath

import (
	"net/http"
	"sync"
	"time"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
)

// TokenServer represents a server that can return authorization tokens for a given client id and secret
type TokenServer struct {
	clientId, clientSecret string
	tokenStore             map[string]*AccessToken
	HttpClient             *http.Client
	authCheckMutex         sync.Mutex
	Logger                 *cpclog.CpCLog
}

// NewTokenServer creates a default TokenServer with a given client ID and Secret
func NewTokenServer(clientId, clientSecret string, logger *cpclog.CpCLog) *TokenServer {
	if clientId == "" || clientSecret == "" {
		panic("cannot have empty clientId or clientSecret")
	}
	return &TokenServer{
		clientId:     clientId,
		clientSecret: clientSecret,
		tokenStore:   map[string]*AccessToken{},
		HttpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		Logger: logger,
	}
}
