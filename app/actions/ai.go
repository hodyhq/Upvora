package actions

import (
	"context"
	"strings"

	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/pkg/validate"
)

var aiProviders = map[string]bool{"claude": true, "openai": true, "custom": true}
var aiClaudeModels = map[string]bool{"haiku": true, "sonnet-5": true}
var aiOpenAIModels = map[string]bool{"luna": true, "terra": true}

// UpdateAISettings is the admin action behind the AI settings tab. An empty
// APIKey keeps the stored key (the key is write-only from the client's view).
type UpdateAISettings struct {
	Enabled       bool   `json:"enabled"`
	Provider      string `json:"provider"`
	APIKey        string `json:"apiKey"`
	Model         string `json:"model"`
	CustomBaseURL string `json:"customBaseUrl"`
	CustomModel   string `json:"customModel"`
}

func (a *UpdateAISettings) IsAuthorized(ctx context.Context, user *entity.User) bool {
	return user != nil && user.IsAdministrator()
}

func (a *UpdateAISettings) Validate(ctx context.Context, user *entity.User) *validate.Result {
	result := validate.Success()
	if a.Provider == "" && !a.Enabled {
		return result // fully off is always valid
	}
	if !aiProviders[a.Provider] {
		result.AddFieldFailure("provider", "Provider must be claude, openai, or custom.")
		return result
	}
	switch a.Provider {
	case "claude":
		if !aiClaudeModels[a.Model] {
			result.AddFieldFailure("model", "Claude model must be haiku or sonnet-5.")
		}
	case "openai":
		if !aiOpenAIModels[a.Model] {
			result.AddFieldFailure("model", "OpenAI model must be luna or terra.")
		}
	case "custom":
		if !strings.HasPrefix(a.CustomBaseURL, "https://") {
			result.AddFieldFailure("customBaseUrl", "Custom base URL must start with https://")
		}
		if strings.TrimSpace(a.CustomModel) == "" {
			result.AddFieldFailure("customModel", "Custom model name is required.")
		}
	}
	if len(a.APIKey) > 500 {
		result.AddFieldFailure("apiKey", "That doesn't look like an API key.")
	}
	return result
}

// UpsertAIAgentAction configures Vora for one product (or the default).
type UpsertAIAgentAction struct {
	ProductID    *int   `json:"productId"`
	Description  string `json:"description"`
	Instructions string `json:"instructions"`
	Enabled      bool   `json:"enabled"`
}

func (a *UpsertAIAgentAction) IsAuthorized(ctx context.Context, user *entity.User) bool {
	return user != nil && user.IsAdministrator()
}

func (a *UpsertAIAgentAction) Validate(ctx context.Context, user *entity.User) *validate.Result {
	result := validate.Success()
	if len(a.Description) > 500 {
		result.AddFieldFailure("description", "Description must be under 500 characters.")
	}
	if len(a.Instructions) > 5000 {
		result.AddFieldFailure("instructions", "Instructions must be under 5000 characters.")
	}
	return result
}

// AIConverse is one turn (or the finalize call) of a Vora session.
type AIConverse struct {
	ProductID int                `json:"productId"`
	Messages  []entity.AIMessage `json:"messages"`
}

func (a *AIConverse) IsAuthorized(ctx context.Context, user *entity.User) bool {
	return user != nil // any signed-in member
}

func (a *AIConverse) Validate(ctx context.Context, user *entity.User) *validate.Result {
	result := validate.Success()
	if len(a.Messages) == 0 {
		result.AddFieldFailure("messages", "Say something first.")
	}
	if len(a.Messages) > 60 {
		result.AddFieldFailure("messages", "This conversation is too long — wrap it up or start fresh.")
	}
	for _, m := range a.Messages {
		if m.Role != "user" && m.Role != "assistant" {
			result.AddFieldFailure("messages", "Invalid message role.")
			break
		}
		if len(m.Content) > 6000 {
			result.AddFieldFailure("messages", "Messages must be under 6000 characters.")
			break
		}
	}
	return result
}
