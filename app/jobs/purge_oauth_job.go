package jobs

import (
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/dto"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/log"
)

// PurgeStaleOAuthJobHandler removes old MCP authorization data.
type PurgeStaleOAuthJobHandler struct{}

func (PurgeStaleOAuthJobHandler) Schedule() string {
	return "0 30 * * * *" // every hour at minute 30
}

func (PurgeStaleOAuthJobHandler) Run(ctx Context) error {
	c := &cmd.PurgeStaleOAuthData{}
	if err := bus.Dispatch(ctx, c); err != nil {
		return err
	}
	log.Debugf(ctx, "@{RowsDeleted} stale OAuth rows were deleted", dto.Props{"RowsDeleted": c.Deleted})
	return nil
}
