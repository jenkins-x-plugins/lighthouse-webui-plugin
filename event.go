package webui

import (
	"strings"
	"time"
)

type Events struct {
	Events []Event
	Counts struct {
		Kinds        map[string]int
		Repositories map[string]int
		Senders      map[string]int
	}
}

type Event struct {
	GUID       string
	Owner      string
	Repository string
	Branch     string
	Kind       string
	Action     string
	Details    string
	URL        string
	Sender     string
	Time       time.Time
}

func (e Event) PullRequestNumber() string {
	if strings.HasPrefix(e.Branch, "PR-") {
		return strings.TrimPrefix(e.Branch, "PR-")
	}
	return ""
}
