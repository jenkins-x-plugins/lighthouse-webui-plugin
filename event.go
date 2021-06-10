package webui

import (
	"strings"
	"time"
)

type Events []Event

type Event struct {
	GUID       string
	Owner      string
	Repository string
	Branch     string
	Kind       string
	Details    string
	Sender     string
	Time       time.Time
}

func (e Event) PullRequestNumber() string {
	if strings.HasPrefix(e.Branch, "PR-") {
		return strings.TrimPrefix(e.Branch, "PR-")
	}
	return ""
}
