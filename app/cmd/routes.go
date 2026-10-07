package cmd

import (
	"net/http"
	"time"

	"github.com/getfider/fider/app/handlers"
	"github.com/getfider/fider/app/handlers/apiv1"
	"github.com/getfider/fider/app/handlers/webhooks"
	"github.com/getfider/fider/app/middlewares"
	"github.com/getfider/fider/app/models/enum"
	"github.com/getfider/fider/app/pkg/env"
	"github.com/getfider/fider/app/pkg/web"
)

func routes(r *web.Engine) *web.Engine {
	r.Worker().Use(middlewares.WorkerSetup())

	r.Get("/_health", handlers.Health())

	r.Use(middlewares.CatchPanic())
	r.Use(middlewares.Instrumentation())

	r.NotFound(func(c *web.Context) error {
		mw := middlewares.Chain(
			middlewares.WebSetup(),
			middlewares.Tenant(),
			middlewares.NoIndex(),
			middlewares.User(),
		)
		next := mw(func(c *web.Context) error {
			return c.NotFound()
		})
		return next(c)
	})

	r.Use(middlewares.Secure())
	r.Use(middlewares.Compress())

	assets := r.Group()
	{
		assets.Use(middlewares.CORS())
		assets.Use(middlewares.ClientCache(365 * 24 * time.Hour))
		assets.Get("/static/favicon", handlers.Favicon())
		assets.Static("/assets/*filepath", "dist")
		assets.Static("/static/assets/*filepath", "static")
	}

	feed := r.Group()
	{
		feed.Use(middlewares.CORS())
		feed.Use(middlewares.WebSetup())
		feed.Use(middlewares.Tenant())
		feed.Use(middlewares.NoIndex())
		feed.Use(middlewares.ClientCache(5 * time.Minute))

		feed.Get("/feed/global.atom", handlers.GlobalFeed())
		feed.Get("/feed/posts/:path", handlers.CommentFeed())
	}

	r.Use(middlewares.Session())

	r.Get("/robots.txt", handlers.RobotsTXT())
	r.Post("/_api/log-error", handlers.LogError())

	r.Use(middlewares.Maintenance())
	r.Use(middlewares.WebSetup())
	r.Use(middlewares.Tenant())
	r.Use(middlewares.NoIndex())
	r.Use(middlewares.User())

	r.Get("/privacy", handlers.LegalPage("Privacy Policy", "privacy.md"))

	// Stripe webhooks (before CSRF middleware)
	stripeWh := r.Group()
	{
		stripeWh.Post("/webhooks/stripe", webhooks.IncomingStripeWebhook())
	}

	// OAuth 2.1 authorization server for MCP clients: discovery, registration
	// and token are called cross-origin without cookies, so they sit before
	// CSRF. Each handler returns 404 unless MCP is enabled for the site.
	oauthAS := r.Group()
	{
		oauthAS.Get("/.well-known/oauth-authorization-server", handlers.OAuthAuthorizationServerMetadata())
		oauthAS.Get("/.well-known/oauth-protected-resource", handlers.OAuthProtectedResourceMetadata())
		oauthAS.Get("/.well-known/oauth-protected-resource/mcp", handlers.OAuthProtectedResourceMetadata())
		oauthAS.Post("/oauth2/register", handlers.OAuthRegister())
		oauthAS.Post("/oauth2/token", handlers.OAuthTokenEndpoint())
	}

	r.Use(middlewares.CSRF())

	r.Get("/terms", handlers.LegalPage("Terms of Service", "terms.md"))

	r.Post("/_api/tenants", handlers.CreateTenant())
	r.Get("/_api/tenants/:subdomain/availability", handlers.CheckAvailability())
	r.Get("/signup", handlers.SignUp())
	r.Get("/oauth/:provider", handlers.SignInByOAuth())
	r.Get("/oauth/:provider/callback", handlers.OAuthCallback())

	// Starting from this step, a Tenant is required
	r.Use(middlewares.RequireTenant())

	r.Get("/sitemap.xml", handlers.Sitemap())

	tenantAssets := r.Group()
	{
		tenantAssets.Use(middlewares.ClientCache(5 * 24 * time.Hour))
		tenantAssets.Get("/static/avatars/letter/:id/:name", handlers.LetterAvatar())
		tenantAssets.Get("/static/avatars/gravatar/:id/*name", handlers.Gravatar())

		tenantAssets.Use(middlewares.ClientCache(30 * 24 * time.Hour))
		tenantAssets.Get("/static/favicon/*bkey", handlers.Favicon())
		tenantAssets.Get("/static/images/*bkey", handlers.ViewUploadedImage())
		tenantAssets.Get("/static/custom/:md5.css", func(c *web.Context) error {
			return c.Blob(http.StatusOK, "text/css", []byte(c.Tenant().CustomCSS))
		})
	}

	r.Get("/_design", handlers.Page("Design System", "A preview of Fider UI elements", "DesignSystem/DesignSystem.page"))
	r.Get("/signup/verify", handlers.VerifySignUpKey())
	r.Post("/_api/signup/resend", handlers.ResendSignUpEmail())
	r.Get("/signout", handlers.SignOut())
	r.Get("/oauth/:provider/token", handlers.OAuthToken())
	r.Get("/oauth/:provider/echo", handlers.OAuthEcho())

	// If tenant is pending, block it from using any other route
	r.Use(middlewares.BlockPendingTenants())

	r.Get("/signin", handlers.SignInPage())
	r.Get("/signin/complete", handlers.CompleteSignInProfilePage())
	r.Get("/loginemailsent", handlers.LoginEmailSentPage())
	r.Get("/not-invited", handlers.NotInvitedPage())
	r.Get("/access-denied", handlers.AccessDeniedPage())
	r.Get("/signin/verify", handlers.VerifySignInKey(enum.EmailVerificationKindSignIn))
	r.Get("/invite/verify", handlers.VerifySignInKey(enum.EmailVerificationKindUserInvitation))
	r.Post("/_api/signin/complete", handlers.CompleteSignInProfile())
	r.Post("/_api/signin", handlers.SignInByEmail())
	r.Post("/_api/signin/newuser", handlers.SignInByEmailWithName())
	r.Post("/_api/signin/verify", handlers.VerifySignInCode())
	r.Post("/_api/signin/resend", handlers.ResendSignInCode())

	// Cancel a scheduled site deletion. Authorised by the unguessable key in the emailed link
	// alone, so it must stay reachable without authentication (it only restores access).
	if !env.IsSingleHostMode() {
		r.Get("/admin/danger-zone/cancel", handlers.CancelTenantDeletion())
	}

	// Block if it's private tenant with unauthenticated user
	r.Use(middlewares.CheckTenantPrivacy())

	r.Get("/", handlers.Index())
	r.Get("/p/:productSlug", handlers.ProductBoard())
	r.Get("/roadmap", handlers.RoadmapPage())
	r.Get("/posts/:number", handlers.PostDetails())
	r.Get("/posts/:number/:slug", handlers.PostDetails())

	ui := r.Group()
	{
		// From this step, a User is required
		ui.Use(middlewares.IsAuthenticated())

		ui.Get("/settings", handlers.UserSettings())
		ui.Get("/notifications", handlers.Notifications())
		ui.Get("/notifications/:id", handlers.ReadNotification())
		ui.Get("/_api/notifications/unread", handlers.GetAllNotifications())
		ui.Get("/change-email/verify", handlers.VerifyChangeEmailKey())

		ui.Delete("/_api/user", handlers.DeleteUser())
		ui.Post("/_api/user/regenerate-apikey", handlers.RegenerateAPIKey())
		ui.Post("/_api/user/settings", handlers.UpdateUserSettings())
		ui.Post("/_api/user/change-email", handlers.ChangeUserEmail())
		ui.Post("/_api/notifications/read-all", handlers.ReadAllNotifications())
		ui.Get("/_api/notifications/unread/total", handlers.TotalUnreadNotifications())
		// Members may publish their own private idea; the handler enforces the rules.
		ui.Post("/_api/posts/:number/privacy", handlers.SetPostPrivacy())
		// MCP client authorization: the user is signed in by now (IsAuthenticated
		// sends them through the site's normal sign-in), then consents here.
		ui.Get("/oauth2/authorize", handlers.OAuthAuthorize())
		ui.Post("/_api/oauth2/authorize", handlers.OAuthAuthorizeDecision())

		// From this step, only Collaborators and Administrators are allowed
		ui.Use(middlewares.IsAuthorized(enum.RoleCollaborator, enum.RoleAdministrator))

		ui.Get("/admin/theme", handlers.ManageThemePage())
		ui.Post("/_api/admin/settings/theme", handlers.UpdateTenantTheme())
		ui.Get("/admin/products", handlers.ManageProductsPage())
		ui.Get("/_api/admin/products", handlers.ListProducts())
		ui.Post("/_api/admin/products", handlers.CreateProduct())
		ui.Put("/_api/admin/products/:id", handlers.UpdateProduct())
		ui.Delete("/_api/admin/products/:id", handlers.DeleteProduct())
		ui.Put("/_api/posts/:number/product", handlers.SetPostProduct())
		ui.Get("/_api/posts/:number/internal-note", handlers.GetInternalNote())
		ui.Put("/_api/posts/:number/internal-note", handlers.SetInternalNote())
		ui.Get("/scorecard", handlers.ScorecardPage())
		ui.Get("/scorecard/:id", handlers.ScorecardCardPage())
		ui.Post("/_api/scorecards", handlers.CreateScorecard())
		ui.Put("/_api/scorecards/:id", handlers.UpdateScorecard())
		ui.Delete("/_api/scorecards/:id", handlers.DeleteScorecard())

		// locale is forced to English for administrative pages.
		// This is meant to be removed when all pages are translated.
		ui.Use(middlewares.SetLocale("en"))

		// Collaborator-accessible settings pages — full edit rights on exactly
		// these four: Invitations, Tags, Scorecard settings, Scorecard fields.
		// /admin (General) is readable so the settings landing page and its side
		// menu work for collaborators — its inputs render disabled for them and
		// the save API below is admin-only. Everything else is admin-only.
		ui.Get("/admin", handlers.GeneralSettingsPage())
		ui.Get("/admin/invitations", handlers.Page("Invitations · Site Settings", "", "Administration/pages/Invitations.page"))
		ui.Get("/admin/tags", handlers.ManageTags())
		ui.Get("/admin/scorecard-settings", handlers.ManageScorecardSettings())
		ui.Get("/admin/scorecard-fields", handlers.LegacyScorecardFieldsRedirect())
		ui.Get("/_api/admin/scorecard-fields", handlers.ListScorecardFields())
		ui.Post("/_api/admin/scorecard-fields", handlers.CreateScorecardField())
		ui.Put("/_api/admin/scorecard-fields/:id", handlers.UpdateScorecardField())
		ui.Delete("/_api/admin/scorecard-fields/:id", handlers.DeleteScorecardField())
		ui.Post("/_api/admin/scorecard-settings", handlers.UpdateScorecardSettings())

		// Pro features (available to self-hosters and pro hosted customers)
		proUi := ui.Group()
		{
			proUi.Use(middlewares.RequirePro())
			proUi.Get("/admin/moderation", handlers.ModerationPage())
		}

		// From this step, only Administrators are allowed
		ui.Use(middlewares.IsAuthorized(enum.RoleAdministrator))

		ui.Get("/admin/ai", handlers.ManageAIPage())
		ui.Post("/_api/admin/settings/ai", handlers.UpdateAISettings())
		ui.Get("/_api/admin/settings/ai/key", handlers.GetAIProviderKey())
		ui.Post("/_api/admin/ai/agents", handlers.UpsertAIAgentHandler())
		ui.Get("/_api/posts/:number/brief/download", handlers.DownloadIdeaBrief())
		ui.Get("/_api/posts/:number/brief/transcript", handlers.GetBriefTranscript())
		ui.Get("/_api/admin/system/status", handlers.SystemStatus())
		ui.Post("/_api/admin/system/update", handlers.SystemTriggerUpdate())
		ui.Get("/admin/advanced", handlers.AdvancedSettingsPage())
		ui.Get("/admin/privacy", handlers.Page("Privacy · Site Settings", "", "Administration/pages/PrivacySettings.page"))
		ui.Get("/admin/users", handlers.ManageMembers())
		ui.Get("/admin/statuses", handlers.ManageStatuses())
		ui.Get("/admin/banner", handlers.ManageBanner())
		ui.Get("/admin/authentication", handlers.ManageAuthentication())
		ui.Get("/_api/admin/oauth/:provider", handlers.GetOAuthConfig())

		// Danger Zone — delete the entire site. Hosted multi-tenant only; owner-only is
		// enforced inside the handlers.
		if !env.IsSingleHostMode() {
			ui.Get("/admin/danger-zone", handlers.DangerZonePage())
			ui.Delete("/_api/admin/tenant", handlers.RequestTenantDeletion())
			ui.Post("/_api/admin/tenant/cancel-deletion", handlers.CancelTenantDeletionByOwner())
		}

		ui.Get("/admin/export", handlers.Page("Export · Site Settings", "", "Administration/pages/Export.page"))
		ui.Get("/admin/export/posts.csv", handlers.ExportPostsToCSV())
		ui.Get("/admin/export/backup.zip", handlers.ExportBackupZip())
		ui.Get("/admin/export/tags.json", handlers.ExportTagsJSON())
		ui.Post("/_api/admin/import/tags", handlers.ImportTagsJSON())
		ui.Get("/admin/webhooks", handlers.ManageWebhooks())
		ui.Post("/_api/admin/webhook", handlers.CreateWebhook())
		ui.Put("/_api/admin/webhook/:id", handlers.UpdateWebhook())
		ui.Delete("/_api/admin/webhook/:id", handlers.DeleteWebhook())
		ui.Post("/_api/admin/webhook/test/:id", handlers.TestWebhook())
		ui.Post("/_api/admin/webhook/preview", handlers.PreviewWebhook())
		ui.Get("/_api/admin/webhook/props/:type", handlers.GetWebhookProps())
		ui.Post("/_api/admin/settings/general", handlers.UpdateSettings())
		ui.Post("/_api/admin/settings/advanced", handlers.UpdateAdvancedSettings())
		ui.Post("/_api/admin/settings/privacy", handlers.UpdatePrivacySettings())
		ui.Post("/_api/admin/settings/mcp", handlers.UpdateMCPSettings())
		ui.Get("/admin/mcp", handlers.ManageMCPPage())
		ui.Get("/_api/admin/mcp/clients", handlers.ListMCPClients())
		ui.Post("/_api/admin/mcp/clients", handlers.CreateMCPClient())
		ui.Delete("/_api/admin/mcp/clients/:id", handlers.DeleteMCPClient())
		ui.Post("/_api/admin/settings/emailauth", handlers.UpdateEmailAuthAllowed())
		ui.Post("/_api/admin/settings/site-banner", handlers.UpdateSiteBanner())
		ui.Get("/_api/admin/statuses", handlers.ListStatuses())
		ui.Post("/_api/admin/statuses", handlers.CreateStatus())
		ui.Put("/_api/admin/statuses/:id", handlers.UpdateStatus())
		ui.Delete("/_api/admin/statuses/:id", handlers.DeleteStatus())

		ui.Post("/_api/admin/oauth", handlers.SaveOAuthConfig())
		ui.Post("/_api/admin/oauth/:provider/status", handlers.SetSystemProviderStatus())
		ui.Post("/_api/admin/roles/:role/users", handlers.ChangeUserRole())
		ui.Put("/_api/admin/users/:userID/block", handlers.BlockUser())
		ui.Delete("/_api/admin/users/:userID/block", handlers.UnblockUser())
		ui.Put("/_api/admin/users/:userID/trust", handlers.TrustUser())
		ui.Delete("/_api/admin/users/:userID/trust", handlers.UntrustUser())

		// Pro features (available to self-hosters and pro hosted customers)
		proAdmin := ui.Group()
		{
			proAdmin.Use(middlewares.RequirePro())
			proAdmin.Get("/_api/admin/moderation/items", handlers.GetModerationItems())
			proAdmin.Get("/_api/admin/moderation/count", handlers.GetModerationCount())
		}

		if env.IsBillingEnabled() {
			ui.Get("/admin/billing", handlers.ManageBilling())
			ui.Post("/_api/admin/billing/portal", handlers.CreateStripePortalSession())
			ui.Post("/_api/admin/billing/checkout", handlers.CreateStripeCheckoutSession())
			ui.Post("/_api/admin/billing/checkout/annual", handlers.CreateStripeAnnualCheckoutSession())
		}
	}

	// Public operations
	// Does not require authentication
	publicApi := r.Group()
	{
		publicApi.Get("/api/v1/similarposts", apiv1.FindSimilarPosts())
		publicApi.Get("/api/v1/posts", apiv1.SearchPosts())
		publicApi.Get("/api/v1/tags", apiv1.ListTags())
		publicApi.Get("/api/v1/posts/:number", apiv1.GetPost())
		publicApi.Get("/api/v1/posts/:number/comments", apiv1.ListComments())
		publicApi.Get("/api/v1/posts/:number/comments/:id", apiv1.GetComment())
		publicApi.Get("/api/v1/taggable-users", apiv1.ListTaggableUsers())
		publicApi.Get("/api/v1/posts/:number/votes", apiv1.ListVotes())
		publicApi.Get("/api/v1/tenant", apiv1.GetTenant())
	}

	// Operations used to manage the content of a site
	// Available to any authenticated user
	membersApi := r.Group()
	{
		membersApi.Use(middlewares.IsAuthenticated())
		membersApi.Use(middlewares.BlockLockedTenants())

		membersApi.Post("/api/v1/posts", apiv1.CreatePost())
		membersApi.Put("/api/v1/posts/:number", apiv1.UpdatePost())
		membersApi.Post("/api/v1/posts/:number/comments/:id/reactions/:reaction", apiv1.ToggleReaction())
		membersApi.Post("/api/v1/posts/:number/comments", apiv1.PostComment())
		membersApi.Put("/api/v1/posts/:number/comments/:id", apiv1.UpdateComment())
		membersApi.Delete("/api/v1/posts/:number/comments/:id", apiv1.DeleteComment())
		membersApi.Post("/api/v1/ai/ideate", handlers.AIIdeate())
		membersApi.Post("/api/v1/ai/finalize", handlers.AIFinalize())
		membersApi.Get("/api/v1/posts/:number/brief", handlers.GetIdeaBriefHandler())
		membersApi.Post("/api/v1/posts/:number/votes", apiv1.AddVote())
		membersApi.Delete("/api/v1/posts/:number/votes", apiv1.RemoveVote())
		membersApi.Post("/api/v1/posts/:number/votes/toggle", apiv1.ToggleVote())
		membersApi.Post("/api/v1/posts/:number/subscription", apiv1.Subscribe())
		membersApi.Delete("/api/v1/posts/:number/subscription", apiv1.Unsubscribe())

		// Self-service and notifications (same handlers as the UI's /_api routes).
		membersApi.Get("/api/v1/notifications", handlers.GetAllNotifications())
		membersApi.Get("/api/v1/notifications/unread/total", handlers.TotalUnreadNotifications())
		membersApi.Post("/api/v1/notifications/read-all", handlers.ReadAllNotifications())
		membersApi.Get("/api/v1/user/settings", apiv1.GetUserSettings())
		membersApi.Post("/api/v1/user/settings", handlers.UpdateUserSettings())
		membersApi.Post("/api/v1/user/change-email", handlers.ChangeUserEmail())
		// Members may publish their own private idea; the handler enforces the rules.
		membersApi.Post("/api/v1/posts/:number/privacy", handlers.SetPostPrivacy())

		membersApi.Use(middlewares.IsAuthorized(enum.RoleCollaborator, enum.RoleAdministrator))
		membersApi.Put("/api/v1/posts/:number/status", apiv1.SetResponse())
	}

	// Operations used to manage a site
	// Available to both collaborators and administrators
	staffApi := r.Group()
	{
		staffApi.Use(middlewares.SetLocale("en"))
		staffApi.Use(middlewares.IsAuthenticated())
		staffApi.Use(middlewares.IsAuthorized(enum.RoleCollaborator, enum.RoleAdministrator))

		staffApi.Get("/api/v1/users", apiv1.ListUsers())
		staffApi.Post("/api/v1/invitations/send", apiv1.SendInvites())
		staffApi.Post("/api/v1/invitations/sample", apiv1.SendSampleInvite())
		staffApi.Get("/api/v1/posts/:number/internal-note", handlers.GetInternalNote())
		staffApi.Get("/api/v1/admin/scorecard-fields", handlers.ListScorecardFields())
		staffApi.Get("/api/v1/admin/products", handlers.ListProducts())
		staffApi.Get("/api/v1/scorecards", apiv1.ListScorecards())
		staffApi.Get("/api/v1/scorecards/:id", apiv1.GetScorecard())
		staffApi.Get("/api/v1/admin/scorecard-settings", apiv1.GetScorecardSettings())

		staffApi.Use(middlewares.BlockLockedTenants())
		staffApi.Post("/api/v1/posts/:number/tags/:slug", apiv1.AssignTag())
		staffApi.Delete("/api/v1/posts/:number/tags/:slug", apiv1.UnassignTag())
		// Tag CRUD is collaborator-editable: collaborators own the Tags settings
		// page with full edit rights.
		staffApi.Post("/api/v1/tags", apiv1.CreateEditTag())
		staffApi.Put("/api/v1/tags/:slug", apiv1.CreateEditTag())
		staffApi.Delete("/api/v1/tags/:slug", apiv1.DeleteTag())
		staffApi.Put("/api/v1/posts/:number/product", handlers.SetPostProduct())
		staffApi.Put("/api/v1/posts/:number/internal-note", handlers.SetInternalNote())
		staffApi.Post("/api/v1/scorecards", handlers.CreateScorecard())
		staffApi.Put("/api/v1/scorecards/:id", handlers.UpdateScorecard())
		staffApi.Delete("/api/v1/scorecards/:id", handlers.DeleteScorecard())
		staffApi.Post("/api/v1/admin/scorecard-fields", handlers.CreateScorecardField())
		staffApi.Put("/api/v1/admin/scorecard-fields/:id", handlers.UpdateScorecardField())
		staffApi.Delete("/api/v1/admin/scorecard-fields/:id", handlers.DeleteScorecardField())
		staffApi.Post("/api/v1/admin/scorecard-settings", handlers.UpdateScorecardSettings())
		// Product handlers enforce admin-only themselves for list and delete.
		staffApi.Post("/api/v1/admin/products", handlers.CreateProduct())
		staffApi.Put("/api/v1/admin/products/:id", handlers.UpdateProduct())
		staffApi.Delete("/api/v1/admin/products/:id", handlers.DeleteProduct())
	}

	// Operations used to manage a site
	// Only available to administrators
	adminApi := r.Group()
	{
		adminApi.Use(middlewares.SetLocale("en"))
		adminApi.Use(middlewares.IsAuthenticated())
		adminApi.Use(middlewares.IsAuthorized(enum.RoleAdministrator))

		adminApi.Post("/api/v1/users", apiv1.CreateUser())
		adminApi.Get("/api/v1/posts/:number/brief/transcript", handlers.GetBriefTranscript())
		adminApi.Get("/api/v1/admin/system/status", handlers.SystemStatus())
		adminApi.Get("/api/v1/admin/statuses", handlers.ListStatuses())
		adminApi.Get("/api/v1/admin/webhooks/props/:type", handlers.GetWebhookProps())
		adminApi.Get("/api/v1/admin/oauth/:provider", handlers.GetOAuthConfig())
		adminApi.Get("/api/v1/admin/oauth", apiv1.ListOAuthProviders())
		adminApi.Get("/api/v1/admin/settings/advanced", apiv1.GetAdvancedSettings())
		adminApi.Get("/api/v1/admin/settings/ai", apiv1.GetAISettings())
		adminApi.Get("/api/v1/admin/webhooks", apiv1.ListWebhooks())
		adminApi.Get("/api/v1/admin/mcp/clients", handlers.ListMCPClients())

		// Billing and site deletion stay reachable on a locked tenant, as in the UI.
		if env.IsBillingEnabled() {
			adminApi.Post("/api/v1/admin/billing/portal", handlers.CreateStripePortalSession())
			adminApi.Get("/api/v1/admin/billing", apiv1.GetBillingState())
			adminApi.Post("/api/v1/admin/billing/checkout", handlers.CreateStripeCheckoutSession())
			adminApi.Post("/api/v1/admin/billing/checkout/annual", handlers.CreateStripeAnnualCheckoutSession())
		}
		if !env.IsSingleHostMode() {
			adminApi.Delete("/api/v1/admin/tenant", handlers.RequestTenantDeletion())
			adminApi.Post("/api/v1/admin/tenant/cancel-deletion", handlers.CancelTenantDeletionByOwner())
		}

		// Pro features (available to self-hosters and pro hosted customers)
		proAdminApi := adminApi.Group()
		{
			proAdminApi.Use(middlewares.RequirePro())
			proAdminApi.Post("/api/v1/admin/moderation/posts/:id/approve-and-verify", apiv1.ApprovePostAndVerify())
			proAdminApi.Post("/api/v1/admin/moderation/posts/:id/decline-and-block", apiv1.DeclinePostAndBlock())
			proAdminApi.Post("/api/v1/admin/moderation/posts/:id/approve", apiv1.ApprovePost())
			proAdminApi.Post("/api/v1/admin/moderation/posts/:id/decline", apiv1.DeclinePost())
			proAdminApi.Post("/api/v1/admin/moderation/comments/:id/approve-and-verify", apiv1.ApproveCommentAndVerify())
			proAdminApi.Post("/api/v1/admin/moderation/comments/:id/decline-and-block", apiv1.DeclineCommentAndBlock())
			proAdminApi.Post("/api/v1/admin/moderation/comments/:id/approve", apiv1.ApproveComment())
			proAdminApi.Post("/api/v1/admin/moderation/comments/:id/decline", apiv1.DeclineComment())
			proAdminApi.Get("/api/v1/admin/moderation/items", handlers.GetModerationItems())
			proAdminApi.Get("/api/v1/admin/moderation/count", handlers.GetModerationCount())
		}

		adminApi.Use(middlewares.BlockLockedTenants())
		adminApi.Delete("/api/v1/posts/:number", apiv1.DeletePost())
		adminApi.Post("/api/v1/admin/settings/ai", handlers.UpdateAISettings())
		adminApi.Post("/api/v1/admin/ai/agents", handlers.UpsertAIAgentHandler())
		adminApi.Post("/api/v1/admin/system/update", handlers.SystemTriggerUpdate())
		adminApi.Post("/api/v1/admin/settings/general", handlers.UpdateSettings())
		adminApi.Post("/api/v1/admin/settings/theme", handlers.UpdateTenantTheme())
		adminApi.Post("/api/v1/admin/settings/advanced", handlers.UpdateAdvancedSettings())
		adminApi.Post("/api/v1/admin/settings/privacy", handlers.UpdatePrivacySettings())
		adminApi.Post("/api/v1/admin/settings/mcp", handlers.UpdateMCPSettings())
		adminApi.Post("/api/v1/admin/mcp/clients", handlers.CreateMCPClient())
		adminApi.Delete("/api/v1/admin/mcp/clients/:id", handlers.DeleteMCPClient())
		adminApi.Post("/api/v1/admin/settings/emailauth", handlers.UpdateEmailAuthAllowed())
		adminApi.Post("/api/v1/admin/settings/site-banner", handlers.UpdateSiteBanner())
		adminApi.Post("/api/v1/admin/statuses", handlers.CreateStatus())
		adminApi.Put("/api/v1/admin/statuses/:id", handlers.UpdateStatus())
		adminApi.Delete("/api/v1/admin/statuses/:id", handlers.DeleteStatus())
		adminApi.Post("/api/v1/admin/webhooks", handlers.CreateWebhook())
		adminApi.Put("/api/v1/admin/webhooks/:id", handlers.UpdateWebhook())
		adminApi.Delete("/api/v1/admin/webhooks/:id", handlers.DeleteWebhook())
		adminApi.Post("/api/v1/admin/webhooks/test/:id", handlers.TestWebhook())
		adminApi.Post("/api/v1/admin/webhooks/preview", handlers.PreviewWebhook())
		adminApi.Post("/api/v1/admin/import/tags", handlers.ImportTagsJSON())
		adminApi.Post("/api/v1/admin/oauth", handlers.SaveOAuthConfig())
		adminApi.Post("/api/v1/admin/oauth/:provider/status", handlers.SetSystemProviderStatus())
		adminApi.Post("/api/v1/admin/roles/:role/users", handlers.ChangeUserRole())
		adminApi.Put("/api/v1/admin/users/:userID/block", handlers.BlockUser())
		adminApi.Delete("/api/v1/admin/users/:userID/block", handlers.UnblockUser())
		adminApi.Put("/api/v1/admin/users/:userID/trust", handlers.TrustUser())
		adminApi.Delete("/api/v1/admin/users/:userID/trust", handlers.UntrustUser())
	}

	return r
}
