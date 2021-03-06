package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/blizz_oath"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/blizzard_api_call"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cache_provider"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/environment_variables"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/blizzard_api_helpers"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/wow_crafting_profits"
)

func main() {

	logger := &cpclog.CpCLog{
		LogLevel: cpclog.GetLevel(environment_variables.LOG_LEVEL),
	}

	logger.Info("Starting cpc-job-worker")

	uri := environment_variables.REDIS_URL

	ctx, cancelContext := context.WithCancel(context.Background())

	cache := cache_provider.NewCacheProvider(ctx, environment_variables.REDIS_URL)
	tokenServer := blizz_oath.NewTokenServer(environment_variables.CLIENT_ID, environment_variables.CLIENT_SECRET, logger)
	api := blizzard_api_call.NewBlizzardApiProvider(tokenServer, logger)
	helper := blizzard_api_helpers.NewBlizzardApiHelper(cache, logger, api)
	cpc := wow_crafting_profits.WoWCpCRunner{
		Helper: helper,
		Logger: logger,
	}

	closeRequested := make(chan os.Signal, 1)
	signal.Notify(closeRequested, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	redis_options, err := redis.ParseURL(uri)
	if err != nil {
		panic("redis cannot be contacted")
	}
	redisClient := redis.NewClient(redis_options)

	running := true

	go func() {
		<-closeRequested
		running = false
		cancelContext()
	}()

	job_error_return, err := json.Marshal(&globalTypes.ReturnError{
		ERROR: "Item Not Found",
	})
	if err != nil {
		panic("failure to create error")
	}

	for running {
		logger.Debug("Trying to get job")
		job_json, popErr := redisClient.BRPop(ctx, time.Minute, globalTypes.CPC_JOB_QUEUE_NAME).Result()

		logger.Sillyf("Got \"%v\" from json : %v.", job_json, popErr)

		go func() {
			if len(job_json) == 0 {
				return
			}

			job := globalTypes.RunJob{}
			err := json.Unmarshal([]byte(job_json[1]), &job)
			if err != nil {
				logger.Error("Error decoding job", err)
				return
			}

			run_id := job.JobId
			run_config := job.JobConfig

			logger.Infof(`Got new job with id %d -> %v`, run_id, run_config)
			config := globalTypes.NewRunConfig(&run_config.AddonData, run_config.Item, run_config.Count)
			job_key := fmt.Sprintf(globalTypes.CPC_JOB_RETURN_FORMAT_STRING, run_id)

			data, err := cpc.RunWithJSONConfig(config)
			if err != nil {
				logger.Info(`Invalid item search`, err)
				redisClient.SetEX(ctx, job_key, job_error_return, time.Hour)
				return
			}

			job_save, err := json.Marshal(&data)
			if err != nil {
				logger.Error("Issue marshaling js ", err)
			}

			redisClient.SetEX(ctx, job_key, job_save, time.Hour)
		}()

	}

	logger.Info("Stopping cpc-job-worker")
}
