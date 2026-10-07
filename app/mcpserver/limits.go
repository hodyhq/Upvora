package mcpserver

import (
	"strconv"
	"sync"
	"time"

	"github.com/getfider/fider/app/pkg/ratelimit"
)

// Tool-call budgets per 10 minutes. A runaway or prompt-injected client hits
// these long before it can hurt the site.
const (
	DefaultPerUser   = 300
	DefaultPerClient = 600
	callWindow       = 10 * time.Minute
)

var (
	limitsMu       sync.Mutex
	perUserCalls   = ratelimit.New(DefaultPerUser, callWindow)
	perClientCalls = ratelimit.New(DefaultPerClient, callWindow)
)

// SetCallLimits replaces the budgets (tests and tuning).
func SetCallLimits(perUser, perClient int) {
	limitsMu.Lock()
	defer limitsMu.Unlock()
	perUserCalls = ratelimit.New(perUser, callWindow)
	perClientCalls = ratelimit.New(perClient, callWindow)
}

// CallAllowed records a tool call and reports whether it is within both the
// user's and the token client's budget on this site.
func CallAllowed(tenantID, userID int, clientID string) bool {
	limitsMu.Lock()
	user, client := perUserCalls, perClientCalls
	limitsMu.Unlock()
	site := strconv.Itoa(tenantID) + "|"
	return user.Allow(site+strconv.Itoa(userID)) && client.Allow(site+clientID)
}
