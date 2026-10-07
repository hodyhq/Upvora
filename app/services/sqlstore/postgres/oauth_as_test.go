package postgres_test

import (
	"testing"
	"time"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/query"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/errors"
)

func registerClient(t *testing.T) string {
	reg := &cmd.RegisterOAuthClient{Name: "Claude", RedirectURIs: []string{"https://claude.ai/api/mcp/auth_callback"}}
	Expect(bus.Dispatch(demoTenantCtx, reg)).IsNil()
	Expect(len(reg.Result.ClientID) >= 32).IsTrue()
	return reg.Result.ClientID
}

func TestOAuthClients_RegisterGetListDelete(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()
	clientID := registerClient(t)

	get := &query.GetOAuthClient{ClientID: clientID}
	Expect(bus.Dispatch(demoTenantCtx, get)).IsNil()
	Expect(get.Result.Name).Equals("Claude")
	Expect(get.Result.RedirectURIs).Equals([]string{"https://claude.ai/api/mcp/auth_callback"})

	// tenant-scoped: another tenant cannot see it
	other := &query.GetOAuthClient{ClientID: clientID}
	Expect(errors.Cause(bus.Dispatch(avengersTenantCtx, other))).Equals(app.ErrNotFound)

	list := &query.ListOAuthClients{}
	Expect(bus.Dispatch(jonSnowCtx, list)).IsNil()
	Expect(list.Result).HasLen(1)

	Expect(bus.Dispatch(jonSnowCtx, &cmd.DeleteOAuthClient{ID: get.Result.ID})).IsNil()
	gone := &query.GetOAuthClient{ClientID: clientID}
	Expect(errors.Cause(bus.Dispatch(demoTenantCtx, gone))).Equals(app.ErrNotFound)
}

func saveCode(t *testing.T, clientID, hash string, expires time.Time) {
	Expect(bus.Dispatch(demoTenantCtx, &cmd.SaveOAuthCode{
		CodeHash: hash, ClientID: clientID, UserID: aryaStark.ID, RedirectURI: "https://claude.ai/api/mcp/auth_callback",
		Scope: "upvora", CodeChallenge: "challenge", ExpiresAt: expires,
	})).IsNil()
}

func TestOAuthCodes_SingleUseAndScoped(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()
	clientID := registerClient(t)
	saveCode(t, clientID, "code-hash-1", time.Now().Add(10*time.Minute))

	wrongClient := &cmd.ConsumeOAuthCode{CodeHash: "code-hash-1", ClientID: "someone-else"}
	Expect(errors.Cause(bus.Dispatch(demoTenantCtx, wrongClient))).Equals(app.ErrNotFound)

	wrongTenant := &cmd.ConsumeOAuthCode{CodeHash: "code-hash-1", ClientID: clientID}
	Expect(errors.Cause(bus.Dispatch(avengersTenantCtx, wrongTenant))).Equals(app.ErrNotFound)

	first := &cmd.ConsumeOAuthCode{CodeHash: "code-hash-1", ClientID: clientID}
	Expect(bus.Dispatch(demoTenantCtx, first)).IsNil()
	Expect(first.Result.UserID).Equals(aryaStark.ID)
	Expect(first.Result.CodeChallenge).Equals("challenge")

	again := &cmd.ConsumeOAuthCode{CodeHash: "code-hash-1", ClientID: clientID}
	Expect(errors.Cause(bus.Dispatch(demoTenantCtx, again))).Equals(app.ErrNotFound)
}

func TestOAuthCodes_Expired(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()
	clientID := registerClient(t)
	saveCode(t, clientID, "code-hash-old", time.Now().Add(-time.Minute))

	c := &cmd.ConsumeOAuthCode{CodeHash: "code-hash-old", ClientID: clientID}
	Expect(errors.Cause(bus.Dispatch(demoTenantCtx, c))).Equals(app.ErrNotFound)
}

func saveRefresh(t *testing.T, clientID, hash, family string) {
	Expect(bus.Dispatch(demoTenantCtx, &cmd.SaveOAuthRefreshToken{
		TokenHash: hash, ClientID: clientID, UserID: aryaStark.ID, Scope: "upvora",
		FamilyID: family, ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	})).IsNil()
}

