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

type MergeHistoryHandler struct {
	Store  *webui.Store
	Render *render.Render
	Logger *logrus.Logger
}

func (h *MergeHistoryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		vars       = mux.Vars(r)
		owner      = vars["owner"]
		repository = vars["repository"]
		branch     = vars["branch"]
		renderYAML = strings.HasSuffix(r.RequestURI, ".yaml")
	)

	records := h.Store.QueryMergeHistory(webui.MergeHistoryQuery{
		Owner:      owner,
		Repository: repository,
		Branch:     branch,
	})

	if renderYAML {
		var keeperRecords []interface{}
		for _, record := range records {
			keeperRecords = append(keeperRecords, record.KeeperRecord)
		}

		enc := yaml.NewEncoder(w)
		if err := enc.Encode(keeperRecords); err != nil {
			h.Logger.WithError(err).Error("failed to encode merge history in YAML")
			return
		}
		if err := enc.Close(); err != nil {
			h.Logger.WithError(err).Error("failed to close YAML encoder for merge history")
			return
		}
		return
	}

	err := h.Render.HTML(w, http.StatusOK, "merge_history", struct {
		Records    []webui.MergeRecord
		Owner      string
		Repository string
		Branch     string
	}{
		records,
		owner,
		repository,
		branch,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
