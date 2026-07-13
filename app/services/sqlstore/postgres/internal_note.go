package postgres

import (
	"context"
	"time"

	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/dbx"
	"github.com/getfider/fider/app/pkg/errors"
)

type dbInternalNote struct {
	Content       string       `db:"content"`
	UpdatedAt     time.Time    `db:"updated_at"`
	UpdatedByName dbx.NullString `db:"updated_by_name"`
}

func (n *dbInternalNote) toModel() *entity.InternalNote {
	note := &entity.InternalNote{
		Content:   n.Content,
		UpdatedAt: n.UpdatedAt,
	}
	if n.UpdatedByName.Valid {
		note.UpdatedByName = n.UpdatedByName.String
	}
	return note
}

func getInternalNote(ctx context.Context, q *query.GetInternalNote) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		row := dbInternalNote{}
		err := trx.Get(&row, `
			SELECT n.content, n.updated_at, u.name AS updated_by_name
			FROM post_internal_notes n
			LEFT JOIN users u ON u.id = n.updated_by AND u.tenant_id = n.tenant_id
			WHERE n.tenant_id = $1 AND n.post_id = $2
		`, tenant.ID, q.PostID)
		if err != nil {
			// no note yet - return an empty one, the client renders a blank editor
			q.Result = &entity.InternalNote{}
			return nil
		}
		q.Result = row.toModel()
		return nil
	})
}

func setInternalNote(ctx context.Context, c *cmd.SetInternalNote) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		now := time.Now()
		_, err := trx.Execute(`
			INSERT INTO post_internal_notes (tenant_id, post_id, content, updated_by, updated_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (tenant_id, post_id)
			DO UPDATE SET content = $3, updated_by = $4, updated_at = $5
		`, tenant.ID, c.Post.ID, c.Content, user.ID, now)
		if err != nil {
			return errors.Wrap(err, "failed to upsert internal note for post %d", c.Post.ID)
		}
		c.Result = &entity.InternalNote{Content: c.Content, UpdatedAt: now, UpdatedByName: user.Name}
		return nil
	})
}
