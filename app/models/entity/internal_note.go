package entity

import "time"

// InternalNote is the single shared team note attached to a post. It is only
// ever visible to collaborators/administrators, and the linked scorecard reads
// the same row - the two surfaces can never drift apart.
type InternalNote struct {
	Content       string    `json:"content"`
	UpdatedAt     time.Time `json:"updatedAt"`
	UpdatedByName string    `json:"updatedByName"`
}
