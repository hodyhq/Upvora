package cmd

import "github.com/getfider/fider/app/models/entity"

// SetInternalNote upserts the shared team note for a post.
type SetInternalNote struct {
	Post    *entity.Post
	Content string

	Result *entity.InternalNote
}
