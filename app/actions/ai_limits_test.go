package actions_test

import (
	"context"
	"strings"
	"testing"

	"github.com/getfider/fider/app/actions"
	"github.com/getfider/fider/app/models/entity"
	. "github.com/getfider/fider/app/pkg/assert"
)

// The whole conversation is capped, not just each message (LLM cost bound).
func TestAIConverse_TotalSizeCapped(t *testing.T) {
	RegisterT(t)
	msgs := []entity.AIMessage{}
	for i := 0; i < 12; i++ {
		msgs = append(msgs, entity.AIMessage{Role: "user", Content: strings.Repeat("x", 5500)}) // 66k in total
	}
	r := (&actions.AIConverse{Messages: msgs}).Validate(context.Background(), &entity.User{})
	ExpectFailed(r, "messages")

	ok := (&actions.AIConverse{Messages: msgs[:10]}).Validate(context.Background(), &entity.User{}) // 55k
	Expect(ok.Ok).IsTrue()
}
