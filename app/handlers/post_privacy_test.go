package handlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/getfider/fider/app/handlers"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/query"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/mock"
)

// setupPrivacy serves one post authored by authorID and records whether the
// privacy command ran.
func setupPrivacy(authorID int, private bool) *bool {
	post := &entity.Post{ID: 1, Number: 1, IsPrivate: private, User: &entity.User{ID: authorID}}
	called := false
	bus.AddHandler(func(ctx context.Context, q *query.GetPostByNumber) error { q.Result = post; return nil })
	bus.AddHandler(func(ctx context.Context, c *cmd.SetPostPrivacy) error { called = true; return nil })
	return &called
}

func tenantWithPublish(allowed bool) *entity.Tenant {
	tenant := *mock.DemoTenant
	tenant.MembersCanPublishPrivate = allowed
	return &tenant
}

func setPrivacy(tenant *entity.Tenant, user *entity.User, body string) int {
	code, _ := mock.NewServer().OnTenant(tenant).AsUser(user).
		AddParam("number", 1).ExecutePost(handlers.SetPostPrivacy(), body)
	return code
}

func TestSetPostPrivacy_AuthorCanPublish(t *testing.T) {
	RegisterT(t)
	called := setupPrivacy(mock.AryaStark.ID, true)
	Expect(setPrivacy(tenantWithPublish(true), mock.AryaStark, `{"isPrivate":false}`)).Equals(http.StatusOK)
	Expect(*called).IsTrue()
}

func TestSetPostPrivacy_AuthorPublishDisabled(t *testing.T) {
	RegisterT(t)
	called := setupPrivacy(mock.AryaStark.ID, true)
	Expect(setPrivacy(tenantWithPublish(false), mock.AryaStark, `{"isPrivate":false}`)).Equals(http.StatusForbidden)
	Expect(*called).IsFalse()
}

func TestSetPostPrivacy_MemberNeverMakesPrivate(t *testing.T) {
	RegisterT(t)
	called := setupPrivacy(mock.AryaStark.ID, false)
	Expect(setPrivacy(tenantWithPublish(true), mock.AryaStark, `{"isPrivate":true}`)).Equals(http.StatusForbidden)
	Expect(*called).IsFalse()
}

func TestSetPostPrivacy_NonAuthorMemberRejected(t *testing.T) {
	RegisterT(t)
	called := setupPrivacy(mock.JonSnow.ID, true)
	Expect(setPrivacy(tenantWithPublish(true), mock.AryaStark, `{"isPrivate":false}`)).Equals(http.StatusNotFound)
	Expect(*called).IsFalse()
}

func TestSetPostPrivacy_CollaboratorAnyDirection(t *testing.T) {
	RegisterT(t)
	called := setupPrivacy(mock.AryaStark.ID, false)
	Expect(setPrivacy(tenantWithPublish(false), mock.JonSnow, `{"isPrivate":true}`)).Equals(http.StatusOK)
	Expect(*called).IsTrue()
}
