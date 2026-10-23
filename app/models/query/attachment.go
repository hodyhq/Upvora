package query

import (
	"github.com/getfider/fider/app/models/entity"
)

type GetAttachments struct {
	Post    *entity.Post
	Comment *entity.Comment

	Result []string
}

// GetPostIDByAttachment finds the idea a file attachment belongs to (ideas
// only, not comments), so access to the file follows access to the idea.
type GetPostIDByAttachment struct {
	BlobKey string

	Result int
}
