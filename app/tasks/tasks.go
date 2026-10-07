package tasks

import (
	"context"
	"fmt"
	"html"

	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/enum"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/worker"
)

func describe(name string, job worker.Job) worker.Task {
	return worker.Task{Name: name, Job: job}
}

// link returns an HTML anchor that displays its own URL. The result is
// rendered unescaped in email templates, so the URL is HTML-escaped here.
func link(baseURL, path string, args ...any) string {
	url := html.EscapeString(baseURL + fmt.Sprintf(path, args...))
	return fmt.Sprintf("<a href='%[1]s'>%[1]s</a>", url)
}

// linkWithText returns an HTML anchor with the given plain text as its label.
// The result is rendered unescaped in email templates, so both the URL and the
// text are HTML-escaped here. Callers must pass plain text, not HTML.
func linkWithText(text, baseURL, path string, args ...any) string {
	url := html.EscapeString(baseURL + fmt.Sprintf(path, args...))
	return fmt.Sprintf("<a href='%s'>%s</a>", url, html.EscapeString(text))
}

func getActiveSubscribers(ctx context.Context, post *entity.Post, channel enum.NotificationChannel, event enum.NotificationEvent) ([]*entity.User, error) {
	q := &query.GetActiveSubscribers{
		Number:  post.Number,
		Channel: channel,
		Event:   event,
	}
	err := bus.Dispatch(ctx, q)
	if err != nil {
		return nil, err
	}
	// A private idea must never notify anyone who cannot see it. This is the
	// single choke point for every notification event (new post, new comment,
	// status change, delete), so it also covers a public idea flipped private
	// while other members are still subscribed.
	if post.IsPrivate {
		return privateAudience(post, q.Result), nil
	}
	return q.Result, nil
}

// privateAudience keeps only the people allowed to see a private idea:
// collaborators/administrators and the idea's own author.
func privateAudience(post *entity.Post, users []*entity.User) []*entity.User {
	out := make([]*entity.User, 0, len(users))
	for _, u := range users {
		if u == nil {
			continue
		}
		if u.IsCollaborator() || (post.User != nil && u.ID == post.User.ID) {
			out = append(out, u)
		}
	}
	return out
}
