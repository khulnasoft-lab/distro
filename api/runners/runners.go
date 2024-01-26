package runners

import (
	"net/http"

	"github.com/gorilla/context"
	"github.com/khulnasoft-lab/distro/api/helpers"
	"github.com/khulnasoft-lab/distro/db"
	"github.com/khulnasoft-lab/distro/lib"
	"github.com/khulnasoft-lab/distro/services/runners"
	"github.com/khulnasoft-lab/distro/util"
)

func RunnerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		token := r.Header.Get("X-API-Token")

		if token == "" {
			helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "Invalid token",
			})
			return
		}

		runnerID, err := helpers.GetIntParam("runner_id", w, r)

		if err != nil {
			helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{
				"error": "runner_id required",
			})
			return
		}

		store := helpers.Store(r)

		runner, err := store.GetGlobalRunner(runnerID)

		if err != nil {
			helpers.WriteJSON(w, http.StatusNotFound, map[string]string{
				"error": "Runner not found",
			})
			return
		}

		if runner.Token != token {
			helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "Invalid token",
			})
			return
		}

		context.Set(r, "runner", runner)
		next.ServeHTTP(w, r)
	})
}

func GetRunner(w http.ResponseWriter, r *http.Request) {
	runner := context.Get(r, "runner").(db.Runner)

	data := runners.RunnerState{
		AccessKeys: make(map[int]db.AccessKey),
	}

	tasks := helpers.TaskPool(r).GetRunningTasks()

	for _, tsk := range tasks {
		if tsk.RunnerID != runner.ID {
			continue
		}

		if tsk.Task.Status == lib.TaskStartingStatus {

			data.NewJobs = append(data.NewJobs, runners.JobData{
				Username:        tsk.Username,
				IncomingVersion: tsk.IncomingVersion,
				Task:            tsk.Task,
				Template:        tsk.Template,
				Inventory:       tsk.Inventory,
				Repository:      tsk.Repository,
				Environment:     tsk.Environment,
			})

			if tsk.Inventory.SSHKeyID != nil {
				err := tsk.Inventory.SSHKey.DeserializeSecret()
				if err != nil {
					// TODO: return error
				}
				data.AccessKeys[*tsk.Inventory.SSHKeyID] = tsk.Inventory.SSHKey
			}

			if tsk.Inventory.BecomeKeyID != nil {
				err := tsk.Inventory.BecomeKey.DeserializeSecret()
				if err != nil {
					// TODO: return error
				}
				data.AccessKeys[*tsk.Inventory.BecomeKeyID] = tsk.Inventory.BecomeKey
			}

			if tsk.Template.VaultKeyID != nil {
				err := tsk.Template.VaultKey.DeserializeSecret()
				if err != nil {
					// TODO: return error
				}
				data.AccessKeys[*tsk.Template.VaultKeyID] = tsk.Template.VaultKey
			}

			data.AccessKeys[tsk.Repository.SSHKeyID] = tsk.Repository.SSHKey

		} else {
			data.CurrentJobs = append(data.CurrentJobs, runners.JobState{
				ID:     tsk.Task.ID,
				Status: tsk.Task.Status,
			})
		}
	}

	helpers.WriteJSON(w, http.StatusOK, data)
}

func UpdateRunner(w http.ResponseWriter, r *http.Request) {

	runner := context.Get(r, "runner").(db.Runner)

	var body runners.RunnerProgress

	if !helpers.Bind(w, r, &body) {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid format",
		})
		return
	}

	taskPool := helpers.TaskPool(r)

	if body.Jobs == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	for _, job := range body.Jobs {
		tsk := taskPool.GetTask(job.ID)

		if tsk == nil {
			// TODO: log
			continue
		}

		if tsk.RunnerID != runner.ID {
			// TODO: add error message
			continue
		}

		for _, logRecord := range job.LogRecords {
			tsk.Log2(logRecord.Message, logRecord.Time)
		}

		tsk.SetStatus(job.Status)
	}

	w.WriteHeader(http.StatusNoContent)
}

func RegisterRunner(w http.ResponseWriter, r *http.Request) {
	var register runners.RunnerRegistration

	if !helpers.Bind(w, r, &register) {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid format",
		})
		return
	}

	if util.Config.RunnerRegistrationToken == "" || register.RegistrationToken != util.Config.RunnerRegistrationToken {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid registration token",
		})
		return
	}

	runner, err := helpers.Store(r).CreateRunner(db.Runner{
		Webhook:          register.Webhook,
		MaxParallelTasks: register.MaxParallelTasks,
	})

	if err != nil {
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Unexpected error",
		})
		return
	}

	res := runners.RunnerConfig{
		RunnerID: runner.ID,
		Token:    runner.Token,
	}

	helpers.WriteJSON(w, http.StatusOK, res)
}
