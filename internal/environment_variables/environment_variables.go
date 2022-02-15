package environment_variables

import (
	"log"
	"os"
	"strconv"
	"strings"
)

var (
	STANDALONE_CONTAINER    string = ""
	DISABLE_AUCTION_HISTORY bool   = false
	//DATABASE_TYPE              string = ""
	//CACHE_DB_FN                string = ""
	//HISTORY_DB_FN              string = ""
	CLIENT_ID     string = ""
	CLIENT_SECRET string = ""
	USE_REDIS     bool   = false
	LOG_LEVEL     string = ""
	//NODE_ENV                   string = ""
	DOCKERIZED  bool   = false
	REDIS_URL   string = ""
	SERVER_PORT uint64 = 0
	//CLUSTER_SIZE               uint64 = 1
	DATABASE_CONNECTION_STRING string = ""
	STATIC_DIR_ROOT            string = ""
)

func getBoolean(variable string) (result bool) {
	switch strings.ToLower(variable) {
	case "true":
		result = true
	case "yes":
		result = true
	default:
		result = false
	}
	return
}

func getWithDefault(variable string, default_value string) (result string) {
	result = default_value
	if val, present := os.LookupEnv(variable); present {
		if val != "" {
			result = val
		}
	}
	return
}

func validateFromArray(check string, options []string) (found bool) {
	found = false
	for _, element := range options {
		if check == element {
			found = true
			break
		}
	}
	return
}

func init() {
	//CACHE_DB_FN = os.Getenv("CACHE_DB_FN")

	//HISTORY_DB_FN = os.Getenv("HISTORY_DB_FN")

	USE_REDIS = getBoolean(os.Getenv("USE_REDIS"))

	DOCKERIZED = getBoolean(os.Getenv("DOCKERIZED"))

	REDIS_URL = os.Getenv("REDIS_URL")

	if val, present := os.LookupEnv("CLIENT_ID"); present {
		CLIENT_ID = val
	} else {
		log.Fatal("must provide a CLIENT_ID environment variable")
	}

	if val, present := os.LookupEnv("CLIENT_SECRET"); present {
		CLIENT_SECRET = val
	} else {
		log.Fatal("must provide a CLIENT_SECRET environment variable")
	}
	LOG_LEVEL = getWithDefault("LOG_LEVEL", "info")

	//NODE_ENV = getWithDefault("NODE_ENV", "development")

	DISABLE_AUCTION_HISTORY = getBoolean("DISABLE_AUCTION_HISTORY")

	/*tempCS, err := strconv.ParseUint(getWithDefault("CLUSTER_SIZE", "1"), 0, 64)
	if err != nil {
		log.Fatal("could not parse CLUSTER_SIZE from environment variable")
	}
	CLUSTER_SIZE = tempCS*/

	tempSP, err := strconv.ParseUint(getWithDefault("SERVER_PORT", "3001"), 0, 64)
	if err != nil {
		log.Fatal("could not parse SERVER_PORT from environment variable")
	}
	SERVER_PORT = tempSP

	DATABASE_CONNECTION_STRING = os.Getenv("DATABASE_CONNECTION_STRING")
	//DATABASE_TYPE = os.Getenv("DATABASE_TYPE")

	var standaloneContainerOptions []string = []string{"normal", "hourly", "worker", "standalone"}
	if fetched_var := getWithDefault("STANDALONE_CONTAINER", "normal"); validateFromArray(fetched_var, standaloneContainerOptions) {
		STANDALONE_CONTAINER = fetched_var
	} else {
		log.Fatalf("STANDALONE_CONTAINER must be one of {%s}", standaloneContainerOptions)
	}

	STATIC_DIR_ROOT = os.Getenv("STATIC_DIR_ROOT")
}
