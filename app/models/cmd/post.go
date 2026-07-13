package cmd

import (
	"github.com/getfider/fider/app/models/entity"
)

type AddNewPost struct {
	ProductID int // 0 = General (unassigned)
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
	StatusSlug string
}
