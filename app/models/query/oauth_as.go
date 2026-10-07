package query

import "github.com/getfider/fider/app/models/entity"

// GetOAuthClient finds a client of the current tenant by client_id.
type GetOAuthClient struct {
	ClientID string

	Result *entity.OAuthClient
}

// ListOAuthClients lists the current tenant's clients.
type ListOAuthClients struct {
	Result []*entity.OAuthClient
}
