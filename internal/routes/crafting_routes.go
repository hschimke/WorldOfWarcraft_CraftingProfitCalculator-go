package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
)

type jsonOutputBodyQueueData struct {
	//AddonData   globalTypes.AddonData `json:"addon_data,omitempty"`
	AddonData         string   `json:"addon_data,omitempty"`
	Type              string   `json:"type,omitempty"`
	ItemId            string   `json:"item_id,omitempty"`
	Count             uint     `json:"count,omitempty"`
	UseAllProfessions bool     `json:"use_all_professions"`
	Professions       []string `json:"professions,omitempty"`
	Server            string   `json:"server,omitempty"`
	Region            string   `json:"region,omitempty"`
}

// Queue up a CPC run
func (routes *CPCRoutes) JsonOutputQueue(w http.ResponseWriter, r *http.Request) {

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

	var adData globalTypes.AddonData

	adDataErr := json.Unmarshal([]byte(data.AddonData), &adData)
	if adDataErr != nil {
		adData = globalTypes.AddonData{}
	}

	switch data.Type {
	case "custom":
		routes.Logger.Debugf(`Custom search for item: %s, server: %s, region: %s, professions: %v. JSON DATA: %d`, data.ItemId, data.Server, data.Region, data.Professions, len(adData.Inventory))
		runJob := globalTypes.RunJob{
			JobId: jobUUID,
			JobConfig: struct {
				Item              globalTypes.ItemSoftIdentity
				Count             uint
				UseAllProfessions bool
				AddonData         globalTypes.AddonData
			}{
				Item:              globalTypes.NewItemFromString(data.ItemId),
				Count:             data.Count,
				UseAllProfessions: data.UseAllProfessions,
				AddonData: globalTypes.AddonData{
					Inventory:   adData.Inventory, //data.AddonData.Inventory,
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
		routes.redisClient.LPush(context.TODO(), globalTypes.CPC_JOB_QUEUE_NAME, rjs)
	case "json":
		routes.Logger.Debug("json search")
		runJob := globalTypes.RunJob{
			JobId: jobUUID,
			JobConfig: struct {
				Item              globalTypes.ItemSoftIdentity
				Count             uint
				UseAllProfessions bool
				AddonData         globalTypes.AddonData
			}{
				Item:              globalTypes.NewItemFromString(data.ItemId),
				Count:             data.Count,
				UseAllProfessions: false,
				AddonData:         adData, //data.AddonData,
			},
		}
		rjs, rjsErr := json.Marshal(runJob)
		if rjsErr != nil {
			http.Error(w, rjsErr.Error(), http.StatusInternalServerError)
			return
		}
		routes.redisClient.LPush(context.TODO(), globalTypes.CPC_JOB_QUEUE_NAME, rjs)
	default:
		http.Error(w, "type must be one of 'custom' or 'json'", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	json.NewEncoder(w).Encode(globalTypes.QueuedJobReturn{
		JobId: jobUUID,
	})
}

// Check to see if a queued CPC run has completed
func (routes *CPCRoutes) JsonOutputCheck(w http.ResponseWriter, r *http.Request) {

	if r.Body == nil {
		http.Error(w, "body required", http.StatusBadRequest)
		return
	}

	var data globalTypes.QueuedJobReturn
	parseErr := json.NewDecoder(r.Body).Decode(&data)
	if parseErr != nil {
		http.Error(w, parseErr.Error(), http.StatusInternalServerError)
		return
	}

	key := fmt.Sprintf(globalTypes.CPC_JOB_RETURN_FORMAT_STRING, data.JobId)
	jobDone := false

	fnd, err := routes.redisClient.Exists(context.TODO(), key).Result()
	if err != nil {
		jobDone = false
	} else {
		jobDone = (fnd == 1)
	}

	if jobDone {
		job, jobErr := routes.redisClient.Get(context.TODO(), key).Result()
		if jobErr != nil {
			json.NewEncoder(w).Encode(globalTypes.ReturnError{
				ERROR: jobErr.Error(),
			})
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Write([]byte(job))
	} else {
		json.NewEncoder(w).Encode(globalTypes.QueuedJobReturn{
			JobId: data.JobId,
		})
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
}
