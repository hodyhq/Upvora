package query

import "github.com/getfider/fider/app/models/entity"

// GetInternalNote loads the shared team note for a post. Result is a zero-value
// note (empty content) when none has been written yet.
type GetInternalNote struct {
	PostID int

	Result *entity.InternalNote
}
