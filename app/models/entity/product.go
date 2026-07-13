package entity

import "time"

// Product is a per-tenant board scope: its own board and roadmap views under
// one install. A lens, not a sub-tenant - statuses/tags/members stay shared.
type Product struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	// Color is a hex like "#38BDF8", or "" meaning "default" (tenant brand).
	Color     string    `json:"color"`
	SortOrder int       `json:"sortOrder"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
}

// ProductInfo is the compact product summary carried on posts.
type ProductInfo struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Slug  string `json:"slug"`
	Color string `json:"color"`
}
