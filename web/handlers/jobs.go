package handlers

import (
	"net/http"
	"strings"

	webui "github.com/jenkins-x-plugins/lighthouse-webui-plugin"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/unrolled/render"
)

type JobsHandler struct {
	Store  *webui.Store
	Render *render.Render
	Logger *logrus.Logger
}

func (h *JobsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	jobs, err := h.Store.QueryJobs(webui.JobsQuery{
		Owner:      owner,
		Repository: repository,
		Branch:     branch,
		Query:      query,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.Render.HTML(w, http.StatusOK, "jobs", struct {
		Jobs       *webui.Jobs
		Owner      string
		Repository string
		Branch     string
		Query      string
	}{
		jobs,
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
