package postgres_test

import (
	"testing"
	"time"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/query"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/errors"
)

// seedPrivacyPosts inserts one public and one private open post authored by
// Jon Snow (admin) in the demo tenant, with distinctive slugs.
func seedPrivacyPosts(t *testing.T) (publicSlug, privateSlug string) {
	now := time.Now()
	publicSlug, privateSlug = "pub-visible-idea", "priv-secret-idea"
	_, err := trx.Execute("INSERT INTO posts (title, slug, number, description, created_at, tenant_id, user_id, status_slug, is_approved, is_private, language) VALUES ('Pub Visible Idea', $1, 9001, 'public one', $2, 1, 1, 'open', true, false, 'english')", publicSlug, now)
	Expect(err).IsNil()
	_, err = trx.Execute("INSERT INTO posts (title, slug, number, description, created_at, tenant_id, user_id, status_slug, is_approved, is_private, language) VALUES ('Priv Secret Idea', $1, 9002, 'private one', $2, 1, 1, 'open', true, true, 'english')", privateSlug, now)
	Expect(err).IsNil()
	return
}

func slugsFrom(posts []*entity.Post) map[string]bool {
	m := map[string]bool{}
	for _, p := range posts {
		m[p.Slug] = true
	}
	return m
}

// Task 2: board/search/roadmap/RSS/API-list — a private post is returned to
// collaborators only. The anonymous path also covers the public RSS feed.
func TestPrivatePost_HiddenFromListing(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()
	pub, priv := seedPrivacyPosts(t)

	anon := &query.SearchPosts{Limit: "50"}
	visitor := &query.SearchPosts{Limit: "50"}
	admin := &query.SearchPosts{Limit: "50"}
	Expect(bus.Dispatch(demoTenantCtx, anon)).IsNil()
	Expect(bus.Dispatch(aryaStarkCtx, visitor)).IsNil()
	Expect(bus.Dispatch(jonSnowCtx, admin)).IsNil()

	anonSlugs, visitorSlugs, adminSlugs := slugsFrom(anon.Result), slugsFrom(visitor.Result), slugsFrom(admin.Result)

	// public post visible to everyone
	Expect(anonSlugs[pub]).IsTrue()
	Expect(visitorSlugs[pub]).IsTrue()
	Expect(adminSlugs[pub]).IsTrue()
	// private post visible ONLY to the collaborator/admin
	Expect(anonSlugs[priv]).IsFalse()
	Expect(visitorSlugs[priv]).IsFalse()
	Expect(adminSlugs[priv]).IsTrue()
}

// Task 3: direct link / single-post fetch — NotFound for a visitor, found for a collaborator.
func TestPrivatePost_SinglePostFetch(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()
	seedPrivacyPosts(t)

	anon := &query.GetPostByNumber{Number: 9002}
	err := bus.Dispatch(demoTenantCtx, anon)
	Expect(errors.Cause(err)).Equals(app.ErrNotFound)

	visitor := &query.GetPostByNumber{Number: 9002}
	err = bus.Dispatch(aryaStarkCtx, visitor)
	Expect(errors.Cause(err)).Equals(app.ErrNotFound)

	admin := &query.GetPostByNumber{Number: 9002}
	Expect(bus.Dispatch(jonSnowCtx, admin)).IsNil()
	Expect(admin.Result.Slug).Equals("priv-secret-idea")
	Expect(admin.Result.IsPrivate).IsTrue()
}

// Task 4: By-Status and By-Product counts drop private for visitors, keep for collaborators.
func TestPrivatePost_CountsExcludePrivateForVisitors(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()
	seedPrivacyPosts(t)

	anonStatus := &query.CountPostPerStatus{}
	adminStatus := &query.CountPostPerStatus{}
	Expect(bus.Dispatch(demoTenantCtx, anonStatus)).IsNil()
	Expect(bus.Dispatch(jonSnowCtx, adminStatus)).IsNil()
	// admin sees exactly one more 'open' post (the private one) than anon
	Expect(adminStatus.Result["open"] - anonStatus.Result["open"]).Equals(1)

	anonProduct := &query.CountPostPerProduct{}
	adminProduct := &query.CountPostPerProduct{}
	Expect(bus.Dispatch(demoTenantCtx, anonProduct)).IsNil()
	Expect(bus.Dispatch(jonSnowCtx, adminProduct)).IsNil()
	// both posts are unassigned (product key 0)
	Expect(adminProduct.Result[0] - anonProduct.Result[0]).Equals(1)
}

