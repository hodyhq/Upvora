package mcpserver

import (
	"context"
	"sync"

	"github.com/getfider/fider/app/models/dto"
	"github.com/getfider/fider/app/pkg/log"
)

// AuditEntry records one tool call: who, where, which tool, how it went.
// Arguments and results are never recorded (they may hold personal data).
type AuditEntry struct {
	TenantID   int
	UserID     int
	ClientID   string
	Tool       string
	Status     int // HTTP status of the replayed /api/v1 call; 429 when rate limited
	DurationMs int64
}

var (
	auditMu   sync.Mutex
	auditSink = logAudit
)

func logAudit(ctx context.Context, e AuditEntry) {
	log.Infof(ctx, "MCP tool @{Tool:magenta} by user @{UserID} (client @{ClientID}) on tenant @{TenantID} finished with @{Status:magenta} in @{ElapsedMs:magenta}ms", dto.Props{
		"Tool": e.Tool, "UserID": e.UserID, "ClientID": e.ClientID, "TenantID": e.TenantID, "Status": e.Status, "ElapsedMs": e.DurationMs,
	})
}

func audit(ctx context.Context, e AuditEntry) {
	auditMu.Lock()
	sink := auditSink
	auditMu.Unlock()
	sink(ctx, e)
}

// SetAuditSink replaces where audit entries go (tests); returns a restore func.
func SetAuditSink(f func(AuditEntry)) func() {
	auditMu.Lock()
	defer auditMu.Unlock()
	previous := auditSink
	auditSink = func(_ context.Context, e AuditEntry) { f(e) }
	return func() {
		auditMu.Lock()
		defer auditMu.Unlock()
		auditSink = previous
	}
}
