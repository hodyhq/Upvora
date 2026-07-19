package cmd

import "github.com/getfider/fider/app/models/entity"

// SetAISettings stores the tenant's provider configuration. An empty APIKey
// means "keep the stored key" — the key is write-only from the client's view.
type SetAISettings struct {
	Enabled       bool
	Provider      string // "claude" | "openai" | "custom"
	APIKey        string
	Model         string // provider-specific selector, e.g. "haiku", "sonnet-5", "luna", "terra"
	CustomBaseURL string
	CustomModel   string

	// Web search
	WebSearchEnabled  bool
	WebSearchProvider string
	WebSearchAPIKey   string // empty = keep stored
	WebSearchBaseURL  string
}

// UpsertAIAgent creates or updates the agent config for a product
// (ProductID nil = the default agent, used for General).
type UpsertAIAgent struct {
	ProductID        *int
	Description      string
	Instructions     string
	Enabled          bool
	WebSearchEnabled bool
}

// SaveIdeaBrief attaches Vora's brief to a post. Content must already carry
// the {{submitter_email}} token instead of a real address.
type SaveIdeaBrief struct {
	PostID          int
	Content         string
	Transcript      string // JSON []entity.AIMessage, already email-scrubbed
	SubmitterUserID int
}

// AIChatCompletion calls the tenant's configured provider with a system
// prompt and conversation, returning the assistant's reply.
type AIChatCompletion struct {
	System    string
	Messages  []entity.AIMessage
	MaxTokens int

	Result string
}

// AIWebSearch runs a web search through the tenant's configured provider
// (Serper.dev or a self-hosted SearXNG). Results are used only as untrusted
// grounding context for Vora — never executed or acted on.
type AIWebSearch struct {
	Query string

	Result []AIWebSearchResult
}

type AIWebSearchResult struct {
	Title   string
	URL     string
	Snippet string
}
