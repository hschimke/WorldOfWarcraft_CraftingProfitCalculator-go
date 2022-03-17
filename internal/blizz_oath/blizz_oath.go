package blizz_oath

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
)

const (
	authorizationUriBase string = "battle.net/oauth/token"
)

// AccessToken represents an access token as returned by Blizzard OATH. Fetched is internal only.
type AccessToken struct {
	Access_token string    `json:"access_token"`
	Token_type   string    `json:"token_type"`
	Expires_in   uint64    `json:"expires_in"`
	Scope        string    `json:"scope"`
	Fetched      time.Time `json:"-"`
}

// CheckExpired checks if the given access token needs to be refreshed
func (at *AccessToken) CheckExpired() (expired bool) {
	expired = true
	current_time := time.Now()
	expire_time := at.Fetched.Add(time.Duration(at.Expires_in))
	if current_time.Before(expire_time) {
		expired = false
	}
	return expired
}

// TokenServer represents a server that can return authorization tokens for a given client id and secret
type TokenServer struct {
	clientId, clientSecret string
	tokenStore             map[string]*AccessToken
	httpClient             *http.Client
	authCheckMutex         sync.Mutex
	logger                 *cpclog.CpCLog
}

// NewTokenServer creates a default TokenServer with a given client ID and Secret
func NewTokenServer(clientId, clientSecret string, logger *cpclog.CpCLog) (*TokenServer, error) {
	if clientId == "" || clientSecret == "" {
		return nil, fmt.Errorf("cannot have empty clientId or clientSecret")
	}
	return &TokenServer{
		clientId:     clientId,
		clientSecret: clientSecret,
		tokenStore:   map[string]*AccessToken{},
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}, nil
}

// GetAuthorizationToken returns an authorization token for a given region, fetches a new one if an existing token isn't found or has expired.
func (ts *TokenServer) GetAuthorizationToken(region string) (*AccessToken, error) {
	if region == "" {
		return nil, fmt.Errorf("cannot have empty region")
	}

	ts.authCheckMutex.Lock()

	if _, found := ts.tokenStore[region]; !found {
		ts.tokenStore[region] = &AccessToken{
			Access_token: "",
			Token_type:   "",
			Expires_in:   0,
			Scope:        "",
			Fetched:      time.Now(),
		}
	}
	token := ts.tokenStore[region]

	if token.CheckExpired() {
		ts.logger.Debug("Access token expired, fetching fresh.")
		uri := fmt.Sprint("https://", region, ".", authorizationUriBase)

		form := url.Values{}
		form.Add("grant_type", "client_credentials")

		req, err := http.NewRequest(http.MethodPost, uri, strings.NewReader(form.Encode()))
		if err != nil {
			ts.logger.Errorf("error getting access token for region: %s, err: %s", region, err)
			return nil, fmt.Errorf("error getting access token for region: %s, err: %s", region, err)
		}
		req.Header.Set("User-Agent", "WorldOfWarcraft_CraftingProfitCalculator-go")
		req.Header.Set("Connection", "keep-alive")
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.URL.User = url.UserPassword(ts.clientId, ts.clientSecret)

		res, getErr := ts.httpClient.Do(req)
		if getErr != nil {
			ts.logger.Error(getErr)
			ts.logger.Error("an error was encountered while retrieving an authorization token: ", getErr.Error())
			return nil, fmt.Errorf("error getting access token for region: %s, err: %s", region, getErr)
		}

		if res.Body != nil {
			defer res.Body.Close()
		}

		new_token := AccessToken{}
		parseErr := json.NewDecoder(res.Body).Decode(&new_token)
		if parseErr != nil {
			ts.logger.Error(parseErr)
			ts.logger.Error("an error was encountered while parsing an authorization token: ", parseErr.Error())
			return nil, fmt.Errorf("error getting access token for region: %s, err: %s", region, parseErr)
		}
		new_token.Fetched = time.Now()
		if new_token.Expires_in == 0 {
			new_token.Expires_in = uint64(time.Hour)
		} else {
			new_token.Expires_in = uint64(time.Duration(time.Second * time.Duration(new_token.Expires_in)))
		}
		ts.tokenStore[region] = &new_token
	}
	return_value := ts.tokenStore[region]

	ts.authCheckMutex.Unlock()

	return return_value, nil
}
