package actions_test

import (
	"context"
	"testing"

	"github.com/getfider/fider/app/actions"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/enum"
	. "github.com/getfider/fider/app/pkg/assert"
)

func TestUpdateTenantMCPSettings_AdminOnlyAndValidRole(t *testing.T) {
	RegisterT(t)
	admin := &entity.User{Role: enum.RoleAdministrator}
	collab := &entity.User{Role: enum.RoleCollaborator}

	a := &actions.UpdateTenantMCPSettings{Enabled: true, MinRole: enum.RoleCollaborator}
	Expect(a.IsAuthorized(context.Background(), admin)).IsTrue()
	Expect(a.IsAuthorized(context.Background(), collab)).IsFalse()
	Expect(a.IsAuthorized(context.Background(), nil)).IsFalse()
	Expect(a.Validate(context.Background(), admin).Ok).IsTrue()

	bad := &actions.UpdateTenantMCPSettings{Enabled: true, MinRole: enum.Role(0)}
	ExpectFailed(bad.Validate(context.Background(), admin), "minRole")
}
