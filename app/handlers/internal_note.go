package handlers

import (
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/web"
)

// GetInternalNote returns the shared team note for a post. Collaborators and
// administrators only - the note must never reach visitors.
func GetInternalNote() web.HandlerFunc {
	return func(c *web.Context) error {
		if c.User() == nil || !c.User().IsCollaborator() {
			return c.NotFound()
		}
		number, err := c.ParamAsInt("number")
		if err != nil {
			return c.NotFound()
		}
		getPost := &query.GetPostByNumber{Number: number}
		if err := bus.Dispatch(c, getPost); err != nil {
			return c.Failure(err)
		}
		note := &query.GetInternalNote{PostID: getPost.Result.ID}
		if err := bus.Dispatch(c, note); err != nil {
			return c.Failure(err)
		}
		return c.Ok(note.Result)
	}
}

// SetInternalNote upserts the shared team note for a post. The scorecard and
// the post page both write through here - one row, always in sync.
func SetInternalNote() web.HandlerFunc {
	return func(c *web.Context) error {
		if c.User() == nil || !c.User().IsCollaborator() {
			return c.NotFound()
		}
		number, err := c.ParamAsInt("number")
		if err != nil {
			return c.NotFound()
		}
		input := new(struct {
			Content string `json:"content"`
		})
		if err := c.Bind(input); err != nil {
			return c.Failure(err)
		}
		if len(input.Content) > 10000 {
			return c.BadRequest(web.Map{"message": "Internal note is limited to 10000 characters."})
		}
		getPost := &query.GetPostByNumber{Number: number}
		if err := bus.Dispatch(c, getPost); err != nil {
			return c.Failure(err)
		}
		set := &cmd.SetInternalNote{Post: getPost.Result, Content: input.Content}
		if err := bus.Dispatch(c, set); err != nil {
			return c.Failure(err)
		}
		return c.Ok(set.Result)
	}
}
