package environment_variables

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
)

var (
	STANDALONE_CONTAINER       string = ""
	DISABLE_AUCTION_HISTORY    bool   = false
	CLIENT_ID                  string = ""
	CLIENT_SECRET              string = ""
	USE_REDIS                  bool   = false
	LOG_LEVEL                  string = ""
	DOCKERIZED                 bool   = false
	REDIS_URL                  string = ""
	SERVER_PORT                uint64 = 0
	DATABASE_CONNECTION_STRING string = ""
	STATIC_DIR_ROOT            string = ""
	EXCLUDE_BEFORE_SHADOWLANDS bool   = false
)

// Return a boolean based on a string
func getBoolean(variable string) (result bool) {
	switch strings.ToLower(variable) {
	case "true":
		result = true
	case "yes":
		result = true
	default:
		result = false
	}
	return result
}

// Get an environment variable with a default value
func getWithDefault(variable string, default_value string) (result string) {
	result = default_value
	if val, present := os.LookupEnv(variable); present {
		if val != "" {
			result = val
		}
	}
	return result
}

// Verify that the value in check is one of the acceptable ones available in options
func validateFromArray(check string, options []string) (found bool) {
	return slices.Contains(options, check)
}

// Load configuration from environment variables.
// This function replaces the previous init() to allow for better testing
// and explicit error handling.
func Load() error {
	var errs []error

	USE_REDIS = getBoolean(os.Getenv("USE_REDIS"))
	DOCKERIZED = getBoolean(os.Getenv("DOCKERIZED"))
	REDIS_URL = os.Getenv("REDIS_URL")

	if val, present := os.LookupEnv("CLIENT_ID"); present {
		CLIENT_ID = val
	} else {
		errs = append(errs, errors.New("must provide a CLIENT_ID environment variable"))
	}

	if val, present := os.LookupEnv("CLIENT_SECRET"); present {
		CLIENT_SECRET = val
	} else {
		errs = append(errs, errors.New("must provide a CLIENT_SECRET environment variable"))
	}

	LOG_LEVEL = getWithDefault("LOG_LEVEL", "info")
	DISABLE_AUCTION_HISTORY = getBoolean(os.Getenv("DISABLE_AUCTION_HISTORY"))

	tempSP, err := strconv.ParseUint(getWithDefault("SERVER_PORT", "3001"), 0, 64)
	if err != nil {
		errs = append(errs, fmt.Errorf("could not parse SERVER_PORT from environment variable: %w", err))
	} else {
		SERVER_PORT = tempSP
	}

	DATABASE_CONNECTION_STRING = os.Getenv("DATABASE_CONNECTION_STRING")

	var standaloneContainerOptions []string = []string{"normal", "hourly", "worker", "standalone"}
	if fetched_var := getWithDefault("STANDALONE_CONTAINER", "normal"); validateFromArray(fetched_var, standaloneContainerOptions) {
		STANDALONE_CONTAINER = fetched_var
	} else {
		errs = append(errs, fmt.Errorf("STANDALONE_CONTAINER must be one of %v", standaloneContainerOptions))
	}

	STATIC_DIR_ROOT = os.Getenv("STATIC_DIR_ROOT")
	EXCLUDE_BEFORE_SHADOWLANDS = getBoolean(getWithDefault("SEARCH_BEFORE_SHADOWLANDS", "false"))

	return errors.Join(errs...)
}
