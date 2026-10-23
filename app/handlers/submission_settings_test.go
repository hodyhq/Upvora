package handlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/getfider/fider/app/handlers"
	"github.com/getfider/fider/app/models/cmd"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/mock"
)

func TestUpdateSubmissionSettings_SendsOnlyWhatWasGiven(t *testing.T) {
	RegisterT(t)
	var got *cmd.UpdateTenantSubmissionSettings
	bus.AddHandler(func(ctx context.Context, c *cmd.UpdateTenantSubmissionSettings) error {
		got = c
		return nil
	})
	code, _ := mock.NewServer().OnTenant(mock.DemoTenant).AsUser(mock.JonSnow).
		ExecutePost(handlers.UpdateSubmissionSettings(), `{"voraRequired": true}`)
	Expect(code).Equals(http.StatusOK)
	Expect(*got.VoraRequired).IsTrue()
	Expect(got.FileAttachmentsEnabled).IsNil()

	code, _ = mock.NewServer().OnTenant(mock.DemoTenant).AsUser(mock.JonSnow).
		ExecutePost(handlers.UpdateSubmissionSettings(), `{"fileAttachmentsEnabled": false}`)
	Expect(code).Equals(http.StatusOK)
	Expect(got.VoraRequired).IsNil()
	Expect(*got.FileAttachmentsEnabled).IsFalse()
}

func TestUpdateSubmissionSettings_NothingGiven(t *testing.T) {
	RegisterT(t)
	code, _ := mock.NewServer().OnTenant(mock.DemoTenant).AsUser(mock.JonSnow).
		ExecutePost(handlers.UpdateSubmissionSettings(), `{}`)
	Expect(code).Equals(http.StatusBadRequest)
}
