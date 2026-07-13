package actions

import (
	"context"
	"regexp"

	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/validate"
)

var productSlugRE = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var productColorRE = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

// CreateProduct is the admin action to add a product.
type CreateProduct struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Color       string `json:"color"` // "" = default (brand)
	SortOrder   int    `json:"sortOrder"`
}

func (a *CreateProduct) IsAuthorized(ctx context.Context, user *entity.User) bool {
	return user != nil && user.IsAdministrator()
}

func (a *CreateProduct) Validate(ctx context.Context, user *entity.User) *validate.Result {
	result := validate.Success()
	validateProductFields(result, a.Name, a.Slug, a.Color)
	if result.Ok {
		existing := &query.GetProductBySlug{Slug: a.Slug}
		if err := bus.Dispatch(ctx, existing); err == nil && existing.Result != nil {
			result.AddFieldFailure("slug", "A product with this slug already exists.")
		}
	}
	return result
}

// UpdateProduct is the admin action to edit a product. Slug is immutable -
// it is a public URL segment.
type UpdateProduct struct {
	ProductID   int    `route:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
	SortOrder   int    `json:"sortOrder"`
	IsActive    bool   `json:"isActive"`
}

func (a *UpdateProduct) IsAuthorized(ctx context.Context, user *entity.User) bool {
	return user != nil && user.IsAdministrator()
}

func (a *UpdateProduct) Validate(ctx context.Context, user *entity.User) *validate.Result {
	result := validate.Success()
	validateProductFields(result, a.Name, "skip-slug", a.Color)
	return result
}

func validateProductFields(result *validate.Result, name, slug, color string) {
	if name == "" {
		result.AddFieldFailure("name", "Name is required.")
	} else if len(name) > 60 {
		result.AddFieldFailure("name", "Name must be at most 60 characters.")
	}
	if slug != "skip-slug" {
		if slug == "" {
			result.AddFieldFailure("slug", "Slug is required.")
		} else if len(slug) > 60 || !productSlugRE.MatchString(slug) {
			result.AddFieldFailure("slug", "Slug must be lowercase letters, digits, and dashes.")
		}
	}
	if color != "" && !productColorRE.MatchString(color) {
		result.AddFieldFailure("color", "Color must be a hex value like #38BDF8, or empty for the brand default.")
	}
}
