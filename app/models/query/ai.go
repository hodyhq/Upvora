package query

import "github.com/getfider/fider/app/models/entity"

// ListAIAgents returns every agent config for the tenant (default + per product).
type ListAIAgents struct {
	Result []*entity.AIAgent
}

// GetAIAgentForProduct resolves the agent for a product, falling back to the
// tenant default (product_id NULL) when the product has no agent of its own.
// Result is nil when nothing applies or the resolved agent is disabled.
type GetAIAgentForProduct struct {
	ProductID int // 0 = General

	Result *entity.AIAgent
}

// GetIdeaBrief loads the brief attached to a post, if any.
type GetIdeaBrief struct {
	PostID int

	Result *entity.IdeaBrief
}
