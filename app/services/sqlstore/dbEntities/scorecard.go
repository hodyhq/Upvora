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
