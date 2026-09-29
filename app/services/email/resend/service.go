package resend

import (
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/env"
)

// endpoint is the Resend "send email" API. Package-level so tests can point it
// at a stub.
var endpoint = "https://api.resend.com/emails"

func init() {
	bus.Register(Service{})
}

type Service struct{}

func (s Service) Name() string {
	return "Resend"
}

func (s Service) Category() string {
	return "email"
}

func (s Service) Enabled() bool {
	return env.Config.Email.Type == "resend"
}

func (s Service) Init() {
	bus.AddListener(sendMail)
}
