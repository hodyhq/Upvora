package postgres

import (
	"context"
	"time"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/dbx"
	"github.com/getfider/fider/app/pkg/errors"
)

type dbProduct struct {
	ID          int       `db:"id"`
	Name        string    `db:"name"`
	Slug        string    `db:"slug"`
	Description string    `db:"description"`
	Color       string    `db:"color"`
	SortOrder   int       `db:"sort_order"`
	IsActive    bool      `db:"is_active"`
	CreatedAt   time.Time `db:"created_at"`
}

func (p *dbProduct) toModel() *entity.Product {
	return &entity.Product{
		ID:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		Color:       p.Color,
		SortOrder:   p.SortOrder,
		IsActive:    p.IsActive,
		CreatedAt:   p.CreatedAt,
	}
}

const productCols = "id, name, slug, description, color, sort_order, is_active, created_at"

func listActiveProducts(ctx context.Context, q *query.ListActiveProducts) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		rows := []*dbProduct{}
		err := trx.Select(&rows, "SELECT "+productCols+" FROM products WHERE tenant_id = $1 AND is_active = TRUE ORDER BY sort_order, id", tenant.ID)
		if err != nil {
			return errors.Wrap(err, "failed to list active products")
		}
		q.Result = make([]*entity.Product, len(rows))
		for i, r := range rows {
			q.Result[i] = r.toModel()
		}
		return nil
	})
}

func listAllProducts(ctx context.Context, q *query.ListAllProducts) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		rows := []*dbProduct{}
		err := trx.Select(&rows, "SELECT "+productCols+" FROM products WHERE tenant_id = $1 ORDER BY sort_order, id", tenant.ID)
		if err != nil {
			return errors.Wrap(err, "failed to list products")
		}
		q.Result = make([]*entity.Product, len(rows))
		for i, r := range rows {
			q.Result[i] = r.toModel()
		}
		return nil
	})
}

func countPostPerProduct(ctx context.Context, q *query.CountPostPerProduct) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		type row struct {
			ProductID dbx.NullInt `db:"product_id"`
			Count     int         `db:"count"`
		}
		rows := []*row{}
		err := trx.Select(&rows, "SELECT product_id, COUNT(*) AS count FROM posts WHERE tenant_id = $1 GROUP BY product_id", tenant.ID)
		if err != nil {
			return errors.Wrap(err, "failed to count posts per product")
		}
		q.Result = make(map[int]int)
		for _, r := range rows {
			id := 0
			if r.ProductID.Valid {
				id = int(r.ProductID.Int64)
			}
			q.Result[id] = r.Count
		}
		return nil
	})
}

func getProductBySlug(ctx context.Context, q *query.GetProductBySlug) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		row := dbProduct{}
		err := trx.Get(&row, "SELECT "+productCols+" FROM products WHERE tenant_id = $1 AND slug = $2", tenant.ID, q.Slug)
		if err == app.ErrNotFound {
			return app.ErrNotFound
		}
		if err != nil {
			return errors.Wrap(err, "failed to get product by slug %q", q.Slug)
		}
		q.Result = row.toModel()
		return nil
	})
}

func createProduct(ctx context.Context, c *cmd.CreateProduct) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		var id int
		err := trx.Get(&id, `
			INSERT INTO products (tenant_id, name, slug, description, color, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id
		`, tenant.ID, c.Name, c.Slug, c.Description, c.Color, c.SortOrder)
		if err != nil {
			return errors.Wrap(err, "failed to create product %q", c.Slug)
		}
		c.Result = &entity.Product{ID: id, Name: c.Name, Slug: c.Slug, Description: c.Description, Color: c.Color, SortOrder: c.SortOrder, IsActive: true, CreatedAt: time.Now()}
		return nil
	})
}

func updateProduct(ctx context.Context, c *cmd.UpdateProduct) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		_, err := trx.Execute(`
			UPDATE products SET name = $1, description = $2, color = $3, sort_order = $4, is_active = $5
			WHERE id = $6 AND tenant_id = $7
		`, c.Name, c.Description, c.Color, c.SortOrder, c.IsActive, c.ProductID, tenant.ID)
		if err != nil {
			return errors.Wrap(err, "failed to update product %d", c.ProductID)
		}
		return nil
	})
}

func setPostProduct(ctx context.Context, c *cmd.SetPostProduct) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		var productID interface{}
		if c.ProductID > 0 {
			productID = c.ProductID
		}
		_, err := trx.Execute("UPDATE posts SET product_id = $1 WHERE id = $2 AND tenant_id = $3", productID, c.Post.ID, tenant.ID)
		if err != nil {
			return errors.Wrap(err, "failed to set product for post %d", c.Post.ID)
		}
		return nil
	})
}

func deleteProduct(ctx context.Context, c *cmd.DeleteProduct) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		// posts fall back to General via ON DELETE SET NULL
		_, err := trx.Execute("DELETE FROM products WHERE id = $1 AND tenant_id = $2", c.ProductID, tenant.ID)
		if err != nil {
			return errors.Wrap(err, "failed to delete product %d", c.ProductID)
		}
		return nil
	})
}
