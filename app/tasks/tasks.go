package tasks

import (
	"context"
	"fmt"

	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/enum"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/worker"
)

func describe(name string, job worker.Job) worker.Task {
	return worker.Task{Name: name, Job: job}
}

func link(baseURL, path string, args ...any) string {
	return fmt.Sprintf("<a href='%[1]s%[2]s'>%[1]s%[2]s</a>", baseURL, fmt.Sprintf(path, args...))
}

func linkWithText(text, baseURL, path string, args ...any) string {
	return fmt.Sprintf("<a href='%s%s'>%s</a>", baseURL, fmt.Sprintf(path, args...), text)
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
	// while non-collaborators are still subscribed.
	if post.IsPrivate {
		return collaboratorsOnly(q.Result), nil
	}
	return q.Result, nil
}

// collaboratorsOnly keeps only collaborators/administrators.
func collaboratorsOnly(users []*entity.User) []*entity.User {
	out := make([]*entity.User, 0, len(users))
	for _, u := range users {
		if u != nil && u.IsCollaborator() {
			out = append(out, u)
		}
	}
	return out
}
