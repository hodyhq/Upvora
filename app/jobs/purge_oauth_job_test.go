package jobs_test

import (
	"context"
	"testing"

	"github.com/getfider/fider/app/jobs"
	"github.com/getfider/fider/app/models/cmd"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/bus"
)

func TestPurgeStaleOAuthJob(t *testing.T) {
	RegisterT(t)
	job := &jobs.PurgeStaleOAuthJobHandler{}
	Expect(job.Schedule()).Equals("0 30 * * * *")

	ran := false
	bus.AddHandler(func(ctx context.Context, c *cmd.PurgeStaleOAuthData) error { ran = true; return nil })
	Expect(job.Run(jobs.Context{Context: context.Background()})).IsNil()
	Expect(ran).IsTrue()
}
