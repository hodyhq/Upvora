package postgres

import (
	"context"
	"time"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/dbx"
	"github.com/getfider/fider/app/pkg/errors"
	"github.com/getfider/fider/app/pkg/rand"
	"github.com/lib/pq"
)

// Storage for the embedded OAuth 2.1 authorization server. Codes and refresh
// tokens are stored only as SHA-256 hashes; every lookup is tenant-scoped and
// bound to the client.

type dbOAuthClient struct {
	ID             int            `db:"id"`
	ClientID       string         `db:"client_id"`
	Name           string         `db:"name"`
	RedirectURIs   pq.StringArray `db:"redirect_uris"`
	CreatedByAdmin bool           `db:"created_by_admin"`
	CreatedAt      time.Time      `db:"created_at"`
}

func (c *dbOAuthClient) toModel() *entity.OAuthClient {
	return &entity.OAuthClient{
		ID:             c.ID,
		ClientID:       c.ClientID,
		Name:           c.Name,
		RedirectURIs:   []string(c.RedirectURIs),
		CreatedByAdmin: c.CreatedByAdmin,
		CreatedAt:      c.CreatedAt,
	}
}

type dbOAuthGrant struct {
	UserID        int    `db:"user_id"`
	ClientID      string `db:"client_id"`
	Scope         string `db:"scope"`
	RedirectURI   string `db:"redirect_uri"`
	CodeChallenge string `db:"code_challenge"`
	FamilyID      string `db:"family_id"`
}

func (g *dbOAuthGrant) toModel() *entity.OAuthGrant {
	return &entity.OAuthGrant{
		UserID:        g.UserID,
		ClientID:      g.ClientID,
		Scope:         g.Scope,
		RedirectURI:   g.RedirectURI,
		CodeChallenge: g.CodeChallenge,
		FamilyID:      g.FamilyID,
	}
}

func registerOAuthClient(ctx context.Context, c *cmd.RegisterOAuthClient) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		row := dbOAuthClient{}
		err := trx.Get(&row, `
			INSERT INTO oauth_clients (tenant_id, client_id, name, redirect_uris, created_by_admin)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, client_id, name, redirect_uris, created_by_admin, created_at`,
			tenant.ID, rand.String(40), c.Name, pq.Array(c.RedirectURIs), c.CreatedByAdmin)
		if err != nil {
			return errors.Wrap(err, "failed to register OAuth client")
		}
		c.Result = row.toModel()
		return nil
	})
}

func getOAuthClient(ctx context.Context, q *query.GetOAuthClient) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		row := dbOAuthClient{}
		err := trx.Get(&row, `
			SELECT id, client_id, name, redirect_uris, created_by_admin, created_at
			FROM oauth_clients WHERE tenant_id = $1 AND client_id = $2`, tenant.ID, q.ClientID)
		if err != nil {
			return errors.Wrap(err, "failed to get OAuth client")
		}
		q.Result = row.toModel()
		return nil
	})
}

func listOAuthClients(ctx context.Context, q *query.ListOAuthClients) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		rows := []*dbOAuthClient{}
		err := trx.Select(&rows, `
			SELECT id, client_id, name, redirect_uris, created_by_admin, created_at
			FROM oauth_clients WHERE tenant_id = $1 ORDER BY created_at DESC`, tenant.ID)
		if err != nil {
			return errors.Wrap(err, "failed to list OAuth clients")
		}
		q.Result = make([]*entity.OAuthClient, len(rows))
		for i, r := range rows {
			q.Result[i] = r.toModel()
		}
		return nil
	})
}

func deleteOAuthClient(ctx context.Context, c *cmd.DeleteOAuthClient) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		var clientID string
		if err := trx.Scalar(&clientID, "SELECT client_id FROM oauth_clients WHERE tenant_id = $1 AND id = $2", tenant.ID, c.ID); err != nil {
			return errors.Wrap(err, "failed to find OAuth client %d", c.ID)
		}
		for _, sql := range []string{
			"DELETE FROM oauth_codes WHERE tenant_id = $1 AND client_id = $2",
			"DELETE FROM oauth_refresh_tokens WHERE tenant_id = $1 AND client_id = $2",
			"DELETE FROM oauth_clients WHERE tenant_id = $1 AND client_id = $2",
		} {
			if _, err := trx.Execute(sql, tenant.ID, clientID); err != nil {
				return errors.Wrap(err, "failed to delete OAuth client %d", c.ID)
			}
		}
		return nil
	})
}

