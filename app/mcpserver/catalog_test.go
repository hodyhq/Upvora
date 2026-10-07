package mcpserver_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/getfider/fider/app/mcpserver"
	. "github.com/getfider/fider/app/pkg/assert"
)

// specTools is the tool list from the design spec (Pillar 3), minus
// upvora_account_regenerate_api_key (key rotation is UI-only).
var specTools = strings.Fields(`
upvora_ideas_search upvora_ideas_get upvora_ideas_create upvora_ideas_update upvora_ideas_delete upvora_ideas_set_status
upvora_ideas_set_privacy upvora_ideas_assign_product upvora_ideas_set_internal_note upvora_ideas_similar
upvora_votes_add upvora_votes_remove upvora_votes_toggle upvora_votes_list upvora_subscription_subscribe upvora_subscription_unsubscribe
upvora_comments_list upvora_comments_get upvora_comments_create upvora_comments_update upvora_comments_delete upvora_reactions_toggle
upvora_notifications_list upvora_notifications_mark_all_read
upvora_tags_list upvora_tags_create upvora_tags_delete upvora_tags_assign upvora_tags_unassign upvora_tags_import
upvora_products_list upvora_products_create upvora_products_update upvora_products_delete
upvora_statuses_list upvora_statuses_create upvora_statuses_update upvora_statuses_delete
upvora_scorecard_get upvora_scorecard_create upvora_scorecard_update upvora_scorecard_delete upvora_scorecard_fields_list
upvora_scorecard_fields_create upvora_scorecard_fields_update upvora_scorecard_fields_delete upvora_scorecard_settings_get upvora_scorecard_settings_set
upvora_ai_ideation_context upvora_ai_submit_brief upvora_ai_ideate upvora_ai_finalize upvora_ai_get_brief upvora_ai_settings_get
upvora_ai_settings_set upvora_ai_agents_list upvora_ai_agents_set
upvora_users_list upvora_users_invite upvora_users_invite_sample upvora_users_block upvora_users_unblock upvora_users_trust
upvora_users_untrust upvora_users_set_role upvora_users_taggable
upvora_webhooks_list upvora_webhooks_create upvora_webhooks_update upvora_webhooks_delete upvora_webhooks_preview upvora_webhooks_test
upvora_oauth_providers_list upvora_oauth_providers_set upvora_oauth_providers_set_status upvora_settings_general_get upvora_settings_general_set
upvora_settings_theme_get upvora_settings_theme_set upvora_settings_advanced_get upvora_settings_advanced_set upvora_settings_privacy_get
upvora_settings_privacy_set upvora_settings_emailauth_get upvora_settings_emailauth_set upvora_settings_banner_set upvora_settings_mcp_get upvora_settings_mcp_set
upvora_billing_get_subscription upvora_billing_create_checkout upvora_billing_create_annual_checkout upvora_billing_create_portal
upvora_system_status upvora_system_update upvora_tenant_get upvora_tenant_update upvora_tenant_cancel_deletion upvora_tenant_delete
upvora_account_get_settings upvora_account_update_settings upvora_account_change_email
`)

func TestCatalogMatchesSpec(t *testing.T) {
	RegisterT(t)
	names := []string{}
	seen := map[string]bool{}
	for _, tool := range mcpserver.Catalog {
		Expect(seen[tool.Name]).IsFalse() // no duplicates
		seen[tool.Name] = true
		names = append(names, tool.Name)
		Expect(tool.Description != "").IsTrue()
		Expect(strings.HasPrefix(tool.Path, "/api/v1/")).IsTrue()
	}
	want := append([]string{}, specTools...)
	sort.Strings(names)
	sort.Strings(want)
	Expect(names).Equals(want)
}

// High-impact tools must say so plainly.
func TestCatalogHighImpactWording(t *testing.T) {
	RegisterT(t)
	for _, tool := range mcpserver.Catalog {
		switch tool.Name {
		case "upvora_system_update", "upvora_tenant_delete", "upvora_users_set_role", "upvora_settings_privacy_set",
			"upvora_webhooks_create", "upvora_webhooks_update", "upvora_oauth_providers_set", "upvora_ideas_delete":
			Expect(strings.Contains(tool.Description, "ADMIN")).IsTrue()
			Expect(strings.Contains(tool.Description, "Confirm with the user")).IsTrue()
		}
	}
}

// Clients use annotations to decide which calls need the user's OK.
func TestCatalogAnnotations(t *testing.T) {
	RegisterT(t)
	destructiveNonDelete := map[string]bool{
		"upvora_users_set_role": true, "upvora_users_block": true, "upvora_oauth_providers_set": true,
		"upvora_oauth_providers_set_status": true, "upvora_settings_privacy_set": true, "upvora_settings_mcp_set": true,
		"upvora_settings_advanced_set": true, "upvora_settings_emailauth_set": true, "upvora_settings_general_set": true,
		"upvora_tenant_update": true, "upvora_system_update": true, "upvora_webhooks_create": true, "upvora_webhooks_update": true,
		"upvora_ai_settings_set": true,
	}
	openWorld := map[string]bool{
		"upvora_webhooks_create": true, "upvora_webhooks_update": true, "upvora_webhooks_test": true,
		"upvora_users_invite": true, "upvora_users_invite_sample": true, "upvora_ai_ideate": true, "upvora_ai_finalize": true,
	}
	for _, tool := range mcpserver.Catalog {
		a := tool.Annotations()
		if tool.Method == "GET" {
			Expect(a.ReadOnlyHint).IsTrue()
			continue
		}
		Expect(a.ReadOnlyHint).IsFalse()
		destructive := tool.Method == "DELETE" || destructiveNonDelete[tool.Name]
		Expect(a.DestructiveHint != nil && *a.DestructiveHint == destructive).IsTrue()
		if destructive {
			Expect(strings.Contains(tool.Description, "Confirm with the user")).IsTrue()
		}
		Expect(a.OpenWorldHint != nil && *a.OpenWorldHint == openWorld[tool.Name]).IsTrue()
	}
}
