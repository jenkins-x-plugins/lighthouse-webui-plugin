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
		return &Event{
			GUID:    event.GUID,
			Details: fmt.Sprintf("pushed to %s", ref),
			Sender:  event.Sender.Login,
			Branch:  ref,
		}
	case *scm.PullRequestHook:
		return &Event{
			GUID:    event.GUID,
			Details: fmt.Sprintf("PR #%d %s", event.PullRequest.Number, event.Action.String()),
			Sender:  event.Sender.Login,
			Branch:  fmt.Sprintf("PR-%d", event.PullRequest.Number),
		}
	case *scm.PullRequestCommentHook:
		comment, _ := goutils.Abbreviate(event.Comment.Body, 30)
		return &Event{
			GUID:    event.GUID,
			Details: fmt.Sprintf("PR #%d comment %s: %s", event.PullRequest.Number, event.Action.String(), comment),
			Sender:  event.Sender.Login,
			Branch:  fmt.Sprintf("PR-%d", event.PullRequest.Number),
		}
	case *scm.IssueCommentHook:
		comment, _ := goutils.Abbreviate(event.Comment.Body, 30)
		e := Event{
			GUID:    event.GUID,
			Details: fmt.Sprintf("Issue #%d comment %s: %s", event.Issue.Number, event.Action.String(), comment),
			Sender:  event.Sender.Login,
		}
		if event.Issue.PullRequest {
			e.Branch = fmt.Sprintf("PR-%d", event.Issue.Number)
		}
		return &e
	default:
		return nil
	}
}