func TestOAuthRefresh_RotateAndReuseRevokesFamily(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()
	clientID := registerClient(t)
	saveRefresh(t, clientID, "rt-1", "fam-1")

	rot := &cmd.RotateOAuthRefreshToken{OldHash: "rt-1", NewHash: "rt-2", ClientID: clientID, NewExpiresAt: time.Now().Add(time.Hour)}
	Expect(bus.Dispatch(demoTenantCtx, rot)).IsNil()
	Expect(rot.Result.UserID).Equals(aryaStark.ID)
	Expect(rot.Result.Scope).Equals("upvora")

	// replaying the rotated token fails and revokes the whole family
	replay := &cmd.RotateOAuthRefreshToken{OldHash: "rt-1", NewHash: "rt-x", ClientID: clientID, NewExpiresAt: time.Now().Add(time.Hour)}
	Expect(errors.Cause(bus.Dispatch(demoTenantCtx, replay))).Equals(app.ErrNotFound)

	// so the newest token in the family no longer works either
	next := &cmd.RotateOAuthRefreshToken{OldHash: "rt-2", NewHash: "rt-3", ClientID: clientID, NewExpiresAt: time.Now().Add(time.Hour)}
	Expect(errors.Cause(bus.Dispatch(demoTenantCtx, next))).Equals(app.ErrNotFound)
}

func TestOAuthRefresh_WrongClientAndDeletedClient(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()
	clientID := registerClient(t)
	saveRefresh(t, clientID, "rt-a", "fam-a")

	wrong := &cmd.RotateOAuthRefreshToken{OldHash: "rt-a", NewHash: "rt-b", ClientID: "other", NewExpiresAt: time.Now().Add(time.Hour)}
	Expect(errors.Cause(bus.Dispatch(demoTenantCtx, wrong))).Equals(app.ErrNotFound)

	get := &query.GetOAuthClient{ClientID: clientID}
	Expect(bus.Dispatch(demoTenantCtx, get)).IsNil()
	Expect(bus.Dispatch(jonSnowCtx, &cmd.DeleteOAuthClient{ID: get.Result.ID})).IsNil()

	after := &cmd.RotateOAuthRefreshToken{OldHash: "rt-a", NewHash: "rt-b", ClientID: clientID, NewExpiresAt: time.Now().Add(time.Hour)}
	Expect(errors.Cause(bus.Dispatch(demoTenantCtx, after))).Equals(app.ErrNotFound)
}

// Rotating the user's security stamp (block, role change, sign-out everywhere)
// kills outstanding codes and refresh tokens, and revokes the family.
func TestOAuth_SecurityStampRotationRevokes(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()
	clientID := registerClient(t)
	_, err := trx.Execute("UPDATE users SET security_stamp = 'stamp-a' WHERE id = $1", aryaStark.ID)
	Expect(err).IsNil()

	Expect(bus.Dispatch(demoTenantCtx, &cmd.SaveOAuthCode{CodeHash: "stamp-code", ClientID: clientID, UserID: aryaStark.ID,
		RedirectURI: "https://claude.ai/api/mcp/auth_callback", Scope: "upvora", CodeChallenge: "c", SecurityStamp: "stamp-a",
		ExpiresAt: time.Now().Add(time.Minute)})).IsNil()
	Expect(bus.Dispatch(demoTenantCtx, &cmd.SaveOAuthRefreshToken{TokenHash: "stamp-rt", ClientID: clientID, UserID: aryaStark.ID,
		Scope: "upvora", FamilyID: "fam-stamp", SecurityStamp: "stamp-a", ExpiresAt: time.Now().Add(time.Hour)})).IsNil()

	_, err = trx.Execute("UPDATE users SET security_stamp = 'stamp-b' WHERE id = $1", aryaStark.ID)
	Expect(err).IsNil()

	consume := &cmd.ConsumeOAuthCode{CodeHash: "stamp-code", ClientID: clientID}
	Expect(errors.Cause(bus.Dispatch(demoTenantCtx, consume))).Equals(app.ErrNotFound)

	rot := &cmd.RotateOAuthRefreshToken{OldHash: "stamp-rt", NewHash: "stamp-rt2", ClientID: clientID, NewExpiresAt: time.Now().Add(time.Hour)}
	Expect(errors.Cause(bus.Dispatch(demoTenantCtx, rot))).Equals(app.ErrNotFound)

	var revoked int
	Expect(trx.Scalar(&revoked, "SELECT COUNT(*) FROM oauth_refresh_tokens WHERE family_id = 'fam-stamp' AND revoked_at IS NOT NULL")).IsNil()
	Expect(revoked).Equals(1)
}

// OAuth 2.1: a reused authorization code revokes what that code already issued.
func TestOAuthCodes_ReplayRevokesIssuedFamily(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()
	clientID := registerClient(t)
	saveCode(t, clientID, "replay-code", time.Now().Add(10*time.Minute))

	first := &cmd.ConsumeOAuthCode{CodeHash: "replay-code", ClientID: clientID}
	Expect(bus.Dispatch(demoTenantCtx, first)).IsNil()
	Expect(bus.Dispatch(demoTenantCtx, &cmd.SaveOAuthRefreshToken{TokenHash: "replay-rt", ClientID: clientID, UserID: aryaStark.ID,
		Scope: "upvora", FamilyID: "fam-replay", FromCodeHash: "replay-code", ExpiresAt: time.Now().Add(time.Hour)})).IsNil()

	again := &cmd.ConsumeOAuthCode{CodeHash: "replay-code", ClientID: clientID}
	Expect(errors.Cause(bus.Dispatch(demoTenantCtx, again))).Equals(app.ErrNotFound)

	var revoked int
	Expect(trx.Scalar(&revoked, "SELECT COUNT(*) FROM oauth_refresh_tokens WHERE family_id = 'fam-replay' AND revoked_at IS NOT NULL")).IsNil()
	Expect(revoked).Equals(1)
}

