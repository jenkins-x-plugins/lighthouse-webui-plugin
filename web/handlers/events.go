package handlers

import (
	"net/http"
	"strings"

	webui "github.com/jenkins-x-plugins/lighthouse-webui-plugin"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/unrolled/render"
)

type EventsHandler struct {
	Store  *webui.Store
	Render *render.Render
	Logger *logrus.Logger
}

func (h *EventsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		vars       = mux.Vars(r)
		owner      = vars["owner"]
		repository = vars["repository"]
		branch     = vars["branch"]
		query      = r.URL.Query().Get("q")
	)

	if strings.HasPrefix(branch, "pr-") {
		branch = strings.ToUpper(branch)
	}

	events, err := h.Store.QueryEvents(webui.EventsQuery{
		Owner:      owner,
		Repository: repository,
		Branch:     branch,
		Query:      query,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.Render.HTML(w, http.StatusOK, "events", struct {
		Events     *webui.Events
		Owner      string
		Repository string
		Branch     string
		Query      string
	}{
		events,
		owner,
		repository,
		branch,
		query,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