// Task 7: the collab toggle flips visibility both ways.
func TestSetPostPrivacy_TogglesVisibility(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()
	pub, _ := seedPrivacyPosts(t)

	// load the public post (admin ctx) so IsPrivate is populated
	loaded := &query.GetPostByNumber{Number: 9001}
	Expect(bus.Dispatch(jonSnowCtx, loaded)).IsNil()

	// make it private
	Expect(bus.Dispatch(jonSnowCtx, &cmd.SetPostPrivacy{Post: loaded.Result, IsPrivate: true})).IsNil()
	visitorAfter := &query.GetPostByNumber{Number: 9001}
	Expect(errors.Cause(bus.Dispatch(demoTenantCtx, visitorAfter))).Equals(app.ErrNotFound)

	// make it public again
	Expect(bus.Dispatch(jonSnowCtx, &cmd.SetPostPrivacy{Post: loaded.Result, IsPrivate: false})).IsNil()
	visitorBack := &query.GetPostByNumber{Number: 9001}
	Expect(bus.Dispatch(demoTenantCtx, visitorBack)).IsNil()
	Expect(visitorBack.Result.Slug).Equals(pub)
}

// Task 8 (security): merging copies a duplicate's content into its original,
// so toggling the original's privacy must carry the whole merged cluster with
// it. Otherwise flipping only the original to public would expose private
// content copied in from a still-private duplicate.
func TestSetPostPrivacy_PropagatesAcrossMergedCluster(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()
	now := time.Now()
	_, err := trx.Execute("INSERT INTO posts (title, slug, number, description, created_at, tenant_id, user_id, status_slug, is_approved, is_private, language) VALUES ('Dup Secret', 'dup-secret', 9003, 'dup body', $1, 1, 1, 'open', true, true, 'english')", now)
	Expect(err).IsNil()
	_, err = trx.Execute("INSERT INTO posts (title, slug, number, description, created_at, tenant_id, user_id, status_slug, is_approved, is_private, language) VALUES ('Orig Secret', 'orig-secret', 9004, 'orig body', $1, 1, 1, 'open', true, true, 'english')", now)
	Expect(err).IsNil()

	dup := &query.GetPostByNumber{Number: 9003}
	orig := &query.GetPostByNumber{Number: 9004}
	Expect(bus.Dispatch(jonSnowCtx, dup)).IsNil()
	Expect(bus.Dispatch(jonSnowCtx, orig)).IsNil()

	// Both private → merge is allowed and copies dup's content into orig.
	Expect(bus.Dispatch(jonSnowCtx, &cmd.MarkPostAsDuplicate{Post: dup.Result, Original: orig.Result})).IsNil()

	// Toggle the original public.
	Expect(bus.Dispatch(jonSnowCtx, &cmd.SetPostPrivacy{Post: orig.Result, IsPrivate: false})).IsNil()

	// The duplicate must have been carried public too — no split cluster.
	dupAfter := &query.GetPostByNumber{Number: 9003}
	Expect(bus.Dispatch(jonSnowCtx, dupAfter)).IsNil()
	Expect(dupAfter.Result.IsPrivate).IsFalse()
}

// Toggling a duplicate also flips its original (and siblings) — the invariant
// holds regardless of which post in the cluster is targeted.
func TestSetPostPrivacy_PropagatesFromDuplicateToRoot(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()
	now := time.Now()
	_, err := trx.Execute("INSERT INTO posts (title, slug, number, description, created_at, tenant_id, user_id, status_slug, is_approved, is_private, language) VALUES ('Dup Two', 'dup-two', 9005, 'dup body', $1, 1, 1, 'open', true, false, 'english')", now)
	Expect(err).IsNil()
	_, err = trx.Execute("INSERT INTO posts (title, slug, number, description, created_at, tenant_id, user_id, status_slug, is_approved, is_private, language) VALUES ('Orig Two', 'orig-two', 9006, 'orig body', $1, 1, 1, 'open', true, false, 'english')", now)
	Expect(err).IsNil()

	dup := &query.GetPostByNumber{Number: 9005}
	orig := &query.GetPostByNumber{Number: 9006}
	Expect(bus.Dispatch(jonSnowCtx, dup)).IsNil()
	Expect(bus.Dispatch(jonSnowCtx, orig)).IsNil()
	Expect(bus.Dispatch(jonSnowCtx, &cmd.MarkPostAsDuplicate{Post: dup.Result, Original: orig.Result})).IsNil()

	// Target the duplicate; the root must flip too.
	Expect(bus.Dispatch(jonSnowCtx, &cmd.SetPostPrivacy{Post: dup.Result, IsPrivate: true})).IsNil()

	origAfter := &query.GetPostByNumber{Number: 9006}
	Expect(bus.Dispatch(jonSnowCtx, origAfter)).IsNil()
	Expect(origAfter.Result.IsPrivate).IsTrue()
}

// Task 8: merging across a privacy boundary is refused.
func TestMarkPostAsDuplicate_RefusesCrossPrivacy(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()
	seedPrivacyPosts(t)

	pubPost := &query.GetPostByNumber{Number: 9001}
	privPost := &query.GetPostByNumber{Number: 9002}
	Expect(bus.Dispatch(jonSnowCtx, pubPost)).IsNil()
	Expect(bus.Dispatch(jonSnowCtx, privPost)).IsNil()

	// merging the private post into the public one must be refused
	err := bus.Dispatch(jonSnowCtx, &cmd.MarkPostAsDuplicate{Post: privPost.Result, Original: pubPost.Result})
	Expect(err).IsNotNil()
	Expect(err.Error()).ContainsSubstring("private")
}
