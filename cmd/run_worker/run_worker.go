package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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
	if err := environment_variables.Load(); err != nil {
		log.Fatalf("failed to load environment variables: %v", err)
	}

	logger := cpclog.NewCpCLog(cpclog.GetLevel(environment_variables.LOG_LEVEL))

	logger.Info("Starting cpc-job-worker")

	uri := environment_variables.REDIS_URL

	ctx, cancelContext := context.WithCancel(context.Background())
	defer cancelContext()

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

	go func() {
		<-closeRequested
		logger.Info("Shutdown signal received, stopping...")
		cancelContext()
	}()

	job_error_return, _ := json.Marshal(&globalTypes.ReturnError{
		ERROR: "Item Not Found or Error Processing",
	})

	for {
		select {
		case <-ctx.Done():
			logger.Info("Worker context cancelled, exiting loop")
			return
		default:
			logger.Debug("Trying to get job")
			job_json, popErr := redisClient.BRPop(ctx, 10*time.Second, globalTypes.CPC_JOB_QUEUE_NAME).Result()

			if popErr != nil {
				if popErr != redis.Nil && !errors.Is(popErr, context.Canceled) {
					logger.Errorf("Error popping job: %v", popErr)
				}
				continue
			}

			if len(job_json) < 2 {
				continue
			}

			// Process job in a goroutine
			go func(jobStr string) {
				job := globalTypes.RunJob{}
				err := json.Unmarshal([]byte(jobStr), &job)
				if err != nil {
					logger.Error("Error decoding job", err)
					return
				}

				run_id := job.JobId
				run_config := job.JobConfig

				logger.Infof(`Got new job with id %s -> %v`, run_id, run_config)
				config := globalTypes.NewRunConfig(&run_config.AddonData, run_config.Item, run_config.Count)
				job_key := fmt.Sprintf(globalTypes.CPC_JOB_RETURN_FORMAT_STRING, run_id)

				// Use worker context for the run
				data, err := cpc.RunWithJSONConfig(ctx, config)
				if err != nil {
					logger.Infof("Job %s failed: %v", run_id, err)
					redisClient.SetEX(ctx, job_key, job_error_return, time.Hour)
					return
				}

				job_save, err := json.Marshal(&data)
				if err != nil {
					logger.Error("Issue marshaling job results: ", err)
					return
				}

				redisClient.SetEX(ctx, job_key, job_save, time.Hour)
			}(job_json[1])
		}
	}
}
