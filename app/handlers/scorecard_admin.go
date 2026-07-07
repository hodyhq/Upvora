package handlers

import (
	"github.com/getfider/fider/app/actions"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/validate"
	"github.com/getfider/fider/app/pkg/web"
)

// ListScorecardFields returns the current tenant's scorecard field catalogue.
func ListScorecardFields() web.HandlerFunc {
	return func(c *web.Context) error {
		q := &query.ListScorecardFieldsForTenant{}
		if err := bus.Dispatch(c, q); err != nil {
			return c.Failure(err)
		}
		return c.Ok(q.Result)
	}
}

// CreateScorecardField adds a new admin-defined field.
func CreateScorecardField() web.HandlerFunc {
	return func(c *web.Context) error {
		action := new(actions.CreateScorecardField)
		if result := c.BindTo(action); !result.Ok {
			return c.HandleValidation(result)
		}
		create := &cmd.CreateScorecardField{
			Key:       action.Key,
			Label:     action.Label,
			GroupKey:  action.GroupKey,
			Type:      action.Type,
			Choices:   action.Choices,
			Weight:    action.Weight,
			Question:  action.Question,
			SortOrder: action.SortOrder,
		}
		if err := bus.Dispatch(c, create); err != nil {
			return c.Failure(err)
		}
		return c.Ok(create.Result)
	}
}

// UpdateScorecardField mutates a field the current tenant owns. System rows
// accept the same PUT payload but the postgres layer ignores locked columns.
func UpdateScorecardField() web.HandlerFunc {
	return func(c *web.Context) error {
		action := new(actions.UpdateScorecardField)
		if result := c.BindTo(action); !result.Ok {
			return c.HandleValidation(result)
		}
		update := &cmd.UpdateScorecardField{
			ID:        action.ID,
			Label:     action.Label,
			Choices:   action.Choices,
			Weight:    action.Weight,
			Question:  action.Question,
			SortOrder: action.SortOrder,
			IsActive:  action.IsActive,
		}
		if err := bus.Dispatch(c, update); err != nil {
			return c.Failure(err)
		}
		return c.Ok(web.Map{})
	}
}

// DeleteScorecardField removes a non-system field. DELETE-with-no-body — bypass
// BindTo for the same reason DeleteStatus does.
func DeleteScorecardField() web.HandlerFunc {
	return func(c *web.Context) error {
		id, err := c.ParamAsInt("id")
		if err != nil {
			r := validate.Success()
			r.AddFieldFailure("id", "Field id must be an integer.")
			return c.HandleValidation(r)
		}
		if err := bus.Dispatch(c, &cmd.DeleteScorecardField{ID: id}); err != nil {
			r := validate.Success()
			r.AddFieldFailure("field", err.Error())
			return c.HandleValidation(r)
		}
		return c.Ok(web.Map{})
	}
}

// UpdateScorecardSettings toggles the feature and writes the 4 band thresholds.
func UpdateScorecardSettings() web.HandlerFunc {
	return func(c *web.Context) error {
		action := new(actions.UpdateScorecardSettings)
		if result := c.BindTo(action); !result.Ok {
			return c.HandleValidation(result)
		}
		set := &cmd.SetTenantScorecardSettings{
			IsEnabled:         action.IsEnabled,
			BandStrong:        action.BandStrong,
			BandGood:          action.BandGood,
			BandRefine:        action.BandRefine,
			BandLow:           action.BandLow,
			TriggerStatusSlug: action.TriggerStatusSlug,
		}
		if err := bus.Dispatch(c, set); err != nil {
			return c.Failure(err)
		}
		return c.Ok(web.Map{})
	}
}
