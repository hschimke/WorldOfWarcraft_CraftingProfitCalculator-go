package blizz_oath

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
)

type AccessToken struct {
	Access_token string    `json:"access_token"`
	Token_type   string    `json:"token_type"`
	Expires_in   uint64    `json:"expires_in"`
	Scope        string    `json:"scope"`
	Fetched      time.Time `json:"-"`
}

func (at *AccessToken) CheckExpired() (expired bool) {
	expired = true
	current_time := time.Now()
	expire_time := at.Fetched.Add(time.Duration(at.Expires_in))
	if current_time.Before(expire_time) {
		expired = false
	}
	return
}

const (
	authorization_uri_base string = "battle.net/oauth/token"
)

var (
	token_store map[string]*AccessToken = map[string]*AccessToken{}
	httpClient  *http.Client            = &http.Client{
		Timeout: 10 * time.Second,
	}
)

func GetAuthorizationToken(client_id string, client_secret string, region string) (*AccessToken, error) {
	if client_id == "" || client_secret == "" || region == "" {
		return nil, nil
	}
	if _, found := token_store[region]; !found {
		token_store[region] = &AccessToken{
			Access_token: "",
			Token_type:   "",
			Expires_in:   0,
			Scope:        "",
			Fetched:      time.Now(),
		}
	}
	token := token_store[region]

	if token.CheckExpired() {
		cpclog.Debug("Access token expired, fetching fresh.")
		uri := fmt.Sprint("https://", region, ".", authorization_uri_base)

		form := url.Values{}
		form.Add("grant_type", "client_credentials")

		req, err := http.NewRequest(http.MethodPost, uri, strings.NewReader(form.Encode()))
		if err != nil {
			cpclog.Errorf("error getting access token for region: %s, err: %s", region, err)
			return nil, fmt.Errorf("error getting access token for region: %s, err: %s", region, err)
		}
		req.Header.Set("User-Agent", "WorldOfWarcraft_CraftingProfitCalculator-go")
		req.Header.Set("Connection", "keep-alive")
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.URL.User = url.UserPassword(client_id, client_secret)

		res, getErr := httpClient.Do(req)
		if getErr != nil {
			cpclog.Error(getErr)
			cpclog.Error("an error was encountered while retrieving an authorization token: ", getErr.Error())
			return nil, fmt.Errorf("error getting access token for region: %s, err: %s", region, getErr)
		}

		if res.Body != nil {
			defer res.Body.Close()
		}

		new_token := AccessToken{}
		parseErr := json.NewDecoder(res.Body).Decode(&new_token)
		if parseErr != nil {
			cpclog.Error(parseErr)
			cpclog.Error("an error was encountered while parsing an authorization token: ", parseErr.Error())
			return nil, fmt.Errorf("error getting access token for region: %s, err: %s", region, parseErr)
		}
		new_token.Fetched = time.Now()
		if new_token.Expires_in == 0 {
			new_token.Expires_in = uint64(time.Hour)
		} else {
			new_token.Expires_in = uint64(time.Duration(time.Second * time.Duration(new_token.Expires_in)))
		}
		token_store[region] = &new_token
	}
	return_value := token_store[region]
	return return_value, nil
}
