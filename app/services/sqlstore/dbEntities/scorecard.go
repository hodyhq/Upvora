package dbEntities

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/getfider/fider/app/models/entity"
)

type ScorecardField struct {
	ID         int             `db:"id"`
	TenantID   int             `db:"tenant_id"`
	Key        string          `db:"key"`
	Label      string          `db:"label"`
	GroupKey   string          `db:"group_key"`
	Type       string          `db:"type"`
	Choices    sql.NullString  `db:"choices"`
	Weight     sql.NullInt64   `db:"weight"`
	Question   sql.NullString  `db:"question"`
	SortOrder  int             `db:"sort_order"`
	IsSystem   bool            `db:"is_system"`
	IsActive   bool            `db:"is_active"`
	CreatedAt  time.Time       `db:"created_at"`
	UpdatedAt  time.Time       `db:"updated_at"`
}

type Scorecard struct {
	ID        int             `db:"id"`
	TenantID  int             `db:"tenant_id"`
	PostID    sql.NullInt64   `db:"post_id"`
	Title     string          `db:"title"`
	Values    string          `db:"values"`
	CreatedAt time.Time       `db:"created_at"`
	UpdatedAt time.Time       `db:"updated_at"`
}

func (s *Scorecard) ToModel() *entity.Scorecard {
	if s == nil {
		return nil
	}
	m := &entity.Scorecard{
		ID:        s.ID,
		Title:     s.Title,
		Values:    json.RawMessage(s.Values),
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
	if s.PostID.Valid {
		pid := int(s.PostID.Int64)
		m.PostID = &pid
	}
	return m
}

// ScorecardListItem is a Scorecard row joined with its linked post's live
// number/slug/author/votes for the dashboard list. Flat struct — dbx's row
// mapper only maps db-tagged direct fields, it does not flatten embedding.
type ScorecardListItem struct {
	ID          int            `db:"id"`
	TenantID    int            `db:"tenant_id"`
	PostID      sql.NullInt64  `db:"post_id"`
	Title       string         `db:"title"`
	Values      string         `db:"values"`
	CreatedAt   time.Time      `db:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at"`
	PostNumber  sql.NullInt64  `db:"post_number"`
	PostSlug    sql.NullString `db:"post_slug"`
	SubmittedBy sql.NullString `db:"submitted_by"`
	PostVotes   int            `db:"post_votes"`
}

func (s *ScorecardListItem) ToModel() *entity.Scorecard {
	if s == nil {
		return nil
	}
	m := &entity.Scorecard{
		ID:        s.ID,
		Title:     s.Title,
		Values:    json.RawMessage(s.Values),
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
		PostVotes: s.PostVotes,
	}
	if s.PostID.Valid {
		pid := int(s.PostID.Int64)
		m.PostID = &pid
	}
	if s.PostNumber.Valid {
		n := int(s.PostNumber.Int64)
		m.PostNumber = &n
	}
	if s.PostSlug.Valid {
		m.PostSlug = s.PostSlug.String
	}
	if s.SubmittedBy.Valid {
		m.SubmittedBy = s.SubmittedBy.String
	}
	return m
}

func (f *ScorecardField) ToModel() *entity.ScorecardField {
	if f == nil {
		return nil
	}
	m := &entity.ScorecardField{
		ID:        f.ID,
		Key:       f.Key,
		Label:     f.Label,
		GroupKey:  f.GroupKey,
		Type:      f.Type,
		SortOrder: f.SortOrder,
		IsSystem:  f.IsSystem,
		IsActive:  f.IsActive,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}
	if f.Choices.Valid && f.Choices.String != "" {
		m.Choices = json.RawMessage(f.Choices.String)
	}
	if f.Weight.Valid {
		w := int(f.Weight.Int64)
		m.Weight = &w
	}
	if f.Question.Valid {
		m.Question = f.Question.String
	}
	return m
}
