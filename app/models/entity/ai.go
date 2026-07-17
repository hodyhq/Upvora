package entity

import "time"

// AIAgent is Vora's per-product configuration. The agent's name is always
// "Vora" by design — admins tune description, instructions and the switch.
type AIAgent struct {
	ID           int    `json:"id"`
	ProductID    *int   `json:"productId"`
	Description  string `json:"description"`
	Instructions string `json:"instructions"`
	Enabled      bool   `json:"enabled"`
}

// IdeaBrief is the markdown document Vora writes with the submitter. The
// content stores the submitter's email only as the {{submitter_email}} token;
// the real address is substituted server-side on admin download.
type IdeaBrief struct {
	PostID          int       `json:"postId"`
	Content         string    `json:"content"`
	SubmitterUserID *int      `json:"-"`
	CreatedAt       time.Time `json:"createdAt"`
}

// AIMessage is one turn of a Vora conversation, relayed to the provider.
type AIMessage struct {
	Role    string `json:"role"` // "user" or "assistant"
	Content string `json:"content"`
}
