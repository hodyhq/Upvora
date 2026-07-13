package handlers

import (
	"net/http"

	"github.com/getfider/fider/app"

	"github.com/getfider/fider/app/actions"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/web"
)

// ProductBoard renders /p/:productSlug - the board scoped to one product.
func ProductBoard() web.HandlerFunc {
	return func(c *web.Context) error {
		slug := c.Param("productSlug")
		product := &query.GetProductBySlug{Slug: slug}
		if err := bus.Dispatch(c, product); err != nil {
			if err == app.ErrNotFound {
				return c.NotFound()
			}
			return c.Failure(err)
		}
		if !product.Result.IsActive {
			return c.NotFound()
		}
		return renderBoard(c, product.Result)
	}
}

// ManageProductsPage renders the Admin - Products page.
func ManageProductsPage() web.HandlerFunc {
	return func(c *web.Context) error {
		list := &query.ListAllProducts{}
		if err := bus.Dispatch(c, list); err != nil {
			return c.Failure(err)
		}
		counts := &query.CountPostPerProduct{}
		if err := bus.Dispatch(c, counts); err != nil {
			counts.Result = map[int]int{}
		}
		return c.Page(http.StatusOK, web.Props{
			Page:  "Administration/pages/ManageProducts.page",
			Title: "Products · Site Settings",
			Data: web.Map{
				"products": list.Result,
				"counts":   counts.Result,
			},
		})
	}
}

// ListProducts returns the full catalogue (admins see inactive too).
func ListProducts() web.HandlerFunc {
	return func(c *web.Context) error {
		if c.User() == nil || !c.User().IsAdministrator() {
			return c.NotFound()
		}
		q := &query.ListAllProducts{}
		if err := bus.Dispatch(c, q); err != nil {
			return c.Failure(err)
		}
		return c.Ok(q.Result)
	}
}

// CreateProduct adds a product to the tenant catalogue.
func CreateProduct() web.HandlerFunc {
	return func(c *web.Context) error {
		action := new(actions.CreateProduct)
		if result := c.BindTo(action); !result.Ok {
			return c.HandleValidation(result)
		}
		create := &cmd.CreateProduct{
			Name:        action.Name,
			Slug:        action.Slug,
			Description: action.Description,
			Color:       action.Color,
			SortOrder:   action.SortOrder,
		}
		if err := bus.Dispatch(c, create); err != nil {
			return c.Failure(err)
		}
		return c.Ok(create.Result)
	}
}

// UpdateProduct edits name/description/color/order/active. Slug is immutable.
func UpdateProduct() web.HandlerFunc {
	return func(c *web.Context) error {
		action := new(actions.UpdateProduct)
		if result := c.BindTo(action); !result.Ok {
			return c.HandleValidation(result)
		}
		update := &cmd.UpdateProduct{
			ProductID:   action.ProductID,
			Name:        action.Name,
			Description: action.Description,
			Color:       action.Color,
			SortOrder:   action.SortOrder,
			IsActive:    action.IsActive,
		}
		if err := bus.Dispatch(c, update); err != nil {
			return c.Failure(err)
		}
		return c.Ok(web.Map{})
	}
}

// DeleteProduct removes a product; its ideas fall back to General.
func DeleteProduct() web.HandlerFunc {
	return func(c *web.Context) error {
		if c.User() == nil || !c.User().IsAdministrator() {
			return c.NotFound()
		}
		id, err := c.ParamAsInt("id")
		if err != nil {
			return c.NotFound()
		}
		if err := bus.Dispatch(c, &cmd.DeleteProduct{ProductID: id}); err != nil {
			return c.Failure(err)
		}
		return c.Ok(web.Map{})
	}
}
