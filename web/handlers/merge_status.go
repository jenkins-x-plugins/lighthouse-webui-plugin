package handlers

import (
	"net/http"
	"strings"

	webui "github.com/jenkins-x-plugins/lighthouse-webui-plugin"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/unrolled/render"
	"gopkg.in/yaml.v2"
)

type MergeStatusHandler struct {
	Store  *webui.Store
	Render *render.Render
	Logger *logrus.Logger
}

func (h *MergeStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		vars       = mux.Vars(r)
		owner      = vars["owner"]
		repository = vars["repository"]
		branch     = vars["branch"]
		renderYAML = strings.HasSuffix(r.RequestURI, ".yaml")
	)

	pools := h.Store.QueryMergeStatus(webui.MergeStatusQuery{
		Owner:      owner,
		Repository: repository,
		Branch:     branch,
	})

	if renderYAML {
		var keeperPools []interface{}
		for _, pool := range pools {
			keeperPools = append(keeperPools, pool.KeeperPool)
		}

		enc := yaml.NewEncoder(w)
		if err := enc.Encode(keeperPools); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := enc.Close(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}

	err := h.Render.HTML(w, http.StatusOK, "merge_status", struct {
		Pools      []webui.MergePool
		Owner      string
		Repository string
		Branch     string
	}{
		pools,
		owner,
		repository,
		branch,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
