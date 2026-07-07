package handlers

import (
	"net/http"

	"github.com/getfider/fider/app/actions"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/web"
)

// ScorecardCardPage renders /scorecard/:id — one card. If PostID is set on
// the card, the linked post is stitched onto scorecard.Post so the client
// can render the read-only header block.
func ScorecardCardPage() web.HandlerFunc {
	return func(c *web.Context) error {
		id, err := c.ParamAsInt("id")
		if err != nil {
			return c.NotFound()
		}
		get := &query.GetScorecardByID{ID: id}
		if err := bus.Dispatch(c, get); err != nil {
			return c.Failure(err)
		}
		if get.Result.PostID != nil {
			p := &query.GetPostByID{PostID: *get.Result.PostID}
			if perr := bus.Dispatch(c, p); perr == nil {
				get.Result.Post = p.Result
			}
			// non-fatal: post may have been deleted; header just falls back
		}
		return c.Page(http.StatusOK, web.Props{
			Page:  "Scorecard/ScorecardCard.page",
			Title: get.Result.Title,
			Data: web.Map{
				"scorecard": get.Result,
			},
		})
	}
}

// CreateScorecard inserts a new card (optionally linked to a post) and returns
// the created row so the client can redirect to /scorecard/:id.
func CreateScorecard() web.HandlerFunc {
	return func(c *web.Context) error {
		action := new(actions.CreateScorecard)
		if result := c.BindTo(action); !result.Ok {
			return c.HandleValidation(result)
		}
		title := action.Title
		if title == "" && action.PostID != nil {
			// Auto-title from the linked post.
			p := &query.GetPostByID{PostID: *action.PostID}
			if err := bus.Dispatch(c, p); err == nil && p.Result != nil {
				title = p.Result.Title
			}
		}
		create := &cmd.CreateScorecard{PostID: action.PostID, Title: title}
		if err := bus.Dispatch(c, create); err != nil {
			return c.Failure(err)
		}
		return c.Ok(create.Result)
	}
}

// UpdateScorecard writes new title + values JSON.
func UpdateScorecard() web.HandlerFunc {
	return func(c *web.Context) error {
		action := new(actions.UpdateScorecardValues)
		if result := c.BindTo(action); !result.Ok {
			return c.HandleValidation(result)
		}
		update := &cmd.UpdateScorecardValues{
			ID:     action.ID,
			Title:  action.Title,
			Values: []byte(action.Values),
		}
		if err := bus.Dispatch(c, update); err != nil {
			return c.Failure(err)
		}
		return c.Ok(web.Map{})
	}
}
