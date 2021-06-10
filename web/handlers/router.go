package handlers

import (
	htmltemplate "html/template"
	"net/http"
	"text/template"

	webui "github.com/jenkins-x-plugins/lighthouse-webui-plugin"
	"github.com/jenkins-x-plugins/lighthouse-webui-plugin/internal/lighthouse"
	"github.com/jenkins-x-plugins/lighthouse-webui-plugin/internal/version"
	"github.com/jenkins-x-plugins/lighthouse-webui-plugin/web/handlers/functions"

	"github.com/Masterminds/sprig/v3"
	"github.com/gorilla/mux"
	lighthousev1alpha1 "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned/typed/lighthouse/v1alpha1"
	"github.com/sirupsen/logrus"
	"github.com/unrolled/render"
	"github.com/urfave/negroni/v2"
)

type Router struct {
	Store                 *webui.Store
	LighthouseHandler     *lighthouse.Handler
	LighthouseJobClient   lighthousev1alpha1.LighthouseJobInterface
	EventTraceURLTemplate string
	Logger                *logrus.Logger
	render                *render.Render
}

func (r Router) Handler() (http.Handler, error) {
	var (
		eventTraceURLTemplate *template.Template
		err                   error
	)
	if len(r.EventTraceURLTemplate) > 0 {
		eventTraceURLTemplate, err = template.New("eventTraceURL").Funcs(sprig.TxtFuncMap()).Parse(r.EventTraceURLTemplate)
		if err != nil {
			return nil, err
		}
	}

	r.render = render.New(render.Options{
		Directory:     "web/templates",
		Layout:        "layout",
		IsDevelopment: version.Version == "dev",
		Funcs: []htmltemplate.FuncMap{
			sprig.HtmlFuncMap(),
			htmltemplate.FuncMap{
				"traceURL":         functions.TraceURLFunc(eventTraceURLTemplate),
				"loadJobsForEvent": functions.LoadJobsForEventFunc(r.Store),
				"loadEventForJob":  functions.LoadEventForJobFunc(r.Store),
				"vdate":            functions.VDate,
				"appVersion":       functions.AppVersion,
			},
		},
	})

	router := mux.NewRouter()
	router.StrictSlash(true)

	router.Handle("/healthz", healthzHandler())
	router.Handle("/lighthouse/events", r.LighthouseHandler) // TODO move to its own server?

	mergeStatusHandler := &MergeStatusHandler{
		Store:  r.Store,
		Render: r.render,
		Logger: r.Logger,
	}
	router.Handle("/merge/status", mergeStatusHandler)
	router.Handle("/merge/status.yaml", mergeStatusHandler)
	router.Handle("/merge/status/{owner}.yaml", mergeStatusHandler)
	router.Handle("/merge/status/{owner}/{repository}.yaml", mergeStatusHandler)
	router.Handle("/merge/status/{owner}/{repository}/{branch}.yaml", mergeStatusHandler)
	router.Handle("/merge/status/{owner}", mergeStatusHandler)
	router.Handle("/merge/status/{owner}/{repository}", mergeStatusHandler)
	router.Handle("/merge/status/{owner}/{repository}/{branch}", mergeStatusHandler)

	mergeHistoryHandler := &MergeHistoryHandler{
		Store:  r.Store,
		Render: r.render,
		Logger: r.Logger,
	}
	router.Handle("/merge/history", mergeHistoryHandler)
	router.Handle("/merge/history.yaml", mergeHistoryHandler)
	router.Handle("/merge/history/{owner}.yaml", mergeHistoryHandler)
	router.Handle("/merge/history/{owner}/{repository}.yaml", mergeHistoryHandler)
	router.Handle("/merge/history/{owner}/{repository}/{branch}.yaml", mergeHistoryHandler)
	router.Handle("/merge/history/{owner}", mergeHistoryHandler)
	router.Handle("/merge/history/{owner}/{repository}", mergeHistoryHandler)
	router.Handle("/merge/history/{owner}/{repository}/{branch}", mergeHistoryHandler)

	jobHandler := &JobHandler{
		LighthouseJobClient: r.LighthouseJobClient,
		Render:              r.render,
		Logger:              r.Logger,
	}
	router.Handle("/job/{job}.yaml", jobHandler)

	jobsHandler := &JobsHandler{
		Store:  r.Store,
		Render: r.render,
		Logger: r.Logger,
	}
	router.Handle("/jobs/", jobsHandler)
	router.Handle("/jobs/{owner}", jobsHandler)
	router.Handle("/jobs/{owner}/{repository}", jobsHandler)
	router.Handle("/jobs/{owner}/{repository}/{branch}", jobsHandler)

	eventsHandler := &EventsHandler{
		Store:  r.Store,
		Render: r.render,
		Logger: r.Logger,
	}
	router.Handle("/events", eventsHandler)
	router.Handle("/events/{owner}", eventsHandler)
	router.Handle("/events/{owner}/{repository}", eventsHandler)
	router.Handle("/events/{owner}/{repository}/{branch}", eventsHandler)

	router.Handle("/", http.RedirectHandler("/events", http.StatusPermanentRedirect))

	handler := negroni.New(
		negroni.NewRecovery(),
		&negroni.Static{
			Dir:       http.Dir("web/static"),
			Prefix:    "/static",
			IndexFile: "index.html",
		},
		negroni.Wrap(router),
	)

	return handler, nil
}

func healthzHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
