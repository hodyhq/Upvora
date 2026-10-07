package handlers_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/getfider/fider/app/handlers"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/enum"
	"github.com/getfider/fider/app/models/query"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/mock"
)

// The sign-in page remembers where to return (same-site paths only), so a
// magic link can bring the user back to e.g. /oauth2/authorize.
func TestSignInPage_RemembersSafeReturnURL(t *testing.T) {
	RegisterT(t)

	code, res := mock.NewServer().OnTenant(mock.DemoTenant).
		WithURL("http://demo.test.fider.io/signin?redirect=%2Foauth2%2Fauthorize%3Fclient_id%3Dabc").
		Execute(handlers.SignInPage())
	Expect(code).Equals(http.StatusOK)
	Expect(res.Header().Get("Set-Cookie")).ContainsSubstring(handlers.SignInReturnCookie + "=%2Foauth2%2Fauthorize%3Fclient_id%3Dabc")

	// a foreign target is reduced to the root
	_, res = mock.NewServer().OnTenant(mock.DemoTenant).
		WithURL("http://demo.test.fider.io/signin?redirect=https%3A%2F%2Fevil.example%2Fx").
		Execute(handlers.SignInPage())
	Expect(res.Header().Get("Set-Cookie")).ContainsSubstring(handlers.SignInReturnCookie + "=%2F;")
}

func TestSignInPage_SignedInUserGoesStraightBack(t *testing.T) {
	RegisterT(t)
	code, res := mock.NewServer().OnTenant(mock.DemoTenant).AsUser(mock.JonSnow).
		WithURL("http://demo.test.fider.io/signin?redirect=%2Foauth2%2Fauthorize%3Fclient_id%3Dabc").
		Execute(handlers.SignInPage())
	Expect(code).Equals(http.StatusTemporaryRedirect)
	Expect(res.Header().Get("Location")).Equals("http://demo.test.fider.io/oauth2/authorize?client_id=abc")
}

func TestVerifySignInKey_ReturnsToRememberedURL(t *testing.T) {
	RegisterT(t)
	bus.AddHandler(func(ctx context.Context, q *query.GetVerificationByKey) error {
		q.Result = &entity.EmailVerification{Key: q.Key, Kind: q.Kind, ExpiresAt: time.Now().Add(5 * time.Minute), Email: mock.JonSnow.Email}
		return nil
	})
	bus.AddHandler(func(ctx context.Context, q *query.GetUserByEmail) error { q.Result = mock.JonSnow; return nil })
	bus.AddHandler(func(ctx context.Context, c *cmd.SetKeyAsVerified) error { return nil })

	code, res := mock.NewServer().OnTenant(mock.DemoTenant).
		WithURL("http://demo.test.fider.io/signin/verify?k=1234567890").
		AddCookie(handlers.SignInReturnCookie, "%2Foauth2%2Fauthorize%3Fclient_id%3Dabc").
		Execute(handlers.VerifySignInKey(enum.EmailVerificationKindSignIn))
	Expect(code).Equals(http.StatusTemporaryRedirect)
	Expect(res.Header().Get("Location")).Equals("http://demo.test.fider.io/oauth2/authorize?client_id=abc")

	// a tampered cookie pointing elsewhere is ignored
	_, res = mock.NewServer().OnTenant(mock.DemoTenant).
		WithURL("http://demo.test.fider.io/signin/verify?k=1234567890").
		AddCookie(handlers.SignInReturnCookie, "https%3A%2F%2Fevil.example%2F").
		Execute(handlers.VerifySignInKey(enum.EmailVerificationKindSignIn))
	Expect(res.Header().Get("Location")).Equals("http://demo.test.fider.io/")
}