func saveOAuthCode(ctx context.Context, c *cmd.SaveOAuthCode) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		_, err := trx.Execute(`
			INSERT INTO oauth_codes (code_hash, tenant_id, client_id, user_id, redirect_uri, scope, code_challenge, expires_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			c.CodeHash, tenant.ID, c.ClientID, c.UserID, c.RedirectURI, c.Scope, c.CodeChallenge, c.ExpiresAt)
		if err != nil {
			return errors.Wrap(err, "failed to save OAuth code")
		}
		return nil
	})
}

func consumeOAuthCode(ctx context.Context, c *cmd.ConsumeOAuthCode) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		row := dbOAuthGrant{}
		// One statement: a concurrent replay cannot also see used_at IS NULL.
		err := trx.Get(&row, `
			UPDATE oauth_codes SET used_at = now()
			WHERE code_hash = $1 AND tenant_id = $2 AND client_id = $3 AND used_at IS NULL AND expires_at > now()
			RETURNING user_id, client_id, scope, redirect_uri, code_challenge, '' AS family_id`,
			c.CodeHash, tenant.ID, c.ClientID)
		if err != nil {
			return errors.Wrap(err, "failed to consume OAuth code")
		}
		c.Result = row.toModel()
		return nil
	})
}

func saveOAuthRefreshToken(ctx context.Context, c *cmd.SaveOAuthRefreshToken) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		_, err := trx.Execute(`
			INSERT INTO oauth_refresh_tokens (token_hash, tenant_id, client_id, user_id, scope, family_id, expires_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			c.TokenHash, tenant.ID, c.ClientID, c.UserID, c.Scope, c.FamilyID, c.ExpiresAt)
		if err != nil {
			return errors.Wrap(err, "failed to save OAuth refresh token")
		}
		return nil
	})
}

func rotateOAuthRefreshToken(ctx context.Context, c *cmd.RotateOAuthRefreshToken) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, _ *entity.User) error {
		row := dbOAuthGrant{}
		err := trx.Get(&row, `
			UPDATE oauth_refresh_tokens SET rotated_at = now()
			WHERE token_hash = $1 AND tenant_id = $2 AND client_id = $3
			  AND rotated_at IS NULL AND revoked_at IS NULL AND expires_at > now()
			RETURNING user_id, client_id, scope, '' AS redirect_uri, '' AS code_challenge, family_id`,
			c.OldHash, tenant.ID, c.ClientID)
		if errors.Cause(err) == app.ErrNotFound {
			// A rotated or revoked token was presented: treat it as stolen and
			// revoke its whole family (RFC 9700 refresh token reuse detection).
			if _, rerr := trx.Execute(`
				UPDATE oauth_refresh_tokens SET revoked_at = now()
				WHERE tenant_id = $1 AND revoked_at IS NULL AND family_id = (
					SELECT family_id FROM oauth_refresh_tokens WHERE token_hash = $2 AND tenant_id = $1 AND client_id = $3
				)`, tenant.ID, c.OldHash, c.ClientID); rerr != nil {
				return errors.Wrap(rerr, "failed to revoke OAuth refresh token family")
			}
			return app.ErrNotFound
		}
		if err != nil {
			return errors.Wrap(err, "failed to rotate OAuth refresh token")
		}
		if _, err := trx.Execute(`
			INSERT INTO oauth_refresh_tokens (token_hash, tenant_id, client_id, user_id, scope, family_id, expires_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			c.NewHash, tenant.ID, row.ClientID, row.UserID, row.Scope, row.FamilyID, c.NewExpiresAt); err != nil {
			return errors.Wrap(err, "failed to save rotated OAuth refresh token")
		}
		c.Result = row.toModel()
		return nil
	})
}
