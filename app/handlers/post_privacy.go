package handlers

import (
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/web"
)

// SetPostPrivacy changes whether an idea is private. Staff may change any idea
// in either direction. A member may only make their own private idea public,
// and only when the tenant allows it; a member can never make an
// already-submitted idea private. Enforced here, not just in the UI.
func SetPostPrivacy() web.HandlerFunc {
	return func(c *web.Context) error {
		user := c.User()
		if user == nil {
			return c.NotFound()
		}
		number, err := c.ParamAsInt("number")
		if err != nil {
			return c.NotFound()
		}
		input := new(struct {
			IsPrivate bool `json:"isPrivate"`
		})
		if err := c.Bind(input); err != nil {
			return c.Failure(err)
		}
		getPost := &query.GetPostByNumber{Number: number}
		if err := bus.Dispatch(c, getPost); err != nil {
			return c.Failure(err)
		}
		post := getPost.Result

		if !user.IsCollaborator() {
			if post.User == nil || post.User.ID != user.ID {
				return c.NotFound()
			}
			if input.IsPrivate || !post.IsPrivate || !c.Tenant().MembersCanPublishPrivate {
				return c.Forbidden()
			}
		}

		set := &cmd.SetPostPrivacy{Post: post, IsPrivate: input.IsPrivate}
		if err := bus.Dispatch(c, set); err != nil {
			return c.Failure(err)
		}
		return c.Ok(web.Map{"isPrivate": input.IsPrivate})
	}
}
