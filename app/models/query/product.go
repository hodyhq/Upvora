package query

import "github.com/getfider/fider/app/models/entity"

// ListActiveProducts returns the tenant's active products in sort order -
// this is what ships to the client session for the switcher.
type ListActiveProducts struct {
	Result []*entity.Product
}

// ListAllProducts includes inactive products (admin catalogue).
type ListAllProducts struct {
	Result []*entity.Product
}

// CountPostPerProduct returns idea counts keyed by product id ("0" = General).
type CountPostPerProduct struct {
	Result map[int]int
}

// GetProductBySlug resolves /p/:slug routes.
type GetProductBySlug struct {
	Slug string

	Result *entity.Product
}
