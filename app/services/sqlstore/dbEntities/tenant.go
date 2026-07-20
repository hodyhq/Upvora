package dbEntities

import (
	"database/sql"
	"encoding/json"

	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/enum"
	"github.com/getfider/fider/app/pkg/dbx"
	"github.com/getfider/fider/app/pkg/env"
)

type Tenant struct {
	ID                         int            `db:"id"`
	Name                       string         `db:"name"`
	Subdomain                  string         `db:"subdomain"`
	CNAME                      string         `db:"cname"`
	Invitation                 string         `db:"invitation"`
	WelcomeMessage             string         `db:"welcome_message"`
	WelcomeHeader              string         `db:"welcome_header"`
	DescriptionTemplate        string         `db:"description_template"`
	Status                     int            `db:"status"`
	Locale                     string         `db:"locale"`
	IsPrivate                  bool           `db:"is_private"`
	LogoBlobKey                string         `db:"logo_bkey"`
	CustomCSS                  string         `db:"custom_css"`
	AllowedSchemes             string         `db:"allowed_schemes"`
	IsEmailAuthAllowed         bool           `db:"is_email_auth_allowed"`
	IsFeedEnabled              bool           `db:"is_feed_enabled"`
	PreventIndexing            bool           `db:"prevent_indexing"`
	IsModerationEnabled        bool           `db:"is_moderation_enabled"`
	IsPro                      bool           `db:"is_pro"`
	HasPaddleSubscription      bool           `db:"has_paddle_subscription"`
	ScheduledDeletionAt        dbx.NullTime   `db:"scheduled_deletion_at"`
	ShareIdeaInstructions      string         `db:"share_idea_instructions"`
	RailCtaHeading             string         `db:"rail_cta_heading"`
	RailCtaText                string         `db:"rail_cta_text"`
	RailCtaButton              string         `db:"rail_cta_button"`
	DefaultTheme               string         `db:"default_theme"`
	ThemePrimary               string         `db:"theme_primary"`
	ThemeAccents               string         `db:"theme_accents"`
	SiteBannerEnabled          bool           `db:"site_banner_enabled"`
	SiteBannerMessage          string         `db:"site_banner_message"`
	SiteBannerVariant          string         `db:"site_banner_variant"`
	IsScorecardEnabled         bool           `db:"is_scorecard_enabled"`
	ScorecardBandStrong        int            `db:"scorecard_band_strong"`
	ScorecardBandGood          int            `db:"scorecard_band_good"`
	ScorecardBandRefine        int            `db:"scorecard_band_refine"`
	ScorecardBandLow           int            `db:"scorecard_band_low"`
	ScorecardBandStrongLabel   string         `db:"scorecard_band_strong_label"`
	ScorecardBandGoodLabel     string         `db:"scorecard_band_good_label"`
	ScorecardBandRefineLabel   string         `db:"scorecard_band_refine_label"`
	ScorecardBandLowLabel      string         `db:"scorecard_band_low_label"`
	ScorecardBandNoneLabel     string         `db:"scorecard_band_none_label"`
	AIEnabled                  bool           `db:"ai_enabled"`
	AIProvider                 string         `db:"ai_provider"`
	AIAPIKey                   string         `db:"ai_api_key"`
	AIModel                    string         `db:"ai_model"`
	AICustomBaseURL            string         `db:"ai_custom_base_url"`
	AICustomModel              string         `db:"ai_custom_model"`
	AIWebSearchEnabled         bool           `db:"ai_web_search_enabled"`
	AIWebSearchProvider        string         `db:"ai_web_search_provider"`
	AIWebSearchAPIKey          string         `db:"ai_web_search_api_key"`
	AIWebSearchBaseURL         string         `db:"ai_web_search_base_url"`
	ScorecardTriggerStatusSlug sql.NullString `db:"scorecard_trigger_status_slug"`
}

func (t *Tenant) ToModel() *entity.Tenant {
	if t == nil {
		return nil
	}

	// Self-hosted: all features are available (isPro = true)
	// Hosted multi-tenant: isPro based on subscription status
	isPro := true
	if env.IsMultiHostMode() {
		isPro = t.IsPro || t.HasPaddleSubscription
	}

	tenant := &entity.Tenant{
		ID:                       t.ID,
		Name:                     t.Name,
		Subdomain:                t.Subdomain,
		CNAME:                    t.CNAME,
		Invitation:               t.Invitation,
		WelcomeMessage:           t.WelcomeMessage,
		WelcomeHeader:            t.WelcomeHeader,
		DescriptionTemplate:      t.DescriptionTemplate,
		Status:                   enum.TenantStatus(t.Status),
		Locale:                   t.Locale,
		IsPrivate:                t.IsPrivate,
		LogoBlobKey:              t.LogoBlobKey,
		CustomCSS:                t.CustomCSS,
		AllowedSchemes:           t.AllowedSchemes,
		IsEmailAuthAllowed:       t.IsEmailAuthAllowed,
		IsFeedEnabled:            t.IsFeedEnabled,
		PreventIndexing:          t.PreventIndexing,
		IsModerationEnabled:      isPro && t.IsModerationEnabled,
		IsPro:                    isPro,
		ShareIdeaInstructions:    t.ShareIdeaInstructions,
		RailCtaHeading:           t.RailCtaHeading,
		RailCtaText:              t.RailCtaText,
		RailCtaButton:            t.RailCtaButton,
		DefaultTheme:             t.DefaultTheme,
		ThemePrimary:             t.ThemePrimary,
		SiteBannerEnabled:        t.SiteBannerEnabled,
		SiteBannerMessage:        t.SiteBannerMessage,
		SiteBannerVariant:        t.SiteBannerVariant,
		IsScorecardEnabled:       t.IsScorecardEnabled,
		ScorecardBandStrong:      t.ScorecardBandStrong,
		ScorecardBandGood:        t.ScorecardBandGood,
		ScorecardBandRefine:      t.ScorecardBandRefine,
		ScorecardBandLow:         t.ScorecardBandLow,
		ScorecardBandStrongLabel: t.ScorecardBandStrongLabel,
		ScorecardBandGoodLabel:   t.ScorecardBandGoodLabel,
		ScorecardBandRefineLabel: t.ScorecardBandRefineLabel,
		ScorecardBandLowLabel:    t.ScorecardBandLowLabel,
		ScorecardBandNoneLabel:   t.ScorecardBandNoneLabel,
		AIEnabled:                t.AIEnabled,
		AIProvider:               t.AIProvider,
		AIAPIKey:                 t.AIAPIKey,
		AIModel:                  t.AIModel,
		AICustomBaseURL:          t.AICustomBaseURL,
		AICustomModel:            t.AICustomModel,
		AIWebSearchEnabled:       t.AIWebSearchEnabled,
		AIWebSearchProvider:      t.AIWebSearchProvider,
		AIWebSearchAPIKey:        t.AIWebSearchAPIKey,
		AIWebSearchBaseURL:       t.AIWebSearchBaseURL,
	}

	if t.ScorecardTriggerStatusSlug.Valid {
		tenant.ScorecardTriggerStatusSlug = t.ScorecardTriggerStatusSlug.String
	}

	if t.ScheduledDeletionAt.Valid {
		tenant.ScheduledDeletionAt = &t.ScheduledDeletionAt.Time
	}

	if t.ThemeAccents != "" && t.ThemeAccents != "{}" {
		accents := map[string]string{}
		if err := json.Unmarshal([]byte(t.ThemeAccents), &accents); err == nil && len(accents) > 0 {
			tenant.ThemeAccents = accents
		}
	}
	return tenant
}
