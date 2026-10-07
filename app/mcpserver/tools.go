package mcpserver

import (
	"strings"

	"github.com/getfider/fider/app/models/enum"
)

// Field helpers keep the table readable.
func pathInt(name, desc string) Field {
	return Field{Name: name, Type: "integer", In: "path", Required: true, Description: desc}
}
func pathStr(name, desc string) Field {
	return Field{Name: name, Type: "string", In: "path", Required: true, Description: desc}
}
func q(name, typ, desc string) Field {
	return Field{Name: name, Type: typ, In: "query", Description: desc}
}
func b(name, typ, desc string) Field {
	return Field{Name: name, Type: typ, In: "body", Description: desc}
}
func req(f Field) Field { f.Required = true; return f }
func arr(name, items, desc string) Field {
	return Field{Name: name, Type: "array", Items: items, In: "body", Description: desc}
}

var (
	number = pathInt("number", "Idea number (the #N shown on the board).")
	member = enum.RoleVisitor
	staff  = enum.RoleCollaborator
	admin  = enum.RoleAdministrator
)

const (
	adminWarn  = " ADMIN action that changes the whole site. Confirm with the user before calling."
	readOnlyNo = " Read-only."
)

// Catalog is every MCP tool. Each replays one /api/v1 route as the caller, so
// that route's own authorization and validation apply.
var Catalog = []Tool{
	// Ideas
	{Name: "upvora_ideas_search", Method: "GET", Path: "/api/v1/posts", MinRole: member, Paged: true,
		Description: "Search and list ideas on the board." + readOnlyNo,
		Fields: []Field{
			q("query", "string", "Full-text search."),
			q("view", "string", "Ordering/filter: all, recent, most-wanted, most-discussed, trending, my-votes, planned, started, completed, declined."),
			q("tags", "string", "Comma-separated tag slugs."),
			q("statuses", "string", "Comma-separated status slugs."),
			q("products", "string", "Comma-separated product ids."),
			q("myvotes", "boolean", "Only ideas I voted for."),
			q("myposts", "boolean", "Only ideas I created."),
			q("limit", "integer", "Max results (default 30, capped at 100)."),
		}},
	{Name: "upvora_ideas_get", Method: "GET", Path: "/api/v1/posts/{number}", MinRole: member,
		Description: "Get one idea with its status, votes and response." + readOnlyNo, Fields: []Field{number}},
	{Name: "upvora_ideas_similar", Method: "GET", Path: "/api/v1/similarposts", MinRole: member,
		Description: "Find up to 5 existing ideas similar to a draft title, to avoid duplicates." + readOnlyNo,
		Fields:      []Field{req(q("query", "string", "Draft title or keywords."))}},
	{Name: "upvora_ideas_create", Method: "POST", Path: "/api/v1/posts", MinRole: member,
		Description: "Create an idea as the user. For a guided idea with an Idea Brief, use upvora_ai_ideation_context then upvora_ai_submit_brief instead.",
		Fields: []Field{
			req(b("title", "string", "10 to 100 characters, unique on the board.")),
			b("description", "string", "Markdown description."),
			arr("tags", "string", "Tag slugs (only if the site allows tags at creation)."),
			b("productId", "integer", "Product id (omit for General)."),
			b("isPrivate", "boolean", "Private idea (staff always; members only if the site allows it)."),
		}},
	{Name: "upvora_ideas_update", Method: "PUT", Path: "/api/v1/posts/{number}", MinRole: member,
		Description: "Edit an idea's title and description (authors within 1 hour; staff any time).",
		Fields:      []Field{number, req(b("title", "string", "10 to 100 characters.")), b("description", "string", "Markdown description.")}},
	{Name: "upvora_ideas_delete", Method: "DELETE", Path: "/api/v1/posts/{number}", MinRole: admin,
		Description: "Delete an idea permanently." + adminWarn,
		Fields:      []Field{number, b("text", "string", "Optional reason shown to subscribers.")}},
	{Name: "upvora_ideas_set_status", Method: "PUT", Path: "/api/v1/posts/{number}/status", MinRole: staff,
		Description: "Set an idea's status and optional public response (staff). Notifies subscribers.",
		Fields: []Field{number,
			req(b("status", "string", "Active status slug (see upvora_statuses_list or the tenant's statuses).")),
			b("text", "string", "Public response text."),
			b("originalNumber", "integer", "Required when status is duplicate: the original idea's number.")}},
	{Name: "upvora_ideas_set_privacy", Method: "POST", Path: "/api/v1/posts/{number}/privacy", MinRole: member,
		Description: "Make an idea private or public. Staff: either way. Members: only their own private idea to public, if allowed.",
		Fields:      []Field{number, req(b("isPrivate", "boolean", "true = private, false = public."))}},
	{Name: "upvora_ideas_assign_product", Method: "PUT", Path: "/api/v1/posts/{number}/product", MinRole: staff,
		Description: "Move an idea to a product (staff).",
		Fields:      []Field{number, req(b("productId", "integer", "Active product id, or 0 for General."))}},
	{Name: "upvora_ideas_set_internal_note", Method: "PUT", Path: "/api/v1/posts/{number}/internal-note", MinRole: staff,
		Description: "Set the team-only internal note on an idea (staff; never shown to members).",
		Fields:      []Field{number, req(b("content", "string", "Markdown, up to 10000 characters."))}},

	// Votes and subscriptions
	{Name: "upvora_votes_add", Method: "POST", Path: "/api/v1/posts/{number}/votes", MinRole: member, Description: "Vote for an idea.", Fields: []Field{number}},
	{Name: "upvora_votes_remove", Method: "DELETE", Path: "/api/v1/posts/{number}/votes", MinRole: member, Description: "Remove my vote from an idea.", Fields: []Field{number}},
	{Name: "upvora_votes_toggle", Method: "POST", Path: "/api/v1/posts/{number}/votes/toggle", MinRole: member, Description: "Toggle my vote on an idea.", Fields: []Field{number}},
	{Name: "upvora_votes_list", Method: "GET", Path: "/api/v1/posts/{number}/votes", MinRole: member, Description: "List who voted for an idea." + readOnlyNo, Fields: []Field{number}},
	{Name: "upvora_subscription_subscribe", Method: "POST", Path: "/api/v1/posts/{number}/subscription", MinRole: member, Description: "Follow an idea (get notifications).", Fields: []Field{number}},
	{Name: "upvora_subscription_unsubscribe", Method: "DELETE", Path: "/api/v1/posts/{number}/subscription", MinRole: member, Description: "Stop following an idea.", Fields: []Field{number}},

	// Comments and reactions
	{Name: "upvora_comments_list", Method: "GET", Path: "/api/v1/posts/{number}/comments", MinRole: member, Description: "List an idea's comments." + readOnlyNo, Fields: []Field{number}},
	{Name: "upvora_comments_get", Method: "GET", Path: "/api/v1/posts/{number}/comments/{id}", MinRole: member, Description: "Get one comment." + readOnlyNo,
		Fields: []Field{number, pathInt("id", "Comment id.")}},
	{Name: "upvora_comments_create", Method: "POST", Path: "/api/v1/posts/{number}/comments", MinRole: member,
		Description: "Comment on an idea as the user.",
		Fields: []Field{number, req(b("content", "string", "Markdown, up to 4000 characters.")),
			b("isInternal", "boolean", "Team-only comment (staff only).")}},
	{Name: "upvora_comments_update", Method: "PUT", Path: "/api/v1/posts/{number}/comments/{id}", MinRole: member,
		Description: "Edit a comment (its author or staff).",
		Fields:      []Field{number, pathInt("id", "Comment id."), req(b("content", "string", "Markdown, up to 4000 characters."))}},
	{Name: "upvora_comments_delete", Method: "DELETE", Path: "/api/v1/posts/{number}/comments/{id}", MinRole: member,
		Description: "Delete a comment (its author or staff).", Fields: []Field{number, pathInt("id", "Comment id.")}},
	{Name: "upvora_reactions_toggle", Method: "POST", Path: "/api/v1/posts/{number}/comments/{id}/reactions/{reaction}", MinRole: member,
		Description: "Toggle a reaction on a comment.",
		Fields:      []Field{number, pathInt("id", "Comment id."), pathStr("reaction", "One of: 👍 👎 😄 🎉 😕 ❤️ 🚀 👀")}},

	// Notifications
	{Name: "upvora_notifications_list", Method: "GET", Path: "/api/v1/notifications", MinRole: member, Description: "List my notifications." + readOnlyNo},
	{Name: "upvora_notifications_mark_all_read", Method: "POST", Path: "/api/v1/notifications/read-all", MinRole: member, Description: "Mark all my notifications as read."},

	// Tags
	{Name: "upvora_tags_list", Method: "GET", Path: "/api/v1/tags", MinRole: member, Description: "List the board's tags." + readOnlyNo},
	{Name: "upvora_tags_create", Method: "POST", Path: "/api/v1/tags", MinRole: staff, Description: "Create a tag (staff).",
		Fields: []Field{req(b("name", "string", "Up to 30 characters.")), req(b("color", "string", "6 hex digits without #, e.g. 3B82F6.")), b("isPublic", "boolean", "Visible to members.")}},
	{Name: "upvora_tags_delete", Method: "DELETE", Path: "/api/v1/tags/{slug}", MinRole: staff, Description: "Delete a tag from the board (staff).", Fields: []Field{pathStr("slug", "Tag slug.")}},
	{Name: "upvora_tags_assign", Method: "POST", Path: "/api/v1/posts/{number}/tags/{slug}", MinRole: staff, Description: "Add a tag to an idea (staff).",
		Fields: []Field{number, pathStr("slug", "Tag slug.")}},
	{Name: "upvora_tags_unassign", Method: "DELETE", Path: "/api/v1/posts/{number}/tags/{slug}", MinRole: staff, Description: "Remove a tag from an idea (staff).",
		Fields: []Field{number, pathStr("slug", "Tag slug.")}},
	{Name: "upvora_tags_import", Method: "POST", Path: "/api/v1/admin/import/tags", MinRole: admin,
		Description: "Import tags in bulk (admin). Existing names are skipped.",
		Fields:      []Field{{Name: "tags", Type: "array", Items: "object", In: "raw", Required: true, Description: "Array of {name, color (6 hex, no #), isPublic}."}}},

	// Products
	{Name: "upvora_products_list", Method: "GET", Path: "/api/v1/admin/products", MinRole: admin, Description: "List products with their settings (admin). Members get products from upvora_ai_ideation_context." + readOnlyNo},
	{Name: "upvora_products_create", Method: "POST", Path: "/api/v1/admin/products", MinRole: admin, Description: "Create a product (admin).",
		Fields: []Field{req(b("name", "string", "Up to 60 characters.")), req(b("slug", "string", "Lowercase words joined by hyphens, unique.")),
			b("description", "string", ""), b("color", "string", "#RGB or #RRGGBB, empty for the brand color."), b("sortOrder", "integer", "")}},
	{Name: "upvora_products_update", Method: "PUT", Path: "/api/v1/admin/products/{id}", MinRole: admin,
		Description: "Update a product (admin). Send every field; a missing isActive means inactive.",
		Fields: []Field{pathInt("id", "Product id."), req(b("name", "string", "")), b("description", "string", ""), b("color", "string", ""),
			b("sortOrder", "integer", ""), b("isActive", "boolean", "")}},
	{Name: "upvora_products_delete", Method: "DELETE", Path: "/api/v1/admin/products/{id}", MinRole: admin, Description: "Delete a product (admin).", Fields: []Field{pathInt("id", "Product id.")}},

	// Statuses
	{Name: "upvora_statuses_list", Method: "GET", Path: "/api/v1/admin/statuses", MinRole: admin, Description: "List all statuses with settings (admin)." + readOnlyNo},
	{Name: "upvora_statuses_create", Method: "POST", Path: "/api/v1/admin/statuses", MinRole: admin, Description: "Create a custom status (admin).",
		Fields: []Field{req(b("slug", "string", "Lowercase, hyphens.")), req(b("label", "string", "")),
			req(b("kind", "string", "open, active, closed-completed, closed-declined or duplicate.")),
			b("color", "string", "blue, green, yellow, red or gray."), b("icon", "string", ""), b("showOnHome", "boolean", ""),
			b("showOnRoadmap", "boolean", ""), b("filterable", "boolean", ""), b("sortOrder", "integer", "")}},
	{Name: "upvora_statuses_update", Method: "PUT", Path: "/api/v1/admin/statuses/{id}", MinRole: admin,
		Description: "Update a status (admin). Slug and kind are fixed. Send every field; missing booleans mean false.",
		Fields: []Field{pathInt("id", "Status id."), req(b("label", "string", "")), b("color", "string", ""), b("icon", "string", ""),
			b("showOnHome", "boolean", ""), b("showOnRoadmap", "boolean", ""), b("filterable", "boolean", ""), b("sortOrder", "integer", ""), b("isActive", "boolean", "")}},
	{Name: "upvora_statuses_delete", Method: "DELETE", Path: "/api/v1/admin/statuses/{id}", MinRole: admin, Description: "Delete a custom status (admin).", Fields: []Field{pathInt("id", "Status id.")}},

	// Scorecard
	{Name: "upvora_scorecard_get", Method: "GET", Path: "/api/v1/scorecards/{id}", MinRole: staff, Description: "Get a scorecard (staff)." + readOnlyNo, Fields: []Field{pathInt("id", "Scorecard id.")}},
	{Name: "upvora_scorecard_create", Method: "POST", Path: "/api/v1/scorecards", MinRole: staff, Description: "Create a scorecard, optionally for an idea (staff).",
		Fields: []Field{b("postId", "integer", "Idea id (not number)."), b("title", "string", "Defaults to the idea's title.")}},
	{Name: "upvora_scorecard_update", Method: "PUT", Path: "/api/v1/scorecards/{id}", MinRole: staff, Description: "Update a scorecard's title and field values (staff).",
		Fields: []Field{pathInt("id", "Scorecard id."), req(b("title", "string", "")), b("values", "object", "Field key to value.")}},
	{Name: "upvora_scorecard_delete", Method: "DELETE", Path: "/api/v1/scorecards/{id}", MinRole: staff, Description: "Delete a scorecard (staff).", Fields: []Field{pathInt("id", "Scorecard id.")}},
	{Name: "upvora_scorecard_fields_list", Method: "GET", Path: "/api/v1/admin/scorecard-fields", MinRole: staff, Description: "List active scorecard fields." + readOnlyNo},
	{Name: "upvora_scorecard_fields_create", Method: "POST", Path: "/api/v1/admin/scorecard-fields", MinRole: staff, Description: "Add a scorecard field (staff).",
		Fields: []Field{req(b("key", "string", "Lowercase letters, digits, underscores.")), req(b("label", "string", "")),
			req(b("groupKey", "string", "header, intake, context, workflow, ownership, classification, scoring or decision.")),
			req(b("type", "string", "text, multiline, note, date, number, url, choice, score (scoring group only) or user.")),
			b("choices", "array", "Required for type choice."), b("weight", "integer", "0 to 100."), b("question", "string", ""), b("sortOrder", "integer", "")}},
	{Name: "upvora_scorecard_fields_update", Method: "PUT", Path: "/api/v1/admin/scorecard-fields/{id}", MinRole: staff, Description: "Update a scorecard field (staff).",
		Fields: []Field{pathInt("id", "Field id."), req(b("label", "string", "")), b("choices", "array", ""), b("weight", "integer", ""),
			b("question", "string", ""), b("sortOrder", "integer", ""), b("isActive", "boolean", "")}},
	{Name: "upvora_scorecard_fields_delete", Method: "DELETE", Path: "/api/v1/admin/scorecard-fields/{id}", MinRole: staff, Description: "Delete a scorecard field (staff).", Fields: []Field{pathInt("id", "Field id.")}},
	{Name: "upvora_scorecard_settings_get", Method: "GET", Path: "/api/v1/admin/scorecard-settings", MinRole: staff, Description: "Get scorecard fields (all) and usage; bands are on upvora_tenant_get." + readOnlyNo},
	{Name: "upvora_scorecard_settings_set", Method: "POST", Path: "/api/v1/admin/scorecard-settings", MinRole: staff,
		Description: "Set scorecard settings (staff). Bands must be strictly descending.",
		Fields: []Field{b("isEnabled", "boolean", ""), b("bandStrong", "integer", ""), b("bandGood", "integer", ""), b("bandRefine", "integer", ""), b("bandLow", "integer", ""),
			b("bandStrongLabel", "string", ""), b("bandGoodLabel", "string", ""), b("bandRefineLabel", "string", ""), b("bandLowLabel", "string", ""),
			b("bandNoneLabel", "string", ""), b("triggerStatusSlug", "string", "Status that auto-creates a scorecard; empty for none.")}},

	// Vora / AI
	{Name: "upvora_ai_ideation_context", Method: "GET", Path: "/api/v1/ai/ideation-context", MinRole: member,
		Description: "Step 1 of submitting an idea: the products, this product's interview guidance (Vora's instructions), the Idea Brief structure, tags and privacy rules. Follow the guidance with the user, then call upvora_ai_submit_brief." + readOnlyNo,
		Fields:      []Field{q("product", "integer", "Product id; omit for General.")}},
	{Name: "upvora_ai_submit_brief", Method: "POST", Path: "/api/v1/posts", MinRole: member,
		Description: "Step 2: submit the idea with its Idea Brief, exactly as Vora would. Show the user the title and brief first.",
		Fields: []Field{req(b("title", "string", "10 to 100 characters.")), b("description", "string", "Short summary."),
			req(b("briefMarkdown", "string", "The Idea Brief, following the structure from upvora_ai_ideation_context.")),
			arr("tags", "string", "Tag slugs."), b("productId", "integer", ""), b("isPrivate", "boolean", ""),
			arr("voraTranscript", "object", "Optional interview transcript: [{role: user|assistant, content}], up to 60 messages.")}},
	{Name: "upvora_ai_ideate", OpenWorld: true, Method: "POST", Path: "/api/v1/ai/ideate", MinRole: member,
		Description: "Delegate one interview turn to the site's own Vora agent (uses the site's AI provider).",
		Fields:      []Field{b("productId", "integer", ""), {Name: "messages", Type: "array", Items: "object", In: "body", Required: true, Description: "[{role: user|assistant, content}], 1 to 60."}}},
	{Name: "upvora_ai_finalize", OpenWorld: true, Method: "POST", Path: "/api/v1/ai/finalize", MinRole: member,
		Description: "Ask the site's Vora agent to turn a conversation into a title, description, brief and tags (does not submit).",
		Fields:      []Field{b("productId", "integer", ""), {Name: "messages", Type: "array", Items: "object", In: "body", Required: true, Description: "[{role, content}]."}}},
	{Name: "upvora_ai_get_brief", Method: "GET", Path: "/api/v1/posts/{number}/brief", MinRole: member, Description: "Get an idea's Idea Brief." + readOnlyNo, Fields: []Field{number}},
	{Name: "upvora_ai_settings_get", Method: "GET", Path: "/api/v1/admin/settings/ai", MinRole: admin, Description: "Get AI (Vora) settings and agents; keys are reported only as present or not." + readOnlyNo},
	{Name: "upvora_ai_settings_set", Destructive: true, Method: "POST", Path: "/api/v1/admin/settings/ai", MinRole: admin,
		Description: "Set AI (Vora) provider settings." + adminWarn,
		Fields: []Field{b("enabled", "boolean", ""), b("provider", "string", "claude, openai or custom."), b("model", "string", ""),
			b("apiKey", "string", "Empty keeps the stored key."), b("customBaseUrl", "string", ""), b("customModel", "string", ""),
			b("webSearchEnabled", "boolean", ""), b("webSearchProvider", "string", "serper or searxng."), b("webSearchApiKey", "string", ""), b("webSearchBaseUrl", "string", "")}},
	{Name: "upvora_ai_agents_list", Method: "GET", Path: "/api/v1/admin/settings/ai", MinRole: admin, Description: "List Vora agents per product with their instructions (admin)." + readOnlyNo},
	{Name: "upvora_ai_agents_set", Method: "POST", Path: "/api/v1/admin/ai/agents", MinRole: admin, Description: "Create or update the Vora agent for a product (admin).",
		Fields: []Field{b("productId", "integer", "Omit for the default agent."), b("description", "string", ""), b("instructions", "string", "Up to 5000 characters."),
			b("enabled", "boolean", ""), b("webSearchEnabled", "boolean", "")}},

	// Members and users
	{Name: "upvora_users_list", Method: "GET", Path: "/api/v1/users", MinRole: staff, Paged: true, Description: "List users with roles (staff)." + readOnlyNo,
		Fields: []Field{q("query", "string", "Name or email contains."), q("roles", "string", "Comma-separated: visitor, collaborator, administrator."),
			q("page", "integer", ""), q("limit", "integer", "Up to 100.")}},
	{Name: "upvora_users_invite", OpenWorld: true, Method: "POST", Path: "/api/v1/invitations/send", MinRole: staff,
		Description: "Email invitations to join the site (staff). Confirm the recipient list with the user first.",
		Fields: []Field{req(b("subject", "string", "Up to 70 characters.")), req(b("message", "string", "Must contain the invitation link placeholder %invite%.")),
			{Name: "recipients", Type: "array", Items: "string", In: "body", Required: true, Description: "1 to 30 email addresses."}}},
	{Name: "upvora_users_invite_sample", OpenWorld: true, Method: "POST", Path: "/api/v1/invitations/sample", MinRole: staff, Description: "Send a sample invitation to myself (staff).",
		Fields: []Field{req(b("subject", "string", "")), req(b("message", "string", ""))}},
	{Name: "upvora_users_block", Destructive: true, Method: "PUT", Path: "/api/v1/admin/users/{userID}/block", MinRole: admin, Description: "Block a user (admin).", Fields: []Field{pathInt("userID", "User id.")}},
	{Name: "upvora_users_unblock", Method: "DELETE", Path: "/api/v1/admin/users/{userID}/block", MinRole: admin, Description: "Unblock a user (admin).", Fields: []Field{pathInt("userID", "User id.")}},
	{Name: "upvora_users_trust", Method: "PUT", Path: "/api/v1/admin/users/{userID}/trust", MinRole: admin, Description: "Mark a user as trusted (admin).", Fields: []Field{pathInt("userID", "User id.")}},
	{Name: "upvora_users_untrust", Method: "DELETE", Path: "/api/v1/admin/users/{userID}/trust", MinRole: admin, Description: "Remove trusted status (admin).", Fields: []Field{pathInt("userID", "User id.")}},
	{Name: "upvora_users_set_role", Destructive: true, Method: "POST", Path: "/api/v1/admin/roles/{role}/users", MinRole: admin,
		Description: "Change a user's role (visitor, collaborator, administrator)." + adminWarn,
		Fields:      []Field{pathStr("role", "visitor, collaborator or administrator."), req(b("userID", "integer", "User id."))}},
	{Name: "upvora_users_taggable", Method: "GET", Path: "/api/v1/taggable-users", MinRole: member, Description: "List users who can be @mentioned." + readOnlyNo},

	// Webhooks
	{Name: "upvora_webhooks_list", Method: "GET", Path: "/api/v1/admin/webhooks", MinRole: admin, Description: "List webhooks (admin)." + readOnlyNo},
	{Name: "upvora_webhooks_create", Destructive: true, OpenWorld: true, Method: "POST", Path: "/api/v1/admin/webhooks", MinRole: admin, Description: "Create a webhook that sends site data to an external URL." + adminWarn, Fields: webhookFields(false)},
	{Name: "upvora_webhooks_update", Destructive: true, OpenWorld: true, Method: "PUT", Path: "/api/v1/admin/webhooks/{id}", MinRole: admin, Description: "Update a webhook." + adminWarn, Fields: webhookFields(true)},
	{Name: "upvora_webhooks_delete", Method: "DELETE", Path: "/api/v1/admin/webhooks/{id}", MinRole: admin, Description: "Delete a webhook (admin).", Fields: []Field{pathInt("id", "Webhook id.")}},
	{Name: "upvora_webhooks_preview", Method: "POST", Path: "/api/v1/admin/webhooks/preview", MinRole: admin, Description: "Preview a webhook's rendered URL and content (admin).",
		Fields: []Field{req(b("type", "string", "new_post, new_comment, change_status or delete_post.")), b("url", "string", ""), b("content", "string", "")}},
	{Name: "upvora_webhooks_test", OpenWorld: true, Method: "POST", Path: "/api/v1/admin/webhooks/test/{id}", MinRole: admin, Description: "Send a test call for a webhook (admin).", Fields: []Field{pathInt("id", "Webhook id.")}},

	// OAuth providers and settings
	{Name: "upvora_oauth_providers_list", Method: "GET", Path: "/api/v1/admin/oauth", MinRole: admin, Description: "List sign-in providers (admin)." + readOnlyNo},
	{Name: "upvora_oauth_providers_set", Destructive: true, Method: "POST", Path: "/api/v1/admin/oauth", MinRole: admin,
		Description: "Add or edit a custom sign-in (OAuth) provider. A wrong setting can lock people out." + adminWarn,
		Fields: []Field{b("provider", "string", "Omit to create; existing key to edit."), req(b("status", "integer", "1 disabled, 2 enabled.")),
			req(b("displayName", "string", "")), req(b("clientID", "string", "")), b("clientSecret", "string", "Required on create; empty keeps the stored secret."),
			req(b("authorizeURL", "string", "")), req(b("tokenURL", "string", "")), req(b("scope", "string", "")), b("profileURL", "string", ""),
			b("isTrusted", "boolean", ""), req(b("jsonUserIDPath", "string", "")), b("jsonUserNamePath", "string", ""), b("jsonUserEmailPath", "string", ""),
			b("jsonUserRolesPath", "string", ""), b("allowedRoles", "string", "")}},
	{Name: "upvora_oauth_providers_set_status", Destructive: true, Method: "POST", Path: "/api/v1/admin/oauth/{provider}/status", MinRole: admin,
		Description: "Enable or disable a sign-in provider (admin). Refused if it would leave no way to sign in.",
		Fields:      []Field{{Name: "provider", Type: "string", In: "both", Required: true, Description: "Provider key."}, req(b("isEnabled", "boolean", ""))}},
	{Name: "upvora_settings_general_get", Method: "GET", Path: "/api/v1/tenant", MinRole: admin, Description: "Get the site's settings (name, welcome text, locale, theme, privacy, MCP)." + readOnlyNo},
	{Name: "upvora_settings_general_set", Destructive: true, Method: "POST", Path: "/api/v1/admin/settings/general", MinRole: admin,
		Description: "Set general settings (admin). Replaces every field: read with upvora_settings_general_get first.", Fields: generalFields()},
	{Name: "upvora_settings_theme_get", Method: "GET", Path: "/api/v1/tenant", MinRole: admin, Description: "Get theme settings (themePrimary, themeAccents, defaultTheme on the site)." + readOnlyNo},
	{Name: "upvora_settings_theme_set", Method: "POST", Path: "/api/v1/admin/settings/theme", MinRole: admin, Description: "Set theme colors (admin).",
		Fields: []Field{b("primary", "string", "#RGB or #RRGGBB, empty for the built-in brand."), b("accents", "object", "Keys buttons, votes, links, header to hex or empty."),
			b("defaultTheme", "string", "light, dark or system.")}},
	{Name: "upvora_settings_advanced_get", Method: "GET", Path: "/api/v1/admin/settings/advanced", MinRole: admin, Description: "Get custom CSS and allowed link schemes." + readOnlyNo},
	{Name: "upvora_settings_advanced_set", Destructive: true, Method: "POST", Path: "/api/v1/admin/settings/advanced", MinRole: admin, Description: "Set custom CSS and allowed link schemes (admin).",
		Fields: []Field{b("customCSS", "string", ""), b("allowedSchemes", "string", "")}},
	{Name: "upvora_settings_privacy_get", Method: "GET", Path: "/api/v1/tenant", MinRole: admin, Description: "Get privacy settings (isPrivate, isFeedEnabled, isModerationEnabled, membersPrivateIdeas, membersCanPublishPrivate)." + readOnlyNo},
	{Name: "upvora_settings_privacy_set", Destructive: true, Method: "POST", Path: "/api/v1/admin/settings/privacy", MinRole: admin,
		Description: "Set privacy settings, e.g. make the whole board private." + adminWarn + " Send every field.",
		Fields: []Field{b("isPrivate", "boolean", ""), b("isFeedEnabled", "boolean", "Cannot be true while private."), b("isModerationEnabled", "boolean", ""),
			b("membersPrivateIdeas", "boolean", ""), b("membersCanPublishPrivate", "boolean", "")}},
	{Name: "upvora_settings_emailauth_get", Method: "GET", Path: "/api/v1/tenant", MinRole: admin, Description: "Get whether email sign-in is allowed (isEmailAuthAllowed)." + readOnlyNo},
	{Name: "upvora_settings_emailauth_set", Destructive: true, Method: "POST", Path: "/api/v1/admin/settings/emailauth", MinRole: admin,
		Description: "Allow or disallow email sign-in (admin). Needs another active sign-in provider.", Fields: []Field{req(b("isEmailAuthAllowed", "boolean", ""))}},
	{Name: "upvora_settings_banner_set", Method: "POST", Path: "/api/v1/admin/settings/site-banner", MinRole: admin, Description: "Set the site-wide banner (admin).",
		Fields: []Field{b("enabled", "boolean", ""), b("message", "string", "Up to 500 characters."), b("variant", "string", "info, success, warning, danger or brand.")}},
	{Name: "upvora_settings_mcp_get", Method: "GET", Path: "/api/v1/tenant", MinRole: admin, Description: "Get MCP settings (mcpEnabled, mcpMinRole, mcpDcrEnabled)." + readOnlyNo},
	{Name: "upvora_settings_mcp_set", Destructive: true, Method: "POST", Path: "/api/v1/admin/settings/mcp", MinRole: admin,
		Description: "Set who may use MCP. Turning it off disconnects every client, including this one." + adminWarn,
		Fields:      []Field{req(b("enabled", "boolean", "")), req(b("minRole", "string", "visitor, collaborator or administrator.")), b("dcrEnabled", "boolean", "")}},

	// Billing
	{Name: "upvora_billing_get_subscription", Method: "GET", Path: "/api/v1/admin/billing", MinRole: admin, Billing: true, Description: "Get the site's subscription state (admin)." + readOnlyNo},
	{Name: "upvora_billing_create_checkout", Method: "POST", Path: "/api/v1/admin/billing/checkout", MinRole: admin, Billing: true, Description: "Create a checkout link for the user to open in a browser (admin)."},
	{Name: "upvora_billing_create_annual_checkout", Method: "POST", Path: "/api/v1/admin/billing/checkout/annual", MinRole: admin, Billing: true, Description: "Create an annual-plan checkout link (admin)."},
	{Name: "upvora_billing_create_portal", Method: "POST", Path: "/api/v1/admin/billing/portal", MinRole: admin, Billing: true, Description: "Create a billing portal link (admin)."},

	// System and tenant
	{Name: "upvora_system_status", Method: "GET", Path: "/api/v1/admin/system/status", MinRole: admin, Description: "Installed version and latest available release (admin)." + readOnlyNo},
	{Name: "upvora_system_update", Destructive: true, Method: "POST", Path: "/api/v1/admin/system/update", MinRole: admin,
		Description: "Update Upvora to the latest release now; the site restarts." + adminWarn},
	{Name: "upvora_tenant_get", Method: "GET", Path: "/api/v1/tenant", MinRole: member, Description: "Get the site's public settings, statuses and products." + readOnlyNo},
	{Name: "upvora_tenant_update", Destructive: true, Method: "POST", Path: "/api/v1/admin/settings/general", MinRole: admin,
		Description: "Update the site's name and general settings (same as upvora_settings_general_set; replaces every field).", Fields: generalFields()},
	{Name: "upvora_tenant_cancel_deletion", Method: "POST", Path: "/api/v1/admin/tenant/cancel-deletion", MinRole: admin, MultiTenant: true, Description: "Cancel a scheduled site deletion (owner)."},
	{Name: "upvora_tenant_delete", Method: "DELETE", Path: "/api/v1/admin/tenant", MinRole: admin, MultiTenant: true,
		Description: "Schedule deletion of the ENTIRE site and all its data (owner only, one-hour grace)." + adminWarn},

	// Account self
	{Name: "upvora_account_get_settings", Method: "GET", Path: "/api/v1/user/settings", MinRole: member, Description: "Get my name, avatar type and notification settings." + readOnlyNo},
	{Name: "upvora_account_update_settings", Method: "POST", Path: "/api/v1/user/settings", MinRole: member,
		Description: "Update my name, avatar type and notification settings (send every field; read them first).",
		Fields: []Field{req(b("name", "string", "Up to 50 characters.")), req(b("avatarType", "string", "letter or gravatar.")),
			b("settings", "object", "event_notification_new_post, _new_comment, _mention, _change_status: \"0\" to \"3\" (1 web, 2 email).")}},
	{Name: "upvora_account_change_email", Method: "POST", Path: "/api/v1/user/change-email", MinRole: member,
		Description: "Start changing my email: a confirmation link is sent to the new address and must be opened while signed in.",
		Fields:      []Field{req(b("email", "string", "New email address."))}},
}

