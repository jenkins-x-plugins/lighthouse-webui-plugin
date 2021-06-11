package webui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/goutils"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/sirupsen/logrus"
)

type EventHandler struct {
	Store  *Store
	Logger *logrus.Logger
}

func (h *EventHandler) HandleWebhook(webhook scm.Webhook) error {
	log := h.Logger.
		WithField("repo", webhook.Repository().FullName).
		WithField("kind", webhook.Kind())

	event := convertWebhookToEvent(webhook)
	if event == nil {
		log.Trace("Ignoring webhook event")
		return nil
	}
	log.Debug("Handling webhook event")

	event.Kind = string(webhook.Kind())
	event.Owner = webhook.Repository().Namespace
	event.Repository = webhook.Repository().Name
	if event.Branch == "" {
		event.Branch = webhook.Repository().Branch
	}
	if event.Time.IsZero() {
		event.Time = time.Now()
	}

	return h.Store.AddEvent(*event)
}

func convertWebhookToEvent(webhook scm.Webhook) *Event {
	switch event := webhook.(type) {
	case *scm.PingHook:
		return &Event{
			GUID:   event.GUID,
			Sender: event.Sender.Login,
		}
	case *scm.PushHook:
		ref := event.Ref
		ref = strings.TrimPrefix(ref, "refs/heads/")
		ref = strings.TrimPrefix(ref, "refs/tags/")
		e := Event{
			GUID:    event.GUID,
			Details: ref,
			Sender:  event.Sender.Login,
			Branch:  ref,
		}
		if event.Deleted {
			e.Action = scm.ActionDelete.String()
		}
		if event.Created {
			e.Action = scm.ActionCreate.String()
		}
		return &e
	case *scm.PullRequestHook:
		var details string
		switch event.Action {
		case scm.ActionLabel, scm.ActionUnlabel:
			details = fmt.Sprintf("%s: %s", event.Action.String(), event.Label.Name)
		case scm.ActionAssigned, scm.ActionUnassigned:
			var assignees []string
			for _, assignee := range event.PullRequest.Assignees {
				assignees = append(assignees, assignee.Login)
			}
			details = fmt.Sprintf("%s. Assignees: %s", event.Action.String(), strings.Join(assignees, ", "))
		default:
			details = event.Action.String()
		}
		return &Event{
			GUID:    event.GUID,
			Action:  event.Action.String(),
			Details: details,
			Sender:  event.Sender.Login,
			Branch:  fmt.Sprintf("PR-%d", event.PullRequest.Number),
			URL:     event.PullRequest.Link,
		}
	case *scm.PullRequestCommentHook:
		comment, _ := goutils.Abbreviate(event.Comment.Body, 50)
		return &Event{
			GUID:    event.GUID,
			Action:  event.Action.String(),
			Details: comment,
			Sender:  event.Sender.Login,
			Branch:  fmt.Sprintf("PR-%d", event.PullRequest.Number),
			URL:     event.Comment.Link,
		}
	case *scm.IssueCommentHook:
		comment, _ := goutils.Abbreviate(event.Comment.Body, 50)
		e := Event{
			GUID:    event.GUID,
			Action:  event.Action.String(),
			Details: comment,
			Sender:  event.Sender.Login,
			URL:     event.Comment.Link,
		}
		if event.Issue.PullRequest {
			e.Branch = fmt.Sprintf("PR-%d", event.Issue.Number)
		}
		return &e
	default:
		return nil
	}
}
