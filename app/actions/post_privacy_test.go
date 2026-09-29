package actions_test

import (
	"context"
	"testing"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/actions"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/enum"
	"github.com/getfider/fider/app/models/query"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/bus"
)

// Only collaborators/admins may create a private idea; the check is server-side.
func TestCreateNewPost_PrivateRequiresCollaborator(t *testing.T) {
	RegisterT(t)
	bus.AddHandler(func(ctx context.Context, q *query.GetPostBySlug) error {
		return app.ErrNotFound
	})

	visitor := &entity.User{Role: enum.RoleVisitor}
	collab := &entity.User{Role: enum.RoleCollaborator}

	// visitor cannot create a private idea
	r1 := (&actions.CreateNewPost{Title: "A perfectly valid idea title", IsPrivate: true}).Validate(context.Background(), visitor)
	ExpectFailed(r1, "isPrivate")

	// collaborator can
	r2 := (&actions.CreateNewPost{Title: "Another valid idea title here", IsPrivate: true}).Validate(context.Background(), collab)
	Expect(r2.Ok).IsTrue()

	// non-private idea is fine for a visitor
	r3 := (&actions.CreateNewPost{Title: "Yet another valid idea title", IsPrivate: false}).Validate(context.Background(), visitor)
	Expect(r3.Ok).IsTrue()
}
