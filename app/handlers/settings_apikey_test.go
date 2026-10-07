package handlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/getfider/fider/app/handlers"
	"github.com/getfider/fider/app/models/cmd"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/mock"
	"github.com/getfider/fider/app/pkg/web"
)

func viaAPIKey(next web.HandlerFunc) web.HandlerFunc {
	return func(c *web.Context) error {
		c.SetAuthenticatedByAPIKey()
		return next(c)
	}
}

// Rotating the API key is a signed-in UI action: a Bearer caller would revoke
// the key it is using, and an impersonating admin would receive someone
// else's new key.
func TestRegenerateAPIKey_RefusedForAPIKeyCaller(t *testing.T) {
	RegisterT(t)
	called := false
	bus.AddHandler(func(ctx context.Context, c *cmd.RegenerateAPIKey) error {
		called = true
		c.Result = "new-key"
		return nil
	})

	code, _ := mock.NewServer().OnTenant(mock.DemoTenant).AsUser(mock.JonSnow).Use(viaAPIKey).
		ExecutePost(handlers.RegenerateAPIKey(), "")
	Expect(code).Equals(http.StatusForbidden)
	Expect(called).IsFalse()

	code, _ = mock.NewServer().OnTenant(mock.DemoTenant).AsUser(mock.JonSnow).
		ExecutePost(handlers.RegenerateAPIKey(), "")
	Expect(code).Equals(http.StatusOK)
	Expect(called).IsTrue()
}
