package handlers

import (
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/web"
)

// SetPostPrivacy toggles whether an idea is private (collaborators/admins
// only). Visitors never reach this — the check is here, not just in the UI.
func SetPostPrivacy() web.HandlerFunc {
	return func(c *web.Context) error {
		if c.User() == nil || !c.User().IsCollaborator() {
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
		set := &cmd.SetPostPrivacy{Post: getPost.Result, IsPrivate: input.IsPrivate}
		if err := bus.Dispatch(c, set); err != nil {
			return c.Failure(err)
		}
		return c.Ok(web.Map{"isPrivate": input.IsPrivate})
	}
}
