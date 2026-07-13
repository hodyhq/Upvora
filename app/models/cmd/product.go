package cmd

import "github.com/getfider/fider/app/models/entity"

type CreateProduct struct {
	Name        string
	Slug        string
	Description string
	Color       string
	SortOrder   int

	Result *entity.Product
}

type UpdateProduct struct {
	ProductID   int
	Name        string
	Description string
	Color       string
	SortOrder   int
	IsActive    bool
}

// SetPostProduct reassigns a post to a product (0 = General/unassigned).
type SetPostProduct struct {
	Post      *entity.Post
	ProductID int
}

// DeleteProduct removes the product; its posts fall back to General via the
// ON DELETE SET NULL foreign key.
type DeleteProduct struct {
	ProductID int
}