func webhookFields(withID bool) []Field {
	fields := []Field{
		req(b("name", "string", "Up to 60 characters.")),
		req(b("type", "string", "new_post, new_comment, change_status or delete_post.")),
		req(b("status", "string", "enabled or disabled.")),
		req(b("url", "string", "Target URL (Go template allowed).")),
		b("content", "string", "Request body template."),
		req(b("http_method", "string", "e.g. POST.")),
		b("http_headers", "object", "Header name to value."),
	}
	if withID {
		return append([]Field{pathInt("id", "Webhook id.")}, fields...)
	}
	return fields
}

func generalFields() []Field {
	return []Field{
		req(b("title", "string", "Site name, up to 60 characters.")), b("invitation", "string", ""), b("welcomeMessage", "string", ""),
		b("welcomeHeader", "string", ""), b("descriptionTemplate", "string", ""), b("shareIdeaInstructions", "string", ""),
		b("railCtaHeading", "string", ""), b("railCtaText", "string", ""), b("railCtaButton", "string", ""),
		b("defaultTheme", "string", "light, dark or system."), req(b("locale", "string", "e.g. en.")), b("cname", "string", ""),
	}
}

// Every destructive tool asks the client to confirm with the user, whether or
// not its description already says so.
func init() {
	for i := range Catalog {
		if Catalog[i].IsDestructive() && !strings.Contains(Catalog[i].Description, "Confirm with the user") {
			Catalog[i].Description += " Confirm with the user before calling."
		}
	}
}
