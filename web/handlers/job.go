package handlers

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	lighthousev1alpha1 "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned/typed/lighthouse/v1alpha1"
	"github.com/sirupsen/logrus"
	"github.com/unrolled/render"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/printers"
)

type JobHandler struct {
	LighthouseJobClient lighthousev1alpha1.LighthouseJobInterface
	Render              *render.Render
	Logger              *logrus.Logger
}

func (h *JobHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		vars    = mux.Vars(r)
		jobName = vars["job"]
	)

	ctx := context.Background()
	job, err := h.LighthouseJobClient.Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if job.APIVersion == "" {
		job.APIVersion = "lighthouse.jenkins.io/v1alpha1"
	}
	if job.Kind == "" {
		job.Kind = "LighthouseJob"
	}
	err = new(printers.YAMLPrinter).PrintObj(job, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
