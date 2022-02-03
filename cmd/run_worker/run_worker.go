package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/environment_variables"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/wow_crafting_profits"
)

type RunJob struct {
	JobId     string
	JobConfig struct {
		Item      globalTypes.ItemSoftIdentity
		Count     uint
		AddonData globalTypes.AddonData
	}
}

type ReturnError struct {
	ERROR string
}

const (
	CPC_JOB_QUEUE_NAME           = "cpc-job-queue:web-jobs"
	CPC_JOB_RETURN_FORMAT_STRING = "cpc-job-queue-results:%s"
)

func main() {
	cpclog.Info("Starting cpc-job-worker")

	uri := environment_variables.REDIS_URL

	ctx := context.Background()

	redis_options, err := redis.ParseURL(uri)
	if err != nil {
		panic("redis cannot be contacted")
	}
	redisClient := redis.NewClient(redis_options)

	running := true

	job_error_return, err := json.Marshal(&ReturnError{
		ERROR: "Item Not Found",
	})
	if err != nil {
		panic("failure to create error")
	}

	for running {
		cpclog.Debug("Trying to get job")
		job_json := redisClient.BRPop(ctx, time.Second*15, CPC_JOB_QUEUE_NAME).String()

		if job_json == "" {
			continue
		}

		job := RunJob{}
		err := json.Unmarshal([]byte(job_json), &job)
		if err != nil {
			cpclog.Error("Error decoding job", err)
		}

		run_id := job.JobId
		run_config := job.JobConfig

		cpclog.Infof(`Got new job with id %d -> %v`, run_id, run_config)
		config := globalTypes.NewRunConfig(&run_config.AddonData, run_config.Item, run_config.Count)
		job_key := fmt.Sprintf(CPC_JOB_RETURN_FORMAT_STRING, run_id)

		data, err := wow_crafting_profits.RunWithJSONConfig(config)
		if err != nil {
			cpclog.Info(`Invalid item search`, err)
			redisClient.SetEX(ctx, job_key, job_error_return, time.Hour)
			continue
		}

		job_save, err := json.Marshal(&data.Intermediate)
		if err != nil {
			cpclog.Error("Issue marshaling js ", err)
		}

		redisClient.SetEX(ctx, job_key, job_save, time.Hour)

	}

	cpclog.Info("Stopping cpc-job-worker")
}
