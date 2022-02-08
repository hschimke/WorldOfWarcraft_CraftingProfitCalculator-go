package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/cpclog"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/internal/environment_variables"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
)

var redisClient *redis.Client

func init() {
	uri := environment_variables.REDIS_URL

	redis_options, err := redis.ParseURL(uri)
	if err != nil {
		panic("redis cannot be contacted")
	}
	redisClient = redis.NewClient(redis_options)
}

type jsonOutputBodyQueueData struct {
	AddonData   globalTypes.AddonData `json:"addon_data,omitempty"`
	Type        string                `json:"type,omitempty"`
	ItemId      string                `json:"item_id,omitempty"`
	Count       uint                  `json:"count,omitempty"`
	Professions []string              `json:"professions,omitempty"`
	Server      string                `json:"server,omitempty"`
	Region      string                `json:"region,omitempty"`
}

type jsonOutputBodyCheckData struct {
	RunId string `json:"run_id,omitempty"`
}

func JsonOutputQueue(w http.ResponseWriter, r *http.Request) {

	if r.Body == nil {
		http.Error(w, "body required", http.StatusBadRequest)
		return
	}

	jobUUID := uuid.New().String()

	var data jsonOutputBodyQueueData
	parseErr := json.NewDecoder(r.Body).Decode(&data)
	if parseErr != nil {
		http.Error(w, parseErr.Error(), http.StatusInternalServerError)
		return
	}

	if data.Type == "custom" {
		cpclog.Debugf(`Custom search for item: %s, server: %s, region: %s, professions: %v. JSON DATA: %d`, data.ItemId, data.Server, data.Region, data.Professions, len(data.AddonData.Inventory))
		runJob := globalTypes.RunJob{
			JobId: jobUUID,
			JobConfig: struct {
				Item      globalTypes.ItemSoftIdentity
				Count     uint
				AddonData globalTypes.AddonData
			}{
				Item:  globalTypes.NewItemFromString(data.ItemId),
				Count: data.Count,
				AddonData: globalTypes.AddonData{
					Inventory:   data.AddonData.Inventory,
					Professions: data.Professions,
					Realm: struct {
						Region_id   uint   "json:\"region_id,omitempty\""
						Region_name string "json:\"region_name,omitempty\""
						Realm_id    uint   "json:\"realm_id,omitempty\""
						Realm_name  string "json:\"realm_name,omitempty\""
					}{
						Realm_name:  data.Server,
						Region_name: data.Region,
					},
				},
			},
		}
		rjs, rjsErr := json.Marshal(runJob)
		if rjsErr != nil {
			http.Error(w, rjsErr.Error(), http.StatusInternalServerError)
			return
		}
		redisClient.LPush(context.TODO(), globalTypes.CPC_JOB_QUEUE_NAME, rjs)
	} else if data.Type == "json" {
		cpclog.Debug("json search")
		runJob := globalTypes.RunJob{
			JobId: jobUUID,
			JobConfig: struct {
				Item      globalTypes.ItemSoftIdentity
				Count     uint
				AddonData globalTypes.AddonData
			}{
				Item:      globalTypes.NewItemFromString(data.ItemId),
				Count:     data.Count,
				AddonData: data.AddonData,
			},
		}
		rjs, rjsErr := json.Marshal(runJob)
		if rjsErr != nil {
			http.Error(w, rjsErr.Error(), http.StatusInternalServerError)
			return
		}
		redisClient.LPush(context.TODO(), globalTypes.CPC_JOB_QUEUE_NAME, rjs)
	} else {
		fmt.Fprint(w, "type must be one of 'custom' or 'json'")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	fmt.Fprintf(w, "{ job_id: \"%s\" }", jobUUID)
}

func JsonOutputCheck(w http.ResponseWriter, r *http.Request) {

	if r.Body == nil {
		http.Error(w, "body required", http.StatusBadRequest)
		return
	}

	var data jsonOutputBodyCheckData
	parseErr := json.NewDecoder(r.Body).Decode(&data)
	if parseErr != nil {
		http.Error(w, parseErr.Error(), http.StatusInternalServerError)
		return
	}

	key := fmt.Sprintf(globalTypes.CPC_JOB_RETURN_FORMAT_STRING, data.RunId)
	jobDone := false

	fnd, err := redisClient.Exists(context.TODO(), key).Result()
	if err != nil {
		jobDone = false
	} else {
		jobDone = (fnd == 1)
	}

	if jobDone {
		job, jobErr := redisClient.Get(context.TODO(), key).Result()
		if jobErr != nil {
			fmt.Fprintf(w, "{ERROR:\"%s\"}", jobErr.Error())
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Write([]byte(job))
	} else {
		fmt.Fprintf(w, "{job_id:\"%s\"}", data.RunId)
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
}
