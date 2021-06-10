package webui

import (
	"context"
	"strings"
	"time"

	lhv1alpha1 "github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	lhclientset "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned"
	lhinformers "github.com/jenkins-x/lighthouse/pkg/client/informers/externalversions"
	"github.com/sirupsen/logrus"
)

type JobInformer struct {
	LHClient       *lhclientset.Clientset
	Namespace      string
	ResyncInterval time.Duration
	Store          *Store
	Logger         *logrus.Logger
}

func (i *JobInformer) Start(ctx context.Context) {
	informerFactory := lhinformers.NewSharedInformerFactoryWithOptions(
		i.LHClient,
		i.ResyncInterval,
		lhinformers.WithNamespace(i.Namespace),
	)
	informerFactory.Lighthouse().V1alpha1().LighthouseJobs().Informer().AddEventHandler(i)
	informerFactory.Start(ctx.Done())
}

func (i *JobInformer) OnAdd(obj interface{}) {
	job, ok := obj.(*lhv1alpha1.LighthouseJob)
	if !ok {
		return
	}

	i.indexJob(job, "index")
}

func (i *JobInformer) OnUpdate(oldObj, newObj interface{}) {
	job, ok := newObj.(*lhv1alpha1.LighthouseJob)
	if !ok {
		return
	}

	i.indexJob(job, "re-index")
}
func (i *JobInformer) OnDelete(obj interface{}) {
	job, ok := obj.(*lhv1alpha1.LighthouseJob)
	if !ok {
		return
	}

	if i.Logger != nil && i.Logger.IsLevelEnabled(logrus.DebugLevel) {
		i.Logger.WithField("Job", job.Name).Debug("Deleting Job")
	}
	err := i.Store.DeleteJob(job.Name)
	if err != nil && i.Logger != nil {
		i.Logger.WithError(err).WithField("Job", job.Name).Error("failed to delete Job")
	}
}

func (i *JobInformer) indexJob(job *lhv1alpha1.LighthouseJob, operation string) {
	if i.Logger != nil && i.Logger.IsLevelEnabled(logrus.DebugLevel) {
		i.Logger.WithField("Job", job.Name).Debugf("%sing Job", strings.Title(operation))
	}
	j := JobFromLighthouseJob(job)
	err := i.Store.AddJob(j)
	if err != nil && i.Logger != nil {
		i.Logger.WithError(err).WithField("Job", job.Name).Errorf("failed to %s Job", operation)
	}
}
