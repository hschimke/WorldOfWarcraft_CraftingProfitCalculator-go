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
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/environment_variables"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/wow_crafting_profits"
)

func main() {

	cpclog.Info("Starting cpc-job-worker")

	uri := environment_variables.REDIS_URL

	ctx, cancelContext := context.WithCancel(context.Background())

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
		cpclog.Debug("Trying to get job")
		job_json, popErr := redisClient.BRPop(ctx, time.Minute, globalTypes.CPC_JOB_QUEUE_NAME).Result()

		cpclog.Sillyf("Got \"%v\" from json : %v.", job_json, popErr)

		//go func() {
		if len(job_json) == 0 {
			return
		}

		job := globalTypes.RunJob{}
		err := json.Unmarshal([]byte(job_json[1]), &job)
		if err != nil {
			cpclog.Error("Error decoding job", err)
			return
		}

		run_id := job.JobId
		run_config := job.JobConfig

		cpclog.Infof(`Got new job with id %d -> %v`, run_id, run_config)
		config := globalTypes.NewRunConfig(&run_config.AddonData, run_config.Item, run_config.Count)
		job_key := fmt.Sprintf(globalTypes.CPC_JOB_RETURN_FORMAT_STRING, run_id)

		data, err := wow_crafting_profits.RunWithJSONConfig(config)
		if err != nil {
			cpclog.Info(`Invalid item search`, err)
			redisClient.SetEX(ctx, job_key, job_error_return, time.Hour)
			return
		}

		job_save, err := json.Marshal(&data.Intermediate)
		if err != nil {
			cpclog.Error("Issue marshaling js ", err)
		}

		redisClient.SetEX(ctx, job_key, job_save, time.Hour)
		//}()

	}

	cpclog.Info("Stopping cpc-job-worker")
}
