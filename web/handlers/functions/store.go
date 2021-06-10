package functions

import (
	webui "github.com/jenkins-x-plugins/lighthouse-webui-plugin"
)

func LoadJobsForEventFunc(store *webui.Store) func(string) webui.Jobs {
	return func(eventGUID string) webui.Jobs {
		return loadJobsForEvent(eventGUID, store)
	}
}

func loadJobsForEvent(eventGUID string, store *webui.Store) webui.Jobs {
	if store == nil {
		return webui.Jobs{}
	}
	if eventGUID == "" {
		return webui.Jobs{}
	}

	jobs, err := store.QueryJobs(webui.JobsQuery{
		EventGUID: eventGUID,
	})
	if err != nil {
		return webui.Jobs{}
	}

	return *jobs
}

func LoadEventForJobFunc(store *webui.Store) func(string) *webui.Event {
	return func(eventGUID string) *webui.Event {
		return loadEventForJob(eventGUID, store)
	}
}

func loadEventForJob(eventGUID string, store *webui.Store) *webui.Event {
	if store == nil {
		return nil
	}
	if eventGUID == "" {
		return nil
	}

	events, err := store.QueryEvents(webui.EventsQuery{
		GUID: eventGUID,
	})
	if err != nil {
		return nil
	}

	if len(*events) == 0 {
		return nil
	}

	return &(*events)[0]
}