// Housekeeping removes stale codes, dead refresh tokens and self-registered
// clients nobody ever used, and keeps everything still in play.
func TestOAuth_PurgeStaleData(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()
	used := registerClient(t)
	saveCode(t, used, "old-code", time.Now().Add(-48*time.Hour))
	saveCode(t, used, "fresh-code", time.Now().Add(10*time.Minute))
	saveRefresh(t, used, "live-rt", "fam-live")
	saveRefresh(t, used, "dead-rt", "fam-dead")
	_, err := trx.Execute("UPDATE oauth_refresh_tokens SET revoked_at = now() - interval '8 days' WHERE token_hash = 'dead-rt'")
	Expect(err).IsNil()

	stale := &cmd.RegisterOAuthClient{Name: "never used", RedirectURIs: []string{"https://x.example/cb"}}
	Expect(bus.Dispatch(demoTenantCtx, stale)).IsNil()
	admin := &cmd.RegisterOAuthClient{Name: "admin made", RedirectURIs: []string{"https://x.example/cb"}, CreatedByAdmin: true}
	Expect(bus.Dispatch(demoTenantCtx, admin)).IsNil()
	_, err = trx.Execute("UPDATE oauth_clients SET created_at = now() - interval '8 days'")
	Expect(err).IsNil()

	purge := &cmd.PurgeStaleOAuthData{}
	Expect(bus.Dispatch(demoTenantCtx, purge)).IsNil()

	count := func(sql string) int {
		var n int
		Expect(trx.Scalar(&n, sql)).IsNil()
		return n
	}
	Expect(count("SELECT COUNT(*) FROM oauth_codes WHERE code_hash = 'old-code'")).Equals(0)
	Expect(count("SELECT COUNT(*) FROM oauth_codes WHERE code_hash = 'fresh-code'")).Equals(1)
	Expect(count("SELECT COUNT(*) FROM oauth_refresh_tokens WHERE token_hash = 'dead-rt'")).Equals(0)
	Expect(count("SELECT COUNT(*) FROM oauth_refresh_tokens WHERE token_hash = 'live-rt'")).Equals(1)
	Expect(count("SELECT COUNT(*) FROM oauth_clients WHERE client_id = '" + stale.Result.ClientID + "'")).Equals(0)
	Expect(count("SELECT COUNT(*) FROM oauth_clients WHERE client_id = '" + admin.Result.ClientID + "'")).Equals(1)
	Expect(count("SELECT COUNT(*) FROM oauth_clients WHERE client_id = '" + used + "'")).Equals(1) // has tokens
}

const mcpOrigin = "https://mcp.demo.test"

func TestOAuthSignInHandoff_SingleUseScopedAndExpiring(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()
	save := func(hash string, expires time.Time) {
		Expect(bus.Dispatch(demoTenantCtx, &cmd.SaveOAuthSignInHandoff{
			CodeHash: hash, UserID: aryaStark.ID, Origin: mcpOrigin, Query: "client_id=c&state=s", FlowHash: "flow-1", ExpiresAt: expires,
		})).IsNil()
	}
	save("ho-1", time.Now().Add(2*time.Minute))

	wrongOrigin := &cmd.RedeemOAuthSignInHandoff{CodeHash: "ho-1", Origin: "https://other.demo.test", FlowHash: "flow-1"}
	Expect(errors.Cause(bus.Dispatch(demoTenantCtx, wrongOrigin))).Equals(app.ErrNotFound)
	wrongTenant := &cmd.RedeemOAuthSignInHandoff{CodeHash: "ho-1", Origin: mcpOrigin, FlowHash: "flow-1"}
	Expect(errors.Cause(bus.Dispatch(avengersTenantCtx, wrongTenant))).Equals(app.ErrNotFound)

	// another browser (no or a different flow cookie) cannot redeem it
	otherBrowser := &cmd.RedeemOAuthSignInHandoff{CodeHash: "ho-1", Origin: mcpOrigin, FlowHash: "flow-2"}
	Expect(errors.Cause(bus.Dispatch(demoTenantCtx, otherBrowser))).Equals(app.ErrNotFound)

	first := &cmd.RedeemOAuthSignInHandoff{CodeHash: "ho-1", Origin: mcpOrigin, FlowHash: "flow-1"}
	Expect(bus.Dispatch(demoTenantCtx, first)).IsNil()
	Expect(first.Result.UserID).Equals(aryaStark.ID)
	Expect(first.Result.Query).Equals("client_id=c&state=s")

	again := &cmd.RedeemOAuthSignInHandoff{CodeHash: "ho-1", Origin: mcpOrigin, FlowHash: "flow-1"}
	Expect(errors.Cause(bus.Dispatch(demoTenantCtx, again))).Equals(app.ErrNotFound)

	save("ho-old", time.Now().Add(-time.Second))
	old := &cmd.RedeemOAuthSignInHandoff{CodeHash: "ho-old", Origin: mcpOrigin, FlowHash: "flow-1"}
	Expect(errors.Cause(bus.Dispatch(demoTenantCtx, old))).Equals(app.ErrNotFound)
}

