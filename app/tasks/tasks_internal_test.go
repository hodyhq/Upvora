package tasks

import (
	"testing"

	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/enum"

	. "github.com/getfider/fider/app/pkg/assert"
)

// A private idea notifies only the people who can see it: staff and its own
// author. Other members (e.g. still subscribed after a public->private flip)
// are excluded.
func TestPrivateAudience_StaffAndAuthorOnly(t *testing.T) {
	RegisterT(t)
	author := &entity.User{ID: 2, Role: enum.RoleVisitor}
	other := &entity.User{ID: 3, Role: enum.RoleVisitor}
	collab := &entity.User{ID: 4, Role: enum.RoleCollaborator}
	admin := &entity.User{ID: 1, Role: enum.RoleAdministrator}
	post := &entity.Post{IsPrivate: true, User: author}

	out := privateAudience(post, []*entity.User{author, other, collab, admin, nil})
	ids := map[int]bool{}
	for _, u := range out {
		ids[u.ID] = true
	}
	Expect(out).HasLen(3)
	Expect(ids[2]).IsTrue()
	Expect(ids[3]).IsFalse()
	Expect(ids[4]).IsTrue()
	Expect(ids[1]).IsTrue()
}
