package actions

import (
	"context"
	"regexp"

	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/pkg/validate"
)

var themeHexRE = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)
var themeAccentKeys = map[string]bool{"buttons": true, "votes": true, "links": true, "header": true}

// UpdateTenantTheme is the admin action behind the Theme settings tab.
type UpdateTenantTheme struct {
	Primary      string            `json:"primary"`      // "" = built-in brand
	Accents      map[string]string `json:"accents"`      // key -> hex; missing/empty = follow brand
	DefaultTheme string            `json:"defaultTheme"` // light | dark | system
}

func (a *UpdateTenantTheme) IsAuthorized(ctx context.Context, user *entity.User) bool {
	return user != nil && user.IsAdministrator()
}

func (a *UpdateTenantTheme) Validate(ctx context.Context, user *entity.User) *validate.Result {
	result := validate.Success()
	if a.Primary != "" && !themeHexRE.MatchString(a.Primary) {
		result.AddFieldFailure("primary", "Primary color must be a hex value like #F97316, or empty for the default.")
	}
	if a.Accents == nil {
		a.Accents = map[string]string{}
	}
	for key, value := range a.Accents {
		if !themeAccentKeys[key] {
			result.AddFieldFailure("accents", "Unknown accent surface: "+key)
			continue
		}
		if value != "" && !themeHexRE.MatchString(value) {
			result.AddFieldFailure("accents", "Accent colors must be hex values like #3B82F6.")
		}
	}
	if a.DefaultTheme == "" {
		a.DefaultTheme = "light"
	}
	if a.DefaultTheme != "light" && a.DefaultTheme != "dark" && a.DefaultTheme != "system" {
		result.AddFieldFailure("defaultTheme", "Default appearance must be light, dark, or system.")
	}
	return result
}