func TestOAuthCodes_BoundToOrigin(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()
	clientID := registerClient(t)
	Expect(bus.Dispatch(demoTenantCtx, &cmd.SaveOAuthCode{
		CodeHash: "oc-1", ClientID: clientID, UserID: aryaStark.ID, RedirectURI: "https://claude.ai/api/mcp/auth_callback",
		Scope: "upvora", CodeChallenge: "challenge", Origin: mcpOrigin, ExpiresAt: time.Now().Add(10 * time.Minute),
	})).IsNil()

	// Presented on another address: refused, and not burned.
	elsewhere := &cmd.ConsumeOAuthCode{CodeHash: "oc-1", ClientID: clientID, Origin: "http://demo.test.fider.io", AllowUnbound: true}
	Expect(errors.Cause(bus.Dispatch(demoTenantCtx, elsewhere))).Equals(app.ErrNotFound)
	here := &cmd.ConsumeOAuthCode{CodeHash: "oc-1", ClientID: clientID, Origin: mcpOrigin}
	Expect(bus.Dispatch(demoTenantCtx, here)).IsNil()

	// A code from before this release (no address) works only on the board's own address.
	saveCode(t, clientID, "oc-legacy", time.Now().Add(10*time.Minute))
	onMCP := &cmd.ConsumeOAuthCode{CodeHash: "oc-legacy", ClientID: clientID, Origin: mcpOrigin}
	Expect(errors.Cause(bus.Dispatch(demoTenantCtx, onMCP))).Equals(app.ErrNotFound)
	onBoard := &cmd.ConsumeOAuthCode{CodeHash: "oc-legacy", ClientID: clientID, Origin: "http://demo.test.fider.io", AllowUnbound: true}
	Expect(bus.Dispatch(demoTenantCtx, onBoard)).IsNil()
}

func TestOAuthRefresh_BoundToOriginWithoutRevoking(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()
	clientID := registerClient(t)
	Expect(bus.Dispatch(demoTenantCtx, &cmd.SaveOAuthRefreshToken{
		TokenHash: "ort-1", ClientID: clientID, UserID: aryaStark.ID, Scope: "upvora",
		FamilyID: "ofam", Origin: mcpOrigin, ExpiresAt: time.Now().Add(time.Hour),
	})).IsNil()

	elsewhere := &cmd.RotateOAuthRefreshToken{OldHash: "ort-1", NewHash: "ort-x", ClientID: clientID,
		Origin: "http://demo.test.fider.io", AllowUnbound: true, NewExpiresAt: time.Now().Add(time.Hour)}
	Expect(errors.Cause(bus.Dispatch(demoTenantCtx, elsewhere))).Equals(app.ErrNotFound)

	// The family survives a wrong-address attempt, and the new token keeps the address.
	here := &cmd.RotateOAuthRefreshToken{OldHash: "ort-1", NewHash: "ort-2", ClientID: clientID,
		Origin: mcpOrigin, NewExpiresAt: time.Now().Add(time.Hour)}
	Expect(bus.Dispatch(demoTenantCtx, here)).IsNil()
	next := &cmd.RotateOAuthRefreshToken{OldHash: "ort-2", NewHash: "ort-3", ClientID: clientID,
		Origin: "http://demo.test.fider.io", AllowUnbound: true, NewExpiresAt: time.Now().Add(time.Hour)}
	Expect(errors.Cause(bus.Dispatch(demoTenantCtx, next))).Equals(app.ErrNotFound)
	nextHere := &cmd.RotateOAuthRefreshToken{OldHash: "ort-2", NewHash: "ort-3", ClientID: clientID,
		Origin: mcpOrigin, NewExpiresAt: time.Now().Add(time.Hour)}
	Expect(bus.Dispatch(demoTenantCtx, nextHere)).IsNil()
}
