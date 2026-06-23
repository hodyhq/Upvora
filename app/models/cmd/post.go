package cmd

import (
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/enum"
)

type AddNewPost struct {
	Title       string
	Description string

	Result *entity.Post
}

type UpdatePost struct {
	Post        *entity.Post
	Title       string
	Description string

	Result *entity.Post
}

type SetPostResponse struct {
	Post       *entity.Post
	Text       string
	Status     enum.PostStatus // legacy_enum for backwards compat (0..6) or 0 for custom statuses
	StatusSlug string          // the tenant-defined slug; written to posts.status_slug
}
