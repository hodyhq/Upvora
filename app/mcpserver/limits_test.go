package mcpserver_test

import (
	"testing"

	"github.com/getfider/fider/app/mcpserver"
	. "github.com/getfider/fider/app/pkg/assert"
)

func TestCallAllowed_PerUserAndPerClient(t *testing.T) {
	RegisterT(t)
	mcpserver.SetCallLimits(2, 3)
	defer mcpserver.SetCallLimits(mcpserver.DefaultPerUser, mcpserver.DefaultPerClient)

	// per user: user 7 gets 2 calls
	Expect(mcpserver.CallAllowed(1, 7, "client-a")).IsTrue()
	Expect(mcpserver.CallAllowed(1, 7, "client-a")).IsTrue()
	Expect(mcpserver.CallAllowed(1, 7, "client-a")).IsFalse()

	// per client: client-b shared by users 8 and 9 gets 3 calls in total
	Expect(mcpserver.CallAllowed(1, 8, "client-b")).IsTrue()
	Expect(mcpserver.CallAllowed(1, 8, "client-b")).IsTrue()
	Expect(mcpserver.CallAllowed(1, 9, "client-b")).IsTrue()
	Expect(mcpserver.CallAllowed(1, 9, "client-b")).IsFalse()

	// tenants are separate
	Expect(mcpserver.CallAllowed(2, 7, "client-a")).IsTrue()
}
