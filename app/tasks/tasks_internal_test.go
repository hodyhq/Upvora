package tasks

import (
	"testing"

	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/enum"

	. "github.com/getfider/fider/app/pkg/assert"
)

// A private idea's notifications must exclude non-collaborators, including a
// visitor still subscribed after a public->private flip.
func TestCollaboratorsOnly(t *testing.T) {
	RegisterT(t)
	visitor := &entity.User{ID: 1, Role: enum.RoleVisitor}
	collab := &entity.User{ID: 2, Role: enum.RoleCollaborator}
	admin := &entity.User{ID: 3, Role: enum.RoleAdministrator}

	out := collaboratorsOnly([]*entity.User{visitor, collab, nil, admin})

	Expect(out).HasLen(2)
	Expect(out[0].ID).Equals(2)
	Expect(out[1].ID).Equals(3)
}
