package webui

import (
	"regexp"
	"strings"
	"time"

	lhv1alpha1 "github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
)

type Jobs []Job

type Job struct {
	Name        string
	Type        string
	EventGUID   string
	Owner       string
	Repository  string
	Branch      string
	Build       string
	Context     string
	State       string
	Description string
	ReportURL   string
	TraceID     string
	Start       time.Time
	End         time.Time
	Duration    time.Duration
}

func (j Job) PullRequestNumber() string {
	if strings.HasPrefix(j.Branch, "PR-") {
		return strings.TrimPrefix(j.Branch, "PR-")
	}
	return ""
}

func JobFromLighthouseJob(lhjob *lhv1alpha1.LighthouseJob) Job {
	j := Job{
		Name:        lhjob.Name,
		Type:        string(lhjob.Spec.Type),
		EventGUID:   lhjob.Labels["event-GUID"],
		Owner:       lhjob.Labels["lighthouse.jenkins-x.io/refs.org"],
		Repository:  lhjob.Labels["lighthouse.jenkins-x.io/refs.repo"],
		Branch:      lhjob.Labels["lighthouse.jenkins-x.io/branch"],
		Build:       lhjob.Labels["lighthouse.jenkins-x.io/buildNum"],
		Context:     lhjob.Labels["lighthouse.jenkins-x.io/context"],
		State:       string(lhjob.Status.State),
		Description: lhjob.Status.Description,
		ReportURL:   lhjob.Status.ReportURL,
		TraceID:     extractTraceIDFromLighthouseJob(lhjob),
		Start:       lhjob.Status.StartTime.Time,
	}
	if lhjob.Status.CompletionTime != nil {
		j.End = lhjob.Status.CompletionTime.Time
		j.Duration = j.End.Sub(j.Start)
	}
	return j
}

var traceCtxRegExp = regexp.MustCompile("^(?P<version>[0-9a-f]{2})-(?P<traceID>[a-f0-9]{32})-(?P<spanID>[a-f0-9]{16})-(?P<traceFlags>[a-f0-9]{2})(?:-.*)?$")

func extractTraceIDFromLighthouseJob(lhjob *lhv1alpha1.LighthouseJob) string {
	if traceID := lhjob.Annotations["lighthouse.jenkins-x.io/traceID"]; traceID != "" {
		return traceID
	}

	traceCtx := lhjob.Annotations["lighthouse.jenkins-x.io/traceparent"]
	if traceCtx == "" {
		return ""
	}

	matches := traceCtxRegExp.FindStringSubmatch(traceCtx)
	if len(matches) < 5 {
		return ""
	}
	if len(matches[2]) != 32 {
		return ""
	}
	return matches[2]
}
